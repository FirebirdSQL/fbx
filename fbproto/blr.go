package fbproto

import (
	"fmt"

	"rdb.red-soft.ru/fbx/internal/fbio"
)

// BLRWriter builds BLR buffer.
type BLRWriter struct {
	buf []byte
}

// NewBLRWriter creates a new BLRWriter.
func NewBLRWriter(capacity int) *BLRWriter {
	return &BLRWriter{buf: make([]byte, 0, capacity)}
}

// PutByte appends 1 byte to BLRWriter.
func (blr *BLRWriter) PutByte(val byte) {
	blr.buf = append(blr.buf, val)
}

// PutTag appends 1 byte to BLRWriter.
func (blr *BLRWriter) PutTag(tag byte) {
	blr.PutByte(tag)
}

// PutInt16 appends int16 to BLRWriter in little-endian format.
func (blr *BLRWriter) PutInt16(val int16) {
	blr.buf = fbio.AppendInt16LE(blr.buf, val)
}

// PutInt32 appends int32 to BLRWriter in little-endian format.
func (blr *BLRWriter) PutInt32(val int32) {
	blr.buf = fbio.AppendInt32LE(blr.buf, val)
}

// PutInt64 appends int64 to BLRWriter in little-endian format.
func (blr *BLRWriter) PutInt64(val int64) {
	blr.buf = fbio.AppendInt64LE(blr.buf, val)
}

// PutBytes appends []byte buffer to BLRWriter.
func (blr *BLRWriter) PutBytes(bytes []byte) {
	blr.buf = append(blr.buf, bytes...)
}

// PutString appends length of string and string to BLRWriter.
// The string length is stored in 1 byte, which is why it cannot
// be greater than 255, otherwise an error will be returned.
func (blr *BLRWriter) PutString(val string) error {
	strBytes := []byte(val)
	strLen := len(strBytes)
	if strLen >= 256 {
		return fmt.Errorf("buffer more then expected")
	}
	blr.PutByte(byte(strLen))
	blr.PutBytes(strBytes)
	return nil
}

// PutTagWithByte appends tag and 1 byte after it to BLRWriter.
func (blr *BLRWriter) PutTagWithByte(tag byte, b byte) {
	blr.PutByte(tag)
	blr.PutByte(b)
}

// Bytes returns BLR buffer.
func (blr *BLRWriter) Bytes() []byte {
	return blr.buf
}

// PutType adds type information to the BLRWriter, taking
// the metadata of the specified type as input. Returns
// an error if the specified type is not defined.
func (blr *BLRWriter) PutType(sqltype, subtype, scale, length int) error {
	switch sqltype {
	case SQL_TYPE_VARYING:
		charsetID := byte(subtype & 0xFF)
		collationID := byte((subtype >> 8) & 0xFF)

		blr.PutTag(Isc_blr_varying2)
		blr.PutByte(charsetID)
		blr.PutByte(collationID)
		blr.PutInt16(int16(length))
	case SQL_TYPE_TEXT:
		charsetID := byte(subtype & 0xFF)
		collationID := byte((subtype >> 8) & 0xFF)

		blr.PutTag(Isc_blr_text2)
		blr.PutByte(charsetID)
		blr.PutByte(collationID)
		blr.PutInt16(int16(length))
	case SQL_TYPE_LONG:
		blr.PutTagWithByte(Isc_blr_long, byte(scale))
	case SQL_TYPE_SHORT:
		blr.PutTagWithByte(Isc_blr_short, byte(scale))
	case SQL_TYPE_INT64:
		blr.PutTagWithByte(Isc_blr_int64, byte(scale))
	case SQL_TYPE_INT128:
		blr.PutTagWithByte(Isc_blr_int128, byte(scale))
	case SQL_TYPE_QUAD:
		blr.PutTagWithByte(Isc_blr_quad, byte(scale))
	case SQL_TYPE_DEC_FIXED: // OBSOLETED
		blr.PutTagWithByte(Isc_blr_int128, byte(scale))
	case SQL_TYPE_DOUBLE:
		blr.PutTag(Isc_blr_double)
	case SQL_TYPE_FLOAT:
		blr.PutTag(Isc_blr_float)
	case SQL_TYPE_D_FLOAT:
		blr.PutTag(Isc_blr_d_float)
	case SQL_TYPE_DATE:
		blr.PutTag(Isc_blr_sql_date)
	case SQL_TYPE_TIME:
		blr.PutTag(Isc_blr_sql_time)
	case SQL_TYPE_TIMESTAMP:
		blr.PutTag(Isc_blr_timestamp)
	case SQL_TYPE_BLOB:
		blr.PutTagWithByte(Isc_blr_quad, 0)
	case SQL_TYPE_ARRAY:
		blr.PutTagWithByte(Isc_blr_quad, 0)
	case SQL_TYPE_BOOLEAN:
		blr.PutTag(Isc_blr_bool)
	case SQL_TYPE_DEC64:
		blr.PutTag(Isc_blr_dec64)
	case SQL_TYPE_DEC128:
		blr.PutTag(Isc_blr_dec128)
	case SQL_TYPE_TIME_TZ:
		blr.PutTag(Isc_blr_sql_time_tz)
	case SQL_TYPE_TIMESTAMP_TZ:
		blr.PutTag(Isc_blr_timestamp_tz)
	case SQL_TIME_TZ_EX:
		blr.PutTag(Isc_blr_ex_time_tz)
	case SQL_TIMESTAMP_TZ_EX:
		blr.PutTag(Isc_blr_ex_timestamp_tz)
	case SQL_TYPE_NULL:
		blr.PutTag(Isc_blr_text)
		blr.PutByte(0)
		blr.PutByte(0)
	default:
		return fmt.Errorf("unknown SQL type %d", sqltype)
	}
	return nil
}

// CalculateBLR generates a BLR buffer from []FieldDescriptor and a set of values.
// If values is not nil, then its length and the length of descriptors must be equal,
// otherwise the method will return an error.
func CalculateBLR(descriptors []FieldDescriptor, values [][]byte) ([]byte, error) {
	if values != nil && len(values) != len(descriptors) {
		return nil, fmt.Errorf(
			"BLR values count mismatch: got %d, want %d",
			len(values),
			len(descriptors),
		)
	}

	parametrsNumber := len(descriptors) * 2
	bufferCapacity := (parametrsNumber * 4) + 8

	blr := NewBLRWriter(bufferCapacity)
	blr.PutTag(Isc_blr_version5)
	blr.PutTag(Isc_blr_begin)
	blr.PutTag(Isc_blr_message)
	blr.PutByte(0) // message number
	blr.PutInt16(int16(parametrsNumber))

	for i, x := range descriptors {
		sqlscale := x.Scale
		if sqlscale < 0 {
			sqlscale += 256
		}
		dataLength := x.Length
		if values != nil {
			if values[i] != nil {
				dataLength = len(values[i])
			} else {
				dataLength = 0
			}
		}
		err := blr.PutType(x.Type, x.Subtype, sqlscale, dataLength)
		if err != nil {
			return nil, err
		}
		blr.PutTagWithByte(Isc_blr_short, 0)
	}
	blr.PutTag(Isc_blr_end)
	blr.PutTag(Isc_blr_eoc)

	return blr.Bytes(), nil
}
