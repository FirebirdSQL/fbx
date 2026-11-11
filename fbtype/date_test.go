package fbtype_test

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/fbtype"
	"github.com/firebirdsql/fbx/fbxtest"
	"github.com/stretchr/testify/require"
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

func TestDateValueNormalizes(t *testing.T) {
	src := fbtype.Date{
		Time:  time.Date(2026, 3, 20, 15, 4, 5, 123456789, time.FixedZone("UTC+3", 3*60*60)),
		Valid: true,
	}

	v, err := src.Value()
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), v)
}

func TestDateCodecScanNullToStringReturnsError(t *testing.T) {
	var c fbtype.DateCodec
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DATE, 0, 0, 0)

	var got string
	planScan := c.PlanScan(nil, typeMetadata, &got)
	require.NotNil(t, planScan)

	err := planScan.Scan(nil, &got)
	require.ErrorContains(t, err, "cannot scan NULL into *string")
}

func TestDateCodecScanNullToNullableTargets(t *testing.T) {
	var c fbtype.DateCodec
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DATE, 0, 0, 0)

	var gotDate fbtype.Date
	planScan := c.PlanScan(nil, typeMetadata, &gotDate)
	require.NotNil(t, planScan)
	require.NoError(t, planScan.Scan(nil, &gotDate))
	require.False(t, gotDate.Valid)

	m := fbtype.NewMap()

	var gotTimePtr *time.Time
	planScan = m.PlanScan(typeMetadata, &gotTimePtr)
	require.NotNil(t, planScan)
	require.NoError(t, planScan.Scan(nil, &gotTimePtr))
	require.Nil(t, gotTimePtr)

	var gotStringPtr *string
	planScan = m.PlanScan(typeMetadata, &gotStringPtr)
	require.NotNil(t, planScan)
	require.NoError(t, planScan.Scan(nil, &gotStringPtr))
	require.Nil(t, gotStringPtr)

	var gotAny any
	planScan = m.PlanScan(typeMetadata, &gotAny)
	require.NotNil(t, planScan)
	require.NoError(t, planScan.Scan(nil, &gotAny))
	require.Nil(t, gotAny)
}

func TestDateCodecRejectsUnsupportedStringFormats(t *testing.T) {
	var c fbtype.DateCodec
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DATE, 0, 0, 0)

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
		planEncode := c.PlanEncode(nil, typeMetadata, input)
		require.NotNil(t, planEncode, "%d: input=%q", i, input)

		_, err := planEncode.Encode(input, nil)
		require.Error(t, err, "%d: input=%q", i, input)
	}
}

func TestDateCodecEncodeRejectsDateOutOfRange(t *testing.T) {
	var c fbtype.DateCodec
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DATE, 0, 0, 0)

	hugeDate := time.Date(11761246, time.January, 22, 0, 0, 0, 0, time.UTC)
	tests := []any{
		hugeDate,
		fbtype.Date{Time: hugeDate, Valid: true},
	}

	for _, input := range tests {
		planEncode := c.PlanEncode(nil, typeMetadata, input)
		require.NotNil(t, planEncode)

		_, err := planEncode.Encode(input, nil)
		require.ErrorContains(t, err, "date out of Firebird range")
	}
}

func TestDateCodecDecodeRejectsDateOutOfRange(t *testing.T) {
	var c fbtype.DateCodec
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DATE, 0, 0, 0)
	src := dateWireBytes(1<<31 - 1)

	var gotDate fbtype.Date
	planScan := c.PlanScan(nil, typeMetadata, &gotDate)
	require.NotNil(t, planScan)
	require.ErrorContains(t, planScan.Scan(src, &gotDate), "date out of Firebird range")

	var gotTime time.Time
	planScan = c.PlanScan(nil, typeMetadata, &gotTime)
	require.NotNil(t, planScan)
	require.ErrorContains(t, planScan.Scan(src, &gotTime), "date out of Firebird range")

	var gotString string
	planScan = c.PlanScan(nil, typeMetadata, &gotString)
	require.NotNil(t, planScan)
	require.ErrorContains(t, planScan.Scan(src, &gotString), "date out of Firebird range")
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

func dateWireBytes(dayOffset int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(dayOffset))
	return buf
}

func TestDateUnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Date
	}{
		{source: "null", result: fbtype.Date{}},
		{source: "\"2012-03-29\"", result: fbtype.Date{Time: time.Date(2012, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Date
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}
