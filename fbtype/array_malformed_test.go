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

func TestArrayDecodeBinaryVaryingElementLengthExceedsRemaining(t *testing.T) {
	t.Parallel()

	ad := newMalformedArrayData(fbproto.SQL_TYPE_VARYING, 8)

	var dst FlatArray[string]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))

	err := ad.decodeBinary(NewMap(), []byte{0, 0, 0, 4, 'a', 'b'}, &dst, nil)
	require.ErrorContains(t, err, "array element 0 length 4 exceeds remaining 2 bytes")
}

func TestArrayDecodeBinaryStaticElementLengthExceedsRemaining(t *testing.T) {
	t.Parallel()

	ad := newMalformedArrayData(fbproto.SQL_TYPE_TEXT, 4)

	var dst FlatArray[string]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))

	err := ad.decodeBinary(NewMap(), []byte{'a', 'b'}, &dst, nil)
	require.ErrorContains(t, err, "array element 0 length 4 exceeds remaining 2 bytes")
}

func newMalformedArrayData(typeID uint32, length uint32) *ArrayData {
	return &ArrayData{
		ElementType: &Type{Name: "array element", TypeID: typeID, Codec: TextCodec{}},
		Dimensions:  []ArrayDimension{{LowerBound: 1, Length: 1}},
		Length:      length,
		TypeId:      typeID,
	}
}
