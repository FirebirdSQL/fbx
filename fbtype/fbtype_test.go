package fbtype_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

var defaultConnTestRunner fbxtest.ConnTestRunner

func init() {
	defaultConnTestRunner = fbxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		return config
	}
}

func getServerMajorVersionForConn(ctx context.Context, t testing.TB, conn *fbx.Conn) int {
	t.Helper()
	return conn.ServerVersion().Major
}

// Renamed helper types used to test underlying-type wrappers.
type _string string
type _uint8 uint8
type _byteSlice []byte

type skipUnderlyingString string

func (skipUnderlyingString) SkipUnderlyingTypePlan() {}

type structStringer struct{}

func (structStringer) String() string {
	return "struct stringer"
}

type stringerInt16 int16

func (s stringerInt16) String() string {
	return fmt.Sprintf("stringer-%d", s)
}

// unregisteredTypeID represents an unknown non-zero Firebird TypeID. Some tests
// use 0 separately to represent missing metadata.
const unregisteredTypeID = uint32(1)

// sqlScannerFunc lets an arbitrary function be used as a sql.Scanner.
type sqlScannerFunc func(src any) error

func (f sqlScannerFunc) Scan(src any) error {
	return f(src)
}

// driverValuerFunc lets an arbitrary function be used as a driver.Valuer.
type driverValuerFunc func() (driver.Value, error)

func (f driverValuerFunc) Value() (driver.Value, error) {
	return f()
}

type arrayGetterEncodeCodec struct{}

func (arrayGetterEncodeCodec) PlanEncode(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, value any) fbtype.EncodePlan {
	if _, ok := value.(fbtype.ArrayGetter); ok {
		return arrayGetterEncodePlan{}
	}
	return nil
}

func (arrayGetterEncodeCodec) PlanScan(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, target any) fbtype.ScanPlan {
	return nil
}

func (arrayGetterEncodeCodec) DecodeDatabaseSQLValue(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, src []byte) (driver.Value, error) {
	return nil, nil
}

func (arrayGetterEncodeCodec) DecodeValue(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, src []byte) (any, error) {
	return nil, nil
}

type arrayGetterEncodePlan struct{}

func (arrayGetterEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	getter := value.(fbtype.ArrayGetter)
	dimensions, err := getter.Dimensions([]int{1, 1, 1, 1})
	if err != nil {
		return nil, err
	}
	if dimensions == nil {
		return nil, nil
	}

	elementCount := 1
	for _, dimension := range dimensions {
		elementCount *= dimension.Length
	}

	values := make([]any, 0, elementCount)
	for i := 0; i < elementCount; i++ {
		values = append(values, getter.Index(i))
	}

	return append(buf, fmt.Sprintf("%v|%v", dimensions, values)...), nil
}

type arraySetterScanCodec struct {
	dimensions []fbtype.ArrayDimension
	values     []any
}

func (arraySetterScanCodec) PlanEncode(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, value any) fbtype.EncodePlan {
	return nil
}

func (c arraySetterScanCodec) PlanScan(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, target any) fbtype.ScanPlan {
	if _, ok := target.(fbtype.ArraySetter); ok {
		return arraySetterScanPlan{dimensions: c.dimensions, values: c.values}
	}
	return nil
}

func (arraySetterScanCodec) DecodeDatabaseSQLValue(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, src []byte) (driver.Value, error) {
	return nil, nil
}

func (arraySetterScanCodec) DecodeValue(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, src []byte) (any, error) {
	return nil, nil
}

type arraySetterScanPlan struct {
	dimensions []fbtype.ArrayDimension
	values     []any
}

