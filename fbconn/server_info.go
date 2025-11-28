package fbconn

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/red-soft/fbx/fbproto"
	"github.com/red-soft/fbx/fbproto/vax"
)

type ServerInfoBufferReader struct {
	buf *bytes.Buffer
}

func NewServerInfoBufferReader(buf []byte) *ServerInfoBufferReader {
	return &ServerInfoBufferReader{bytes.NewBuffer(buf)}
}

func (r *ServerInfoBufferReader) ReadByte() (byte, error) {
	return r.buf.ReadByte()
}

func (r *ServerInfoBufferReader) ReadInt32ValueNoPrefix() (int, error) {
	if r.buf.Len() < 2 {
		return 0, io.ErrUnexpectedEOF
	}
	res := vax.Integer2(r.buf.Bytes(), 0)
	r.buf.Next(2)
	return res, nil
}

func (r *ServerInfoBufferReader) ReadInt32Value() (int, error) {
	l, err := r.ReadInt32ValueNoPrefix()
	if err != nil {
		return 0, err
	}
	if r.buf.Len() < l {
		return 0, io.ErrUnexpectedEOF
	}
	val := vax.Integer(r.buf.Bytes(), 0, l)
	r.buf.Next(l)
	return val, nil
}

func (r *ServerInfoBufferReader) ReadStringValue() (string, error) {
	if r.buf.Len() < 2 {
		return "", io.ErrUnexpectedEOF
	}
	l, err := r.ReadInt32ValueNoPrefix()
	if err != nil {
		return "", err
	}
	if r.buf.Len() < l {
		return "", io.ErrUnexpectedEOF
	}
	return string(r.buf.Next(l)), nil
}

func (r *ServerInfoBufferReader) ReadInt64Value() (int64, error) {
	if r.buf.Len() < 2 {
		return 0, io.ErrUnexpectedEOF
	}

	length, err := r.ReadInt32ValueNoPrefix()
	if err != nil {
		return 0, err
	}

	if length == 0 {
		return 0, nil
	}
	if length > 8 {
		return 0, fmt.Errorf("invalid int64 length: %d", length)
	}
	if r.buf.Len() < length {
		return 0, io.ErrUnexpectedEOF
	}

	value := vax.Long(r.buf.Bytes(), 0, length)
	r.buf.Next(length)

	return value, nil
}

func (fbConn *FbConn) getServerInfo(statementHandle int, itemsBuilder func(int) []byte, itemsParser func([]byte) error) error {
	var err error
	var continueIndex int
	infoSize := 10240
	for {
		if err = fbConn.client.WriteMessage(&fbproto.OpInfoSQL{
			StatementHandle:           statementHandle,
			StatementInformationItems: itemsBuilder(continueIndex),
			MaxBufferLength:           infoSize,
		}); err != nil {
			return err
		}
		if err = fbConn.writer.Flush(); err != nil {
			return err
		}

		msg, err := fbConn.receiveMessage()
		if err != nil {
			fbConn.asyncClose()
			return err
		}
		var respData []byte
		switch m := msg.(type) {
		case *fbproto.OpResponse:
			respData = m.Data
		case *fbproto.ErrorResponse:
			return ErrorResponseToFbError(m)
		default:
			return fmt.Errorf("received unexpected message: %v", m)
		}

		err = itemsParser(respData)
		if err != nil {
			truncError := InfoTruncationError{}
			if errors.As(err, &truncError) {
				fmt.Printf("truncation error: %v\n", truncError)
				continueIndex = truncError.Index
				infoSize *= 2
			} else {
				return fmt.Errorf("failed to parse server info: %v", err)
			}
		} else {
			break
		}
	}
	return nil
}
