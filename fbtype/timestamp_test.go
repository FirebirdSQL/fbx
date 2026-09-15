package fbtype_test

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestTimestampCodec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "timestamp", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: "2025-12-17", Result: new(time.Time), Test: isExpectedEqTime(time.Date(2025, 12, 17, 0, 0, 0, 0, time.UTC))},

		{Param: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))},

		// Nanosecond truncation
		{Param: time.Date(2020, 1, 1, 0, 0, 0, 9999999, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2020, 1, 1, 0, 0, 0, 9900000, time.UTC))},

		{Param: fbtype.Timestamp{}, Result: new(fbtype.Timestamp), Test: isExpectedEq(fbtype.Timestamp{})},
		{Param: nil, Result: new(*time.Time), Test: isExpectedEq((*time.Time)(nil))},
	})
}

func TestTimestampCodecWithScanLocationUTC(t *testing.T) {
	connTestRunner := defaultConnTestRunner
	connTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		conn.TypeMap().RegisterType(&fbtype.Type{
			Name:   "timestamp",
			TypeID: fbproto.SQL_TYPE_TIMESTAMP,
			Codec:  &fbtype.TimestampCodec{ScanLocation: time.UTC},
		})
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, connTestRunner, nil, "timestamp", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Timestamp{Time: time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local), Valid: true}, Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
	})
}

func TestTimestampCodecWithScanLocationLocal(t *testing.T) {
	connTestRunner := defaultConnTestRunner
	connTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		conn.TypeMap().RegisterType(&fbtype.Type{
			Name:   "timestamp",
			TypeID: fbproto.SQL_TYPE_TIMESTAMP,
			Codec:  &fbtype.TimestampCodec{ScanLocation: time.Local},
		})
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, connTestRunner, nil, "timestamp", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local))},
	})
}

func TestTimestampTranscodeBigTimeBinary(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		in := &fbtype.Timestamp{Time: time.Date(9999, 12, 31, 23, 59, 59, 999900000, time.UTC), Valid: true}
		var out fbtype.Timestamp

		err := conn.QueryRow(ctx, "select cast(? as timestamp) from rdb$database", in).Scan(&out)
		if err != nil {
			t.Fatal(err)
		}

		require.Equal(t, in.Valid, out.Valid)
		require.Truef(t, in.Time.Equal(out.Time), "expected %v got %v", in.Time, out.Time)
	})
}

func TestTimestampCodecDecodeTextInvalid(t *testing.T) {
	c := &fbtype.TimestampCodec{}
	var ts fbtype.Timestamp
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)
	plan := c.PlanScan(nil, typeMetadata, &ts)
	err := plan.Scan([]byte(`eeeee`), &ts)
	require.Error(t, err)
}

func TestTimestampScanStringSupportedFormats(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		year   int
		month  time.Month
		day    int
		hour   int
		minute int
		second int
		nsec   int
		offset int
	}{
		{
			name:  "firebird date only",
			input: "2025-12-17",
			year:  2025,
			month: time.December,
			day:   17,
		},
		{
			name:  "firebird alternative date only",
			input: "17.12.2025",
			year:  2025,
			month: time.December,
			day:   17,
		},
		{
			name:   "firebird full",
			input:  "2025-12-17 10:16:43",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
			second: 43,
		},
		{
			name:   "firebird hour minute",
			input:  "2025-12-17 10:16",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
		},
		{
			name:   "firebird single digit fields",
			input:  "2025-12-17 1:2:3",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   1,
			minute: 2,
			second: 3,
		},
		{
			name:   "firebird alternative date format",
			input:  "17.12.2025 10:16:43.1234",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
			second: 43,
			nsec:   123400000,
		},
		{
			name:   "json without zone",
			input:  "2025-12-17T10:16:43.123456",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
			second: 43,
			nsec:   123456000,
		},
		{
			name:   "rfc3339 with zone",
			input:  "2025-12-17T10:16:43.123456+03:00",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
			second: 43,
			nsec:   123456000,
			offset: 3 * 60 * 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got fbtype.Timestamp
			err := got.Scan(tt.input)
			assert.NoError(t, err)
			assert.True(t, got.Valid)
			assertTimestampClock(t, got.Time, tt.year, tt.month, tt.day, tt.hour, tt.minute, tt.second, tt.nsec)
			_, offset := got.Time.Zone()
			assert.Equal(t, tt.offset, offset)
		})
	}
}

