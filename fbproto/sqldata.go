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

func ReadRows(x *XDRReader, rowDescriptor []FieldDescriptor) (*SQLData, error) {
	var err error
	count := len(rowDescriptor)
	rowData := make([][]byte, count)
	l := int(math.Ceil(float64(count) / 8))
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
	return &SQLData{nullsBitset, rowData, rowDescriptor}, nil
}
