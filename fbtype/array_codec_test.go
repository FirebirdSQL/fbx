package fbtype_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
)

func TestArrayCodec(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		for i, tt := range []struct {
			expected any
		}{
			{[]int16(nil)},
			{[]int16{}},
			{[]int16{1, 2, 3}},
		} {
			var actual []int16
			err := conn.QueryRow(
				ctx,
				"select $1::smallint[]",
				tt.expected,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)
		}

		newInt16 := func(n int16) *int16 { return &n }

		for i, tt := range []struct {
			expected any
		}{
			{[]*int16{newInt16(1), nil, newInt16(3), nil, newInt16(5)}},
		} {
			var actual []*int16
			err := conn.QueryRow(
				ctx,
				"select $1::smallint[]",
				tt.expected,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)
		}
	})
}

func TestArrayCodecFlatArrayString(t *testing.T) {
	t.Skip()
	testCases := []struct {
		input []string
	}{
		{nil},
		{[]string{}},
		{[]string{"a"}},
		{[]string{"a", "b"}},
		// previously had a bug with whitespace handling
		{[]string{"\v", "\t", "\n", "\r", "\f", " "}},
		{[]string{"a\vb", "a\tb", "a\nb", "a\rb", "a\fb", "a b"}},
	}

	queryModes := []fbx.QueryExecMode{fbx.QueryExecModeExec}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		for i, testCase := range testCases {
			for _, queryMode := range queryModes {
				var out []string
				err := conn.QueryRow(ctx, "select $1::text[]", queryMode, testCase.input).Scan(&out)
				if err != nil {
					t.Fatalf("i=%d input=%#v queryMode=%s: Scan failed: %s",
						i, testCase.input, queryMode, err)
				}
				if !reflect.DeepEqual(out, testCase.input) {
					t.Errorf("i=%d input=%#v queryMode=%s: not equal output=%#v",
						i, testCase.input, queryMode, out)
				}
			}
		}
	})
}

func TestArrayCodecArray(t *testing.T) {
	t.Skip()
	ctr := defaultConnTestRunner
	ctr.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
	}

	ctr.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		for i, tt := range []struct {
			expected any
		}{
			{fbtype.Array[int32]{
				Elements: []int32{1, 2, 3, 4},
				Dims: []fbtype.ArrayDimension{
					{Length: 2, LowerBound: 2},
					{Length: 2, LowerBound: 2},
				},
				Valid: true,
			}},
		} {
			var actual fbtype.Array[int32]
			err := conn.QueryRow(
				ctx,
				"select $1::int[]",
				tt.expected,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)
		}
	})
}

func TestArrayCodecNamedSliceType(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		type _int16Slice []int16

		for i, tt := range []struct {
			expected any
		}{
			{_int16Slice(nil)},
			{_int16Slice{}},
			{_int16Slice{1, 2, 3}},
		} {
			var actual _int16Slice
			err := conn.QueryRow(
				ctx,
				"select $1::smallint[]",
				tt.expected,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)
		}
	})
}

// https://github.com/jackc/pgx/issues/1488
func TestArrayCodecAnySliceArgument(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		type _int16Slice []int16

		for i, tt := range []struct {
			arg      any
			expected []int16
		}{
			{[]any{1, 2, 3}, []int16{1, 2, 3}},
		} {
			var actual []int16
			err := conn.QueryRow(
				ctx,
				"select $1::smallint[]",
				tt.arg,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)
		}
	})
}

// https://github.com/jackc/pgx/issues/1442
func TestArrayCodecAnyArray(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		type _point3 [3]float32

		for i, tt := range []struct {
			expected any
		}{
			{_point3{0, 0, 0}},
			{_point3{1, 2, 3}},
		} {
			var actual _point3
			err := conn.QueryRow(
				ctx,
				"select $1::float4[]",
				tt.expected,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)
		}
	})
}