func TestTimestampScanStringRejectsInvalidAndResetsValue(t *testing.T) {
	tests := []string{
		"",
		"not-a-timestamp",
		"2025-02-30",
		"2025-13-17",
		"10:16:43",
		"2025-12-17 10",
		"2025-13-17 10:16:43",
		"2025-12-32 10:16:43",
		"2025-12-17 24:16:43",
		"2025-12-17 10:60:43",
		"2025-12-17 10:16:60",
		"2025-12-17 10:16:43.",
		"2025-12-17 10:16:43.1234567",
		"2025-12-17 10:16:43 Europe/Moscow",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := fbtype.Timestamp{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true}
			err := got.Scan(input)
			assert.Error(t, err)
			assert.Equal(t, fbtype.Timestamp{}, got)
		})
	}
}

func TestTimestampScanUnsupportedTypeResetsValue(t *testing.T) {
	got := fbtype.Timestamp{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true}
	err := got.Scan(1)
	assert.Error(t, err)
	assert.Equal(t, fbtype.Timestamp{}, got)
}

func TestTimestampCodecEncodeStringSupportsFirebirdFormats(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)
	codec := &fbtype.TimestampCodec{}

	tests := []struct {
		name   string
		input  string
		year   int
		month  time.Month
		day    int
		hour   int
		minute int
		second int
		nsec   int
	}{
		{
			name:  "firebird date only",
			input: "17.12.2025",
			year:  2025,
			month: time.December,
			day:   17,
		},
		{
			name:   "firebird hour minute",
			input:  "2025-12-17 10:16",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
		},
		{
			name:   "firebird fractional",
			input:  "17.12.2025 10:16:43.123456",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
			second: 43,
			nsec:   123400000,
		},
		{
			name:   "firebird single digit fields",
			input:  "2025-12-17 1:2:3",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   1,
			minute: 2,
			second: 3,
		},
		{
			name:   "rfc3339 wall clock",
			input:  "2025-12-17T10:16:43.123456+03:00",
			year:   2025,
			month:  time.December,
			day:    17,
			hour:   10,
			minute: 16,
			second: 43,
			nsec:   123400000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := codec.PlanEncode(nil, typeMetadata, tt.input)
			require.NotNil(t, plan)

			buf, err := plan.Encode(tt.input, nil)
			require.NoError(t, err)

			var got time.Time
			err = codec.PlanScan(nil, typeMetadata, &got).Scan(buf, &got)
			require.NoError(t, err)

			assertTimestampClock(t, got, tt.year, tt.month, tt.day, tt.hour, tt.minute, tt.second, tt.nsec)
		})
	}
}

func TestTimestampCodecScanStringPreservesScanLocationWallClock(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)

	encodeCodec := &fbtype.TimestampCodec{}
	input := time.Date(2025, 12, 17, 10, 16, 43, 123400000, time.UTC)
	buf, err := encodeCodec.PlanEncode(nil, typeMetadata, input).Encode(input, nil)
	require.NoError(t, err)

	scanCodec := &fbtype.TimestampCodec{ScanLocation: time.FixedZone("MSK", 3*60*60)}
	var got string
	err = scanCodec.PlanScan(nil, typeMetadata, &got).Scan(buf, &got)
	require.NoError(t, err)

	assert.Equal(t, "2025-12-17 10:16:43.1234", got)
}

func TestTimestampCodecScanNullToNullableTargets(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)
	m := fbtype.NewMap()

	var gotTimestamp fbtype.Timestamp
	err := m.Scan(typeMetadata, nil, &gotTimestamp)
	assert.NoError(t, err)
	assert.False(t, gotTimestamp.Valid)

	var gotNullTime sql.NullTime
	err = m.Scan(typeMetadata, nil, &gotNullTime)
	assert.NoError(t, err)
	assert.False(t, gotNullTime.Valid)

	var gotTimePtr *time.Time
	err = m.Scan(typeMetadata, nil, &gotTimePtr)
	assert.NoError(t, err)
	assert.Nil(t, gotTimePtr)

	var gotStringPtr *string
	err = m.Scan(typeMetadata, nil, &gotStringPtr)
	assert.NoError(t, err)
	assert.Nil(t, gotStringPtr)

	var gotAny any
	err = m.Scan(typeMetadata, nil, &gotAny)
	assert.NoError(t, err)
	assert.Nil(t, gotAny)
}

