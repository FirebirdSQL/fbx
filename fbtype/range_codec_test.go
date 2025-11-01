package fbtype_test

import (
	"context"
	"testing"

	pgx "github.com/red-soft/fbx"
	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/fbxtest"
	"github.com/stretchr/testify/require"
)

func TestRangeCodecTranscode(t *testing.T) {
	skipCockroachDB(t, "Server does not support range types (see https://github.com/cockroachdb/cockroach/issues/27791)")

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int4range", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Range[fbtype.Int4]{LowerType: fbtype.Empty, UpperType: fbtype.Empty, Valid: true}, Result: new(fbtype.Range[fbtype.Int4]), Test: isExpectedEq(fbtype.Range[fbtype.Int4]{LowerType: fbtype.Empty, UpperType: fbtype.Empty, Valid: true})},
		{Param: fbtype.Range[fbtype.Int4]{
			LowerType: fbtype.Inclusive,
			Lower:     fbtype.Int4{Int32: 1, Valid: true},
			Upper:     fbtype.Int4{Int32: 5, Valid: true},
			UpperType: fbtype.Exclusive, Valid: true,
		}, Result: new(fbtype.Range[fbtype.Int4]), Test: isExpectedEq(fbtype.Range[fbtype.Int4]{
			LowerType: fbtype.Inclusive,
			Lower:     fbtype.Int4{Int32: 1, Valid: true},
			Upper:     fbtype.Int4{Int32: 5, Valid: true},
			UpperType: fbtype.Exclusive, Valid: true,
		}),
		},
		{Param: fbtype.Range[fbtype.Int4]{}, Result: new(fbtype.Range[fbtype.Int4]), Test: isExpectedEq(fbtype.Range[fbtype.Int4]{})},
		{Param: nil, Result: new(fbtype.Range[fbtype.Int4]), Test: isExpectedEq(fbtype.Range[fbtype.Int4]{})},
	})
}

func TestRangeCodecTranscodeCompatibleRangeElementTypes(t *testing.T) {
	ctr := defaultConnTestRunner
	ctr.AfterConnect = func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		fbxtest.SkipCockroachDB(t, conn, "Server does not support range types (see https://github.com/cockroachdb/cockroach/issues/27791)")
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, ctr, nil, "numrange", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Range[fbtype.Float8]{LowerType: fbtype.Empty, UpperType: fbtype.Empty, Valid: true}, Result: new(fbtype.Range[fbtype.Float8]), Test: isExpectedEq(fbtype.Range[fbtype.Float8]{LowerType: fbtype.Empty, UpperType: fbtype.Empty, Valid: true})},
		{Param: fbtype.Range[fbtype.Float8]{
			LowerType: fbtype.Inclusive,
			Lower:     fbtype.Float8{Float64: 1, Valid: true},
			Upper:     fbtype.Float8{Float64: 5, Valid: true},
			UpperType: fbtype.Exclusive, Valid: true,
		}, Result: new(fbtype.Range[fbtype.Float8]), Test: isExpectedEq(fbtype.Range[fbtype.Float8]{
			LowerType: fbtype.Inclusive,
			Lower:     fbtype.Float8{Float64: 1, Valid: true},
			Upper:     fbtype.Float8{Float64: 5, Valid: true},
			UpperType: fbtype.Exclusive, Valid: true,
		}),
		},
		{Param: fbtype.Range[fbtype.Float8]{}, Result: new(fbtype.Range[fbtype.Float8]), Test: isExpectedEq(fbtype.Range[fbtype.Float8]{})},
		{Param: nil, Result: new(fbtype.Range[fbtype.Float8]), Test: isExpectedEq(fbtype.Range[fbtype.Float8]{})},
	})
}

func TestRangeCodecScanRangeTwiceWithUnbounded(t *testing.T) {
	skipCockroachDB(t, "Server does not support range types (see https://github.com/cockroachdb/cockroach/issues/27791)")

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {

		var r fbtype.Range[fbtype.Int4]

		err := conn.QueryRow(context.Background(), `select '[1,5)'::int4range`).Scan(&r)
		require.NoError(t, err)

		require.Equal(
			t,
			fbtype.Range[fbtype.Int4]{
				Lower:     fbtype.Int4{Int32: 1, Valid: true},
				Upper:     fbtype.Int4{Int32: 5, Valid: true},
				LowerType: fbtype.Inclusive,
				UpperType: fbtype.Exclusive,
				Valid:     true,
			},
			r,
		)

		err = conn.QueryRow(ctx, `select '[1,)'::int4range`).Scan(&r)
		require.NoError(t, err)

		require.Equal(
			t,
			fbtype.Range[fbtype.Int4]{
				Lower:     fbtype.Int4{Int32: 1, Valid: true},
				Upper:     fbtype.Int4{},
				LowerType: fbtype.Inclusive,
				UpperType: fbtype.Unbounded,
				Valid:     true,
			},
			r,
		)

		err = conn.QueryRow(ctx, `select 'empty'::int4range`).Scan(&r)
		require.NoError(t, err)

		require.Equal(
			t,
			fbtype.Range[fbtype.Int4]{
				Lower:     fbtype.Int4{},
				Upper:     fbtype.Int4{},
				LowerType: fbtype.Empty,
				UpperType: fbtype.Empty,
				Valid:     true,
			},
			r,
		)
	})
}

func TestRangeCodecDecodeValue(t *testing.T) {
	skipCockroachDB(t, "Server does not support range types (see https://github.com/cockroachdb/cockroach/issues/27791)")

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, _ testing.TB, conn *pgx.Conn) {

		for _, tt := range []struct {
			sql      string
			expected any
		}{
			{
				sql: `select '[1,5)'::int4range`,
				expected: fbtype.Range[any]{
					Lower:     int32(1),
					Upper:     int32(5),
					LowerType: fbtype.Inclusive,
					UpperType: fbtype.Exclusive,
					Valid:     true,
				},
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
