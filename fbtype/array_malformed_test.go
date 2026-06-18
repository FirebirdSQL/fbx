package fbtype

import (
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestArrayDecodeBinaryVaryingTruncatedLength(t *testing.T) {
	t.Parallel()

	ad := newMalformedArrayData(fbproto.SQL_TYPE_VARYING, 8)

	var dst FlatArray[string]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))

	err := ad.decodeBinary(NewMap(), []byte{0, 0}, &dst, nil)
	require.ErrorContains(t, err, "array body truncated at element 0")
}

func newMalformedArrayData(typeID uint32, length uint32) *ArrayData {
	return &ArrayData{
		ElementType: &Type{Name: "array element", TypeID: typeID, Codec: TextCodec{}},
		Dimensions:  []ArrayDimension{{LowerBound: 1, Length: 1}},
		Length:      length,
		TypeId:      typeID,
	}
}
