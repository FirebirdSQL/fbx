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

func TestArray(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table single_dim(id integer, val smallint[1:3])")
		for i, tt := range []struct {
			input    any
			expected any
		}{
			{[]int16(nil), []int16(nil)},
			{[]int16{}, []int16{0, 0, 0}},
			{[]int16{1, 2, 3}, []int16{1, 2, 3}},
		} {
			var actual []int16
			_, err := conn.Exec(
				ctx,
				"insert into single_dim(id, val) values(?, ?)",
				i,
				tt.input,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from single_dim where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)

		}
	})
}

func TestArrayFlatArrayVarchar(t *testing.T) {
	testCases := []struct {
		input    []string
		expected []string
	}{
		{nil, []string(nil)},
		{[]string{}, []string{"", "", "", "", "", ""}},
		{[]string{"a"}, []string{"a", "", "", "", "", ""}},
		{[]string{"a", "b"}, []string{"a", "b", "", "", "", ""}},
		{[]string{"\v", "\t", "\n", "\r", "\f", " "}, []string{"\v", "\t", "\n", "\r", "\f", " "}},
		{[]string{"a\vb", "a\tb", "a\nb", "a\rb", "a\fb", "a b"}, []string{"a\vb", "a\tb", "a\nb", "a\rb", "a\fb", "a b"}},
	}

	queryModes := []fbx.QueryExecMode{fbx.QueryExecModeExec}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table single_dim_varchar(id  integer, val varchar(10)[1:6])")
		for i, testCase := range testCases {
			for _, queryMode := range queryModes {
				var out []string
				_, err := conn.Exec(
					ctx,
					"insert into single_dim_varchar(id, val) values(?, ?)",
					i,
					testCase.input,
				)
				if err != nil {
					t.Fatalf("i=%d input=%#v queryMode=%s: Scan failed: %s",
						i, testCase.input, queryMode, err)
				}

				err = conn.QueryRow(
					ctx,
					"select val from single_dim_varchar where id = ?",
					i,
				).Scan(&out)

				if err != nil {
					t.Fatalf("i=%d input=%#v queryMode=%s: Scan failed: %s",
						i, testCase.input, queryMode, err)
				}
				if !reflect.DeepEqual(out, testCase.expected) {
					t.Errorf("i=%d input=%#v queryMode=%s: not equal output=%#v",
						i, testCase.expected, queryMode, out)
				}
			}
		}
	})
}

func TestArrayArray(t *testing.T) {
	ctr := defaultConnTestRunner
	ctr.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
	}

	ctr.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table two_dim_Array(id integer, val smallint[2,2])")
		for i, tt := range []struct {
			value any
		}{
			{fbtype.Array[int32]{
				Elements: []int32{1, 2, 3, 4},
				Dims: []fbtype.ArrayDimension{
					{LowerBound: 1, Length: 2},
					{LowerBound: 1, Length: 2},
				},
				Valid: true,
			}},
		} {
			var actual fbtype.Array[int32]
			_, err := conn.Exec(
				ctx,
				"insert into two_dim_Array(id, val) values(?, ?)",
				i,
				tt.value,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from two_dim_Array where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.value, actual, "%d", i)
		}
	})
}

func TestArrayNamedSliceType(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table single_dim_namedSliceType(id integer, val smallint[1:3])")
		type _int16Slice []int16

		for i, tt := range []struct {
			input    any
			expected any
		}{
			{_int16Slice(nil), _int16Slice(nil)},
			{_int16Slice{}, _int16Slice{0, 0, 0}},
			{_int16Slice{1, 2, 3}, _int16Slice{1, 2, 3}},
		} {
			var actual _int16Slice
			_, err := conn.Exec(
				ctx,
				"insert into single_dim_namedSliceType(id, val) values(?, ?)",
				i,
				tt.input,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from single_dim_namedSliceType where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.expected, actual, "%d", i)
		}
	})
}

func TestArrayAnySliceArgument(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table single_dim_AnySliceArgument(id integer, val smallint[1:3])")

		for i, tt := range []struct {
			arg   any
			value []int16
		}{
			{[]any{1, 2, 3}, []int16{1, 2, 3}},
		} {
			var actual []int16
			_, err := conn.Exec(
				ctx,
				"insert into single_dim_AnySliceArgument(id, val) values(?, ?)",
				i,
				tt.value,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from single_dim_AnySliceArgument where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.value, actual, "%d", i)
		}
	})
}

func TestArrayAnyArray(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table single_dim_AnyArray(id integer, val float[1:3])")
		type _point3 [3]float32

		for i, tt := range []struct {
			value any
		}{
			{_point3{0, 0, 0}},
			{_point3{1, 2, 3}},
		} {
			var actual _point3
			_, err := conn.Exec(
				ctx,
				"insert into single_dim_AnyArray(id, val) values(?, ?)",
				i,
				tt.value,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from single_dim_AnyArray where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt.value, actual, "%d", i)
		}
	})
}

func TestArrayEncodeMultipleDimensions(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table multi_dim_Array(val integer[3,4])")
		_, err := conn.Exec(ctx, `insert into multi_dim_Array(val) values(?)`, [][]int32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}})
		assert.NoError(t, err)

		var out [][]int32
		rows, err := conn.Query(
			ctx,
			"select val from multi_dim_Array",
		)
		assert.NoError(t, err)
		for rows.Next() {
			err = rows.Scan(&out)
			require.NoError(t, err)
			require.Equal(t, [][]int32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}, out)
		}
	})
}

func TestArrayEncodeMultipleDimensionsRagged(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table two_dim_ArrayRagged(val integer[3,4])")
		_, err := conn.Exec(ctx, `insert into two_dim_ArrayRagged(val) values(?)`, [][]int32{{1, 2, 3, 4}, {5}, {9, 10, 11, 12}})
		require.Error(t, err, "cannot convert to ArrayGetter because it is a ragged multi-dimensional")
	})
}
