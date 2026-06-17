package fbx_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestDateTranscode(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		dates := []time.Time{
			time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(1600, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(1700, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(1800, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2001, 1, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2004, 2, 29, 0, 0, 0, 0, time.UTC),
			time.Date(2013, 7, 4, 0, 0, 0, 0, time.UTC),
			time.Date(2013, 12, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2081, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2096, 2, 29, 0, 0, 0, 0, time.UTC),
			time.Date(2550, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC),
		}

		for _, actualDate := range dates {
			var d time.Time

			err := conn.QueryRow(context.Background(), "select cast(? as date) from rdb$database", actualDate).Scan(&d)
			if err != nil {
				t.Fatalf("Unexpected failure on QueryRow Scan: %v", err)
			}
			if !actualDate.Equal(d) {
				t.Errorf("Did not transcode date successfully: %v is not %v", d, actualDate)
			}
		}
	})
}

func TestTimestampTzTranscode(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, t, conn) <= 3 {
			t.Skip("timestamp with time zone is not supported by server engine version <= 3")
		}

		inputTime := time.Date(2013, 1, 2, 3, 4, 5, 600000, time.Local)

		var outputTime time.Time

		err := conn.QueryRow(context.Background(), "select cast(? as timestamp with time zone) from rdb$database", inputTime).Scan(&outputTime)
		if err != nil {
			t.Fatalf("QueryRow Scan failed: %v", err)
		}
		if !inputTime.Equal(outputTime) {
			t.Errorf("Did not transcode time successfully: %v is not %v", outputTime, inputTime)
		}
	})
}

func TestPointerPointer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		type allTypes struct {
			s   *string
			i16 *int16
			i32 *int32
			i64 *int64
			f32 *float32
			f64 *float64
			b   *bool
			t   *time.Time
		}

		var actual, zero, expected allTypes

		{
			s := "foo"
			expected.s = &s
			i16 := int16(1)
			expected.i16 = &i16
			i32 := int32(1)
			expected.i32 = &i32
			i64 := int64(1)
			expected.i64 = &i64
			f32 := float32(1.23)
			expected.f32 = &f32
			f64 := float64(1.23)
			expected.f64 = &f64
			b := true
			expected.b = &b
			t := time.Unix(123, 5000)
			expected.t = &t
		}

		tests := []struct {
			sql       string
			queryArgs []any
			scanArgs  []any
			expected  allTypes
		}{
			{"select cast(? as varchar(255)) from rdb$database", []any{expected.s}, []any{&actual.s}, allTypes{s: expected.s}},
			{"select cast(? as varchar(255)) from rdb$database", []any{zero.s}, []any{&actual.s}, allTypes{}},
			{"select cast(? as smallint) from rdb$database", []any{expected.i16}, []any{&actual.i16}, allTypes{i16: expected.i16}},
			{"select cast(? as smallint) from rdb$database", []any{zero.i16}, []any{&actual.i16}, allTypes{}},
			{"select cast(? as integer) from rdb$database", []any{expected.i32}, []any{&actual.i32}, allTypes{i32: expected.i32}},
			{"select cast(? as integer) from rdb$database", []any{zero.i32}, []any{&actual.i32}, allTypes{}},
			{"select cast(? as bigint) from rdb$database", []any{expected.i64}, []any{&actual.i64}, allTypes{i64: expected.i64}},
			{"select cast(? as bigint) from rdb$database", []any{zero.i64}, []any{&actual.i64}, allTypes{}},
			{"select cast(? as float) from rdb$database", []any{expected.f32}, []any{&actual.f32}, allTypes{f32: expected.f32}},
			{"select cast(? as float) from rdb$database", []any{zero.f32}, []any{&actual.f32}, allTypes{}},
			{"select cast(? as double precision) from rdb$database", []any{expected.f64}, []any{&actual.f64}, allTypes{f64: expected.f64}},
			{"select cast(? as double precision) from rdb$database", []any{zero.f64}, []any{&actual.f64}, allTypes{}},
			{"select cast(? as boolean) from rdb$database", []any{expected.b}, []any{&actual.b}, allTypes{b: expected.b}},
			{"select cast(? as boolean) from rdb$database", []any{zero.b}, []any{&actual.b}, allTypes{}},
			//FIXME: fbx need properly converting to sqldata buffer
			// {"select cast(? as timestamp with time zone) from rdb$database", []any{expected.t}, []any{&actual.t}, allTypes{t: expected.t}},
			// {"select cast(? as timestamp with time zone) from rdb$database", []any{zero.t}, []any{&actual.t}, allTypes{}},
		}

		for i, tt := range tests {
			actual = zero

			err := conn.QueryRow(context.Background(), tt.sql, tt.queryArgs...).Scan(tt.scanArgs...)
			if err != nil {
				t.Errorf("%d. Unexpected failure: %v (sql -> %v, queryArgs -> %v)", i, err, tt.sql, tt.queryArgs)
			}

			assert.Equal(t, tt.expected.s, actual.s)
			assert.Equal(t, tt.expected.i16, actual.i16)
			assert.Equal(t, tt.expected.i32, actual.i32)
			assert.Equal(t, tt.expected.i64, actual.i64)
			assert.Equal(t, tt.expected.f32, actual.f32)
			assert.Equal(t, tt.expected.f64, actual.f64)
			assert.Equal(t, tt.expected.b, actual.b)
			if tt.expected.t != nil || actual.t != nil {
				assert.True(t, tt.expected.t.Equal(*actual.t))
			}

			ensureConnValid(t, conn)
		}
	})
}

