package fbtype_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

var defaultConnTestRunner fbxtest.ConnTestRunner
var (
	serverMajorVersionCheckOnce sync.Once
	serverMajorVersionValue     int
	engineMajorRegexp           = regexp.MustCompile(`\d+`)
)

func init() {
	defaultConnTestRunner = fbxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		return config
	}
}

func getServerMajorVersionForConn(ctx context.Context, t testing.TB, conn interface {
	QueryRow(ctx context.Context, sql string, args ...any) fbx.Row
}) int {
	t.Helper()
	serverMajorVersionCheckOnce.Do(func() {
		var version string
		err := conn.QueryRow(ctx, "select rdb$get_context('SYSTEM','ENGINE_VERSION') from rdb$database").Scan(&version)
		require.NoError(t, err)

		major, ok := parseEngineMajorVersion(version)
		require.Truef(t, ok, "failed to parse engine version: %q", version)
		serverMajorVersionValue = major
	})
	return serverMajorVersionValue
}

func parseEngineMajorVersion(version string) (int, bool) {
	match := engineMajorRegexp.FindString(version)
	if match == "" {
		return 0, false
	}
	major, err := strconv.Atoi(match)
	if err != nil {
		return 0, false
	}
	return major, true
}

// Test for renamed types
type _string string
type _bool bool
type _uint8 uint8
type _int8 int8
type _int16 int16
type _int16Slice []int16
type _int32Slice []int32
type _int64Slice []int64
type _float32Slice []float32
type _float64Slice []float64
type _byteSlice []byte

type stringerInt16 int16

func (s stringerInt16) String() string {
	return fmt.Sprintf("stringer-%d", s)
}

// unregisteredTypeID represents an actual type that is not registered. Cannot use 0 because that represents that the type
// is not known (e.g. when using the simple protocol).
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

type fieldIndexScanner interface {
	ScanNull() error
	ScanIndex(i int) any
}

type fieldIndexScanCodec struct{}

func (fieldIndexScanCodec) PlanEncode(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, value any) fbtype.EncodePlan {
	return nil
}

func (fieldIndexScanCodec) PlanScan(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, target any) fbtype.ScanPlan {
	if _, ok := target.(fieldIndexScanner); ok {
		return fieldIndexScanPlan{}
	}
	return nil
}

func (fieldIndexScanCodec) DecodeDatabaseSQLValue(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, src []byte) (driver.Value, error) {
	return nil, nil
}

func (fieldIndexScanCodec) DecodeValue(m *fbtype.Map, typeMetadata fbtype.TypeMetadata, src []byte) (any, error) {
	return nil, nil
}

type fieldIndexScanPlan struct{}

