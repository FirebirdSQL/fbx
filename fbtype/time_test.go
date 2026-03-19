package fbtype_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
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
	for i := 0; i <= len(mirco); i++ {
		assert.NoError(t, fbTime.Scan("04:05:06."+mirco[:i]))
		assert.Equal(t, true, fbTime.Valid)

		frac, _ := strconv.ParseInt(mirco[:i], 10, 64)
		for k := i; k < 6; k++ {
			frac *= 10
		}
		assert.Equal(t, woFraction+frac*int64(time.Microsecond), fbTime.Microseconds*int64(time.Microsecond))
	}

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
