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
		config, err := fbx.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
		require.NoError(t, err)
		return config
	}
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

func TestMapScanNilIsNoOp(t *testing.T) {
	m := fbtype.NewMap()

	err := m.Scan(fbproto.SQL_TYPE_TEXT, 0, 3, 0, fbx.BinaryFormatCode, []byte("foo"), nil)
	assert.NoError(t, err)
}

func TestMapScanTextFormatInterfacePtr(t *testing.T) {
	m := fbtype.NewMap()
	var got any
	err := m.Scan(fbproto.SQL_TYPE_TEXT, 0, 3, 0, fbx.BinaryFormatCode, []byte("foo"), &got)
	require.NoError(t, err)
	assert.Equal(t, "foo", got)
}

func TestMapScanBinaryFormatInterfacePtr(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	var got any
	err := m.Scan(fbproto.SQL_TYPE_TEXT, 0, 3, 0, fbx.BinaryFormatCode, []byte("foo"), &got)
	require.NoError(t, err)
	assert.Equal(t, "foo", got)
}

func TestMapScanUnknownTypeIDToStringsAndBytes(t *testing.T) {
	t.Skip("")
	unknownTypeID := uint32(999999)
	srcBuf := []byte("foo")
	m := fbtype.NewMap()

	var s string
	err := m.Scan(unknownTypeID, 0, 3, 0, fbx.TextFormatCode, srcBuf, &s)
	assert.NoError(t, err)
	assert.Equal(t, "foo", s)

	var rs _string
	err = m.Scan(unknownTypeID, 0, 3, 0, fbx.TextFormatCode, srcBuf, &rs)
	assert.NoError(t, err)
	assert.Equal(t, "foo", string(rs))

	var b []byte
	err = m.Scan(unknownTypeID, 0, 3, 0, fbx.BinaryFormatCode, srcBuf, &b)
	assert.NoError(t, err)
	assert.Equal(t, []byte("foo"), b)

	var rb _byteSlice
	err = m.Scan(unknownTypeID, 0, 3, 0, fbx.BinaryFormatCode, srcBuf, &rb)
	assert.NoError(t, err)
	assert.Equal(t, []byte("foo"), []byte(rb))
}

func TestMapScanPointerToNilStructDoesNotCrash(t *testing.T) {
	m := fbtype.NewMap()

	type myStruct struct{}
	var p *myStruct
	err := m.Scan(0, 0, 9, 0, fbx.BinaryFormatCode, []byte("(foo,bar)"), &p)
	require.NotNil(t, err)
}

func TestMapScanUnknownTypeIDIntoSQLScanner(t *testing.T) {
	m := fbtype.NewMap()

	var s sql.NullString
	err := m.Scan(0, 0, 0, 0, fbx.BinaryFormatCode, []byte(nil), &s)
	assert.NoError(t, err)
	assert.Equal(t, "", s.String)
	assert.False(t, s.Valid)
}

type scannerString string

func (ss *scannerString) Scan(v any) error {
	*ss = scannerString("scanned")
	return nil
}

func TestMapScanUnregisteredTypeIDIntoRenamedStringSQLScanner(t *testing.T) {
	m := fbtype.NewMap()

	var s scannerString
	err := m.Scan(unregisteredTypeID, 0, 0, 0, fbx.BinaryFormatCode, []byte(nil), &s)
	assert.NoError(t, err)
	assert.Equal(t, "scanned", string(s))
}

type pgCustomInt int64

func (ci *pgCustomInt) Scan(src interface{}) error {
	*ci = pgCustomInt(src.(int64))
	return nil
}

func TestScanPlanBinaryInt32ScanScanner(t *testing.T) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v pgCustomInt

	plan := m.PlanScan(fbproto.SQL_TYPE_SHORT, 0, 4, 0, fbtype.BinaryFormatCode, &v)
	err := plan.Scan(src, &v)
	require.NoError(t, err)
	require.EqualValues(t, 42, v)

	ptr := new(pgCustomInt)
	plan = m.PlanScan(fbproto.SQL_TYPE_SHORT, 0, 4, 0, fbtype.BinaryFormatCode, &ptr)
	err = plan.Scan(src, &ptr)
	require.NoError(t, err)
	require.EqualValues(t, 42, *ptr)

	ptr = new(pgCustomInt)
	err = plan.Scan(nil, &ptr)
	require.NoError(t, err)
	assert.Nil(t, ptr)

	ptr = nil
	plan = m.PlanScan(fbproto.SQL_TYPE_SHORT, 0, 4, 0, fbtype.BinaryFormatCode, &ptr)
	err = plan.Scan(src, &ptr)
	require.NoError(t, err)
	require.EqualValues(t, 42, *ptr)

	ptr = nil
	plan = m.PlanScan(fbproto.SQL_TYPE_SHORT, 0, 4, 0, fbtype.BinaryFormatCode, &ptr)
	err = plan.Scan(nil, &ptr)
	require.NoError(t, err)
	assert.Nil(t, ptr)
}

func TestScanPlanInterface(t *testing.T) {
	m := fbtype.NewMap()
	src := []byte{0, 42}
	var v interface{}
	plan := m.PlanScan(fbproto.SQL_TYPE_SHORT, 0, 4, 0, fbtype.BinaryFormatCode, v)
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
	plan := m.PlanScan(fbproto.SQL_TYPE_TEXT, 0, 0, 0, fbtype.BinaryFormatCode, &v)
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
	buf, err := m.Encode(fbproto.SQL_TYPE_TEXT, 0, 3, 0, fbtype.BinaryFormatCode, src, nil)
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

func TestMapEncodeTextFormatDatabaseValuerThatIsFmtStringer(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	src := databaseValuerFmtStringer("")
	buf, err := m.Encode(fbproto.SQL_TYPE_TEXT, 0, 0, 0, fbtype.BinaryFormatCode, src, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
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
	buf, err := m.Encode(fbproto.SQL_TYPE_SHORT, 0, 0, 0, fbtype.BinaryFormatCode, src, nil)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 42}, buf)
}

