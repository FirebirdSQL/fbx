package fbtype_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/red-soft/fbx"
	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/fbxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimestampCodec(t *testing.T) {
	skipCockroachDB(t, "Server does not support infinite timestamps (see https://github.com/cockroachdb/cockroach/issues/41564)")

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "timestamp", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(-100, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(-100, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC))},

		{Param: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))},

		// Nanosecond truncation
		{Param: time.Date(2020, 1, 1, 0, 0, 0, 999999999, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2020, 1, 1, 0, 0, 0, 999999000, time.UTC))},
		{Param: time.Date(2020, 1, 1, 0, 0, 0, 999999001, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2020, 1, 1, 0, 0, 0, 999999000, time.UTC))},

		{Param: fbtype.Timestamp{InfinityModifier: fbtype.Infinity, Valid: true}, Result: new(fbtype.Timestamp), Test: isExpectedEq(fbtype.Timestamp{InfinityModifier: fbtype.Infinity, Valid: true})},
		{Param: fbtype.Timestamp{InfinityModifier: fbtype.NegativeInfinity, Valid: true}, Result: new(fbtype.Timestamp), Test: isExpectedEq(fbtype.Timestamp{InfinityModifier: fbtype.NegativeInfinity, Valid: true})},
		{Param: fbtype.Timestamp{}, Result: new(fbtype.Timestamp), Test: isExpectedEq(fbtype.Timestamp{})},
		{Param: nil, Result: new(*time.Time), Test: isExpectedEq((*time.Time)(nil))},
	})
}

func TestTimestampCodecWithScanLocationUTC(t *testing.T) {
	skipCockroachDB(t, "Server does not support infinite timestamps (see https://github.com/cockroachdb/cockroach/issues/41564)")

	connTestRunner := defaultConnTestRunner
	connTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		conn.TypeMap().RegisterType(&fbtype.Type{
			Name:  "timestamp",
			OID:   fbtype.TimestampOID,
			Codec: &fbtype.TimestampCodec{ScanLocation: time.UTC},
		})
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, connTestRunner, nil, "timestamp", []fbxtest.ValueRoundTripTest{
		// Have to use pgtype.Timestamp instead of time.Time as source because otherwise the simple and exec query exec
		// modes will encode the time for timestamptz. That is, they will convert it from local time zone.
		{Param: fbtype.Timestamp{Time: time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local), Valid: true}, Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
	})
}

func TestTimestampCodecWithScanLocationLocal(t *testing.T) {
	skipCockroachDB(t, "Server does not support infinite timestamps (see https://github.com/cockroachdb/cockroach/issues/41564)")

	connTestRunner := defaultConnTestRunner
	connTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		conn.TypeMap().RegisterType(&fbtype.Type{
			Name:  "timestamp",
			OID:   fbtype.TimestampOID,
			Codec: &fbtype.TimestampCodec{ScanLocation: time.Local},
		})
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, connTestRunner, nil, "timestamp", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local))},
	})
}

// https://github.com/jackc/pgx/v4/pgtype/pull/128
func TestTimestampTranscodeBigTimeBinary(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		in := &fbtype.Timestamp{Time: time.Date(294276, 12, 31, 23, 59, 59, 999999000, time.UTC), Valid: true}
		var out fbtype.Timestamp

		err := conn.QueryRow(ctx, "select $1::timestamp", in).Scan(&out)
		if err != nil {
			t.Fatal(err)
		}

		require.Equal(t, in.Valid, out.Valid)
		require.Truef(t, in.Time.Equal(out.Time), "expected %v got %v", in.Time, out.Time)
	})
}

// https://github.com/jackc/pgtype/issues/74
func TestTimestampCodecDecodeTextInvalid(t *testing.T) {
	c := &fbtype.TimestampCodec{}
	var ts fbtype.Timestamp
	plan := c.PlanScan(nil, fbtype.TimestampOID, fbtype.TextFormatCode, &ts)
	err := plan.Scan([]byte(`eeeee`), &ts)
	require.Error(t, err)
}

func TestTimestampMarshalJSON(t *testing.T) {

	tsStruct := struct {
		TS fbtype.Timestamp `json:"ts"`
	}{}

	tm := time.Date(2012, 3, 29, 10, 5, 45, 0, time.UTC)
	tsString := "\"" + tm.Format("2006-01-02T15:04:05") + "\"" //  `"2012-03-29T10:05:45"`
	var pgt fbtype.Timestamp
	_ = pgt.Scan(tm)

	successfulTests := []struct {
		source fbtype.Timestamp
		result string
	}{
		{source: fbtype.Timestamp{}, result: "null"},
		{source: fbtype.Timestamp{Time: tm, Valid: true}, result: tsString},
		{source: pgt, result: tsString},
		{source: fbtype.Timestamp{Time: tm.Add(time.Second * 555 / 1000), Valid: true}, result: `"2012-03-29T10:05:45.555"`},
		{source: fbtype.Timestamp{InfinityModifier: fbtype.Infinity, Valid: true}, result: "\"infinity\""},
		{source: fbtype.Timestamp{InfinityModifier: fbtype.NegativeInfinity, Valid: true}, result: "\"-infinity\""},
	}
	for i, tt := range successfulTests {
		r, err := tt.source.MarshalJSON()
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if !assert.Equal(t, tt.result, string(r)) {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, string(r))
		}
		tsStruct.TS = tt.source
		b, err := json.Marshal(tsStruct)
		assert.NoErrorf(t, err, "failed to marshal %v %s", tt.source, err)
		t2 := tsStruct
		t2.TS = fbtype.Timestamp{} // Clear out the value so that we can compare after unmarshalling
		err = json.Unmarshal(b, &t2)
		assert.NoErrorf(t, err, "failed to unmarshal %v with %s", tt.source, err)
		assert.True(t, tsStruct.TS.Time.Unix() == t2.TS.Time.Unix())
	}
}

func TestTimestampUnmarshalJSONErrors(t *testing.T) {
	tsStruct := struct {
		TS fbtype.Timestamp `json:"ts"`
	}{}
	goodJson1 := []byte(`{"ts":"2012-03-29T10:05:45"}`)
	assert.NoError(t, json.Unmarshal(goodJson1, &tsStruct))
	goodJson2 := []byte(`{"ts":"2012-03-29T10:05:45Z"}`)
	assert.NoError(t, json.Unmarshal(goodJson2, &tsStruct))
	badJson := []byte(`{"ts":"2012-03-29"}`)
	assert.Error(t, json.Unmarshal(badJson, &tsStruct))
}

func TestTimestampUnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Timestamp
	}{
		{source: "null", result: fbtype.Timestamp{}},
		{source: "\"2012-03-29T10:05:45\"", result: fbtype.Timestamp{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.UTC), Valid: true}},
		{source: "\"2012-03-29T10:05:45.555\"", result: fbtype.Timestamp{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.UTC), Valid: true}},
		{source: "\"infinity\"", result: fbtype.Timestamp{InfinityModifier: fbtype.Infinity, Valid: true}},
		{source: "\"-infinity\"", result: fbtype.Timestamp{InfinityModifier: fbtype.NegativeInfinity, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Timestamp
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if !r.Time.Equal(tt.result.Time) || r.Valid != tt.result.Valid || r.InfinityModifier != tt.result.InfinityModifier {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}
