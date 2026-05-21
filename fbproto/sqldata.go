package fbproto

import (
	"fmt"
)

type SQLData struct {
	Nulls         *BitSet
	RowData       [][]byte
	RowDescriptor []FieldDescriptor
}

type rowScratch struct {
	buf   []byte
	pos   int
	data  [][]byte
	nulls []byte
}

const rowScratchMinBufSize = 256

func (s *rowScratch) reset(cols int, nullsLen int) {
	s.pos = 0
	if cap(s.data) < cols {
		s.data = make([][]byte, cols)
	} else {
		s.data = s.data[:cols]
		for i := range s.data {
			s.data[i] = nil
		}
	}
	if cap(s.nulls) < nullsLen {
		s.nulls = make([]byte, nullsLen)
	} else {
		s.nulls = s.nulls[:nullsLen]
	}
	if cap(s.buf) < rowScratchMinBufSize {
		s.buf = make([]byte, 0, rowScratchMinBufSize)
	} else if len(s.buf) != 0 {
		s.buf = s.buf[:0]
	}
}

func (s *rowScratch) take(i int, n int) []byte {
	if n == 0 {
		empty := s.buf[s.pos:s.pos:s.pos]
		s.data[i] = empty
		return empty
	}
	start := s.pos
	need := s.pos + n
	if cap(s.buf) < need {
		newCap := cap(s.buf) * 2
		if newCap < need {
			newCap = need
		}
		newBuf := make([]byte, need, newCap)
		copy(newBuf, s.buf[:start])
		s.buf = newBuf
	} else if len(s.buf) < need {
		s.buf = s.buf[:need]
	}
	dst := s.buf[start:need:need]
	s.pos = need
	s.data[i] = dst
	return dst
}