func TestMapEncodeDatabaseValuerThatReturnsStringIntoUnregisteredTypeTextFormat(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	buf, err := m.Encode(unregisteredTypeID, 0, 0, 0, fbtype.BinaryFormatCode, driverValuerFunc(func() (driver.Value, error) { return "foo", nil }), nil)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), buf)
}

func TestMapEncodeStringIntoUnregisteredTypeTextFormat(t *testing.T) {
	m := fbtype.NewMap()
	buf, err := m.Encode(unregisteredTypeID, 0, 3, 0, fbtype.BinaryFormatCode, "foo", nil)
	require.NoError(t, err)
	require.Equal(t, []byte("foo"), buf)
}

func TestMapEncodeByteSliceIntoUnregisteredTypeTextFormat(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	buf, err := m.Encode(unregisteredTypeID, 0, 4, 0, fbtype.BinaryFormatCode, []byte{0, 1, 2, 3}, nil)
	require.NoError(t, err)
	require.Equal(t, []byte(`\x00010203`), buf)
}

func TestMapEncodeNamedTypeOfByteSliceIntoTextTextFormat(t *testing.T) {
	m := fbtype.NewMap()
	buf, err := m.Encode(fbproto.SQL_TYPE_TEXT, 0, 14, 0, fbtype.TextFormatCode, json.RawMessage(`{"foo": "bar"}`), nil)
	require.NoError(t, err)
	require.Equal(t, []byte(`{"foo": "bar"}`), buf)
}

func TestMapScanPointerToRenamedType(t *testing.T) {
	srcBuf := []byte("foo")
	m := fbtype.NewMap()

	var rs *_string
	err := m.Scan(fbproto.SQL_TYPE_TEXT, 0, 3, 0, fbx.BinaryFormatCode, srcBuf, &rs)
	assert.NoError(t, err)
	require.NotNil(t, rs)
	assert.Equal(t, "foo", string(*rs))
}

func TestMapScanNullToWrongType(t *testing.T) {
	m := fbtype.NewMap()

	var n *int32
	err := m.Scan(fbproto.SQL_TYPE_TEXT, 0, 0, 0, fbx.BinaryFormatCode, nil, &n)
	assert.NoError(t, err)
	assert.Nil(t, n)

	var pn fbtype.Int4
	err = m.Scan(fbproto.SQL_TYPE_TEXT, 0, 0, 0, fbx.BinaryFormatCode, nil, &pn)
	assert.NoError(t, err)
	assert.False(t, pn.Valid)
}

func TestScanToSliceOfRenamedUint8(t *testing.T) {
	t.Skip()
	m := fbtype.NewMap()
	var ruint8 []_uint8
	err := m.Scan(fbproto.SQL_TYPE_SHORT, 0, 0, 0, fbx.BinaryFormatCode, []byte("{2,4}"), &ruint8)
	assert.NoError(t, err)
	assert.Equal(t, []_uint8{2, 4}, ruint8)
}

func TestMapScanBinaryToBool(t *testing.T) {
	m := fbtype.NewMap()

	var b bool

	err := m.Scan(fbproto.SQL_TYPE_BOOLEAN, 0, 1, 0, fbtype.BinaryFormatCode, []byte{1}, &b)
	require.NoError(t, err)
	require.True(t, b)

	err = m.Scan(fbproto.SQL_TYPE_BOOLEAN, 0, 1, 0, fbtype.BinaryFormatCode, []byte{0}, &b)
	require.NoError(t, err)
	require.False(t, b)
}

func TestMapScanBinaryToBoolError_InvalidLength(t *testing.T) {
	m := fbtype.NewMap()
	var b bool

	err := m.Scan(fbproto.SQL_TYPE_BOOLEAN, 0, 1, 0, fbtype.BinaryFormatCode, []byte{0, 1}, &b)
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
	err = m.Scan(fbproto.SQL_TYPE_TEXT, 0, 14, 0, fbx.BinaryFormatCode, []byte("{foo,bar,baz}"), &tags)
	require.NoError(t, err)

	var cells [][]string
	err = m.Scan(fbproto.SQL_TYPE_TEXT, 0, 21, 0, fbx.BinaryFormatCode, []byte("{{foo,bar},{baz,quz}}"), &cells)
	require.NoError(t, err)
}

func BenchmarkMapScanInt4IntoBinaryDecoder(b *testing.B) {
	m := fbtype.NewMap()
	src := []byte{0, 0, 0, 42}
	var v fbtype.Int4

	for i := 0; i < b.N; i++ {
		v = fbtype.Int4{}
		err := m.Scan(fbproto.SQL_TYPE_LONG, 0, 4, 0, fbtype.BinaryFormatCode, src, &v)
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
		err := m.Scan(fbproto.SQL_TYPE_LONG, 0, 4, 0, fbtype.BinaryFormatCode, src, &v)
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

	plan := m.PlanScan(fbproto.SQL_TYPE_LONG, 0, 4, 0, fbtype.BinaryFormatCode, &v)

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

	plan := m.PlanScan(fbproto.SQL_TYPE_LONG, 0, 4, 0, fbtype.BinaryFormatCode, &v)

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
