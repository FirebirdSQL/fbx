package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-row-data

import (
	"fmt"
)

type FieldDescriptor struct {
	Index             int
	Type              int
	Scale             int
	Subtype           int
	Length            int
	NullOk            bool
	Name              string
	TableAliasName    string
	RelName           string
	OwnerName         string
	OriginalName      string
	OriginalTableName string
	Nullable          bool
}

// GetIOLength returns physical length of field by its type
// If length is 0 - field length should be obtained from stream
// If length < 0 - field data used as is without padding
// If length > 0 - padding bytes must be obtained after field data (note +1 to length below to distinguish from 0, it should be subtracted later)
func (f *FieldDescriptor) GetIOLength() int {
	switch f.Type {
	case SQL_TYPE_TEXT:
		return f.Length + 1 // +1 to distinguish from 0
	case SQL_TYPE_VARYING:
		return 0
	case SQL_TYPE_SHORT, SQL_TYPE_LONG, SQL_TYPE_FLOAT, SQL_TYPE_TIME, SQL_TYPE_DATE:
		return -4
	case SQL_TYPE_DOUBLE, SQL_TYPE_TIMESTAMP, SQL_TYPE_BLOB, SQL_TYPE_ARRAY, SQL_TYPE_QUAD, SQL_TYPE_INT64, SQL_TYPE_DEC64, SQL_TYPE_TIME_TZ:
		return -8
	case SQL_TYPE_TIMESTAMP_TZ, SQL_TIME_TZ_EX:
		return -12
	case SQL_TIMESTAMP_TZ_EX, SQL_TYPE_DEC128, SQL_TYPE_INT128:
		return -16
	case SQL_TYPE_NULL:
		return 0
	case SQL_TYPE_BOOLEAN:
		return 1 + 1 // +1 to distinguish from 0
	default:
		panic(fmt.Sprintf("can not get IO length of unknown field descriptor type: %d", f.Type))
	}
}

// CalculateIoLength returns physical length of field by its type and data
// See GetIOLength
func (f *FieldDescriptor) CalculateIoLength(fieldData []byte) int {
	t := f.Type
	if t == SQL_TYPE_TEXT {
		return len(fieldData) + 1
	}
	return f.GetIOLength()
}

func (f *FieldDescriptor) GetPaddingByte() byte {
	switch f.Type {
	case SQL_TYPE_TEXT, SQL_TYPE_VARYING:
		{
			if f.GetCharacterSetId() == CS_BINARY {
				return 0x00
			}
			return 0x20
		}
	default:
		return 0x00
	}
}

func (f *FieldDescriptor) GetCharacterSetId() int {
	switch f.Type {
	case SQL_TYPE_TEXT, SQL_TYPE_VARYING:
		return f.Subtype & 0xFF
	case SQL_TYPE_BLOB:
		if f.Subtype == BLOB_SUB_TYPE_TEXT {
			return f.Scale & 0xFF
		}
		return CS_BINARY
	default:
		return CS_dynamic
	}
}
