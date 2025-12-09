package fbtype_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestTimestamptzCodec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "timestamptz", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(-100, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(-100, 1, 1, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(-1, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(-1, 1, 1, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(0, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(0, 1, 1, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(1, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1, 1, 1, 0, 0, 0, 0, time.Local))},

		{Param: time.Date(1900, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1900, 1, 1, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(1999, 12, 31, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1999, 12, 31, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local))},
		{Param: time.Date(2200, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2200, 1, 1, 0, 0, 0, 0, time.Local))},

		// Nanosecond truncation
		{Param: time.Date(2020, 1, 1, 0, 0, 0, 999999999, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2020, 1, 1, 0, 0, 0, 999999000, time.Local))},
		{Param: time.Date(2020, 1, 1, 0, 0, 0, 999999001, time.Local), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2020, 1, 1, 0, 0, 0, 999999000, time.Local))},

		{Param: fbtype.Timestamptz{}, Result: new(fbtype.Timestamptz), Test: isExpectedEq(fbtype.Timestamptz{})},
		{Param: nil, Result: new(*time.Time), Test: isExpectedEq((*time.Time)(nil))},
	})
}

func TestTimestamptzCodecWithLocationUTC(t *testing.T) {
	connTestRunner := defaultConnTestRunner
	connTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		conn.TypeMap().RegisterType(&fbtype.Type{
			Name:  "timestamptz",
			OID:   fbproto.SQL_TYPE_TIMESTAMP_TZ,
			Codec: &fbtype.TimestamptzCodec{ScanLocation: time.UTC},
		})
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, connTestRunner, nil, "timestamptz", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
	})
}

func TestTimestamptzCodecWithLocationLocal(t *testing.T) {
	connTestRunner := defaultConnTestRunner
	connTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		conn.TypeMap().RegisterType(&fbtype.Type{
			Name:  "timestamptz",
			OID:   fbproto.SQL_TYPE_TIMESTAMP_TZ,
			Codec: &fbtype.TimestamptzCodec{ScanLocation: time.Local},
		})
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, connTestRunner, nil, "timestamptz", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local))},
	})
}

// https://github.com/jackc/pgx/v4/pgtype/pull/128
func TestTimestamptzTranscodeBigTimeBinary(t *testing.T) {
	t.Skip()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		in := &fbtype.Timestamptz{Time: time.Date(294276, 12, 31, 23, 59, 59, 999999000, time.UTC), Valid: true}
		var out fbtype.Timestamptz

		err := conn.QueryRow(ctx, "select $1::timestamptz", in).Scan(&out)
		if err != nil {
			t.Fatal(err)
		}

		require.Equal(t, in.Valid, out.Valid)
		require.Truef(t, in.Time.Equal(out.Time), "expected %v got %v", in.Time, out.Time)
	})
}

// https://github.com/jackc/pgtype/issues/74
func TestTimestamptzDecodeTextInvalid(t *testing.T) {
	t.Skip()
	c := &fbtype.TimestamptzCodec{}
	var tstz fbtype.Timestamptz
	plan := c.PlanScan(nil, fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0, fbtype.TextFormatCode, &tstz)
	err := plan.Scan([]byte(`eeeee`), &tstz)
	require.Error(t, err)
}

func TestTimestamptzMarshalJSON(t *testing.T) {
	t.Skip()
	successfulTests := []struct {
		source fbtype.Timestamptz
		result string
	}{
		{source: fbtype.Timestamptz{}, result: "null"},
		{source: fbtype.Timestamptz{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.FixedZone("", -6*60*60)), Valid: true}, result: "\"2012-03-29T10:05:45-06:00\""},
		{source: fbtype.Timestamptz{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.FixedZone("", -6*60*60)), Valid: true}, result: "\"2012-03-29T10:05:45.555-06:00\""},
	}
	for i, tt := range successfulTests {
		r, err := tt.source.MarshalJSON()
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if string(r) != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, string(r))
		}
	}
}

func TestTimestamptzUnmarshalJSON(t *testing.T) {
	t.Skip()
	successfulTests := []struct {
		source string
		result fbtype.Timestamptz
	}{
		{source: "null", result: fbtype.Timestamptz{}},
		{source: "\"2012-03-29T10:05:45-06:00\"", result: fbtype.Timestamptz{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.FixedZone("", -6*60*60)), Valid: true}},
		{source: "\"2012-03-29T10:05:45.555-06:00\"", result: fbtype.Timestamptz{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.FixedZone("", -6*60*60)), Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Timestamptz
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if !r.Time.Equal(tt.result.Time) || r.Valid != tt.result.Valid {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}
