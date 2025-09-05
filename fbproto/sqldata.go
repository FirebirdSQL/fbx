package fbproto

import (
	"math"

	"github.com/red-soft/fbx/fbproto/utils"
)

type SQLData struct {
	Nulls   *utils.BitSet
	RowData [][]byte
}

func (d *SQLData) Read(x *XDRReader, row []FieldDescriptor) error {
	var err error
	count := len(row)
	d.RowData = make([][]byte, count)
	l := int(math.Ceil(float64(count) / 8))
	pad := (4 - l) & 3
	if nulls, err := x.ReadBytes(int64(l + pad)); err == nil {
		d.Nulls = utils.NewBitSetFromBytes(l, nulls)
	} else {
		return err
	}
	for i := 0; i < count; i++ {
		if d.Nulls.Get(i) {
			d.RowData[i] = nil
		} else {
			ioLen := row[i].GetIOLength()
			var data []byte
			if ioLen == 0 {
				data, err = x.ReadBuffer()
			} else if ioLen < 0 {
				data, err = x.ReadBytes(int64(-ioLen))
			} else {
				data, err = x.ReadBytes(int64(ioLen - 1))
				if err != nil {
					return err
				}
				err = x.SkipPadding(int64(ioLen - 1))
			}
			if err != nil {
				return err
			}
			d.RowData[i] = data
		}
	}
	return nil
}
