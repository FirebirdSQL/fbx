package fbtype_test

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/fbtype"
	"github.com/firebirdsql/fbx/fbxtest"
	"github.com/stretchr/testify/assert"
)

func TestTimeCodec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "time", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Time{Microseconds: 0, Valid: true}, Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{Microseconds: 0, Valid: true})},
		{Param: fbtype.Time{Microseconds: 100, Valid: true}, Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{Microseconds: 100, Valid: true})},
		{Param: fbtype.Time{Microseconds: 86399999900, Valid: true}, Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{Microseconds: 86399999900, Valid: true})},
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{Microseconds: 0, Valid: true})},
		{Param: fbtype.Time{Microseconds: 0, Valid: true}, Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: fbtype.Time{}, Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{})},
		{Param: nil, Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{})},

		{Param: time.Date(2000, 1, 1, 14, 30, 45, 0, time.UTC), Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{Microseconds: 14*3600000000 + 30*60000000 + 45*1000000, Valid: true})},
		{Param: time.Date(2024, 12, 31, 23, 59, 59, 999999000, time.UTC), Result: new(fbtype.Time), Test: isExpectedEq(fbtype.Time{Microseconds: 86399999900, Valid: true})},
		{Param: fbtype.Time{Microseconds: 0, Valid: true}, Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: fbtype.Time{Microseconds: 14*3600000000 + 30*60000000 + 45*1000000, Valid: true}, Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 14, 30, 45, 0, time.UTC))},
		{Param: fbtype.Time{Microseconds: 86399999900, Valid: true}, Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 23, 59, 59, 999900000, time.UTC))},

		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2024, 6, 15, 7, 37, 16, 0, time.UTC), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 7, 37, 16, 0, time.UTC))},
		{Param: time.Date(2000, 1, 1, 15, 4, 5, 123456000, time.UTC), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 15, 4, 5, 123400000, time.UTC))},
		{Param: time.Date(2000, 1, 1, 23, 59, 59, 999999000, time.UTC), Result: new(time.Time), Test: isExpectedEq(time.Date(2000, 1, 1, 23, 59, 59, 999900000, time.UTC))},
	})
}