func (p arraySetterScanPlan) Scan(src []byte, target any) error {
	setter := target.(fbtype.ArraySetter)
	if src == nil {
		return setter.SetDimensions(nil)
	}

	if err := setter.SetDimensions(p.dimensions); err != nil {
		return err
	}

	for i, value := range p.values {
		dst := reflect.ValueOf(setter.ScanIndex(i))
		if dst.Kind() != reflect.Ptr || dst.IsNil() {
			return fmt.Errorf("array scan target %d is %T, not a non-nil pointer", i, setter.ScanIndex(i))
		}

		srcValue := reflect.ValueOf(value)
		if !srcValue.Type().AssignableTo(dst.Elem().Type()) {
			if !srcValue.Type().ConvertibleTo(dst.Elem().Type()) {
				return fmt.Errorf("cannot assign %T to %v", value, dst.Elem().Type())
			}
			srcValue = srcValue.Convert(dst.Elem().Type())
		}
		dst.Elem().Set(srcValue)
	}

	return nil
}

func TestMapScanNilIsNoOp(t *testing.T) {
	m := fbtype.NewMap()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	err := m.Scan(typeMetadata, []byte("foo"), nil)
	assert.NoError(t, err)
}

func TestStringerWithUnderlyingIntEncodesAsInt(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 2, 0)

	buf, err := m.Encode(typeMetadata, stringerInt16(200), nil)
	require.NoError(t, err)
	assert.Equal(t, []byte{0, 0, 0, 200}, buf)
}

func TestArraySetDimensionsOverflowDoesNotPanic(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	var a fbtype.FlatArray[int]
	var err error

	require.NotPanics(t, func() {
		err = (&a).SetDimensions([]fbtype.ArrayDimension{
			{LowerBound: 1, Length: maxInt/2 + 1},
			{LowerBound: 1, Length: 3},
		})
	})
	require.NoError(t, err)
	assert.Len(t, a, 0)
}

func TestMapScanTextIntoAnyReturnsString(t *testing.T) {
	m := fbtype.NewMap()
	var got any
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	err := m.Scan(typeMetadata, []byte("foo"), &got)
	require.NoError(t, err)
	assert.Equal(t, "foo", got)
}

func TestMapScanTextIntoStringWithUnsupportedCharsetReturnsError(t *testing.T) {
	m := fbtype.NewMap()
	var got string

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 99, 3, 0)
	err := m.Scan(typeMetadata, []byte("foo"), &got)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot scan")
}

func TestMapScanSmallintIntoAnyReturnsInt16(t *testing.T) {
	m := fbtype.NewMap()
	var got any

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 4, 0)
	err := m.Scan(typeMetadata, []byte{0, 0, 0, 1}, &got)

	require.NoError(t, err)
	assert.Equal(t, int16(1), got)
}

func TestMapScanIntegerIntoAnyReturnsInt32(t *testing.T) {
	m := fbtype.NewMap()
	var got any

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)
	err := m.Scan(typeMetadata, []byte{0, 0, 0, 1}, &got)

	require.NoError(t, err)
	assert.Equal(t, int32(1), got)
}

func TestMapScanUnknownTypeIDToPtrToAny(t *testing.T) {
	unknownTypeID := uint32(999999)
	srcBuf := []byte("foo")
	m := fbtype.NewMap()

	var a any
	typeMetadata := fbtype.NewTypeMetadata(unknownTypeID, 0, 3, 0)
	err := m.Scan(typeMetadata, srcBuf, &a)
	assert.NoError(t, err)
	assert.Equal(t, []byte("foo"), a)
}

func TestMapScanUnknownTypeIDToStringsAndBytes(t *testing.T) {
	unknownTypeID := uint32(999999)
	srcBuf := []byte("foo")
	m := fbtype.NewMap()

	var s string
	typeMetadata := fbtype.NewTypeMetadata(unknownTypeID, 0, 3, 0)
	err := m.Scan(typeMetadata, srcBuf, &s)
	assert.NoError(t, err)
	assert.Equal(t, "foo", s)

	var rs _string
	err = m.Scan(typeMetadata, srcBuf, &rs)
	assert.NoError(t, err)
	assert.Equal(t, "foo", string(rs))

	var b []byte
	err = m.Scan(typeMetadata, srcBuf, &b)
	assert.NoError(t, err)
	assert.Equal(t, []byte("foo"), b)

	var rb _byteSlice
	err = m.Scan(typeMetadata, srcBuf, &rb)
	assert.NoError(t, err)
	assert.Equal(t, []byte("foo"), []byte(rb))
}

