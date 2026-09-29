package fbtype_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
	"rdb.red-soft.ru/fbx/stdlib"
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

func TestArrayNumericAndDecimalElementsUseSubtypeCodec(t *testing.T) {
	fractional := []fbtype.Numeric{
		mustArrayNumeric(t, "1.23"),
		mustArrayNumeric(t, "-4.56"),
		mustArrayNumeric(t, "78.90"),
	}
	whole := []fbtype.Numeric{
		mustArrayNumeric(t, "1"),
		mustArrayNumeric(t, "-4"),
		mustArrayNumeric(t, "78"),
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		const table = "array_numeric_subtype_codec"
		columns := []struct {
			name    string
			sqlType string
		}{
			{name: "numeric_smallint", sqlType: "numeric(4,2)"},
			{name: "numeric_integer", sqlType: "numeric(9,2)"},
			{name: "numeric_bigint", sqlType: "numeric(18,2)"},
			{name: "decimal_integer", sqlType: "decimal(9,2)"},
			{name: "decimal_bigint", sqlType: "decimal(18,2)"},
		}

		definitions := []string{"id integer primary key"}
		for _, column := range columns {
			definitions = append(definitions, column.name+" "+column.sqlType+"[1:3]")
		}
		_, err := conn.Exec(ctx, "recreate table "+table+" ("+strings.Join(definitions, ", ")+")")
		require.NoError(tb, err)
		defer func() {
			_, _ = conn.Exec(context.Background(), "drop table "+table)
		}()

		placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(columns)+1), ", ")
		insert := func(id int, values []fbtype.Numeric) {
			args := make([]any, 1, len(columns)+1)
			args[0] = id
			for range columns {
				args = append(args, values)
			}
			_, err := conn.Exec(ctx, "insert into "+table+" values ("+placeholders+")", args...)
			require.NoError(tb, err)
		}
		insert(1, fractional)
		insert(2, whole)

		db := stdlib.OpenDB(*conn.Config())
		defer func() { require.NoError(tb, db.Close()) }()
		sqlTypeMap := fbtype.NewMap()
		paths := []struct {
			name string
			scan func(query string, id int, target any) error
		}{
			{
				name: "fbx",
				scan: func(query string, id int, target any) error {
					return conn.QueryRow(ctx, query, id).Scan(target)
				},
			},
			{
				name: "database/sql",
				scan: func(query string, id int, target any) error {
					return db.QueryRowContext(ctx, query, id).Scan(sqlTypeMap.SQLScanner(target))
				},
			},
		}

		for _, column := range columns {
			query := "select " + column.name + " from " + table + " where id = ?"
			for _, path := range paths {
				label := fmt.Sprintf("%s through %s", column.sqlType, path.name)

				var numerics []fbtype.Numeric
				err = path.scan(query, 1, &numerics)
				require.NoError(tb, err, label)
				require.Equal(tb, fractional, numerics, label)

				var text []string
				err = path.scan(query, 1, &text)
				require.NoError(tb, err, label)
				require.Equal(tb, []string{"1.23", "-4.56", "78.90"}, text, label)

				var floats []float64
				err = path.scan(query, 1, &floats)
				require.NoError(tb, err, label)
				require.Len(tb, floats, 3, label)
				assert.InDelta(tb, 1.23, floats[0], 1e-9, label)
				assert.InDelta(tb, -4.56, floats[1], 1e-9, label)
				assert.InDelta(tb, 78.9, floats[2], 1e-9, label)

				var integers []int64
				err = path.scan(query, 2, &integers)
				require.NoError(tb, err, label)
				require.Equal(tb, []int64{1, -4, 78}, integers, label)

				err = path.scan(query, 1, &integers)
				require.ErrorContains(tb, err, "failed to scan array element 0", label)
			}
		}
	})
}

func mustArrayNumeric(t testing.TB, value string) fbtype.Numeric {
	t.Helper()
	var numeric fbtype.Numeric
	require.NoError(t, numeric.Scan(value))
	return numeric
}

