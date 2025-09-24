package fbproto

import (
	"math"

	"github.com/red-soft/fbx/fbproto/utils"
)

type SQLData struct {
	Nulls         *utils.BitSet
	RowData       [][]byte
	RowDescriptor []FieldDescriptor
}

func ReadRowData(x *XDRReader, rowDescriptor []FieldDescriptor) (*SQLData, error) {
	var err error
	count := len(rowDescriptor)
	rowData := make([][]byte, count)
	l := int(math.Ceil(float64(count+7) / 8))
	pad := (4 - l) & 3
	var nullsBitset *utils.BitSet
	if nulls, err := x.ReadBytes(int64(l + pad)); err == nil {
		nullsBitset = utils.NewBitSetFromBytes(l, nulls)
	} else {
		return nil, err
	}
	for i := 0; i < count; i++ {
		if nullsBitset.Get(i) {
			rowData[i] = nil
		} else {
			ioLen := rowDescriptor[i].GetIOLength()
			var data []byte
			if ioLen == 0 {
				data, err = x.ReadBuffer()
			} else if ioLen < 0 {
				data, err = x.ReadBytes(int64(-ioLen))
			} else {
				data, err = x.ReadBytes(int64(ioLen - 1))
				if err != nil {
					return nil, err
				}
				err = x.SkipPadding(int64(ioLen - 1))
			}
			if err != nil {
				return nil, err
			}
			rowData[i] = data
		}
	}

	return &SQLData{Nulls: nullsBitset, RowData: rowData, RowDescriptor: rowDescriptor}, nil
}

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