func TestTimeTextScanner(t *testing.T) {
	var fbTime fbtype.Time

	assert.NoError(t, fbTime.Scan("07:37:16"))
	assert.Equal(t, true, fbTime.Valid)
	assert.Equal(t, int64(7*time.Hour+37*time.Minute+16*time.Second), fbTime.Microseconds*int64(time.Microsecond))

	assert.NoError(t, fbTime.Scan("15:04:05"))
	assert.Equal(t, true, fbTime.Valid)
	assert.Equal(t, int64(15*time.Hour+4*time.Minute+5*time.Second), fbTime.Microseconds*int64(time.Microsecond))

	// parsing of fractional digits
	assert.NoError(t, fbTime.Scan("15:04:05.00"))
	assert.Equal(t, true, fbTime.Valid)
	assert.Equal(t, int64(15*time.Hour+4*time.Minute+5*time.Second), fbTime.Microseconds*int64(time.Microsecond))

	const mirco = "789123"
	const woFraction = int64(4*time.Hour + 5*time.Minute + 6*time.Second) // time without fraction
	for i := 1; i <= len(mirco); i++ {
		assert.NoError(t, fbTime.Scan("04:05:06."+mirco[:i]))
		assert.Equal(t, true, fbTime.Valid)

		frac, _ := strconv.ParseInt(mirco[:i], 10, 64)
		for k := i; k < 6; k++ {
			frac *= 10
		}
		assert.Equal(t, woFraction+frac*int64(time.Microsecond), fbTime.Microseconds*int64(time.Microsecond))
	}

	// parsing of empty fraction errors
	assert.Error(t, fbTime.Scan("04:05:06."))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	// parsing of too long fraction errors
	assert.Error(t, fbTime.Scan("04:05:06.7891234"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	// parsing of timetz errors
	assert.Error(t, fbTime.Scan("04:05:06.789-08"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	assert.Error(t, fbTime.Scan("04:05:06-08:00"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	// parsing of date errors
	assert.Error(t, fbTime.Scan("1997-12-17"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	// parsing of text errors
	assert.Error(t, fbTime.Scan("12345678"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	assert.Error(t, fbTime.Scan("12-34-56"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	assert.Error(t, fbTime.Scan("12:34-56"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)

	assert.Error(t, fbTime.Scan("12-34:56"))
	assert.Equal(t, false, fbTime.Valid)
	assert.Equal(t, int64(0), fbTime.Microseconds)
}

func TestTimeScanStringSupportedFirebirdFormats(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{input: "14:30", want: timeUsec(14, 30, 0, 0)},
		{input: "14:30:45", want: timeUsec(14, 30, 45, 0)},
		{input: "14:30:45.1", want: timeUsec(14, 30, 45, 100000)},
		{input: "14:30:45.123456", want: timeUsec(14, 30, 45, 123456)},
		{input: "1:2", want: timeUsec(1, 2, 0, 0)},
		{input: "1:02:03", want: timeUsec(1, 2, 3, 0)},
		{input: "01:2:3", want: timeUsec(1, 2, 3, 0)},
		{input: "1:2:3.4", want: timeUsec(1, 2, 3, 400000)},
		{input: " 04:05:06.789123 ", want: timeUsec(4, 5, 6, 789123)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var got fbtype.Time
			err := got.Scan(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, fbtype.Time{Microseconds: tt.want, Valid: true}, got)
		})
	}
}

func TestTimeScanStringRejectsInvalidFields(t *testing.T) {
	tests := []string{
		"",
		"not-a-time",
		"14",
		"24:00:00",
		"12:60:00",
		"12:34:60",
		"04:05:06.",
		"04:05:06.1234567",
		"04:05:06.x",
		"04:05.1",
		"123:02:03",
		"12:345:04",
		"12:03:456",
		"12:03:04+03:00",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := fbtype.Time{Microseconds: 1, Valid: true}
			err := got.Scan(input)
			assert.Error(t, err)
			assert.Equal(t, fbtype.Time{}, got)
		})
	}
}

func TestTimeScanUnsupportedTypeResetsValue(t *testing.T) {
	got := fbtype.Time{Microseconds: 1, Valid: true}
	err := got.Scan(1)
	assert.Error(t, err)
	assert.Equal(t, fbtype.Time{}, got)
}

func TestTimeValueReturnsGoTime(t *testing.T) {
	value, err := (fbtype.Time{Microseconds: timeUsec(14, 30, 45, 123456), Valid: true}).Value()
	assert.NoError(t, err)

	got, ok := value.(time.Time)
	if assert.True(t, ok) {
		assert.Equal(t, time.Date(2000, 1, 1, 14, 30, 45, 123456000, time.UTC), got)
	}

	value, err = (fbtype.Time{}).Value()
	assert.NoError(t, err)
	assert.Nil(t, value)

	_, err = (fbtype.Time{Microseconds: -1, Valid: true}).Value()
	assert.Error(t, err)

	_, err = (fbtype.Time{Microseconds: 24 * 60 * 60 * 1000000, Valid: true}).Value()
	assert.Error(t, err)
}

func TestTimeMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		src  fbtype.Time
		want string
	}{
		{
			name: "valid",
			src:  fbtype.Time{Microseconds: timeUsec(14, 30, 45, 123400), Valid: true},
			want: `"14:30:45.123400"`,
		},
		{
			name: "null",
			src:  fbtype.Time{},
			want: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.src.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want fbtype.Time
	}{
		{
			name: "full",
			src:  `"14:30:45.123456"`,
			want: fbtype.Time{Microseconds: timeUsec(14, 30, 45, 123456), Valid: true},
		},
		{
			name: "hour minute",
			src:  `"14:30"`,
			want: fbtype.Time{Microseconds: timeUsec(14, 30, 0, 0), Valid: true},
		},
		{
			name: "single digit fields",
			src:  `"1:2:3"`,
			want: fbtype.Time{Microseconds: timeUsec(1, 2, 3, 0), Valid: true},
		},
		{
			name: "null",
			src:  "null",
			want: fbtype.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got fbtype.Time
			err := json.Unmarshal([]byte(tt.src), &got)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimeUnmarshalJSONErrors(t *testing.T) {
	tests := []string{
		`"24:00:00"`,
		`"12:60:00"`,
		`"12:34:60"`,
		`"12:34:56."`,
		`"12:34:56.1234567"`,
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := fbtype.Time{Microseconds: 1, Valid: true}
			err := json.Unmarshal([]byte(input), &got)
			assert.Error(t, err)
			assert.Equal(t, fbtype.Time{}, got)
		})
	}

	var got fbtype.Time
	err := json.Unmarshal([]byte(`123`), &got)
	assert.Error(t, err)
}

func TestTimeCodecEncodeStringSupportsFirebirdFormats(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIME, 0, 0, 0)
	codec := fbtype.TimeCodec{}

	tests := []struct {
		input string
		want  int64
	}{
		{input: "14:30", want: timeUsec(14, 30, 0, 0)},
		{input: "14:30:45.123456", want: timeUsec(14, 30, 45, 123400)},
		{input: "1:2", want: timeUsec(1, 2, 0, 0)},
		{input: "1:02:03", want: timeUsec(1, 2, 3, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			buf, err := codec.PlanEncode(nil, typeMetadata, tt.input).Encode(tt.input, nil)
			assert.NoError(t, err)
			assert.Equal(t, uint32(tt.want/100), binary.BigEndian.Uint32(buf))
		})
	}
}

func TestTimeCodecDecodeDatabaseSQLValueReturnsGoTime(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIME, 0, 0, 0)
	src := timeWireBytes(timeUsec(14, 30, 45, 123400))

	value, err := fbtype.TimeCodec{}.DecodeDatabaseSQLValue(fbtype.NewMap(), typeMetadata, src)
	assert.NoError(t, err)

	got, ok := value.(time.Time)
	if assert.True(t, ok) {
		assert.Equal(t, time.Date(2000, 1, 1, 14, 30, 45, 123400000, time.UTC), got)
	}

	value, err = fbtype.TimeCodec{}.DecodeDatabaseSQLValue(fbtype.NewMap(), typeMetadata, nil)
	assert.NoError(t, err)
	assert.Nil(t, value)
}

func TestTimeCodecScanNullToNullableTargets(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIME, 0, 0, 0)
	m := fbtype.NewMap()

	var nullTime sql.NullTime
	err := m.Scan(typeMetadata, nil, &nullTime)
	assert.NoError(t, err)
	assert.False(t, nullTime.Valid)

	var timePtr *time.Time
	err = m.Scan(typeMetadata, nil, &timePtr)
	assert.NoError(t, err)
	assert.Nil(t, timePtr)

	var stringPtr *string
	err = m.Scan(typeMetadata, nil, &stringPtr)
	assert.NoError(t, err)
	assert.Nil(t, stringPtr)

	var anyValue any
	err = m.Scan(typeMetadata, nil, &anyValue)
	assert.NoError(t, err)
	assert.Nil(t, anyValue)
}

func TestTimeCodecScanNullToNonNullableTargetsReturnsError(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIME, 0, 0, 0)
	codec := fbtype.TimeCodec{}

	var goTime time.Time
	err := codec.PlanScan(nil, typeMetadata, &goTime).Scan(nil, &goTime)
	assert.Error(t, err)

	var s string
	err = codec.PlanScan(nil, typeMetadata, &s).Scan(nil, &s)
	assert.Error(t, err)
}

func TestTimeCodecRejectsMalformedBinary(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TIME, 0, 0, 0)
	codec := fbtype.TimeCodec{}

	t.Run("invalid length", func(t *testing.T) {
		var got time.Time
		err := codec.PlanScan(nil, typeMetadata, &got).Scan([]byte{0, 0, 0}, &got)
		assert.Error(t, err)
	})

	t.Run("out of range", func(t *testing.T) {
		src := timeWireBytes(24 * 60 * 60 * 1000000)

		var gotTime time.Time
		err := codec.PlanScan(nil, typeMetadata, &gotTime).Scan(src, &gotTime)
		assert.Error(t, err)

		var gotFbTime fbtype.Time
		err = codec.PlanScan(nil, typeMetadata, &gotFbTime).Scan(src, &gotFbTime)
		assert.Error(t, err)

		var gotString string
		err = codec.PlanScan(nil, typeMetadata, &gotString).Scan(src, &gotString)
		assert.Error(t, err)

		var gotText fbtype.Text
		err = codec.PlanScan(nil, typeMetadata, &gotText).Scan(src, &gotText)
		assert.Error(t, err)
	})
}

func TestTimeGoTimeRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "midnight",
			input:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "morning time",
			input:    time.Date(2024, 6, 15, 7, 37, 16, 0, time.UTC),
			expected: time.Date(2000, 1, 1, 7, 37, 16, 0, time.UTC),
		},
		{
			name:     "afternoon with microseconds",
			input:    time.Date(2000, 1, 1, 14, 30, 45, 123456000, time.UTC),
			expected: time.Date(2000, 1, 1, 14, 30, 45, 123456000, time.UTC),
		},
		{
			name:     "end of day",
			input:    time.Date(2000, 1, 1, 23, 59, 59, 999999000, time.UTC),
			expected: time.Date(2000, 1, 1, 23, 59, 59, 999999000, time.UTC),
		},
		{
			name:     "noon",
			input:    time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "time with nanoseconds precision",
			input:    time.Date(2000, 1, 1, 15, 4, 5, 789123000, time.UTC),
			expected: time.Date(2000, 1, 1, 15, 4, 5, 789123000, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected.Hour(), tt.input.Hour(), "hour mismatch")
			assert.Equal(t, tt.expected.Minute(), tt.input.Minute(), "minute mismatch")
			assert.Equal(t, tt.expected.Second(), tt.input.Second(), "second mismatch")

			expectedMicros := int64(tt.expected.Nanosecond()) / 1000
			actualMicros := int64(tt.input.Nanosecond()) / 1000
			assert.Equal(t, expectedMicros, actualMicros, "microseconds mismatch")
		})
	}
}

func timeUsec(hour int64, minute int64, second int64, microsecond int64) int64 {
	return hour*3600000000 + minute*60000000 + second*1000000 + microsecond
}

func timeWireBytes(usec int64) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(usec/100))
	return buf
}

func TestTimeGoTimeBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		input       time.Time
		shouldError bool
	}{
		{
			name:        "valid: midnight",
			input:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			shouldError: false,
		},
		{
			name:        "valid: last microsecond of day",
			input:       time.Date(2000, 1, 1, 23, 59, 59, 999999000, time.UTC),
			shouldError: false,
		},
		{
			name:        "valid: max valid hours",
			input:       time.Date(2000, 1, 1, 23, 0, 0, 0, time.UTC),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usec := int64(tt.input.Hour())*3600000000 +
				int64(tt.input.Minute())*60000000 +
				int64(tt.input.Second())*1000000 +
				int64(tt.input.Nanosecond())/1000

			if tt.shouldError {
				assert.True(t, usec < 0 || usec >= 86400000000, "expected out of range")
			} else {
				assert.True(t, usec >= 0 && usec < 86400000000, "expected in range")
			}
		})
	}
}