func TestArrayEncodePlansAreNotMemoized(t *testing.T) {
	m := fbtype.NewMap()
	metadata := newArrayEncodePlanTestMetadata(t, []map[string]int{{"Lower": 1, "Upper": 3}})
	value := []int32{1, 2, 3}

	buf, err := m.Encode(metadata, value, nil)
	require.NoError(t, err)
	require.NotEmpty(t, buf)
	metadataCount, planCount := arrayEncodePlanCacheSize(m)

	for range 20 {
		metadata = newArrayEncodePlanTestMetadata(t, []map[string]int{{"Lower": 1, "Upper": 3}})
		buf, err = m.Encode(metadata, &value, nil)
		require.NoError(t, err)
		require.NotEmpty(t, buf)
	}

	actualMetadataCount, actualPlanCount := arrayEncodePlanCacheSize(m)
	require.Equal(t, metadataCount, actualMetadataCount)
	require.Equal(t, planCount, actualPlanCount)
}

func TestArrayEncodeDoesNotValidateFieldBounds(t *testing.T) {
	m := fbtype.NewMap()

	twoElements := newArrayEncodePlanTestMetadata(t, []map[string]int{{"Lower": 1, "Upper": 2}})
	value := fbtype.Array[int32]{
		Elements: []int32{1, 2, 3},
		Dims:     []fbtype.ArrayDimension{{LowerBound: 0, Length: 3}},
		Valid:    true,
	}
	plan := m.PlanEncode(twoElements, value)
	require.NotNil(t, plan)
	buf, err := plan.Encode(value, nil)
	require.NoError(t, err)
	require.Len(t, buf, 12)
}

func newArrayEncodePlanTestMetadata(t *testing.T, dimensions []map[string]int) fbtype.TypeMetadata {
	t.Helper()
	arrayData, err := fbtype.NewArrayData(fbproto.SQL_TYPE_LONG, 0, 4, 0, dimensions)
	require.NoError(t, err)
	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_ARRAY, 0, 8, 0)
	metadata.SetArrayMetadata(arrayData)
	return metadata
}

func arrayEncodePlanCacheSize(m *fbtype.Map) (metadataCount, planCount int) {
	cache := reflect.ValueOf(m).Elem().FieldByName("memoizedEncodePlans")
	for _, key := range cache.MapKeys() {
		planCount += cache.MapIndex(key).Len()
	}
	return cache.Len(), planCount
}

func TestArrayUsesExecutionTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		const table = "array_execution_transaction"
		_, err := conn.Exec(ctx, "recreate table "+table+" (id integer primary key, val integer[1:3])")
		require.NoError(t, err)
		defer func() {
			_, _ = conn.Exec(context.Background(), "drop table "+table)
		}()

		txA, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = txA.Rollback(context.Background()) }()

		txB, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = txB.Rollback(context.Background()) }()

		first := []int32{1, 2, 3}
		_, err = txA.Exec(ctx, "insert into "+table+" (id, val) values (?, ?)", 1, first)
		require.NoError(t, err)

		var actual []int32
		err = txA.QueryRow(ctx, "select val from "+table+" where id = ?", 1).Scan(&actual)
		require.NoError(t, err)
		require.Equal(t, first, actual)

		require.NoError(t, txB.Rollback(ctx))

		second := []int32{4, 5, 6}
		_, err = txA.Exec(ctx, "insert into "+table+" (id, val) values (?, ?)", 2, second)
		require.NoError(t, err)

		actual = nil
		err = txA.QueryRow(ctx, "select val from "+table+" where id = ?", 2).Scan(&actual)
		require.NoError(t, err)
		require.Equal(t, second, actual)

		require.NoError(t, txA.Commit(ctx))
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
				tt.arg,
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

