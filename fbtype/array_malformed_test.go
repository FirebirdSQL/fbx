package fbtype

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/internal/fbio"
	"github.com/stretchr/testify/require"
)

func TestArrayDecodeBinaryVaryingTruncatedLength(t *testing.T) {
	t.Parallel()

	ad := newMalformedArrayData(fbproto.SQL_TYPE_VARYING, 8)

	var dst FlatArray[string]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))

	err := ad.decodeBinary(NewMap(), []byte{0, 0}, &dst, nil)
	require.ErrorIs(t, err, fbio.ErrInsufficientBytes)
	require.ErrorContains(t, err, "array body truncated at element 0")
}

func TestArrayDecodeBinaryVaryingElementLengthExceedsRemaining(t *testing.T) {
	t.Parallel()

	ad := newMalformedArrayData(fbproto.SQL_TYPE_VARYING, 8)

	var dst FlatArray[string]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))

	err := ad.decodeBinary(NewMap(), []byte{0, 0, 0, 4, 'a', 'b'}, &dst, nil)
	require.ErrorIs(t, err, fbio.ErrInsufficientBytes)
	require.ErrorContains(t, err, "array element 0 length 4 exceeds remaining 2 bytes")
}

func TestArrayDecodeBinaryStaticElementLengthExceedsRemaining(t *testing.T) {
	t.Parallel()

	ad := newMalformedArrayData(fbproto.SQL_TYPE_TEXT, 4)

	var dst FlatArray[string]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))

	err := ad.decodeBinary(NewMap(), []byte{'a', 'b'}, &dst, nil)
	require.ErrorIs(t, err, fbio.ErrInsufficientBytes)
	require.ErrorContains(t, err, "array element 0")
}

func TestArrayCheckedReadRejectsMalformedVarying(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		src     []byte
		element int
	}{
		{name: "one length byte", src: []byte{0}},
		{name: "two length bytes", src: []byte{0, 0}},
		{name: "three length bytes", src: []byte{0, 0, 0}},
		{name: "missing value", src: []byte{0, 0, 0, 1}},
		{name: "truncated value", src: []byte{0, 0, 0, 3, 'a', 'b'}},
		{name: "length exceeds int32", src: []byte{0x80, 0, 0, 0}},
		{name: "maximum uint32 length", src: []byte{0xff, 0xff, 0xff, 0xff}},
		{name: "second length truncated", src: []byte{0, 0, 0, 0, 0, 0}, element: 1},
		{name: "second value truncated", src: []byte{0, 0, 0, 0, 0, 0, 0, 2, 'a'}, element: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ad := newMalformedArrayData(fbproto.SQL_TYPE_VARYING, 8)
			metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
			metadata.SetArrayMetadata(ad)
			m := NewMap()

			// The counting pass must reject malformed input before resetting dst.
			dst := []string{"unchanged"}
			err := m.Scan(metadata, tt.src, &dst)
			require.ErrorIs(t, err, fbio.ErrInsufficientBytes)
			require.ErrorContains(t, err, fmt.Sprintf("element %d", tt.element))
			require.Equal(t, []string{"unchanged"}, dst)

			// Exercise the second pass separately: failed reads must not become NULL.
			initial := Text{String: "unchanged", Valid: true}
			elements := FlatArray[Text]{initial, initial}
			err = ad.decodeBinary(m, tt.src, &elements, []realIndexes{{start: 0, end: 1}})
			require.ErrorIs(t, err, fbio.ErrInsufficientBytes)
			require.ErrorContains(t, err, fmt.Sprintf("element %d", tt.element))
			require.Equal(t, initial, elements[tt.element])
		})
	}
}