// ReadRowData - read fields null-bitset and data in 13+ protocol version
// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-row-data-format
func ReadRowData(x *XDRReader, rowDescriptor []FieldDescriptor, scratch *rowScratch) (*SQLData, error) {
	var err error
	count := len(rowDescriptor)
	l := int(float64(count+7) / 8)
	pad := (4 - l) & 3
	if scratch == nil {
		scratch = &rowScratch{}
	}
	scratch.reset(count, l+pad)
	var nullsBitset *BitSet
	if err = x.ReadBytesInto(scratch.nulls); err == nil {
		nullsBitset = NewBitSetFromBytes(l, scratch.nulls[:l])
	} else {
		return nil, err
	}
	for i := 0; i < count; i++ {
		if nullsBitset.Get(i) {
			scratch.data[i] = nil
		} else {
			ioLen := rowDescriptor[i].GetIOLength()
			var data []byte
			if ioLen == 0 {
				var n int
				n, err = x.ReadInt32()
				if err != nil {
					return nil, err
				}
				data = scratch.take(i, n)
				if err = x.ReadBytesInto(data); err != nil {
					return nil, err
				}
				if err = x.SkipPadding(int64(n)); err != nil {
					return nil, err
				}
			} else if ioLen < 0 {
				data = scratch.take(i, -ioLen)
				err = x.ReadBytesInto(data)
			} else {
				data = scratch.take(i, ioLen-1)
				if err = x.ReadBytesInto(data); err != nil {
					return nil, err
				}
				err = x.SkipPadding(int64(ioLen - 1))
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return &SQLData{Nulls: nullsBitset, RowData: scratch.data, RowDescriptor: rowDescriptor}, nil
}

// WriteRowData - write fields null-bitset and data in 13+ protocol version
// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-row-data-format
func WriteRowData(x *XDRWriter, rowDescriptor []FieldDescriptor, values [][]byte) error {
	var err error
	nulls := NewBitSet(len(rowDescriptor))
	for n, value := range values {
		if value == nil {
			nulls.Set(n)
		}
	}
	if err = x.WriteBytes(nulls.AlignedBuf()); err != nil {
		return err
	}

	for n, field := range rowDescriptor {
		if !nulls.Get(n) {
			if field.Type == SQL_TYPE_NULL {
				continue
			}
			valueBuf := values[n]
			valueBufLen := len(valueBuf)
			l := field.CalculateIoLength(values[n])
			if l == 0 {
				if err = x.WriteInt32(valueBufLen); err != nil {
					return err
				}
				if err = x.WriteBytes(valueBuf); err != nil {
					return err
				}
				if err = x.WriteCustomPadding((4-valueBufLen)&3, field.GetPaddingByte()); err != nil {
					return err
				}
			} else if l < 0 {
				realFieldLen := -l
				if len(valueBuf) != realFieldLen {
					panic(fmt.Sprintf("invalid length of value buffer, expected %d, but got %d", realFieldLen, len(valueBuf)))
				}
				if err = x.WriteBytes(valueBuf); err != nil {
					return err
				}
			} else {
				realFieldLen := l - 1

				if valueBufLen >= realFieldLen {
					if err = x.WriteBytes(valueBuf[:realFieldLen]); err != nil {
						return err
					}
					if err = x.WriteCustomPadding((4-realFieldLen)&3, field.GetPaddingByte()); err != nil {
						return err
					}
				} else {
					if err = x.WriteBytes(valueBuf); err != nil {
						return err
					}
					if err = x.WriteCustomPadding(realFieldLen-valueBufLen+((4-realFieldLen)&3), field.GetPaddingByte()); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func CalculateBatchMessageLength(rowDescriptor []FieldDescriptor) int {
	length := 0
	for _, field := range rowDescriptor {
		fieldLength := field.Length
		align := 1

		switch field.Type {
		case SQL_TYPE_TEXT, SQL_TYPE_NULL, SQL_TYPE_BOOLEAN:
			align = 1
		case SQL_TYPE_SHORT:
			align = 2
		case SQL_TYPE_VARYING:
			align = 2
			fieldLength += 2
		case SQL_TYPE_FLOAT, SQL_TYPE_LONG, SQL_TYPE_DATE, SQL_TYPE_TIME, SQL_TYPE_TIMESTAMP,
			SQL_TIME_TZ_EX, SQL_TYPE_BLOB, SQL_TYPE_ARRAY, SQL_TYPE_QUAD, SQL_TIMESTAMP_TZ_EX,
			SQL_TYPE_TIMESTAMP_TZ, SQL_TYPE_TIME_TZ:
			align = 4
		case SQL_TYPE_DOUBLE, SQL_TYPE_D_FLOAT, SQL_TYPE_INT64, SQL_TYPE_DEC64, SQL_TYPE_DEC128, SQL_TYPE_INT128:
			align = 8
		default:
			panic(fmt.Sprintf("can not get batch message length of unknown field descriptor type: %d", field.Type))
		}

		if align > 1 {
			length = (length + align - 1) & -align
		}
		length += fieldLength
		length = ((length + 1) & -2) + 2
	}
	return length
}

// func AppendRowData(buf []byte, rowDescriptor []FieldDescriptor, values [][]byte) []byte {
// 	nulls := NewBitSet(len(rowDescriptor))
// 	for i, value := range values {
// 		if value == nil {
// 			nulls.Set(i)
// 		}
// 	}
// 	buf = append(buf, nulls.AlignedBuf()...)

// 	for i, field := range rowDescriptor {
// 		if nulls.Get(i) {
// 			continue
// 		}
// 		if field.Type == SQL_TYPE_NULL {
// 			continue
// 		}

// 		valueBuf := values[i]
// 		valueBufLen := len(valueBuf)
// 		l := field.CalculateIoLength(valueBuf)

// 		switch {
// 		case l == 0:
// 			buf = fbio.AppendInt32BE(buf, int32(valueBufLen))
// 			buf = append(buf, valueBuf...)
// 			for pad := (4 - valueBufLen) & 3; pad > 0; pad-- {
// 				buf = append(buf, field.GetPaddingByte())
// 			}
// 		case l < 0:
// 			realFieldLen := -l
// 			if valueBufLen != realFieldLen {
// 				panic(fmt.Sprintf("invalid length of value buffer, expected %d, but got %d", realFieldLen, valueBufLen))
// 			}
// 			buf = append(buf, valueBuf...)
// 		default:
// 			realFieldLen := l - 1
// 			if valueBufLen >= realFieldLen {
// 				buf = append(buf, valueBuf[:realFieldLen]...)
// 				for pad := (4 - realFieldLen) & 3; pad > 0; pad-- {
// 					buf = append(buf, field.GetPaddingByte())
// 				}
// 			} else {
// 				buf = append(buf, valueBuf...)
// 				padding := realFieldLen - valueBufLen + ((4 - realFieldLen) & 3)
// 				for ; padding > 0; padding-- {
// 					buf = append(buf, field.GetPaddingByte())
// 				}
// 			}
// 		}
// 	}

// 	return buf
// }