func TestArrayMetadataUsesSchema(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) < 6 {
			tb.Skip("schemas are supported only in Firebird/RedDatabase 6+")
		}

		_, _ = conn.Exec(ctx, `drop table if exists FBX_ARRAY_SCHEMA_1.T_ARRAY_META`)
		_, _ = conn.Exec(ctx, `drop table if exists FBX_ARRAY_SCHEMA_2.T_ARRAY_META`)
		_, _ = conn.Exec(ctx, `drop schema if exists FBX_ARRAY_SCHEMA_1`)
		_, _ = conn.Exec(ctx, `drop schema if exists FBX_ARRAY_SCHEMA_2`)

		require.NoError(tb, execNoStatus(ctx, conn, `create schema FBX_ARRAY_SCHEMA_1`))
		require.NoError(tb, execNoStatus(ctx, conn, `create schema FBX_ARRAY_SCHEMA_2`))

		require.NoError(tb, execNoStatus(ctx, conn, `
			create table FBX_ARRAY_SCHEMA_1.T_ARRAY_META (
				ID integer,
				VAL smallint[1:2]
			)
		`))
		require.NoError(tb, execNoStatus(ctx, conn, `
			create table FBX_ARRAY_SCHEMA_2.T_ARRAY_META (
				ID integer,
				VAL smallint[1:3]
			)
		`))

		rows, err := conn.Query(ctx, `
			select S1.VAL, S2.VAL
			from FBX_ARRAY_SCHEMA_1.T_ARRAY_META S1
			cross join FBX_ARRAY_SCHEMA_2.T_ARRAY_META S2
		`)
		require.NoError(tb, err)
		defer rows.Close()

		fields := rows.FieldDescriptions()
		require.Len(tb, fields, 2)
		require.NotNil(tb, fields[0].ArrayData)
		require.NotNil(tb, fields[1].ArrayData)

		assert.Equal(tb, []map[string]int{{"Lower": 1, "Upper": 2}}, fields[0].ArrayData.Dimensions)
		assert.Equal(tb, []map[string]int{{"Lower": 1, "Upper": 3}}, fields[1].ArrayData.Dimensions)

		require.Equal(tb, "FBX_ARRAY_SCHEMA_1", fields[0].Schema)
		require.Equal(tb, "FBX_ARRAY_SCHEMA_2", fields[1].Schema)

		assert.Equal(tb, "FBX_ARRAY_SCHEMA_1", fields[0].ArrayData.Schema)
		assert.Equal(tb, "FBX_ARRAY_SCHEMA_2", fields[1].ArrayData.Schema)
	})
}

func execNoStatus(ctx context.Context, conn *fbx.Conn, sql string, args ...any) error {
	_, err := conn.Exec(ctx, sql, args...)
	return err
}

func TestArraySingleQuoteInName(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, _ = conn.Exec(ctx, `recreate table "WEIRD'NAME"(id integer, val smallint[1:3])`)
		input := []int16{1, 2, 3}

		var output []int16
		_, err := conn.Exec(
			ctx,
			`insert into "WEIRD'NAME"(id, val) values(?, ?)`,
			1,
			input,
		)
		assert.NoError(t, err)

		err = conn.QueryRow(
			ctx,
			`select val from "WEIRD'NAME" where id = ?`,
			1,
		).Scan(&output)
		assert.NoError(t, err)
		assert.Equal(t, input, output)
	})
}