func TestMapScanPointerToNilStructReturnsError(t *testing.T) {
	m := fbtype.NewMap()

	type myStruct struct{}
	var p *myStruct
	typeMetadata := fbtype.NewTypeMetadata(0, 0, 9, 0)
	err := m.Scan(typeMetadata, []byte("(foo,bar)"), &p)
	require.Error(t, err)
}

func TestMapScanMissingTypeMetadataIntoSQLScannerReceivesNil(t *testing.T) {
	m := fbtype.NewMap()

	var s sql.NullString
	typeMetadata := fbtype.NewTypeMetadata(0, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte(nil), &s)
	assert.NoError(t, err)
	assert.Equal(t, "", s.String)
	assert.False(t, s.Valid)
}

func TestMapScanRegisteredTypeIntoSQLScannerReceivesDatabaseSQLValue(t *testing.T) {
	m := fbtype.NewMap()

	n, err := fbtype.ParseNumeric("12.34")
	require.NoError(t, err)

	src, err := fbtype.EncodeNumericBinaryFB(nil, n, -2, 4)
	require.NoError(t, err)

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, -2)
	var got any
	scanner := sqlScannerFunc(func(src any) error {
		got = src
		return nil
	})

	err = m.Scan(typeMetadata, src, scanner)
	require.NoError(t, err)
	require.IsType(t, float64(0), got)
	assert.Equal(t, 12.34, got)

	got = "not nil"
	err = m.Scan(typeMetadata, nil, scanner)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapScanUnregisteredTypeIntoSQLScannerReceivesRawBytes(t *testing.T) {
	m := fbtype.NewMap()

	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 3, 0)
	src := []byte{1, 2, 3}
	var got any
	scanner := sqlScannerFunc(func(src any) error {
		got = src
		return nil
	})

	err := m.Scan(typeMetadata, src, scanner)
	require.NoError(t, err)
	gotBytes, ok := got.([]byte)
	require.True(t, ok)
	assert.Equal(t, src, gotBytes)

	got = "not nil"
	err = m.Scan(typeMetadata, nil, scanner)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapScanUndecodedBytesCopiesRawBytes(t *testing.T) {
	m := fbtype.NewMap()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)
	src := []byte{0, 0, 0, 42}
	var got fbtype.UndecodedBytes

	err := m.Scan(typeMetadata, src, &got)
	require.NoError(t, err)
	assert.Equal(t, []byte{0, 0, 0, 42}, []byte(got))

	src[3] = 99
	assert.Equal(t, []byte{0, 0, 0, 42}, []byte(got))

	err = m.Scan(typeMetadata, nil, &got)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapScanPointerPointerAllocatesAndClears(t *testing.T) {
	m := fbtype.NewMap()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)
	src := []byte{0, 0, 0, 42}
	var got *int32

	err := m.Scan(typeMetadata, src, &got)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int32(42), *got)

	err = m.Scan(typeMetadata, nil, &got)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapScanRenamedStringUsesUnderlyingType(t *testing.T) {
	m := fbtype.NewMap()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	var got _string

	err := m.Scan(typeMetadata, []byte("foo"), &got)
	require.NoError(t, err)
	assert.Equal(t, "foo", string(got))
}

func TestMapScanRenamedByteSliceUsesUnderlyingType(t *testing.T) {
	m := fbtype.NewMap()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	var got _byteSlice

	err := m.Scan(typeMetadata, []byte("foo"), &got)
	require.NoError(t, err)
	assert.Equal(t, _byteSlice("foo"), got)
}