func (fieldIndexScanPlan) Scan(src []byte, target any) error {
	scanner := target.(fieldIndexScanner)
	if src == nil {
		return scanner.ScanNull()
	}

	*scanner.ScanIndex(0).(*int32) = 42
	*scanner.ScanIndex(1).(*string) = "forty-two"
	return nil
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

func TestMapScanBinaryFormatInterfacePtr(t *testing.T) {
	m := fbtype.NewMap()
	var got any
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	err := m.Scan(typeMetadata, []byte("foo"), &got)
	require.NoError(t, err)
	assert.Equal(t, "foo", got)
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
	t.Skip("")
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

func TestMapScanPointerToNilStructDoesNotCrash(t *testing.T) {
	m := fbtype.NewMap()

	type myStruct struct{}
	var p *myStruct
	typeMetadata := fbtype.NewTypeMetadata(0, 0, 9, 0)
	err := m.Scan(typeMetadata, []byte("(foo,bar)"), &p)
	require.NotNil(t, err)
}

func TestMapScanUnknownTypeIDIntoSQLScanner(t *testing.T) {
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

func TestMapScanStructWrapperProvidesExportedFieldTargets(t *testing.T) {
	m := fbtype.NewMap()
	typeID := uint32(999998)
	m.RegisterType(&fbtype.Type{Name: "field_index_scan_test", TypeID: typeID, Codec: fieldIndexScanCodec{}})

	var got struct {
		ID     int32
		Name   string
		hidden string
	}

	typeMetadata := fbtype.NewTypeMetadata(typeID, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte{1}, &got)
	require.NoError(t, err)
	assert.Equal(t, int32(42), got.ID)
	assert.Equal(t, "forty-two", got.Name)
	assert.Empty(t, got.hidden)

	err = m.Scan(typeMetadata, nil, &got)
	assert.Error(t, err)
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

func TestMapScanPtrSliceWrapperSkipsMultiDimensionalSlices(t *testing.T) {
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

func TestScanPlanBinaryInt32ScanScanner(t *testing.T) {
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

func TestScanPlanInterface(t *testing.T) {
	m := fbtype.NewMap()
	src := []byte{0, 42}
	var v interface{}
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 4, 0)
	plan := m.PlanScan(typeMetadata, v)
	err := plan.Scan(src, v)
	assert.Error(t, err)
}

// func TestPointerPointerStructScan(t *testing.T) {
// 	t.Skip("need implementation")
// 	m := fbtype.NewMap()
// 	type composite struct {
// 		ID int
// 	}

// 	int4Type, _ := m.TypeForTypeID(fbtype.Int4TypeID)
// 	pgt := &fbtype.Type{
// 		Codec: &fbtype.CompositeCodec{
// 			Fields: []fbtype.CompositeCodecField{
// 				{
// 					Name: "id",
// 					Type: int4Type,
// 				},
// 			},
// 		},
// 		Name: "composite",
// 		TypeID:  215333,
// 	}
// 	m.RegisterType(pgt)

// 	var c *composite
// 	plan := m.PlanScan(pgt.TypeID, fbtype.BinaryFormatCode, &c)
// 	err := plan.Scan([]byte("(1)"), &c)
// 	require.NoError(t, err)
// 	require.Equal(t, 1, c.ID)
// }

func TestMapScanPtrToPtrToSlice(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	src := []byte("{foo,bar}")
	var v *[]string
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 0, 0)
	plan := m.PlanScan(typeMetadata, &v)
	err := plan.Scan(src, &v)
	require.NoError(t, err)
	require.Equal(t, []string{"foo", "bar"}, *v)
}

// func TestMapScanPtrToPtrToSliceOfStruct(t *testing.T) {
// 	t.Skip()
// 	type Team struct {
// 		TeamID int
// 		Name   string
// 	}

// 	// Have to use binary format because text format doesn't include type information.
// 	m := fbtype.NewMap()
// 	src := []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x8, 0xc9, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x1e, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x17, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x19, 0x0, 0x0, 0x0, 0x6, 0x74, 0x65, 0x61, 0x6d, 0x20, 0x31, 0x0, 0x0, 0x0, 0x1e, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x17, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x19, 0x0, 0x0, 0x0, 0x6, 0x74, 0x65, 0x61, 0x6d, 0x20, 0x32}
// 	var v *[]Team
// 	plan := m.PlanScan(fbtype.RecordArrayTypeID, 0, 0, 0, fbtype.BinaryFormatCode, &v)
// 	err := plan.Scan(src, &v)
// 	require.NoError(t, err)
// 	require.Equal(t, []Team{{1, "team 1"}, {2, "team 2"}}, *v)
// }

type databaseValuerString string

func (s databaseValuerString) Value() (driver.Value, error) {
	return fmt.Sprintf("%d", len(s)), nil
}

func TestMapEncodeTextFormatDatabaseValuerThatIsRenamedSimpleType(t *testing.T) {
	m := fbtype.NewMap()
	src := databaseValuerString("foo")
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 3, 0)
	buf, err := m.Encode(typeMetadata, src, nil)
	require.NoError(t, err)
	require.Equal(t, "3", string(buf))
}

type databaseValuerFmtStringer string

func (s databaseValuerFmtStringer) Value() (driver.Value, error) {
	return nil, nil
}

func (s databaseValuerFmtStringer) String() string {
	return "foobar"
}

type databaseValuerStringFormat struct {
	n int32
}

func (v databaseValuerStringFormat) Value() (driver.Value, error) {
	return fmt.Sprint(v.n), nil
}

func TestMapEncodeBinaryFormatDatabaseValuerThatReturnsString(t *testing.T) {
	m := fbtype.NewMap()
	src := databaseValuerStringFormat{n: 42}
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, src, nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 42}, buf)
}

func TestMapEncodeDatabaseValuerThatReturnsStringIntoUnregisteredTypeTextFormat(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 0, 0)
	buf, err := m.Encode(typeMetadata, driverValuerFunc(func() (driver.Value, error) { return "foo", nil }), nil)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), buf)
}

func TestMapEncodeStringIntoUnregisteredTypeTextFormat(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 3, 0)
	buf, err := m.Encode(typeMetadata, "foo", nil)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), buf)
}

func TestMapEncodeByteSliceIntoUnregisteredTypeTextFormat(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(unregisteredTypeID, 0, 4, 0)
	buf, err := m.Encode(typeMetadata, []byte{0, 1, 2, 3}, nil)
	require.NoError(t, err)
	require.Equal(t, []byte(`\x00010203`), buf)
}

func TestMapEncodeNamedTypeOfByteSliceIntoTextTextFormat(t *testing.T) {
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

func TestScanToSliceOfRenamedUint8(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	var ruint8 []_uint8
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 0, 0)
	err := m.Scan(typeMetadata, []byte("{2,4}"), &ruint8)
	assert.NoError(t, err)
	assert.Equal(t, []_uint8{2, 4}, ruint8)
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

type databaseValuerUUID [16]byte

func (v databaseValuerUUID) Value() (driver.Value, error) {
	return fmt.Sprintf("%x", v), nil
}

// PlanScan previously used a cache to improve performance. However, the cache could get confused in certain cases. The
// example below was one such failure case.
func TestCachedPlanScanConfusion(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	var err error

	var tags any
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 14, 0)
	err = m.Scan(typeMetadata, []byte("{foo,bar,baz}"), &tags)
	require.NoError(t, err)

	var cells [][]string
	typeMetadata = fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, 0, 21, 0)
	err = m.Scan(typeMetadata, []byte("{{foo,bar},{baz,quz}}"), &cells)
	require.NoError(t, err)
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

func isPtrExpectedEq(a any) func(any) bool {
	return func(v any) bool {
		val := reflect.ValueOf(v)
		return a == val.Elem().Interface()
	}
}
