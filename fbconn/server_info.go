package fbconn

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/utils"
	"rdb.red-soft.ru/fbx/fbproto/vax"
)

type ServerInfoBufferReader struct {
	buf *bytes.Buffer
}

type InfoTruncationError struct {
	Index int
}

func (s InfoTruncationError) Error() string {
	if s.Index > 0 {
		return fmt.Sprintf("info truncated at field with index %d", s.Index)
	}
	return fmt.Sprintf("info truncated")
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

func (fbConn *FbConn) getInfoStatement(statementHandle int, itemsParser func([]byte) error) error {
	var continueIndex int
	infoSize := 512 * 1024
	msgBuilder := func() fbproto.ClientMessage {
		return &fbproto.OpInfoSQL{
			StatementHandle:           statementHandle,
			StatementInformationItems: GetSelectDescribeInfoItemsContinue(continueIndex),
			MaxBufferLength:           infoSize,
		}
	}
	truncationHandler := func(truncationError InfoTruncationError) {
		continueIndex = truncationError.Index
		infoSize *= 2
	}
	return fbConn.getServerInfo(msgBuilder, truncationHandler, itemsParser)
}

func (fbConn *FbConn) getInfoSQlRecords(statementHandle int, itemsParser func([]byte) error) error {
	infoSize := 64
	msgBuilder := func() fbproto.ClientMessage {
		return &fbproto.OpInfoSQL{
			StatementHandle:           statementHandle,
			StatementInformationItems: []byte{fbproto.Isc_info_sql_records},
			MaxBufferLength:           infoSize,
		}
	}
	truncationHandler := func(truncationError InfoTruncationError) {
		infoSize *= 2
	}
	return fbConn.getServerInfo(msgBuilder, truncationHandler, itemsParser)
}

func (fbConn *FbConn) getServerInfo(msgBuilder func() fbproto.ClientMessage, truncationHandler func(err InfoTruncationError), itemsParser func([]byte) error) error {
	var err error
	for {
		if err = fbConn.withWriteLockAndFlush(func() error {
			return fbConn.client.WriteMessage(msgBuilder())
		}); err != nil {
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
				truncationHandler(truncError)
			} else {
				return fmt.Errorf("failed to parse server info: %v", err)
			}
		} else {
			break
		}
	}
	return nil
}

func GetSelectDescribeInfoItemsContinue(index int) []byte {
	return append(append([]byte{fbproto.Isc_info_sql_sqlda_start, 2}, utils.Int16_to_bytes(int16(index))...), GetSelectDescribeInfoItems()...)
}

func GetSelectDescribeInfoItems() []byte {
	return []byte{
		fbproto.Isc_info_sql_stmt_type,
		fbproto.Isc_info_sql_select,
		fbproto.Isc_info_sql_describe_vars,
		fbproto.Isc_info_sql_sqlda_seq,
		fbproto.Isc_info_sql_type,
		fbproto.Isc_info_sql_sub_type,
		fbproto.Isc_info_sql_scale,
		fbproto.Isc_info_sql_length,
		fbproto.Isc_info_sql_null_ind,
		fbproto.Isc_info_sql_field,
		fbproto.Isc_info_sql_relation,
		fbproto.Isc_info_sql_owner,
		fbproto.Isc_info_sql_alias,
		fbproto.Isc_info_sql_describe_end,

		fbproto.Isc_info_sql_bind,
		fbproto.Isc_info_sql_describe_vars,
		fbproto.Isc_info_sql_sqlda_seq,
		fbproto.Isc_info_sql_type,
		fbproto.Isc_info_sql_sub_type,
		fbproto.Isc_info_sql_scale,
		fbproto.Isc_info_sql_length,
		fbproto.Isc_info_sql_describe_end,
	}
}