func TestMapScanNumericIntoBuiltinWrappers(t *testing.T) {
	m := fbtype.NewMap()

	scanNumeric := func(value string, scale int32, dst any) {
		t.Helper()

		n, err := fbtype.ParseNumeric(value)
		require.NoError(t, err)

		buf, err := fbtype.EncodeNumericBinaryFB(nil, n, scale, 4)
		require.NoError(t, err)

		typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, scale)
		err = m.Scan(typeMetadata, buf, dst)
		require.NoError(t, err)
	}

	var i32 int32
	scanNumeric("42", 0, &i32)
	assert.Equal(t, int32(42), i32)

	var u64 uint64
	scanNumeric("42", 0, &u64)
	assert.Equal(t, uint64(42), u64)

	var f32 float32
	scanNumeric("12.5", -1, &f32)
	assert.Equal(t, float32(12.5), f32)
}

func TestMapScanPtrSliceWrapperUsesFlatArrayForCommonTypes(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999997)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_int16_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{{LowerBound: 1, Length: 3}},
			values:     []any{int16(1), int16(2), int16(3)},
		},
	})

	var got []int16
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.NoError(t, err)
	assert.Equal(t, []int16{1, 2, 3}, got)

	err = m.Scan(typeMetadata, nil, &got)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapScanPtrSliceWrapperUsesReflectForUncommonTypes(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999996)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_bool_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{{LowerBound: 1, Length: 3}},
			values:     []any{true, false, true},
		},
	})

	var got []bool
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, true}, got)

	err = m.Scan(typeMetadata, nil, &got)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapEncodeSliceWrapperUsesFlatArrayForCommonTypes(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999986)
	m.RegisterType(&fbtype.Type{Name: "array_getter_encode_int16_test", TypeID: typeID, Codec: arrayGetterEncodeCodec{}})

	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, []int16{1, 2, 3}, nil)
	require.NoError(t, err)
	assert.Equal(t, "[{1 3}]|[1 2 3]", string(buf))

	buf, err = m.Encode(typeMetadata, []int16(nil), nil)
	require.NoError(t, err)
	assert.Nil(t, buf)
}

func TestMapEncodeSliceWrapperUsesReflectForUncommonTypes(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999985)
	m.RegisterType(&fbtype.Type{Name: "array_getter_encode_bool_test", TypeID: typeID, Codec: arrayGetterEncodeCodec{}})

	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, []bool{true, false, true}, nil)
	require.NoError(t, err)
	assert.Equal(t, "[{1 3}]|[true false true]", string(buf))
}

func TestMapScanMultiDimensionalSliceUsesMultiDimensionalWrapper(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999995)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_multi_dim_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{
				{LowerBound: 1, Length: 2},
				{LowerBound: 1, Length: 2},
			},
			values: []any{int16(1), int16(2), int16(3), int16(4)},
		},
	})

	var got [][]int16
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.NoError(t, err)
	assert.Equal(t, [][]int16{{1, 2}, {3, 4}}, got)
}

func TestMapEncodeMultiDimensionalSliceUsesMultiDimensionalWrapper(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999984)
	m.RegisterType(&fbtype.Type{Name: "array_getter_encode_multi_dim_test", TypeID: typeID, Codec: arrayGetterEncodeCodec{}})

	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, [][]int16{{1, 2}, {3, 4}}, nil)
	require.NoError(t, err)
	assert.Equal(t, "[{1 2} {1 2}]|[1 2 3 4]", string(buf))
}

func TestMapEncodeArrayWrapperUsesReflectForFixedArray(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999983)
	m.RegisterType(&fbtype.Type{Name: "array_getter_encode_fixed_array_test", TypeID: typeID, Codec: arrayGetterEncodeCodec{}})

	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, [3]int16{1, 2, 3}, nil)
	require.NoError(t, err)
	assert.Equal(t, "[{1 3}]|[1 2 3]", string(buf))
}