func TestArrayMapScanRejectsTruncatedFixedElements(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		typeID uint32
		length uint32
		src    []byte
		want   any
	}{
		{"smallint", fbproto.SQL_TYPE_SHORT, 2, []byte{0xff, 0xff, 0xff, 0xff}, int16(-1)},
		{"integer", fbproto.SQL_TYPE_LONG, 4, []byte{0, 0, 0, 42}, int32(42)},
		{"boolean", fbproto.SQL_TYPE_BOOLEAN, 1, []byte{1, 0, 0, 0}, true},
		{"char", fbproto.SQL_TYPE_TEXT, 2, []byte{'a', 'b', 0, 0}, "ab"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
			metadata.SetArrayMetadata(newMalformedArrayData(tt.typeID, tt.length))
			m := NewMap()
			body := append(append([]byte(nil), tt.src...), tt.src...)

			for n := 1; n < len(body); n++ {
				if n == len(tt.src) {
					continue
				}
				for _, initial := range [][]any{nil, {"unchanged", "unchanged"}} {
					dst := append([]any(nil), initial...)
					var err error
					require.NotPanics(t, func() {
						err = m.Scan(metadata, body[:n], &dst)
					}, "body length %d", n)
					require.Error(t, err, "body length %d", n)
					require.Equal(t, initial, dst, "body length %d", n)
				}
			}

			var dst []any
			require.NoError(t, m.Scan(metadata, tt.src, &dst))
			require.Equal(t, []any{tt.want}, dst)
		})
	}
}

func TestArrayDecodeBinaryRejectsTruncatedPaddingBeforeScan(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		typeID uint32
		length uint32
		src    []byte
	}{
		{"boolean", fbproto.SQL_TYPE_BOOLEAN, 1, []byte{1, 0, 0, 0}},
		{"char", fbproto.SQL_TYPE_TEXT, 2, []byte{'a', 'b', 0, 0}},
	} {
		for n := int(tt.length); n < len(tt.src); n++ {
			t.Run(fmt.Sprintf("%s/%d_bytes", tt.name, n), func(t *testing.T) {
				ad := newMalformedArrayData(tt.typeID, tt.length)
				dst := FlatArray[any]{"unchanged", "unchanged"}
				err := ad.decodeBinary(NewMap(), tt.src[:n], &dst, []realIndexes{{start: 0, end: 1}})
				require.ErrorIs(t, err, fbio.ErrInsufficientBytes)
				require.Equal(t, FlatArray[any]{"unchanged", "unchanged"}, dst)
			})
		}
	}
}

func TestArrayDecodeBinaryRejectsTrailingBytes(t *testing.T) {
	t.Parallel()

	ad := newMalformedArrayData(fbproto.SQL_TYPE_LONG, 4)
	var dst FlatArray[int32]
	require.NoError(t, (&dst).SetDimensions(ad.Dimensions))
	err := ad.decodeBinary(NewMap(), []byte{0, 0, 0, 1, 0, 0, 0, 2, 3}, &dst, []realIndexes{{start: 0, end: 1}})
	require.ErrorContains(t, err, "1 unexpected trailing bytes")
}

