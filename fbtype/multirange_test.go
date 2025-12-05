package fbtype_test

import (
	"context"
	"reflect"
	"testing"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
	"github.com/stretchr/testify/require"
)

func TestMultirangeCodecTranscode(t *testing.T) {
	skipPostgreSQLVersionLessThan(t, 14)
	skipCockroachDB(t, "Server does not support range types (see https://github.com/cockroachdb/cockroach/issues/27791)")

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int4multirange", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Multirange[fbtype.Range[fbtype.Int4]](nil), Result: new(fbtype.Multirange[fbtype.Range[fbtype.Int4]]), Test: func(a any) bool { return reflect.DeepEqual(fbtype.Multirange[fbtype.Range[fbtype.Int4]](nil), a) }},
		{Param: fbtype.Multirange[fbtype.Range[fbtype.Int4]]{}, Result: new(fbtype.Multirange[fbtype.Range[fbtype.Int4]]), Test: func(a any) bool { return reflect.DeepEqual(fbtype.Multirange[fbtype.Range[fbtype.Int4]]{}, a) }},
		{Param: fbtype.Multirange[fbtype.Range[fbtype.Int4]]{
			{
				Lower:     fbtype.Int4{Int32: 1, Valid: true},
				Upper:     fbtype.Int4{Int32: 5, Valid: true},
				LowerType: fbtype.Inclusive,
				UpperType: fbtype.Exclusive,
				Valid:     true,
			},
			{
				Lower:     fbtype.Int4{Int32: 7, Valid: true},
				Upper:     fbtype.Int4{Int32: 9, Valid: true},
				LowerType: fbtype.Inclusive,
				UpperType: fbtype.Exclusive,
				Valid:     true,
			},
		}, Result: new(fbtype.Multirange[fbtype.Range[fbtype.Int4]]), Test: func(a any) bool {
			return reflect.DeepEqual(fbtype.Multirange[fbtype.Range[fbtype.Int4]]{
				{
					Lower:     fbtype.Int4{Int32: 1, Valid: true},
					Upper:     fbtype.Int4{Int32: 5, Valid: true},
					LowerType: fbtype.Inclusive,
					UpperType: fbtype.Exclusive,
					Valid:     true,
				},
				{
					Lower:     fbtype.Int4{Int32: 7, Valid: true},
					Upper:     fbtype.Int4{Int32: 9, Valid: true},
					LowerType: fbtype.Inclusive,
					UpperType: fbtype.Exclusive,
					Valid:     true,
				},
			}, a)
		},
		},
	})
}

func TestMultirangeCodecDecodeValue(t *testing.T) {
	skipPostgreSQLVersionLessThan(t, 14)
	skipCockroachDB(t, "Server does not support range types (see https://github.com/cockroachdb/cockroach/issues/27791)")

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, _ testing.TB, conn *fbx.Conn) {

		for _, tt := range []struct {
			sql      string
			expected any
		}{
			{
				sql: `select int4multirange(int4range(1, 5), int4range(7,9))`,
				expected: fbtype.Multirange[fbtype.Range[any]]{
					{
						Lower:     int32(1),
						Upper:     int32(5),
						LowerType: fbtype.Inclusive,
						UpperType: fbtype.Exclusive,
						Valid:     true,
					},
					{
						Lower:     int32(7),
						Upper:     int32(9),
						LowerType: fbtype.Inclusive,
						UpperType: fbtype.Exclusive,
						Valid:     true,
					},
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