func TestMapScanPtrMultiDimSliceWrapperOverwritesRaggedDestination(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999994)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_ragged_multi_dim_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{
				{LowerBound: 1, Length: 2},
				{LowerBound: 1, Length: 2},
			},
			values: []any{int16(1), int16(2), int16(3), int16(4)},
		},
	})

	got := [][]int16{{9}, {8, 7}}
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.NoError(t, err)
	assert.Equal(t, [][]int16{{1, 2}, {3, 4}}, got)

	err = m.Scan(typeMetadata, nil, &got)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapScanPtrArrayWrapperFillsFixedArray(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999993)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_fixed_array_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{{LowerBound: 1, Length: 3}},
			values:     []any{int16(1), int16(2), int16(3)},
		},
	})

	var got [3]int16
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.NoError(t, err)
	assert.Equal(t, [3]int16{1, 2, 3}, got)

	err = m.Scan(typeMetadata, nil, &got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot scan NULL")
}

func TestMapScanPtrArrayWrapperRejectsLengthMismatch(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999992)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_fixed_array_length_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{{LowerBound: 1, Length: 2}},
			values:     []any{int16(1), int16(2)},
		},
	})

	var got [3]int16
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot scan array with length 2")
}

type scannerString string

func (ss *scannerString) Scan(v any) error {
	*ss = scannerString("scanned")
	return nil
}

func TestMapScanUnregisteredTypeIDIntoRenamedStringSQLScanner(t *testing.T) {
	m := fbtype.NewMap()

	var s scannerString
	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte(nil), &s)
	assert.NoError(t, err)
	assert.Equal(t, "scanned", string(s))
}

type fbCustomInt int64

func (ci *fbCustomInt) Scan(src interface{}) error {
	*ci = fbCustomInt(src.(int64))
	return nil
}

func TestMapScanSQLScannerAndPointerPointerTargets(t *testing.T) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v fbCustomInt

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 4, 0)
	plan := m.PlanScan(typeMetadata, &v)
	err := plan.Scan(src, &v)
	require.NoError(t, err)
	require.EqualValues(t, 42, v)

	ptr := new(fbCustomInt)
	plan = m.PlanScan(typeMetadata, &ptr)
	err = plan.Scan(src, &ptr)
	require.NoError(t, err)
	require.EqualValues(t, 42, *ptr)

	ptr = new(fbCustomInt)
	err = plan.Scan(nil, &ptr)
	require.NoError(t, err)
	assert.Nil(t, ptr)

	ptr = nil
	plan = m.PlanScan(typeMetadata, &ptr)
	err = plan.Scan(src, &ptr)
	require.NoError(t, err)
	require.EqualValues(t, 42, *ptr)

	ptr = nil
	plan = m.PlanScan(typeMetadata, &ptr)
	err = plan.Scan(nil, &ptr)
	require.NoError(t, err)
	assert.Nil(t, ptr)
}

func TestPlanScanNilInterfaceTargetFails(t *testing.T) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v interface{}
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 4, 0)
	plan := m.PlanScan(typeMetadata, v)
	err := plan.Scan(src, v)
	assert.Error(t, err)
}

func TestMapScanPtrToPtrToSliceAllocatesAndClears(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999991)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_ptr_to_ptr_slice_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{{LowerBound: 1, Length: 2}},
			values:     []any{"foo", "bar"},
		},
	})

	var v *[]string
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &v)
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, []string{"foo", "bar"}, *v)

	err = m.Scan(typeMetadata, nil, &v)
	require.NoError(t, err)
	assert.Nil(t, v)
}

type databaseValuerString string

func (s databaseValuerString) Value() (driver.Value, error) {
	return fmt.Sprintf("%d", len(s)), nil
}

func TestMapEncodeRenamedDriverValuerUsesValue(t *testing.T) {
	m := fbtype.NewMap()
	src := databaseValuerString("foo")
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	buf, err := m.Encode(typeMetadata, src, nil)
	require.NoError(t, err)
	require.Equal(t, "3", string(buf))
}