func TestArrayCheckedReadPreservesElementsAndDimensions(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		ad   ArrayData
		src  []byte
		dst  any
		want any
	}{
		{
			name: "smallint uses four wire bytes",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_SHORT, Length: 2},
			src:  []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 2},
			dst:  &[]int16{}, want: &[]int16{-1, 2},
		},
		{
			name: "boolean padding",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_BOOLEAN, Length: 1},
			src:  []byte{1, 0, 0, 0, 0, 0, 0, 0},
			dst:  &[]bool{}, want: &[]bool{true, false},
		},
		{
			name: "fixed octets padding",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_TEXT, SubType: fbproto.CS_BINARY, Length: 2},
			src:  []byte{0xff, 0, 0, 0, 1, 2, 0, 0},
			dst:  &FlatArray[[]byte]{}, want: &FlatArray[[]byte]{{0xff, 0}, {1, 2}},
		},
		{
			name: "varchar has no alignment padding in decoded body",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_VARYING, Length: 8},
			src:  []byte{0, 0, 0, 2, 'a', 'b', 0, 0, 0, 1, 'c', 0, 0, 0, 0},
			dst:  &[]string{}, want: &[]string{"ab", "c", ""},
		},
		{
			name: "partial varchar preserves lower bound",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_VARYING, Length: 8},
			src:  []byte{0, 0, 0, 0},
			dst:  &Array[string]{},
			want: &Array[string]{Elements: []string{""}, Dims: []ArrayDimension{{LowerBound: -2, Length: 1}}, Valid: true},
		},
		{
			name: "numeric preserves storage length and scale",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_SHORT, SubType: 1, Length: 2, Scale: -2},
			src:  []byte{0xff, 0xff, 0xcf, 0xc7},
			dst:  &[]Numeric{}, want: &[]Numeric{{Int: big.NewInt(-12345), Exp: -2, Valid: true}},
		},
		{
			name: "decimal preserves subtype and scale",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_LONG, SubType: 2, Length: 4, Scale: -2},
			src:  []byte{0xff, 0xff, 0xcf, 0xc7},
			dst:  &[]Numeric{}, want: &[]Numeric{{Int: big.NewInt(-12345), Exp: -2, Valid: true}},
		},
		{
			name: "partial multidimensional array skips unused positions",
			ad: ArrayData{
				TypeId: fbproto.SQL_TYPE_LONG, Length: 4,
				Dimensions: []ArrayDimension{{LowerBound: 1, Length: 2}, {LowerBound: 1, Length: 2}},
			},
			src: []byte{0, 0, 0, 1, 0, 0, 0, 99, 0, 0, 0, 2},
			dst: &[][]int32{}, want: &[][]int32{{1}, {2}},
		},
		{
			name: "empty array",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_LONG, Length: 4},
			src:  []byte{},
			dst:  &[]int32{99}, want: &[]int32{},
		},
		{
			name: "null array",
			ad:   ArrayData{TypeId: fbproto.SQL_TYPE_LONG, Length: 4},
			src:  nil,
			dst:  &[]int32{99}, want: new([]int32),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ad.Dimensions == nil {
				tt.ad.Dimensions = []ArrayDimension{{LowerBound: -2, Length: 3}}
			}
			metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
			metadata.SetArrayMetadata(&tt.ad)
			m := NewMap()

			require.NoError(t, m.Scan(metadata, tt.src, tt.dst))
			require.Equal(t, tt.want, tt.dst)
			require.NoError(t, m.SQLScanner(tt.dst).Scan(ArraySQLScanner{ArrayData: &tt.ad, ArrayBuffer: tt.src}))
			require.Equal(t, tt.want, tt.dst)
		})
	}
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

func TestArrayEncodeVaryingRejectsZeroByte(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		subType int32
		value   any
	}{
		{
			name:    "text",
			subType: fbproto.CS_NONE,
			value: Array[string]{
				Elements: []string{"a\x00b", "ok"},
				Dims:     []ArrayDimension{{LowerBound: 1, Length: 2}},
				Valid:    true,
			},
		},
		{
			name:    "octets",
			subType: fbproto.CS_BINARY,
			value: Array[[]byte]{
				Elements: [][]byte{{0x01, 0x00}, {0x02}},
				Dims:     []ArrayDimension{{LowerBound: 1, Length: 2}},
				Valid:    true,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			m := NewMap()
			arrayData, err := NewArrayData(
				fbproto.SQL_TYPE_VARYING,
				testCase.subType,
				3,
				0,
				[]map[string]int{{"Lower": 1, "Upper": 2}},
			)
			require.NoError(t, err)
			metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
			metadata.SetArrayMetadata(arrayData)

			_, err = m.Encode(metadata, testCase.value, nil)

			require.ErrorContains(t, err, "cannot encode array element at index 0")
			require.ErrorContains(t, err, "VARCHAR array elements cannot contain zero bytes")
		})
	}
}

func TestArrayEncodeVaryingAllowsEmptyAndRegularStrings(t *testing.T) {
	t.Parallel()

	m := NewMap()
	arrayData, err := NewArrayData(
		fbproto.SQL_TYPE_VARYING,
		fbproto.CS_NONE,
		3,
		0,
		[]map[string]int{{"Lower": 1, "Upper": 2}},
	)
	require.NoError(t, err)
	metadata := NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
	metadata.SetArrayMetadata(arrayData)
	value := Array[string]{
		Elements: []string{"", "abc"},
		Dims:     []ArrayDimension{{LowerBound: 1, Length: 2}},
		Valid:    true,
	}

	buf, err := m.Encode(metadata, value, nil)

	require.NoError(t, err)
	require.Equal(t, []byte{
		0, 0, 0, 0,
		0, 0, 0, 3, 'a', 'b', 'c', 0,
	}, buf)
}

func newMalformedArrayData(typeID uint32, length uint32) *ArrayData {
	return &ArrayData{
		Dimensions: []ArrayDimension{{LowerBound: 1, Length: 2}},
		Length:     length,
		TypeId:     typeID,
	}
}