func TestPointerPointerNonZero(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		f := "foo"
		dest := &f

		err := conn.QueryRow(ctx, "select cast(? as varchar(255)) from rdb$database", nil).Scan(&dest)
		if err != nil {
			t.Errorf("Unexpected failure scanning: %v", err)
		}
		if dest != nil {
			t.Errorf("Expected dest to be nil, got %#v", dest)
		}
	})
}

func TestEncodeTypeRename(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		type _int int
		inInt := _int(1)
		var outInt _int

		type _int8 int8
		inInt8 := _int8(2)
		var outInt8 _int8

		type _int16 int16
		inInt16 := _int16(3)
		var outInt16 _int16

		type _int32 int32
		inInt32 := _int32(4)
		var outInt32 _int32

		type _int64 int64
		inInt64 := _int64(5)
		var outInt64 _int64

		type _uint uint
		inUint := _uint(6)
		var outUint _uint

		type _uint8 uint8
		inUint8 := _uint8(7)
		var outUint8 _uint8

		type _uint16 uint16
		inUint16 := _uint16(8)
		var outUint16 _uint16

		type _uint32 uint32
		inUint32 := _uint32(9)
		var outUint32 _uint32

		type _uint64 uint64
		inUint64 := _uint64(10)
		var outUint64 _uint64

		type _string string
		inString := _string("foo")
		var outString _string

		type _bool bool
		inBool := _bool(true)
		var outBool _bool

		// fbx.QueryExecModeExec requires all types to be registered.
		conn.TypeMap().RegisterDefaultFbType(inInt, "integer")
		conn.TypeMap().RegisterDefaultFbType(inInt8, "smallint")
		conn.TypeMap().RegisterDefaultFbType(inInt16, "smallint")
		conn.TypeMap().RegisterDefaultFbType(inInt32, "integer")
		conn.TypeMap().RegisterDefaultFbType(inInt64, "bigint")
		conn.TypeMap().RegisterDefaultFbType(inUint, "integer")
		conn.TypeMap().RegisterDefaultFbType(inUint8, "smallint")
		conn.TypeMap().RegisterDefaultFbType(inUint16, "smallint")
		conn.TypeMap().RegisterDefaultFbType(inUint32, "integer")
		conn.TypeMap().RegisterDefaultFbType(inUint64, "bigint")
		conn.TypeMap().RegisterDefaultFbType(inString, "varchar")
		conn.TypeMap().RegisterDefaultFbType(inBool, "boolean")

		err := conn.QueryRow(ctx, "select cast(? as bigint), cast(? as bigint), cast(? as smallint), cast(? as integer), cast(? as bigint), cast(? as bigint), cast(? as bigint), cast(? as bigint), cast(? as bigint), cast(? as bigint), cast(? as varchar(255)), cast(? as boolean) from rdb$database",
			inInt, inInt8, inInt16, inInt32, inInt64, inUint, inUint8, inUint16, inUint32, inUint64, inString, inBool,
		).Scan(&outInt, &outInt8, &outInt16, &outInt32, &outInt64, &outUint, &outUint8, &outUint16, &outUint32, &outUint64, &outString, &outBool)
		if err != nil {
			t.Fatalf("Failed with type rename: %v", err)
		}

		if inInt != outInt {
			t.Errorf("int rename: expected %v, got %v", inInt, outInt)
		}

		if inInt8 != outInt8 {
			t.Errorf("int8 rename: expected %v, got %v", inInt8, outInt8)
		}

		if inInt16 != outInt16 {
			t.Errorf("int16 rename: expected %v, got %v", inInt16, outInt16)
		}

		if inInt32 != outInt32 {
			t.Errorf("int32 rename: expected %v, got %v", inInt32, outInt32)
		}

		if inInt64 != outInt64 {
			t.Errorf("int64 rename: expected %v, got %v", inInt64, outInt64)
		}

		if inUint != outUint {
			t.Errorf("uint rename: expected %v, got %v", inUint, outUint)
		}

		if inUint8 != outUint8 {
			t.Errorf("uint8 rename: expected %v, got %v", inUint8, outUint8)
		}

		if inUint16 != outUint16 {
			t.Errorf("uint16 rename: expected %v, got %v", inUint16, outUint16)
		}

		if inUint32 != outUint32 {
			t.Errorf("uint32 rename: expected %v, got %v", inUint32, outUint32)
		}

		if inUint64 != outUint64 {
			t.Errorf("uint64 rename: expected %v, got %v", inUint64, outUint64)
		}

		if inString != outString {
			t.Errorf("string rename: expected %v, got %v", inString, outString)
		}

		if inBool != outBool {
			t.Errorf("bool rename: expected %v, got %v", inBool, outBool)
		}
	})
}