func TestArrayCodecDecodeValue(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, _ testing.TB, conn *fbx.Conn) {
		for _, tt := range []struct {
			sql      string
			expected any
		}{
			{
				sql:      `select '{}'::int4[]`,
				expected: []any{},
			},
			{
				sql:      `select '{1,2}'::int8[]`,
				expected: []any{int64(1), int64(2)},
			},
			{
				sql:      `select '{foo,bar}'::text[]`,
				expected: []any{"foo", "bar"},
			},
		} {
			t.Run(tt.sql, func(t *testing.T) {
				rows, err := conn.Query(ctx, tt.sql)
				require.NoError(t, err)

				for rows.Next() {
					values, err := rows.Values()
					require.NoError(t, err)
					require.Len(t, values, 1)
					require.Equal(t, tt.expected, values[0])
				}

				require.NoError(t, rows.Err())
			})
		}
	})
}

func TestArrayCodecScanMultipleDimensions(t *testing.T) {
	t.Skip()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {

		rows, err := conn.Query(ctx, `select '{{1,2,3,4}, {5,6,7,8}, {9,10,11,12}}'::int4[]`)
		require.NoError(t, err)

		for rows.Next() {
			var ss [][]int32
			err := rows.Scan(&ss)
			require.NoError(t, err)
			require.Equal(t, [][]int32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}, ss)
		}

		require.NoError(t, rows.Err())
	})
}

func TestArrayCodecScanMultipleDimensionsEmpty(t *testing.T) {
	t.Skip()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, err := conn.Query(ctx, `select '{}'::int4[]`)
		require.NoError(t, err)

		for rows.Next() {
			var ss [][]int32
			err := rows.Scan(&ss)
			require.NoError(t, err)
			require.Equal(t, [][]int32{}, ss)
		}

		require.NoError(t, rows.Err())
	})
}

func TestArrayCodecScanWrongMultipleDimensions(t *testing.T) {
	t.Skip()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, err := conn.Query(ctx, `select '{{1,2,3,4}, {5,6,7,8}, {9,10,11,12}}'::int4[]`)
		require.NoError(t, err)

		for rows.Next() {
			var ss [][][]int32
			err := rows.Scan(&ss)
			require.Error(t, err, "can't scan into dest[0]: Firebird array has 2 dimensions but slice has 3 dimensions")
		}
	})
}

func TestArrayCodecEncodeMultipleDimensions(t *testing.T) {
	t.Skip()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, err := conn.Query(ctx, `select $1::int4[]`, [][]int32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}})
		require.NoError(t, err)

		for rows.Next() {
			var ss [][]int32
			err := rows.Scan(&ss)
			require.NoError(t, err)
			require.Equal(t, [][]int32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}, ss)
		}

		require.NoError(t, rows.Err())
	})
}

func TestArrayCodecEncodeMultipleDimensionsRagged(t *testing.T) {
	t.Skip()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, err := conn.Query(ctx, `select $1::int4[]`, [][]int32{{1, 2, 3, 4}, {5}, {9, 10, 11, 12}})
		require.Error(t, err, "cannot convert [][]int32 to ArrayGetter because it is a ragged multi-dimensional")
		defer rows.Close()
	})
}

// https://github.com/jackc/pgx/issues/1494
func TestArrayCodecDecodeTextArrayWithTextOfNULL(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		{
			var actual []string
			err := conn.QueryRow(ctx, `select '{"foo", "NULL", " NULL "}'::text[]`).Scan(&actual)
			require.NoError(t, err)
			require.Equal(t, []string{"foo", "NULL", " NULL "}, actual)
		}

		{
			var actual []fbtype.Text
			err := conn.QueryRow(ctx, `select '{"foo", "NULL", NULL, " NULL "}'::text[]`).Scan(&actual)
			require.NoError(t, err)
			require.Equal(t, []fbtype.Text{
				{String: "foo", Valid: true},
				{String: "NULL", Valid: true},
				{},
				{String: " NULL ", Valid: true},
			}, actual)
		}
	})
}

func TestArrayCodecDecodeTextArrayPrefersBinaryFormat(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		sd, err := conn.Prepare(ctx, "", `select '{"foo", "NULL", " NULL "}'::text[]`)
		require.NoError(t, err)
		require.Equal(t, int16(1), conn.TypeMap().FormatCodeForOID(sd.Fields[0].Type))
	})
}