func TestTimestampCodecScanNullToNonNullableTargetsReturnsError(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)
	codec := &fbtype.TimestampCodec{}

	var goTime time.Time
	err := codec.PlanScan(nil, typeMetadata, &goTime).Scan(nil, &goTime)
	assert.Error(t, err)

	var s string
	err = codec.PlanScan(nil, typeMetadata, &s).Scan(nil, &s)
	assert.Error(t, err)
}

func TestTimestampCodecScanPlanTargetTypeChanged(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)
	codec := &fbtype.TimestampCodec{}
	src := timestampWireBytes(0, 0)

	var goTime time.Time
	plan := codec.PlanScan(nil, typeMetadata, &goTime)

	var s string
	err := plan.Scan(src, &s)
	assert.ErrorIs(t, err, fbtype.ErrScanTargetTypeChanged)
}

func TestTimestampCodecRejectsMalformedBinary(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)
	codec := &fbtype.TimestampCodec{}

	t.Run("invalid length", func(t *testing.T) {
		var got fbtype.Timestamp
		err := codec.PlanScan(nil, typeMetadata, &got).Scan([]byte{0, 0, 0}, &got)
		assert.Error(t, err)
	})

	t.Run("time out of range", func(t *testing.T) {
		src := timestampWireBytes(0, int64(24*time.Hour/time.Microsecond))

		var gotTime time.Time
		err := codec.PlanScan(nil, typeMetadata, &gotTime).Scan(src, &gotTime)
		assert.Error(t, err)

		var gotTimestamp fbtype.Timestamp
		err = codec.PlanScan(nil, typeMetadata, &gotTimestamp).Scan(src, &gotTimestamp)
		assert.Error(t, err)

		var gotString string
		err = codec.PlanScan(nil, typeMetadata, &gotString).Scan(src, &gotString)
		assert.Error(t, err)
	})

	t.Run("date out of range", func(t *testing.T) {
		src := timestampWireBytes(3_000_000, 0)

		var got fbtype.Timestamp
		err := codec.PlanScan(nil, typeMetadata, &got).Scan(src, &got)
		assert.Error(t, err)
	})
}

func TestTimestampCodecEncodeRejectsDateOutOfRange(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP, 0, 0, 0)
	codec := &fbtype.TimestampCodec{}

	hugeDate := time.Date(11761246, time.January, 22, 10, 0, 0, 0, time.UTC)
	tests := []any{
		fbtype.Timestamp{
			Time:  time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
		fbtype.Timestamp{
			Time:  hugeDate,
			Valid: true,
		},
		hugeDate,
	}

	for _, in := range tests {
		_, err := codec.PlanEncode(nil, typeMetadata, in).Encode(in, nil)
		assert.Error(t, err)
	}
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
	badJson := []byte(`{"ts":"not-a-timestamp"}`)
	assert.Error(t, json.Unmarshal(badJson, &tsStruct))
}

func TestTimestampUnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Timestamp
	}{
		{source: "null", result: fbtype.Timestamp{}},
		{source: `"2025-12-17"`, result: fbtype.Timestamp{Time: time.Date(2025, 12, 17, 0, 0, 0, 0, time.UTC), Valid: true}},
		{source: "\"2012-03-29T10:05:45\"", result: fbtype.Timestamp{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.UTC), Valid: true}},
		{source: "\"2012-03-29T10:05:45.555\"", result: fbtype.Timestamp{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.UTC), Valid: true}},
		{source: "\"2025-12-17 1:2:3\"", result: fbtype.Timestamp{Time: time.Date(2025, 12, 17, 1, 2, 3, 0, time.UTC), Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Timestamp
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if !r.Time.Equal(tt.result.Time) || r.Valid != tt.result.Valid {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}

func assertTimestampClock(t *testing.T, got time.Time, year int, month time.Month, day int, hour int, minute int, second int, nsec int) {
	t.Helper()

	assert.Equal(t, year, got.Year())
	assert.Equal(t, month, got.Month())
	assert.Equal(t, day, got.Day())
	assert.Equal(t, hour, got.Hour())
	assert.Equal(t, minute, got.Minute())
	assert.Equal(t, second, got.Second())
	assert.Equal(t, nsec, got.Nanosecond())
}

func timestampWireBytes(dayOffset int32, usec int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], uint32(dayOffset))
	binary.BigEndian.PutUint32(buf[4:8], uint32(usec/100))
	return buf
}