func TestRowsScanNilThenScanValue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		sql := `select cast(null as integer) as a, cast(null as integer) as b from rdb$database
union all
select 1, 2 from rdb$database
`
		rows, err := conn.Query(ctx, sql)
		require.NoError(t, err)

		require.True(t, rows.Next())

		err = rows.Scan(nil, nil)
		require.NoError(t, err)

		require.True(t, rows.Next())

		var a int
		var b int
		err = rows.Scan(&a, &b)
		require.NoError(t, err)

		require.EqualValues(t, 1, a)
		require.EqualValues(t, 2, b)

		rows.Close()
		require.NoError(t, rows.Err())
	})
}

func TestScanIntoByteSlice(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	// Success cases
	for _, tt := range []struct {
		name   string
		sql    string
		output []byte
	}{
		{"int - binary", "select 42 from rdb$database", []byte("42")},
		{"text - binary", "select cast('hi' as varchar(2)) from rdb$database", []byte("hi")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			err := conn.QueryRow(context.Background(), tt.sql).Scan(&buf)
			require.NoError(t, err)
			require.Equal(t, tt.output, buf)
		})
	}
}

func TestEncodeIntoNullField(t *testing.T) {
	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)

	rows, err := conn.Query(context.Background(), "select first_name from phone_list where ? is null", nil)
	require.NoError(t, err)
	res, err := fbx.CollectRows(rows, func(row fbx.CollectableRow) (string, error) {
		var s string
		err := row.Scan(&s)
		return s, err
	})
	require.NoError(t, err)
	assert.NotZero(t, len(res))

	rows, err = conn.Query(context.Background(), "select first_name from phone_list where 1 = ? and ? is null and 3 = ?", 1, 2, 3)
	require.NoError(t, err)
	res, err = fbx.CollectRows(rows, func(row fbx.CollectableRow) (string, error) {
		var s string
		err := row.Scan(&s)
		return s, err
	})
	require.NoError(t, err)
	assert.Zero(t, len(res))
}