func TestMapPlanEncodeDerefPointer(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	value := "foo"

	plan := m.PlanEncode(typeMetadata, &value)
	require.NotNil(t, plan)

	buf, err := plan.Encode(&value, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", string(buf))

	var nilValue *string
	buf, err = plan.Encode(nilValue, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

func TestMapEncodeMemoizationUsesFullTypeMetadata(t *testing.T) {
	m := fbtype.NewMap()

	numericScale2 := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, -2)
	buf, err := m.Encode(numericScale2, "12", nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 4, 176}, buf)

	numericScale0 := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, 0)
	buf, err = m.Encode(numericScale0, "12", nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 12}, buf)
}

func TestMapEncodeRenamedStringUsesUnderlyingType(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)

	buf, err := m.Encode(typeMetadata, _string("foo"), nil)
	require.NoError(t, err)
	require.Equal(t, "foo", string(buf))
}

func TestMapEncodeSkipUnderlyingTypePlannerDoesNotUseUnderlyingType(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)

	buf, err := m.Encode(typeMetadata, skipUnderlyingString("foo"), nil)
	require.Error(t, err)
	require.Nil(t, buf)
	require.Contains(t, err.Error(), "cannot find encode plan")
}

func TestTryWrapStringerEncodePlanWrapsAsTextValuer(t *testing.T) {
	value := structStringer{}

	wrapperPlan, nextValue, ok := fbtype.TryWrapStringerEncodePlan(value)
	require.True(t, ok)
	require.NotNil(t, wrapperPlan)

	textValuer, ok := nextValue.(fbtype.TextValuer)
	require.True(t, ok)
	text, err := textValuer.TextValue()
	require.NoError(t, err)
	require.Equal(t, fbtype.Text{String: "struct stringer", Valid: true}, text)

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 15, 0)
	nextPlan := fbtype.TextCodec{}.PlanEncode(fbtype.NewMap(), typeMetadata, nextValue)
	require.NotNil(t, nextPlan)
	wrapperPlan.SetNext(nextPlan)

	buf, err := wrapperPlan.Encode(value, nil)
	require.NoError(t, err)
	require.Equal(t, "struct stringer", string(buf))
}

func TestTryWrapStringerEncodePlanSkipsDriverValuer(t *testing.T) {
	wrapperPlan, nextValue, ok := fbtype.TryWrapStringerEncodePlan(databaseValuerFmtStringer("foo"))

	require.False(t, ok)
	require.Nil(t, wrapperPlan)
	require.Nil(t, nextValue)
}

func TestTryWrapBuiltinTypeEncodePlanWrapsInt16AsInt64Valuer(t *testing.T) {
	wrapperPlan, nextValue, ok := fbtype.TryWrapBuiltinTypeEncodePlan(int16(200))
	require.True(t, ok)
	require.NotNil(t, wrapperPlan)

	int64Valuer, ok := nextValue.(fbtype.Int64Valuer)
	require.True(t, ok)
	int64Value, err := int64Valuer.Int64Value()
	require.NoError(t, err)
	require.Equal(t, fbtype.Int8{Int64: 200, Valid: true}, int64Value)

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 2, 0)
	nextPlan := fbtype.Int4Codec{}.PlanEncode(fbtype.NewMap(), typeMetadata, nextValue)
	require.NotNil(t, nextPlan)
	wrapperPlan.SetNext(nextPlan)

	buf, err := wrapperPlan.Encode(int16(200), nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 200}, buf)
}

