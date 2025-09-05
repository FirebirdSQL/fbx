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
	"crypto/rc4"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/nakagami/chacha20"
	"golang.org/x/exp/slices"
	//"unsafe"
)

type Client struct {
	Encoding       *Encoding
	reader         io.Reader
	writer         io.Writer
	plugin         string
	rc4reader      *rc4.Cipher
	rc4writer      *rc4.Cipher
	chacha20reader *chacha20.Cipher
	chacha20writer *chacha20.Cipher
	xdrReader      *XDRReader
	xdrWriter      *XDRWriter
}

func NewClient(r io.Reader, w io.Writer, encoding *Encoding) *Client {
	c := Client{}
	c.reader = r
	c.writer = w
	c.xdrReader = NewXDRReader(c.reader)
	c.xdrWriter = NewXDRWriter(c.writer)
	c.Encoding = encoding
	return &c
}

func (c *Client) setCryptKey(plugin string, sessionKey []byte, nonce []byte) (err error) {
	c.plugin = plugin
	if slices.Contains([]string{"ChaCha64", "ChaCha"}, plugin) {
		digest := sha256.New()
		digest.Write(sessionKey)
		key := digest.Sum(nil)
		c.chacha20reader, err = chacha20.NewCipher(key, nonce, 0)
		c.chacha20writer, err = chacha20.NewCipher(key, nonce, 0)
	} else if plugin == "Arc4" {
		c.rc4reader, err = rc4.NewCipher(sessionKey)
		c.rc4writer, err = rc4.NewCipher(sessionKey)
	} else {
		err = errors.New(fmt.Sprintf("Unknown wire encrypto plugin name:%s", plugin))
	}

	return
}

func (c *Client) Read(buf []byte) (n int, err error) {
	if c.plugin != "" {
		src := make([]byte, len(buf))
		n, err = c.reader.Read(src)
		if slices.Contains([]string{"ChaCha64", "ChaCha"}, c.plugin) {
			c.chacha20reader.XORKeyStream(buf, src[0:n])
		} else if c.plugin == "Arc4" {
			c.rc4reader.XORKeyStream(buf, src[0:n])
		}
		return
	}
	return c.reader.Read(buf)
}

func (c *Client) Write(buf []byte) (n int, err error) {
	if c.plugin != "" {
		dst := make([]byte, len(buf))
		if slices.Contains([]string{"ChaCha64", "ChaCha"}, c.plugin) {
			c.chacha20writer.XORKeyStream(dst, buf)
		} else if c.plugin == "Arc4" {
			c.rc4writer.XORKeyStream(dst, buf)
		}
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
	return msg.Write(c.xdrWriter)
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
	case op_dummy:
		return &OpDummy{}, nil
	case op_cond_accept:
		msg = &OpCondAccept{Encoding: c.Encoding}
	case op_accept_data:
		msg = &OpDataAccept{OpCondAccept{Encoding: c.Encoding}}
	case op_response:
		msg = &OpResponse{Encoding: c.Encoding}
	case op_reject:
		msg = &OpReject{}
	case op_fetch_response:
		msg = &OpFetchResponse{}
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
				return msg, fmt.Errorf(m.Message)
			}
		case *OpReject:
			return msg, fmt.Errorf("server rejected connection")
		}
		break
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
		return nil, fmt.Errorf("wrong response message type %v instead expected OpResponse", m)
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
		return nil, fmt.Errorf("wrong response message type %v instead expected OpFetchResponse", m)
	}
}

func (c *Client) ReceiveSQLData(row []FieldDescriptor) (*SQLData, error) {
	sqlData := SQLData{}
	if err := sqlData.Read(c.xdrReader, row); err != nil {
		return nil, err
	}
	return &sqlData, nil
}