func TestArrayDecoding(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	defaultConnTestRunner.RunTest(ctx, t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Exec(ctx, `recreate table TestArrayDecoding(id int,
		boolArr boolean[3],
		smallintArr smallint[4],
		intArr integer[4],
		bigintArr bigint[4],
		floatArr float[4],
		doubleArr double precision[4],
		varcharArr varchar(5)[3],
		timestampArr timestamp[2])`)
		assert.NoError(t, err)

		tests := []struct {
			sqlInsert string
			sqlSelect string
			query     any
			scan      any
			assert    func(testing.TB, any, any)
		}{
			{
				"insert into TestArrayDecoding(id, boolArr) values(?, ?)",
				"select boolArr from TestArrayDecoding where id = ?",
				[]bool{true, false, true}, &[]bool{},
				func(t testing.TB, query, scan any) {
					if !reflect.DeepEqual(query, *(scan.(*[]bool))) {
						t.Errorf("failed to encode bool[]")
					}
				},
			},
			{
				"insert into TestArrayDecoding(id, smallintArr) values(?, ?)",
				"select smallintArr from TestArrayDecoding where id = ?",
				[]int16{2, 4, 484, 32767}, &[]int16{},
				func(t testing.TB, query, scan any) {
					if !reflect.DeepEqual(query, *(scan.(*[]int16))) {
						t.Errorf("failed to encode smallint[]")
					}
				},
			},
			{
				"insert into TestArrayDecoding(id, intArr) values(?, ?)",
				"select intArr from TestArrayDecoding where id = ?",
				[]uint32{2, 4, 484, 2147483647}, &[]uint32{},
				func(t testing.TB, query, scan any) {
					if !reflect.DeepEqual(query, *(scan.(*[]uint32))) {
						t.Errorf("failed to encode int[]")
					}
				},
			},
			{
				"insert into TestArrayDecoding(id, bigintArr) values(?, ?)",
				"select bigintArr from TestArrayDecoding where id = ?",
				[]uint64{2, 4, 484, 9223372036854775807}, &[]uint64{},
				func(t testing.TB, query, scan any) {
					if !reflect.DeepEqual(query, *(scan.(*[]uint64))) {
						t.Errorf("failed to encode bigint[]")
					}
				},
			},
			{
				"insert into TestArrayDecoding(id, floatArr) values(?, ?)",
				"select floatArr from TestArrayDecoding where id = ?",
				[]float32{0, 1.1, -1111.1111, 0.123456}, &[]float32{},
				func(t testing.TB, query, scan any) {
					if !reflect.DeepEqual(query, *(scan.(*[]float32))) {
						t.Errorf("failed to encode float[]")
					}
				},
			},
			{
				"insert into TestArrayDecoding(id, doubleArr) values(?, ?)",
				"select doubleArr from TestArrayDecoding where id = ?",
				[]float64{0, -0.123456789, 1234554.346, 321243.23445}, &[]float64{},
				func(t testing.TB, query, scan any) {
					if !reflect.DeepEqual(query, *(scan.(*[]float64))) {
						t.Errorf("failed to encode double[]")
					}
				},
			},
			{
				"insert into TestArrayDecoding(id, varcharArr) values(?, ?)",
				"select varcharArr from TestArrayDecoding where id = ?",
				[]string{"it's", "over", "9000!"}, &[]string{},
				func(t testing.TB, query, scan any) {
					if !reflect.DeepEqual(query, *(scan.(*[]string))) {
						t.Errorf("failed to encode text[]")
					}
				},
			},
			{
				"insert into TestArrayDecoding(id, timestampArr) values(?, ?)",
				"select timestampArr from TestArrayDecoding where id = ?",
				[]time.Time{time.Unix(323232, 0).UTC(), time.Unix(3239949334, 00).UTC()}, &[]time.Time{},
				func(t testing.TB, query, scan any) {
					queryTimeSlice := query.([]time.Time)
					scanTimeSlice := *(scan.(*[]time.Time))
					require.Equal(t, len(queryTimeSlice), len(scanTimeSlice))
					for i := range queryTimeSlice {
						assert.Truef(t, queryTimeSlice[i].Equal(scanTimeSlice[i]), "%d", i)
					}
				},
			},
		}

		for i, tt := range tests {
			_, err := conn.Exec(ctx, tt.sqlInsert, i, tt.query)
			assert.NoError(t, err)
			err = conn.QueryRow(ctx, tt.sqlSelect, i).Scan(tt.scan)
			assert.NoError(t, err)
			tt.assert(t, tt.query, tt.scan)
		}
	})
}
