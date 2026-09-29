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

func TestArrayDecodeBinaryFixedOctetsExcludesAlignmentPadding(t *testing.T) {
	t.Parallel()

	ad := &ArrayData{
		Dimensions: []ArrayDimension{{LowerBound: 1, Length: 2}},
		SubType:    fbproto.CS_BINARY,
		Length:     2,
		TypeId:     fbproto.SQL_TYPE_TEXT,
	}
	var dst FlatArray[[]byte]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))

	err := ad.decodeBinary(
		NewMap(),
		[]byte{0xff, 0x00, 0x00, 0x00, 0x01, 0x02, 0x00, 0x00},
		&dst,
		[]realIndexes{{start: 0, end: 1}},
	)

	require.NoError(t, err)
	require.Equal(t, FlatArray[[]byte]{{0xff, 0x00}, {0x01, 0x02}}, dst)
}

func TestArrayEncodeRejectsOctetsElementLongerThanDeclared(t *testing.T) {
	t.Parallel()

	for _, typeID := range []uint32{fbproto.SQL_TYPE_TEXT, fbproto.SQL_TYPE_VARYING} {
		m := NewMap()
		arrayData, err := NewArrayData(
			typeID,
			fbproto.CS_BINARY,
			2,
			0,
			[]map[string]int{{"Lower": 1, "Upper": 2}},
		)
		require.NoError(t, err)
		metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
		metadata.SetArrayMetadata(arrayData)
		value := Array[[]byte]{
			Elements: [][]byte{{0x01, 0x02, 0x03}, {0x04, 0x05}},
			Dims:     []ArrayDimension{{LowerBound: 1, Length: 2}},
			Valid:    true,
		}

		_, err = m.Encode(metadata, value, nil)

		require.ErrorContains(t, err, "cannot encode array element at index 0")
		require.ErrorContains(t, err, "encoded array element length 3 exceeds declared length 2")
	}
}

func TestArrayEncodeFixedOctetsPadsWithZero(t *testing.T) {
	t.Parallel()

	m := NewMap()
	arrayData, err := NewArrayData(
		fbproto.SQL_TYPE_TEXT,
		fbproto.CS_BINARY,
		3,
		0,
		[]map[string]int{{"Lower": 1, "Upper": 2}},
	)
	require.NoError(t, err)
	metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
	metadata.SetArrayMetadata(arrayData)
	value := Array[[]byte]{
		Elements: [][]byte{{0x01}, {0x02, 0x03}},
		Dims:     []ArrayDimension{{LowerBound: 1, Length: 2}},
		Valid:    true,
	}

	buf, err := m.Encode(metadata, value, nil)

	require.NoError(t, err)
	require.Equal(t, []byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x03, 0x00, 0x00}, buf)
}

func TestArrayEncodeVaryingOctetsRejectsZeroByte(t *testing.T) {
	t.Parallel()

	m := NewMap()
	arrayData, err := NewArrayData(
		fbproto.SQL_TYPE_VARYING,
		fbproto.CS_BINARY,
		3,
		0,
		[]map[string]int{{"Lower": 1, "Upper": 2}},
	)
	require.NoError(t, err)
	metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
	metadata.SetArrayMetadata(arrayData)
	value := Array[[]byte]{
		Elements: [][]byte{{0x01, 0x00}, {0x02}},
		Dims:     []ArrayDimension{{LowerBound: 1, Length: 2}},
		Valid:    true,
	}

	_, err = m.Encode(metadata, value, nil)

	require.ErrorContains(t, err, "cannot encode array element at index 0")
	require.ErrorContains(t, err, "VARCHAR OCTETS array elements cannot contain zero bytes")
}

func newMalformedArrayData(typeID uint32, length uint32) *ArrayData {
	return &ArrayData{
		ElementType: &Type{Name: "array element", TypeID: typeID, Codec: TextCodec{}},
		Dimensions:  []ArrayDimension{{LowerBound: 1, Length: 2}},
		Length:      length,
		TypeId:      typeID,
	}
}