func TestTryWrapBuiltinTypeEncodePlanWrapsByteSliceAsTextValuer(t *testing.T) {
	value := []byte("foo")
	wrapperPlan, nextValue, ok := fbtype.TryWrapBuiltinTypeEncodePlan(value)
	require.True(t, ok)
	require.NotNil(t, wrapperPlan)

	textValuer, ok := nextValue.(fbtype.TextValuer)
	require.True(t, ok)
	text, err := textValuer.TextValue()
	require.NoError(t, err)
	require.Equal(t, fbtype.Text{String: "foo", Valid: true}, text)

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	nextPlan := fbtype.TextCodec{}.PlanEncode(fbtype.NewMap(), typeMetadata, nextValue)
	require.NotNil(t, nextPlan)
	wrapperPlan.SetNext(nextPlan)

	buf, err := wrapperPlan.Encode(value, nil)
	require.NoError(t, err)
	require.Equal(t, "foo", string(buf))

	var nilValue []byte
	buf, err = wrapperPlan.Encode(nilValue, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

type databaseValuerFmtStringer string

func (s databaseValuerFmtStringer) Value() (driver.Value, error) {
	return nil, nil
}

func (s databaseValuerFmtStringer) String() string {
	return "foobar"
}

func TestMapEncodeDatabaseValuerPreferredOverStringer(t *testing.T) {
	m := fbtype.NewMap()
	src := databaseValuerFmtStringer("foo")
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 6, 0)

	buf, err := m.Encode(typeMetadata, src, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

type databaseValuerStringFormat struct {
	n int32
}

func (v databaseValuerStringFormat) Value() (driver.Value, error) {
	return fmt.Sprint(v.n), nil
}

func TestMapEncodeDatabaseValuerStringIntoSmallint(t *testing.T) {
	m := fbtype.NewMap()
	src := databaseValuerStringFormat{n: 42}
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, src, nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 42}, buf)
}

func TestMapEncodeDatabaseValuerReturnsError(t *testing.T) {
	m := fbtype.NewMap()
	wantErr := fmt.Errorf("valuer failed")
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)

	buf, err := m.Encode(typeMetadata, driverValuerFunc(func() (driver.Value, error) {
		return nil, wantErr
	}), nil)

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, buf)
}

type valueReceiverNilValuer string

func (v valueReceiverNilValuer) Value() (driver.Value, error) {
	return "value", nil
}

type pointerReceiverNilValuer struct{}

func (*pointerReceiverNilValuer) Value() (driver.Value, error) {
	return "pointer", nil
}

func TestMapEncodeNilDriverValuerCompatibility(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 7, 0)

	var valueReceiver *valueReceiverNilValuer
	buf, err := m.Encode(typeMetadata, valueReceiver, nil)
	require.NoError(t, err)
	require.Nil(t, buf)

	var pointerReceiver *pointerReceiverNilValuer
	buf, err = m.Encode(typeMetadata, pointerReceiver, nil)
	require.NoError(t, err)
	require.Equal(t, "pointer", string(buf))
}

func TestMapPlanEncodeNilDriverValuerWithValueReceiverIsNull(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 5, 0)
	var value *valueReceiverNilValuer

	plan := m.PlanEncode(typeMetadata, value)
	require.NotNil(t, plan)

	buf, err := plan.Encode(value, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

func TestMapEncodeDatabaseValuerStringWithUnknownTypeIDUsesDefaultText(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, driverValuerFunc(func() (driver.Value, error) { return "foo", nil }), nil)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), buf)
}

func TestMapEncodeStringWithUnknownTypeIDUsesDefaultText(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 3, 0)
	buf, err := m.Encode(typeMetadata, "foo", nil)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), buf)
}

func TestMapEncodeByteSliceWithUnknownTypeIDUsesDefaultOctets(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 4, 0)
	buf, err := m.Encode(typeMetadata, []byte{0, 1, 2, 3}, nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 1, 2, 3}, buf)
}

func TestMapEncodeNamedByteSliceIntoTextUsesUnderlyingBytes(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 14, 0)
	buf, err := m.Encode(typeMetadata, json.RawMessage(`{"foo": "bar"}`), nil)
	require.NoError(t, err)
	require.Equal(t, []byte(`{"foo": "bar"}`), buf)
}

func TestMapScanPointerToRenamedType(t *testing.T) {
	srcBuf := []byte("foo")
	m := fbtype.NewMap()

	var rs *_string
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	err := m.Scan(typeMetadata, srcBuf, &rs)
	assert.NoError(t, err)
	require.NotNil(t, rs)
	assert.Equal(t, "foo", string(*rs))
}

