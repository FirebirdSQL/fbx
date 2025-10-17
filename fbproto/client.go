/*******************************************************************************
The MIT License (MIT)

Copyright (c) 2013-2025 Hajime Nakagami

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*******************************************************************************/

package fbproto

import (
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
	if err == nil {
		switch msg.(type) {
		//FIXME: fbx should be implicit or explicit?
		case *OpFreeStatement:
			c.lazyCounter++
		}
	}
	return err
	//var buf []byte
	//buf, err := msg.Pack(buf)
	//if err != nil {
	//	return err
	//}
	//_, err = c.Write(buf)
	//return err
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
		switch m := msg.(type) {
		case *OpDummy:
			continue
		case *OpResponse:
			if m.Failure {
				if c.lazyCounter == 0 {
					return &ErrorResponse{Object: m.Object, BlobId: m.BlobId, GdsCodes: m.GdsCodes, SqlCode: m.SqlCode, SqlState: m.SqlState, Message: m.Message}, nil
				} else {
					//FIXME: fbx need some callback for lazy response error?
					fmt.Printf("Ignoring lazy response: %s", m.Message)
				}
			}
		case *OpReject:
			return msg, fmt.Errorf("server rejected connection")
		}
		if c.lazyCounter == 0 {
			break
		}
		c.lazyCounter--
	}
	return msg, nil
}

func (c *Client) ReceiveResponse() (*OpResponse, error) {
	msg, err := c.ReceiveMessage()
	if err != nil {
		return nil, err
	}
	switch m := msg.(type) {
	case *OpResponse:
		return m, nil
	default:
		return nil, fmt.Errorf("wrong response message type %T instead expected OpResponse", m)
	}
}

func (c *Client) ReceiveFetchResponse() (*OpFetchResponse, error) {
	msg, err := c.ReceiveMessage()
	if err != nil {
		return nil, err
	}
	switch m := msg.(type) {
	case *OpFetchResponse:
		return m, nil
	default:
		return nil, fmt.Errorf("wrong response message type %T instead expected OpFetchResponse", m)
	}
}

func (c *Client) ReceiveSQLData(row []FieldDescriptor) (*SQLData, *OpFetchResponse, error) {
	var sqlData *SQLData
	var err error
	if sqlData, err = ReadRowData(c.xdrReader, row); err != nil {
		return nil, nil, err
	}
	if fetchResp, err := c.ReceiveFetchResponse(); err != nil {
		return nil, nil, err
	} else {
		return sqlData, fetchResp, nil
	}
}
