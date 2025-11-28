package fbproto

import (
	"bytes"
	"fmt"
	"io"
)

type Client struct {
	Encoding    *Encoding
	reader      io.Reader
	writer      io.Writer
	plugin      string
	codec       StreamCodec
	xdrReader   *XDRReader
	xdrWriter   *XDRWriter
	tracer      *tracer
	lazyCounter int
}

// StreamCodec is a minimal interface for symmetric wire encryption.
// Implementations are provided by higher-level code to avoid package cycles.
type StreamCodec interface {
	Encrypt(dst, src []byte)
	Decrypt(dst, src []byte)
}

func NewClient(r io.Reader, w io.Writer, encoding *Encoding) *Client {
	c := Client{}
	c.reader = r
	c.writer = w
	c.xdrReader = NewXDRReader(&c)
	c.xdrWriter = NewXDRWriter(&c)
	c.Encoding = encoding
	return &c
}

// SetCodec sets the active wire encryption codec.
func (c *Client) SetCodec(plugin string, codec StreamCodec) {
	c.plugin = plugin
	c.codec = codec
}

func (c *Client) Read(buf []byte) (n int, err error) {
	if c.codec != nil {
		src := make([]byte, len(buf))
		n, err = c.reader.Read(src)
		if n > 0 {
			c.codec.Decrypt(buf, src[:n])
		}
		return
	}
	return c.reader.Read(buf)
}

func (c *Client) Write(buf []byte) (n int, err error) {
	if c.codec != nil {
		dst := make([]byte, len(buf))
		c.codec.Encrypt(dst, buf)
		written := 0
		for written < len(buf) {
			n, err = c.writer.Write(dst[written:])
			if err != nil {
				return
			}
			written += n
		}
		n = written
	} else {
		n, err = c.writer.Write(buf)
	}
	return
}

func (c *Client) WriteMessage(msg ClientMessage) error {
	err := msg.Write(c.xdrWriter)
	if err != nil {
		return err
	}

	switch msg.(type) {
	//FIXME: fbx should be implicit or explicit?
	case *OpFreeStatement:
		c.lazyCounter++
	}

	if c.tracer != nil {
		c.tracer.traceRequest('C', msg)
	}
	return nil
}

func (c *Client) ReadPackets(n int) ([]byte, error) {
	buf := make([]byte, n)
	var err error
	read := 0
	totalRead := 0

	padding := n % 4
	if padding > 0 {
		padding = 4 - padding
	}
	paddedLen := n + padding

	for totalRead < paddedLen {
		read, err = c.Read(buf[totalRead:paddedLen])
		if err != nil {
			return buf, err
		}
		totalRead += read
	}
	return buf, nil
}

func (c *Client) ReadMessage() (ServerMessage, error) {
	opcode, err := c.xdrReader.ReadInt32()
	if err != nil {
		return nil, err
	}
	var msg ServerMessage
	switch opcode {
	case Op_dummy:
		return &OpDummy{}, nil
	case Op_cond_accept:
		msg = &OpCondAccept{Encoding: c.Encoding}
	case Op_accept_data:
		msg = &OpDataAccept{OpCondAccept{Encoding: c.Encoding}}
	case Op_response:
		msg = &OpResponse{Encoding: c.Encoding}
	case Op_reject:
		msg = &OpReject{}
	case Op_fetch_response:
		msg = &OpFetchResponse{}
	case Op_sql_response:
		msg = &OpSqlResponse{}
	case Op_cont_auth:
		msg = &OpContAuth{Encoding: c.Encoding}
	default:
		return nil, fmt.Errorf("unknown opcode: %d", opcode)
	}
	err = msg.Read(c.xdrReader)
	return msg, err
}

func (c *Client) ReceiveMessage() (ServerMessage, error) {
	var msg ServerMessage
	var err error
	for {
		msg, err = c.ReadMessage()
		if err != nil {
			return nil, err
		}

		if c.tracer != nil {
			c.tracer.traceResponse('S', msg)
		}

		switch m := msg.(type) {
		case *OpDummy:
			continue
		case *OpResponse:
			if c.lazyCounter == 0 {
				if m.Failure {
					return &ErrorResponse{Object: m.Object, BlobId: m.BlobId, GdsCodes: m.GdsCodes, SqlErr: m.SqlErr, SqlState: m.SqlState, Message: m.Message, Line: m.Line, Position: m.Position}, nil
				}
			} else {
				if m.Failure {
					//FIXME: fbx need some callback for lazy response error?
					fmt.Printf("Ignoring lazy response with error: %s", m.Message)
				}
				c.lazyCounter--
				continue
			}
		case *OpReject:
			return msg, fmt.Errorf("server rejected connection")
		default:
			if c.lazyCounter > 0 {
				panic(fmt.Sprintf("unexpected response of type %T but expected lazy op_response", m))
			}
		}
		break
	}

	return msg, nil
}

func (c *Client) ReceiveSQLData(row []FieldDescriptor) (*SQLData, ServerMessage, error) {
	sqlData, err := ReadRowData(c.xdrReader, row)
	if err != nil {
		return nil, nil, err
	}

	if c.tracer != nil {
		c.tracer.traceRowData('S', sqlData, len(row))
	}

	msg, err := c.ReceiveMessage()
	if err != nil {
		return nil, nil, err
	}
	return sqlData, msg, nil
}

func (c *Client) Trace(w io.Writer, options TracerOptions) {
	c.tracer = &tracer{
		w:             w,
		buf:           &bytes.Buffer{},
		TracerOptions: options,
	}
}

func (c *Client) Untrace() {
	c.tracer = nil
}