func TestMapScanNullToWrongType(t *testing.T) {
	m := fbtype.NewMap()

	var n *int32
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 0, 0)
	err := m.Scan(typeMetadata, nil, &n)
	assert.NoError(t, err)
	assert.Nil(t, n)

	var pn fbtype.Int4
	err = m.Scan(typeMetadata, nil, &pn)
	assert.NoError(t, err)
	assert.False(t, pn.Valid)
}

func TestMapScanArrayToSliceOfRenamedUint8(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999990)
	m.RegisterType(&fbtype.Type{
		Name:   "array_setter_scan_renamed_uint8_test",
		TypeID: typeID,
		Codec: arraySetterScanCodec{
			dimensions: []fbtype.ArrayDimension{{LowerBound: 1, Length: 2}},
			values:     []any{uint8(2), uint8(4)},
		},
	})

	var got []_uint8
	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.NoError(t, err)
	assert.Equal(t, []_uint8{2, 4}, got)

	err = m.Scan(typeMetadata, nil, &got)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMapScanBinaryToBool(t *testing.T) {
	m := fbtype.NewMap()

	var b bool

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_BOOLEAN, 0, 1, 0)
	err := m.Scan(typeMetadata, []byte{1}, &b)
	require.NoError(t, err)
	require.True(t, b)

	err = m.Scan(typeMetadata, []byte{0}, &b)
	require.NoError(t, err)
	require.False(t, b)
}

func TestMapScanBinaryToBoolError_InvalidLength(t *testing.T) {
	m := fbtype.NewMap()
	var b bool

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_BOOLEAN, 0, 1, 0)
	err := m.Scan(typeMetadata, []byte{0, 1}, &b)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid length for bool")
}

// PlanScan previously used a cache to improve performance. However, a cache
// keyed only by Firebird metadata could reuse a plan selected for one target
// type with a different target type.
func TestCachedPlanScanConfusion(t *testing.T) {
	m := fbtype.NewMap()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)

	var first any
	err := m.Scan(typeMetadata, []byte{0, 0, 0, 42}, &first)
	require.NoError(t, err)
	require.Equal(t, int32(42), first)

	var second int32
	err = m.Scan(typeMetadata, []byte{0, 0, 0, 7}, &second)
	require.NoError(t, err)
	require.Equal(t, int32(7), second)
}

func BenchmarkMapScanInt4IntoBinaryDecoder(b *testing.B) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v fbtype.Int4

	for i := 0; i < b.N; i++ {
		v = fbtype.Int4{}
		typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)
		err := m.Scan(typeMetadata, src, &v)
		if err != nil {
			b.Fatal(err)
		}
		if v != (fbtype.Int4{Int32: 42, Valid: true}) {
			b.Fatal("scan failed due to bad value")
		}
	}
}

func BenchmarkMapScanInt4IntoGoInt32(b *testing.B) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v int32

	for i := 0; i < b.N; i++ {
		v = 0
		typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)
		err := m.Scan(typeMetadata, src, &v)
		if err != nil {
			b.Fatal(err)
		}
		if v != 42 {
			b.Fatal("scan failed due to bad value")
		}
	}
}

func BenchmarkScanPlanScanInt4IntoBinaryDecoder(b *testing.B) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v fbtype.Int4

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)
	plan := m.PlanScan(typeMetadata, &v)

	for i := 0; i < b.N; i++ {
		v = fbtype.Int4{}
		err := plan.Scan(src, &v)
		if err != nil {
			b.Fatal(err)
		}
		if v != (fbtype.Int4{Int32: 42, Valid: true}) {
			b.Fatal("scan failed due to bad value")
		}
	}
}

func BenchmarkScanPlanScanInt4IntoGoInt32(b *testing.B) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v int32

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)
	plan := m.PlanScan(typeMetadata, &v)

	for i := 0; i < b.N; i++ {
		v = 0
		err := plan.Scan(src, &v)
		if err != nil {
			b.Fatal(err)
		}
		if v != 42 {
			b.Fatal("scan failed due to bad value")
		}
	}
}

func isExpectedEq(a any) func(any) bool {
	return func(v any) bool {
		return a == v
	}
}
