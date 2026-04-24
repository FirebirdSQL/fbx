package fbtype_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
)

func TestArray(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table single_dim(id integer, val smallint[1:3])")
		for i, tt := range []any{
			[]int16(nil),
			[]int16{},
			[]int16{1, 2, 3},
			[]int16{1, 2},
		} {
			var actual []int16
			_, err := conn.Exec(
				ctx,
				"insert into single_dim(id, val) values(?, ?)",
				i,
				tt,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from single_dim where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt, actual, "%d", i)

		}
	})
}

func TestArrayFlatArrayVarchar(t *testing.T) {
	testCases := [][]string{
		nil,
		{},
		{"a"},
		{"a", "b"},
		{"\v", "\t", "\n", "\r", "\f", " "},
		{"a\vb", "a\tb", "a\nb", "a\rb", "a\fb", "a b"},
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
					testCase,
				)
				if err != nil {
					t.Fatalf("i=%d input=%#v queryMode=%s: Scan failed: %s",
						i, testCase, queryMode, err)
				}

				err = conn.QueryRow(
					ctx,
					"select val from single_dim_varchar where id = ?",
					i,
				).Scan(&out)

				if err != nil {
					t.Fatalf("i=%d input=%#v queryMode=%s: Scan failed: %s",
						i, testCase, queryMode, err)
				}
				if !reflect.DeepEqual(out, testCase) {
					t.Errorf("i=%d input=%#v queryMode=%s: not equal output=%#v",
						i, testCase, queryMode, out)
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
			{fbtype.Array[int32]{
				Elements: []int32{1, 2},
				Dims: []fbtype.ArrayDimension{
					{LowerBound: 1, Length: 1},
					{LowerBound: 1, Length: 2},
				},
				Valid: true,
			}},
			{fbtype.Array[int32]{
				Elements: []int32{1, 2},
				Dims: []fbtype.ArrayDimension{
					{LowerBound: 1, Length: 2},
					{LowerBound: 1, Length: 1},
				},
				Valid: true,
			}},
			{fbtype.Array[int32]{
				Valid: false,
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

		for i, tt := range []any{
			_int16Slice(nil),
			_int16Slice{},
			_int16Slice{1, 2, 3},
			_int16Slice{1, 2},
		} {
			var actual _int16Slice
			_, err := conn.Exec(
				ctx,
				"insert into single_dim_namedSliceType(id, val) values(?, ?)",
				i,
				tt,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from single_dim_namedSliceType where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt, actual, "%d", i)
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
			{[]any(nil), []int16(nil)},
			{[]any{}, []int16{}},
			{[]any{1, 2, 3}, []int16{1, 2, 3}},
			{[]any{1, 2}, []int16{1, 2}},
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
		_, _ = conn.Exec(ctx, "recreate table multi_dim_Array(id integer, val integer[2,3,4])")
		for i, tt := range []any{
			[][][]int32(nil),
			[][][]int32{},
			[][][]int32{{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}, {{13, 14, 15, 16}, {17, 18, 19, 20}, {21, 22, 23, 24}}},
			[][][]int32{{{1, 2, 3, 4}, {5, 6, 7, 8}}, {{9, 10, 11, 12}, {13, 14, 15, 16}}},
			[][][]int32{{{1, 2}, {3, 4}, {5, 6}}, {{7, 8}, {9, 10}, {11, 12}}},
			[][][]int32{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}},
			[][][]int32{{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}},
			[][][]int32{{{1, 2}, {3, 4}, {5, 6}}},
			[][][]int32{{{1, 2}, {3, 4}}},
			[][][]int32{{{1}}, {{2}}},
		} {
			var actual [][][]int32
			_, err := conn.Exec(
				ctx,
				"insert into multi_dim_Array values(?, ?)",
				i,
				tt,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = conn.QueryRow(
				ctx,
				"select val from multi_dim_Array where id = ?",
				i,
			).Scan(&actual)
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt, actual, "%d", i)

		}
	})
}

func TestArrayEncodeMultipleDimensionsRagged(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, "recreate table three_dim_ArrayRagged(id integer, val integer[2,3,4])")
		for i, tt := range []any{
			[][][]int32{{{1, 2, 3, 4}, {5}, {9, 10, 11, 12}}},
			[][][]int32{{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}, {{13, 14, 15}, {16, 17, 18}, {19, 20, 21}}},
			[][][]int32{{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}, {{10, 11, 12, 13}, {14, 15, 16, 17}, {18, 19, 20, 21}}},
			[][][]int32{{{1, 2, 3, 4}, {5, 6, 7, 8}}, {{9, 10, 11}, {12, 13, 14}}},
			[][][]int32{{{1, 2}, {3}, {4, 5}}, {{6, 7}, {8, 9}, {10, 11}}},
			[][][]int32{{{1, 2}, {3, 4}, {5, 6}}, {{7}, {8, 9}, {10, 11}}},
			[][][]int32{{{1, 2}, {3, 4}}, {{5, 6}, {7}}},
			[][][]int32{{{1, 2}, {3}}, {{4, 5}, {6, 7}}},
			[][][]int32{{{1}}, {{2, 3}}},
		} {
			_, err := conn.Exec(
				ctx,
				"insert into three_dim_ArrayRagged values(?, ?)",
				i,
				tt,
			)
			require.EqualErrorf(t, err, "cannot convert to ArrayGetter because it is a ragged multi-dimensional", "test number %d", i)

		}
	})
}

func TestArrayInteractionWithCharArraysInDifferentEncodings(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, `recreate table different_encodings_char_array(
		id integer,
		encoding_NONE char(10)[2] character set NONE,
		encoding_ASCII char(10)[2] character set ASCII,
		encoding_UNICODE_FSS char(10)[2] character set UNICODE_FSS,
		encoding_UTF8 char(10)[2] character set UTF8
		)`)
		for i, tt := range [][]string{
			{"a"},
			{"a", "b"},
			{"\v", "\t"},
			{"a\vb", "a\tb"},
			{"tensymbol", "tensymbol"},
		} {
			_, err := conn.Exec(
				ctx,
				"insert into different_encodings_char_array values(?, ?, ?, ?, ?)",
				i,
				tt,
				tt,
				tt,
				tt,
			)
			assert.NoErrorf(t, err, "%d", i)

			var actual_NONE, actual_ASCII, actual_UNICODE_FSS, actual_UTF8 []string
			err = conn.QueryRow(
				ctx,
				"select encoding_NONE, encoding_ASCII, encoding_UNICODE_FSS, encoding_UTF8 from different_encodings_char_array where id = ?",
				i,
			).Scan(&actual_NONE, &actual_ASCII, &actual_UNICODE_FSS, &actual_UTF8)
			assert.NoErrorf(t, err, "%d", i)
			var expected []string
			for _, b := range tt {
				padding := strings.Repeat(" ", 10-len(b))
				expected = append(expected, b+padding)
			}
			assert.Equalf(t, expected, actual_NONE, "%d", i)
			assert.Equalf(t, expected, actual_ASCII, "%d", i)
			assert.Equalf(t, expected, actual_UNICODE_FSS, "%d", i)
			assert.Equalf(t, expected, actual_UTF8, "%d", i)
		}
	})
}
