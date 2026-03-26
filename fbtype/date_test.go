package fbtype_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func isExpectedEqTime(a any) func(any) bool {
	return func(v any) bool {
		at := a.(time.Time)
		vt := v.(time.Time)

		return at.Equal(vt)
	}
}

func TestDateCodec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "date", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: fbtype.Date{}, Result: new(fbtype.Date), Test: isExpectedEq(fbtype.Date{})},
		{Param: nil, Result: new(*time.Time), Test: isExpectedEq((*time.Time)(nil))},
	})
}

func TestDateCodecStringInputFormats(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "date", []fbxtest.ValueRoundTripTest{
		{Param: "2026-03-20", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))},
		{Param: "2026-3-2", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC))},
		{Param: "20.03.2026", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))},
		{Param: "2.3.2026", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC))},
		{Param: "2026.03.20", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))},
		{Param: "2026.3.2", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC))},
		{Param: "2026/03/20", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))},
		{Param: "2026/3/2", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC))},
	})
}

func TestDateScanTimeNormalizes(t *testing.T) {
	var d fbtype.Date

	src := time.Date(2026, 3, 20, 15, 4, 5, 123456789, time.FixedZone("UTC+3", 3*60*60))
	err := d.Scan(src)
	require.NoError(t, err)

	require.True(t, d.Valid)
	require.Equal(t, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), d.Time)
}

func TestDateCodecRejectsUnsupportedStringFormats(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		sql := "select cast(? as date) from rdb$database"
		tests := []string{
			"not-a-date",
			"2026-13-01",
			"2026-02-30",
			"2023-02-29",
			"20/03/2026",
			"03-20-2026",
			"",
			"0000-12-31",
			"10000-01-01",
		}

		for i, input := range tests {
			var got time.Time
			err := conn.QueryRow(ctx, sql, input).Scan(&got)
			require.Error(tb, err, "%d: input=%q", i, input)
		}
	})
}

func TestDateMarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Date
		result string
	}{
		{source: fbtype.Date{}, result: "null"},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}, result: "\"2012-03-29\""},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.FixedZone("", -6*60*60)), Valid: true}, result: "\"2012-03-29\""},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.FixedZone("", -6*60*60)), Valid: true}, result: "\"2012-03-29\""},
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

func TestDateUnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Date
	}{
		{source: "null", result: fbtype.Date{}},
		{source: "\"2012-03-29\"", result: fbtype.Date{Time: time.Date(2012, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}},
		{source: "\"2012-03-29\"", result: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.FixedZone("", -6*60*60)), Valid: true}},
		{source: "\"2012-03-29\"", result: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.FixedZone("", -6*60*60)), Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Date
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r.Time.Year() != tt.result.Time.Year() || r.Time.Month() != tt.result.Time.Month() || r.Time.Day() != tt.result.Time.Day() || r.Valid != tt.result.Valid {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}
