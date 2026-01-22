package fbproto

import (
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto/utils"
)

type SQLData struct {
	Nulls         *utils.BitSet
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
	var nullsBitset *utils.BitSet
	if err = x.ReadBytesInto(scratch.nulls); err == nil {
		nullsBitset = utils.NewBitSetFromBytes(l, scratch.nulls[:l])
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
	nulls := utils.NewBitSet(len(rowDescriptor))
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
