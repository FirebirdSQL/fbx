package fbtype

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %q: %v", name, err)
	}
	return loc
}

func assertSameInstant(t *testing.T, got, want time.Time) {
	t.Helper()

	if !got.UTC().Equal(want.UTC()) {
		t.Fatalf("instants differ:\n  got = %s (UTC=%s)\n want = %s (UTC=%s)",
			got, got.UTC(), want, want.UTC())
	}
}

func assertSameOffset(t *testing.T, got, want time.Time) {
	t.Helper()

	_, offGot := got.Zone()
	_, offWant := want.Zone()
	if offGot != offWant {
		t.Fatalf("offsets differ:\n  got  = %d sec\n  want = %d sec", offGot, offWant)
	}
}

func TestTimestamptzCodecRoundTripTime(t *testing.T) {
	var c TimestamptzCodec

	locMSK := mustLoadLocation(t, "Europe/Moscow")
	locNY := mustLoadLocation(t, "America/New_York")
	locKolkata := mustLoadLocation(t, "Asia/Kolkata")
	locKiritimati := mustLoadLocation(t, "Pacific/Kiritimati")

	tests := []struct {
		name string
		time time.Time
	}{
		{
			name: "Y2K Europe/Moscow",
			time: time.Date(2000, 1, 1, 0, 0, 0, 0, locMSK),
		},
		{
			name: "DST America/New_York",
			time: time.Date(2025, 6, 1, 12, 0, 0, 0, locNY),
		},
		{
			name: "Asia/Kolkata (+05:30)",
			time: time.Date(2025, 12, 10, 18, 0, 0, 0, locKolkata),
		},
		{
			name: "max +14 (9999-12-31)",
			time: time.Date(9999, 12, 31, 23, 59, 59, 900000000, locKiritimati),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
			planEnc := c.PlanEncode(nil, typeMetadata, tt.time)
			if planEnc == nil {
				t.Fatalf("PlanEncode returned nil for time.Time")
			}

			buf, err := planEnc.Encode(tt.time, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			var got time.Time
			planScan := c.PlanScan(nil, typeMetadata, &got)
			if planScan == nil {
				t.Fatalf("PlanScan returned nil for *time.Time")
			}

			if err := planScan.Scan(buf, &got); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			assertSameInstant(t, got, tt.time)
			assertSameOffset(t, got, tt.time)
			if !reflect.DeepEqual(got, tt.time) {
				t.Fatalf("time values differ after roundtrip:\n  got  = %#v\n  want = %#v", got, tt.time)
			}
		})
	}
}

func TestTimestamptzCodecRoundTripOffsetLocation(t *testing.T) {
	var c TimestamptzCodec

	tests := []struct {
		name          string
		time          time.Time
		wantLocName   string
		wantDeepEqual bool
	}{
		{
			name:          "canonical offset",
			time:          time.Date(2025, 12, 10, 18, 0, 0, 123400000, time.FixedZone("+04:00", 4*60*60)),
			wantLocName:   "+04:00",
			wantDeepEqual: true,
		},
		{
			name:          "custom fixed zone name",
			time:          time.Date(2025, 12, 10, 18, 0, 0, 123400000, time.FixedZone("MSK", 4*60*60)),
			wantLocName:   "+04:00",
			wantDeepEqual: false,
		},
		{
			name:          "negative half-hour offset",
			time:          time.Date(2025, 12, 10, 18, 0, 0, 123400000, time.FixedZone("-05:30", -(5*60+30)*60)),
			wantLocName:   "-05:30",
			wantDeepEqual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
			planEnc := c.PlanEncode(nil, typeMetadata, tt.time)
			if planEnc == nil {
				t.Fatalf("PlanEncode returned nil for time.Time")
			}

			buf, err := planEnc.Encode(tt.time, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			var got time.Time
			planScan := c.PlanScan(nil, typeMetadata, &got)
			if planScan == nil {
				t.Fatalf("PlanScan returned nil for *time.Time")
			}

			if err := planScan.Scan(buf, &got); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			assertSameInstant(t, got, tt.time)
			assertSameOffset(t, got, tt.time)
			if got.Location().String() != tt.wantLocName {
				t.Fatalf("location name = %q, want %q", got.Location().String(), tt.wantLocName)
			}
			if tt.wantDeepEqual && !reflect.DeepEqual(got, tt.time) {
				t.Fatalf("time values differ after roundtrip:\n  got  = %#v\n  want = %#v", got, tt.time)
			}
			if !tt.wantDeepEqual && reflect.DeepEqual(got, tt.time) {
				t.Fatalf("custom fixed zone name was unexpectedly preserved")
			}
		})
	}
}

func TestTimestamptzCodecRoundTripString(t *testing.T) {
	var c TimestamptzCodec

	tests := []struct {
		name        string
		str         string
		wantLocName string
	}{
		{
			name:        "plus 14",
			str:         "2025-12-10T10:00:00+14:00",
			wantLocName: "+14:00",
		},
		{
			name:        "minus 12",
			str:         "2025-12-10T23:00:00-12:00",
			wantLocName: "-12:00",
		},
		{
			name:        "with fraction",
			str:         "2025-12-10T18:00:00.1234+03:00",
			wantLocName: "+03:00",
		},
		{
			name:        "Firebird named zone",
			str:         "2025-12-17 10:16:43.1234 Europe/Moscow",
			wantLocName: "Europe/Moscow",
		},
		{
			name:        "Firebird single digit fields with named zone",
			str:         "2025-12-17 1:2:3 Europe/Moscow",
			wantLocName: "Europe/Moscow",
		},
		{
			name:        "Firebird compact offset",
			str:         "2025-12-17 10:16:43.1234+03:00",
			wantLocName: "+03:00",
		},
		{
			name:        "Firebird single digit fields with compact offset",
			str:         "2025-12-17 1:2:3+03:00",
			wantLocName: "+03:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
			planEnc := c.PlanEncode(nil, typeMetadata, tt.str)
			if planEnc == nil {
				t.Fatalf("PlanEncode returned nil for string")
			}

			buf, err := planEnc.Encode(tt.str, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if buf == nil {
				t.Fatalf("Encode returned nil buf for non-empty string")
			}

			var got time.Time
			planScan := c.PlanScan(nil, typeMetadata, &got)
			if planScan == nil {
				t.Fatalf("PlanScan returned nil for *time.Time")
			}

			if err := planScan.Scan(buf, &got); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			want, err := parseTimestamptzString(tt.str)
			if err != nil {
				t.Fatalf("parseTimestamptzString failed for %q: %v", tt.str, err)
			}

			assertSameInstant(t, got, want.Time)
			assertSameOffset(t, got, want.Time)
			if got.Location().String() != tt.wantLocName {
				t.Fatalf("location name = %q, want %q", got.Location().String(), tt.wantLocName)
			}
		})
	}
}

func TestTimestamptzCodecEncodeInvalidString(t *testing.T) {
	var c TimestamptzCodec

	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
	planEnc := c.PlanEncode(nil, typeMetadata, "not-a-timestamp")
	if planEnc == nil {
		t.Fatalf("PlanEncode returned nil for string")
	}

	_, err := planEnc.Encode("not-a-timestamp", nil)
	if err == nil {
		t.Fatalf("Encode expected error, got nil")
	}
}

func TestTimestamptzCodecRoundTripTimestamptzStruct(t *testing.T) {
	var c TimestamptzCodec

	locUTC := time.UTC
	locMSK := mustLoadLocation(t, "Europe/Moscow")
	locNY := mustLoadLocation(t, "America/New_York")

	tests := []struct {
		name string
		in   Timestamptz
	}{
		{
			name: "UTC",
			in: Timestamptz{
				Time:  time.Date(2025, 1, 1, 0, 0, 0, 0, locUTC),
				Valid: true,
			},
		},
		{
			name: "Europe/Moscow",
			in: Timestamptz{
				Time:  time.Date(2025, 12, 17, 10, 16, 43, 0, locMSK),
				Valid: true,
			},
		},
		{
			name: "America/New_York DST",
			in: Timestamptz{
				Time:  time.Date(2025, 6, 1, 12, 0, 0, 0, locNY),
				Valid: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
			planEnc := c.PlanEncode(nil, typeMetadata, tt.in)
			if planEnc == nil {
				t.Fatalf("PlanEncode returned nil for TimestamptzValuer")
			}

			buf, err := planEnc.Encode(tt.in, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if buf == nil {
				t.Fatalf("Encode returned nil buf for valid Timestamptz")
			}

			var got Timestamptz
			planScan := c.PlanScan(nil, typeMetadata, &got)
			if planScan == nil {
				t.Fatalf("PlanScan returned nil for *Timestamptz")
			}

			if err := planScan.Scan(buf, &got); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			if !got.Valid {
				t.Fatalf("got.Valid == false, want true")
			}
			assertSameInstant(t, got.Time, tt.in.Time)
			assertSameOffset(t, got.Time, tt.in.Time)
			if !reflect.DeepEqual(got.Time, tt.in.Time) {
				t.Fatalf("time values differ after roundtrip:\n  got  = %#v\n  want = %#v", got.Time, tt.in.Time)
			}
		})
	}
}

func TestTimestamptzCodecEncodeNullStruct(t *testing.T) {
	var c TimestamptzCodec

	in := Timestamptz{Valid: false}

	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
	planEnc := c.PlanEncode(nil, typeMetadata, in)
	if planEnc == nil {
		t.Fatalf("PlanEncode returned nil for TimestamptzValuer")
	}

	buf, err := planEnc.Encode(in, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if buf != nil {
		t.Fatalf("Encode for invalid Timestamptz must return nil buf, got %v", buf)
	}
}

func TestTimestamptzCodecScanNullToTimeReturnsError(t *testing.T) {
	var c TimestamptzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)

	var got time.Time
	planScan := c.PlanScan(nil, typeMetadata, &got)
	if planScan == nil {
		t.Fatalf("PlanScan returned nil for *time.Time")
	}

	if err := planScan.Scan(nil, &got); err == nil {
		t.Fatalf("Scan expected error for NULL into *time.Time, got nil")
	}
}

func TestTimestamptzCodecScanNullToStringReturnsError(t *testing.T) {
	var c TimestamptzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)

	var got string
	planScan := c.PlanScan(nil, typeMetadata, &got)
	if planScan == nil {
		t.Fatalf("PlanScan returned nil for *string")
	}

	if err := planScan.Scan(nil, &got); err == nil {
		t.Fatalf("Scan expected error for NULL into *string, got nil")
	}
}

func TestTimestamptzCodecScanNullToNullableTargets(t *testing.T) {
	var c TimestamptzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)

	var gotTimestamptz Timestamptz
	if err := c.PlanScan(nil, typeMetadata, &gotTimestamptz).Scan(nil, &gotTimestamptz); err != nil {
		t.Fatalf("Scan NULL into *Timestamptz failed: %v", err)
	}
	if gotTimestamptz.Valid {
		t.Fatalf("gotTimestamptz.Valid = true, want false")
	}

	m := NewMap()

	var gotTimePtr *time.Time
	planScan := m.PlanScan(typeMetadata, &gotTimePtr)
	if planScan == nil {
		t.Fatalf("Map PlanScan returned nil for **time.Time")
	}
	if err := planScan.Scan(nil, &gotTimePtr); err != nil {
		t.Fatalf("Map Scan NULL into **time.Time failed: %v", err)
	}
	if gotTimePtr != nil {
		t.Fatalf("gotTimePtr = %v, want nil", gotTimePtr)
	}

	var gotStringPtr *string
	planScan = m.PlanScan(typeMetadata, &gotStringPtr)
	if planScan == nil {
		t.Fatalf("Map PlanScan returned nil for **string")
	}
	if err := planScan.Scan(nil, &gotStringPtr); err != nil {
		t.Fatalf("Map Scan NULL into **string failed: %v", err)
	}
	if gotStringPtr != nil {
		t.Fatalf("gotStringPtr = %v, want nil", gotStringPtr)
	}

	var gotAny any
	planScan = m.PlanScan(typeMetadata, &gotAny)
	if planScan == nil {
		t.Fatalf("Map PlanScan returned nil for *any")
	}
	if err := planScan.Scan(nil, &gotAny); err != nil {
		t.Fatalf("Map Scan NULL into *any failed: %v", err)
	}
	if gotAny != nil {
		t.Fatalf("gotAny = %T(%v), want nil", gotAny, gotAny)
	}
}

func TestTimestamptzCodecScansStringPreservesNamedZone(t *testing.T) {
	var c TimestamptzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)

	locMoscow := mustLoadLocation(t, "Europe/Moscow")
	in := Timestamptz{
		Time:  time.Date(2025, 12, 17, 10, 16, 43, 123400000, locMoscow),
		Valid: true,
	}

	buf, err := c.PlanEncode(nil, typeMetadata, in).Encode(in, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var got string
	if err := c.PlanScan(nil, typeMetadata, &got).Scan(buf, &got); err != nil {
		t.Fatalf("Scan string failed: %v", err)
	}

	want := "2025-12-17T10:16:43.123400+03:00[Europe/Moscow]"
	if got != want {
		t.Fatalf("string scan = %q, want %q", got, want)
	}
}

func TestDecodeTimestamptzBinaryRejectsInvalidInput(t *testing.T) {
	if _, err := decodeTimestamptzBinary([]byte{1, 2, 3}); err == nil {
		t.Fatalf("decodeTimestamptzBinary expected length error, got nil")
	}

	src := make([]byte, 12)
	binary.BigEndian.PutUint32(src[4:8], uint32(microsecondsPerDay/100))
	if _, err := decodeTimestamptzBinary(src); err == nil {
		t.Fatalf("decodeTimestamptzBinary expected time-of-day range error, got nil")
	}

	src = make([]byte, 12)
	binary.BigEndian.PutUint32(src[0:4], uint32(1<<31-1))
	if _, err := decodeTimestamptzBinary(src); err == nil {
		t.Fatalf("decodeTimestamptzBinary expected date range error, got nil")
	}
}

func TestTimestamptzCodecEncodeRejectsUTCDateOutOfRange(t *testing.T) {
	var c TimestamptzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)

	locPlus14 := time.FixedZone("+14:00", 14*60*60)
	locMinus12 := time.FixedZone("-12:00", -12*60*60)
	hugeDate := time.Date(11761246, time.January, 22, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "huge time.Time",
			input: hugeDate,
		},
		{
			name: "huge Timestamptz",
			input: Timestamptz{
				Time:  hugeDate,
				Valid: true,
			},
		},
		{
			name:  "minimum date crosses below UTC range",
			input: time.Date(1, time.January, 1, 0, 0, 0, 0, locPlus14),
		},
		{
			name:  "maximum date crosses above UTC range",
			input: time.Date(9999, time.December, 31, 23, 59, 59, 999900000, locMinus12),
		},
		{
			name:  "minimum string crosses below UTC range",
			input: "0001-01-01 00:00:00 +14:00",
		},
		{
			name:  "maximum string crosses above UTC range",
			input: "9999-12-31 23:59:59.9999 -12:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planEnc := c.PlanEncode(nil, typeMetadata, tt.input)
			if planEnc == nil {
				t.Fatalf("PlanEncode returned nil for %T", tt.input)
			}

			if _, err := planEnc.Encode(tt.input, nil); err == nil {
				t.Fatalf("Encode expected date range error, got nil")
			}
		})
	}
}

func TestTimestamptzMarshalJSON(t *testing.T) {
	locMinus6 := time.FixedZone("TEST-6", -6*60*60)
	locUTC := time.UTC

	tests := []struct {
		name string
		ts   Timestamptz
		want string
	}{
		{
			name: "valid with offset -06:00",
			ts: Timestamptz{
				Time:  time.Date(2012, 1, 2, 3, 4, 5, 6000, locMinus6),
				Valid: true,
			},
			want: `"2012-01-02T03:04:05.000006-06:00"`,
		},
		{
			name: "valid UTC",
			ts: Timestamptz{
				Time:  time.Date(1970, 1, 1, 0, 0, 0, 0, locUTC),
				Valid: true,
			},
			want: `"1970-01-01T00:00:00.000000+00:00[UTC]"`,
		},
		{
			name: "null",
			ts:   Timestamptz{},
			want: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.ts)
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
			if string(b) != tt.want {
				t.Fatalf("unexpected JSON:\n  got  = %s\n  want = %s", string(b), tt.want)
			}
		})
	}
}

func TestTimestamptzUnmarshalJSON(t *testing.T) {
	type tc struct {
		name        string
		json        string
		wantValid   bool
		wantTime    string
		wantLocName string
		wantErr     bool
	}

	tests := []tc{
		{
			name:        "valid RFC3339 with offset -06:00",
			json:        `"2012-01-02T03:04:05.000006-06:00"`,
			wantValid:   true,
			wantTime:    "2012-01-02T03:04:05.000006-06:00",
			wantLocName: "-06:00",
		},
		{
			name:        "valid RFC3339 UTC",
			json:        `"1970-01-01T00:00:00Z"`,
			wantValid:   true,
			wantTime:    "1970-01-01T00:00:00Z",
			wantLocName: "UTC",
		},
		{
			name:        "valid Firebird named zone",
			json:        `"2025-12-17 10:16:43.123400 Europe/Moscow"`,
			wantValid:   true,
			wantTime:    "2025-12-17 10:16:43.123400 Europe/Moscow",
			wantLocName: "Europe/Moscow",
		},
		{
			name:        "valid single digit fields",
			json:        `"2025-12-17 1:2:3 Europe/Moscow"`,
			wantValid:   true,
			wantTime:    "2025-12-17 1:2:3 Europe/Moscow",
			wantLocName: "Europe/Moscow",
		},
		{
			name:      "null",
			json:      `null`,
			wantValid: false,
		},
		{
			name:    "invalid string",
			json:    `"not-a-timestamp"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts Timestamptz
			err := json.Unmarshal([]byte(tt.json), &ts)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}

			if ts.Valid != tt.wantValid {
				t.Fatalf("ts.Valid = %v, want %v", ts.Valid, tt.wantValid)
			}

			if tt.wantValid {
				want, err := parseTimestamptzString(tt.wantTime)
				if err != nil {
					t.Fatalf("parseTimestamptzString(%q) failed: %v", tt.wantTime, err)
				}
				assertSameInstant(t, ts.Time, want.Time)
				assertSameOffset(t, ts.Time, want.Time)
				if ts.Time.Location().String() != tt.wantLocName {
					t.Fatalf("location = %q, want %q", ts.Time.Location().String(), tt.wantLocName)
				}
			}
		})
	}
}

func TestTimestamptzMarshalJSONPreservesNamedZone(t *testing.T) {
	locMoscow := mustLoadLocation(t, "Europe/Moscow")
	ts := Timestamptz{
		Time:  time.Date(2025, 12, 17, 10, 16, 43, 123400000, locMoscow),
		Valid: true,
	}

	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	want := `"2025-12-17T10:16:43.123400+03:00[Europe/Moscow]"`
	if string(b) != want {
		t.Fatalf("JSON = %s, want %s", b, want)
	}

	var decoded Timestamptz
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if !decoded.Valid {
		t.Fatalf("decoded.Valid = false, want true")
	}
	if decoded.Time.Location().String() != "Europe/Moscow" {
		t.Fatalf("decoded location = %q, want Europe/Moscow", decoded.Time.Location().String())
	}
	assertSameInstant(t, decoded.Time, ts.Time)
}

func TestTimestamptzMarshalJSONRejectsSubMinuteOffset(t *testing.T) {
	ts := Timestamptz{
		Time:  time.Date(2025, 12, 10, 12, 0, 0, 0, time.FixedZone("+00:00:30", 30)),
		Valid: true,
	}

	if _, err := json.Marshal(ts); err == nil {
		t.Fatalf("MarshalJSON expected sub-minute offset error, got nil")
	}
}

func TestTimestamptzScanAndValue(t *testing.T) {
	type tc struct {
		name string
		src  time.Time
	}

	locUTC := time.UTC
	locPlus3 := time.FixedZone("PLUS3", 3*60*60)
	locMinus5 := time.FixedZone("MINUS5", -5*60*60)

	tests := []tc{
		{
			name: "UTC",
			src:  time.Date(2025, 1, 1, 0, 0, 0, 0, locUTC),
		},
		{
			name: "offset +03:00",
			src:  time.Date(2025, 12, 10, 18, 0, 0, 0, locPlus3),
		},
		{
			name: "offset -05:00",
			src:  time.Date(2012, 6, 1, 8, 30, 0, 0, locMinus5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts Timestamptz

			if err := ts.Scan(tt.src); err != nil {
				t.Fatalf("Scan(%s) failed: %v", tt.name, err)
			}
			if !ts.Valid {
				t.Fatalf("ts.Valid == false, want true")
			}
			assertSameInstant(t, ts.Time, tt.src)

			v, err := ts.Value()
			if err != nil {
				t.Fatalf("Value() failed: %v", err)
			}
			tv, ok := v.(time.Time)
			if !ok {
				t.Fatalf("Value() type = %T, want time.Time", v)
			}
			assertSameInstant(t, tv, tt.src)

			if err := ts.Scan(nil); err != nil {
				t.Fatalf("Scan(nil) failed: %v", err)
			}
			if ts.Valid {
				t.Fatalf("ts.Valid == true, want false after Scan(nil)")
			}

			v, err = ts.Value()
			if err != nil {
				t.Fatalf("Value() for invalid failed: %v", err)
			}
			if v != nil {
				t.Fatalf("Value() for invalid Timestamptz must be nil, got %v", v)
			}
		})
	}
}

func TestTimestamptzScanUnsupportedType(t *testing.T) {
	var ts Timestamptz
	if err := ts.Scan("not-a-time"); err == nil {
		t.Fatalf("Scan(string) expected error, got nil")
	}
}

func TestTzRawFromTimeRejectsOutOfRangeOffset(t *testing.T) {
	tm := time.Date(2025, 1, 1, 0, 0, 0, 0, time.FixedZone("TOO-FAR", 24*60*60))

	if _, err := tzRawFromTime(tm); err == nil {
		t.Fatalf("tzRawFromTime expected offset range error, got nil")
	}
}

func TestTzRawFromTimeRejectsSubMinuteOffset(t *testing.T) {
	tm := time.Date(2025, 1, 1, 12, 0, 0, 0, time.FixedZone("+00:00:30", 30))

	if _, err := tzRawFromTime(tm); err == nil {
		t.Fatalf("tzRawFromTime expected sub-minute offset error, got nil")
	}
}

func TestTzRawFromTimeAcceptsMaxPositiveOffset(t *testing.T) {
	tm := time.Date(2025, 1, 1, 12, 0, 0, 0, time.FixedZone("+23:59", 1439*60))

	tzRaw, err := tzRawFromTime(tm)
	if err != nil {
		t.Fatalf("tzRawFromTime failed: %v", err)
	}

	if tzRaw != 2878 {
		t.Fatalf("tzRawFromTime = %d, want 2878", tzRaw)
	}
}

func TestTzRawFromTimeFallsBackToOffsetWhenLocationNameOffsetMismatches(t *testing.T) {
	tm := time.Date(2025, 1, 1, 0, 0, 0, 0, time.FixedZone("Europe/Moscow", 9*60*60))

	tzRaw, err := tzRawFromTime(tm)
	if err != nil {
		t.Fatalf("tzRawFromTime failed: %v", err)
	}

	regionRaw, ok := fbproto.ResolveTzRawByLocationName("Europe/Moscow")
	if !ok {
		t.Fatalf("Europe/Moscow timezone ID is not registered")
	}
	if tzRaw == regionRaw {
		t.Fatalf("tzRawFromTime encoded mismatched fixed zone as Europe/Moscow region")
	}

	const wantOffsetRaw = uint16(9*60 + 1439)
	if tzRaw != wantOffsetRaw {
		t.Fatalf("tzRawFromTime = %d, want offset raw %d", tzRaw, wantOffsetRaw)
	}
}

func TestTzRawFromTimePrefersNamedUTC(t *testing.T) {
	tm := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tzRaw, err := tzRawFromTime(tm)
	if err != nil {
		t.Fatalf("tzRawFromTime failed: %v", err)
	}

	wantTzRaw, ok := fbproto.ResolveTzRawByLocationName("UTC")
	if !ok {
		t.Fatalf("UTC timezone ID is not registered")
	}
	if tzRaw != wantTzRaw {
		t.Fatalf("tzRawFromTime = %d, want named UTC raw %d", tzRaw, wantTzRaw)
	}
}

func TestTzRawRoundTripWithResolve(t *testing.T) {
	type tc struct {
		name    string
		locName string
		year    int
		month   time.Month
		day     int
		hour    int
		min     int
		sec     int
	}

	tests := []tc{
		{
			name:    "UTC",
			locName: "UTC",
			year:    2025, month: time.January, day: 1,
			hour: 0, min: 0, sec: 0,
		},
		{
			name:    "Europe/Moscow",
			locName: "Europe/Moscow",
			year:    2025, month: time.December, day: 10,
			hour: 18, min: 0, sec: 0,
		},
		{
			name:    "America/New_York_DST",
			locName: "America/New_York",
			year:    2025, month: time.June, day: 1,
			hour: 12, min: 0, sec: 0,
		},
		{
			name:    "America/New_York_Standard",
			locName: "America/New_York",
			year:    2025, month: time.January, day: 15,
			hour: 12, min: 0, sec: 0,
		},
		{
			name:    "Asia/Kolkata_0530",
			locName: "Asia/Kolkata",
			year:    2025, month: time.December, day: 10,
			hour: 18, min: 0, sec: 0,
		},
		{
			name:    "Pacific/Kiritimati_1400",
			locName: "Pacific/Kiritimati",
			year:    2025, month: time.December, day: 31,
			hour: 23, min: 59, sec: 59,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loc *time.Location
			if tt.locName == "UTC" {
				loc = time.UTC
			} else {
				loc = mustLoadLocation(t, tt.locName)
			}

			tm := time.Date(
				tt.year, tt.month, tt.day,
				tt.hour, tt.min, tt.sec, 0,
				loc,
			)

			tzRaw, err := tzRawFromTime(tm)
			if err != nil {
				t.Fatalf("tzRawFromTime failed for %s: %v", tt.name, err)
			}

			resolvedLoc := fbproto.ResolveLocationByTzRaw(tzRaw)

			_, offOrig := tm.Zone()
			_, offResolved := tm.In(resolvedLoc).Zone()

			if offOrig != offResolved {
				t.Fatalf("%s: offset mismatch after tzRaw roundtrip: orig=%d, resolved=%d",
					tt.name, offOrig, offResolved)
			}
		})
	}
}

func TestTimestamptzStringRoundTripPreservesInstant(t *testing.T) {
	cases := []struct {
		name, zone, utc, text string
	}{
		{"Berlin first", "Europe/Berlin", "2025-10-26T00:30:00Z", "2025-10-26T02:30:00.000000+02:00[Europe/Berlin]"},
		{"Berlin second", "Europe/Berlin", "2025-10-26T01:30:00Z", "2025-10-26T02:30:00.000000+01:00[Europe/Berlin]"},
		{"New York first", "America/New_York", "2025-11-02T05:30:00Z", "2025-11-02T01:30:00.000000-04:00[America/New_York]"},
		{"New York second", "America/New_York", "2025-11-02T06:30:00Z", "2025-11-02T01:30:00.000000-05:00[America/New_York]"},
		{"Lord Howe first", "Australia/Lord_Howe", "2025-04-05T14:45:00Z", "2025-04-06T01:45:00.000000+11:00[Australia/Lord_Howe]"},
		{"Lord Howe second", "Australia/Lord_Howe", "2025-04-05T15:15:00Z", "2025-04-06T01:45:00.000000+10:30[Australia/Lord_Howe]"},
		{"historical seconds", "Europe/Berlin", "1890-01-01T00:00:00Z", "1890-01-01T00:53:28.000000+00:53:28[Europe/Berlin]"},
		{"fraction", "Europe/Moscow", "2025-12-17T07:16:43.123400Z", "2025-12-17T10:16:43.123400+03:00[Europe/Moscow]"},
		{"named UTC", "UTC", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00.000000+00:00[UTC]"},
		{"fixed zero", "+00:00", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00.000000+00:00"},
		{"max positive offset", "+23:59", "2025-01-01T00:00:00Z", "2025-01-01T23:59:00.000000+23:59"},
		{"max negative offset", "-23:59", "2025-01-01T00:00:00Z", "2024-12-31T00:01:00.000000-23:59"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instant, err := time.Parse(time.RFC3339Nano, tc.utc)
			if err != nil {
				t.Fatal(err)
			}
			loc, tzRaw, err := parseTimetzZone(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			original := Timestamptz{Time: instant.In(loc), tzRaw: tzRaw, Valid: true}
			m := NewMap()
			metadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
			wire, err := m.Encode(metadata, original, nil)
			if err != nil {
				t.Fatal(err)
			}
			var text string
			if err := m.Scan(metadata, wire, &text); err != nil {
				t.Fatal(err)
			}
			if text != tc.text {
				t.Fatalf("string = %q, want %q", text, tc.text)
			}
			reencoded, err := m.Encode(metadata, text, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(wire, reencoded) {
				t.Fatalf("string round-trip changed wire value: %x -> %x", wire, reencoded)
			}
			b, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			wantJSON, _ := json.Marshal(tc.text)
			if !bytes.Equal(b, wantJSON) {
				t.Fatalf("JSON = %s, want %s", b, wantJSON)
			}
			var decoded Timestamptz
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatal(err)
			}
			if !decoded.Valid || decoded.tzRaw != tzRaw || decoded.Time.Location().String() != tc.zone {
				t.Fatalf("JSON changed validity or zone: %#v", decoded)
			}
			assertSameInstant(t, decoded.Time, original.Time)
			assertSameOffset(t, decoded.Time, original.Time)
		})
	}
}

func TestTimestamptzLegacyZoneRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		want         string
		wantLocation string
	}{
		{
			name:         "ICU alias in winter",
			input:        "2025-01-15 10:16:43 CST",
			want:         "2025-01-15T10:16:43.000000-06:00[CST]",
			wantLocation: "America/Chicago",
		},
		{
			name:         "ICU alias in summer",
			input:        "2025-07-15 10:16:43 CST",
			want:         "2025-07-15T10:16:43.000000-05:00[CST]",
			wantLocation: "America/Chicago",
		},
		{
			name:         "fixed SystemV zone",
			input:        "2025-07-15 10:16:43 SystemV/CST6",
			want:         "2025-07-15T10:16:43.000000-06:00[SystemV/CST6]",
			wantLocation: "SystemV/CST6",
		},
	}

	m := NewMap()
	metadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire, err := m.Encode(metadata, tt.input, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			var text string
			if err := m.Scan(metadata, wire, &text); err != nil {
				t.Fatalf("Scan string failed: %v", err)
			}
			if text != tt.want {
				t.Fatalf("string = %q, want %q", text, tt.want)
			}

			var decoded Timestamptz
			if err := m.Scan(metadata, wire, &decoded); err != nil {
				t.Fatalf("Scan Timestamptz failed: %v", err)
			}
			if decoded.Time.Location().String() != tt.wantLocation {
				t.Fatalf("location = %q, want %q", decoded.Time.Location(), tt.wantLocation)
			}
		})
	}
}

func TestTimestamptzRejectsInvalidZonedStrings(t *testing.T) {
	cases := []struct{ text, errorPart string }{
		{"2025-10-26T02:30:00+03:00[Europe/Berlin]", "offset"},
		{"2025-03-30T02:30:00+01:00[Europe/Berlin]", "offset"},
		{"2025-10-26T02:30:00[Europe/Berlin]", "offset"},
		{"2025-10-26T02:30:00+02:00[America/Not_A_Zone]", "unknown"},
		{"2025-10-26T02:30:00+02:00[]", "zone suffix"},
		{"2025-10-26T02:30:00+02:00[Europe/Berlin", "zone suffix"},
		{"2025-10-26T02:30:00+02:00Europe/Berlin]", "zone suffix"},
		{"2025-10-26T02:30:00+02:00[[Europe/Berlin]]", "zone suffix"},
		{"2025-10-26T02:30:00+02:00[Europe/Berlin]extra", "zone suffix"},
		{"2025-10-26T02:30:00+02:00[Europe/Berlin][UTC]", "zone suffix"},
		{"2025-10-26T02:30:00+01:60[Europe/Berlin]", "offset"},
		{"2025-10-26T02:30:00+00:59:60[Europe/Berlin]", "offset"},
		{"2025-10-26T02:30:00+24:00[Europe/Berlin]", "offset"},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			m := NewMap()
			metadata := NewTypeMetadata(fbproto.SQL_TYPE_TIMESTAMP_TZ, 0, 0, 0)
			if _, err := m.Encode(metadata, tc.text, nil); err == nil || !strings.Contains(err.Error(), tc.errorPart) {
				t.Fatalf("Encode error = %v, want %q", err, tc.errorPart)
			}
			b, _ := json.Marshal(tc.text)
			var decoded Timestamptz
			if err := json.Unmarshal(b, &decoded); err == nil || !strings.Contains(err.Error(), tc.errorPart) {
				t.Fatalf("Unmarshal error = %v, want %q", err, tc.errorPart)
			}
		})
	}
}

func TestTimestamptzExplicitOffsetFormats(t *testing.T) {
	cases := []struct{ text, utc, zone string }{
		{"2025-10-26T02:30+02:00[Europe/Berlin]", "2025-10-26T00:30:00Z", "Europe/Berlin"},
		{"2025-10-26T02:30+01:00[Europe/Berlin]", "2025-10-26T01:30:00Z", "Europe/Berlin"},
		{"2025-10-26T00:30:00Z[Europe/Berlin]", "2025-10-26T00:30:00Z", "Europe/Berlin"},
		{"2025-10-26T01:30:00Z[Europe/Berlin]", "2025-10-26T01:30:00Z", "Europe/Berlin"},
		{"1890-01-01T00:53+00:53:28[Europe/Berlin]", "1889-12-31T23:59:32Z", "Europe/Berlin"},
		{"2025-10-26T02:30+02:00", "2025-10-26T00:30:00Z", "+02:00"},
		{"2025-10-26T00:30:00+00:00", "2025-10-26T00:30:00Z", "+00:00"},
		{"2025-10-26T00:30:00Z", "2025-10-26T00:30:00Z", "UTC"},
		{"2025-10-26T02:30:00.123456+02:00[Europe/Berlin]", "2025-10-26T00:30:00.123456Z", "Europe/Berlin"},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			b, _ := json.Marshal(tc.text)
			var decoded Timestamptz
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatal(err)
			}
			want, err := time.Parse(time.RFC3339Nano, tc.utc)
			if err != nil {
				t.Fatal(err)
			}
			assertSameInstant(t, decoded.Time, want)
			if decoded.Time.Location().String() != tc.zone {
				t.Fatalf("zone = %s, want %s", decoded.Time.Location(), tc.zone)
			}
		})
	}
}

func TestTimestamptzLegacyLocalTimeResolution(t *testing.T) {
	cases := []struct{ text, utc, errorPart string }{
		{"2025-10-26 02:30 Europe/Berlin", "", "ambiguous"},
		{"2025-11-02 01:30 America/New_York", "", "ambiguous"},
		{"2025-04-06 01:45 Australia/Lord_Howe", "", "ambiguous"},
		{"2025-03-30 02:30 Europe/Berlin", "", "nonexistent"},
		{"2025-10-05 02:15 Australia/Lord_Howe", "", "nonexistent"},
		{"2011-12-30 12:00 Pacific/Apia", "", "nonexistent"},
		{"2025-10-26 01:59:59 Europe/Berlin", "2025-10-25T23:59:59Z", ""},
		{"2025-10-26 03:00 Europe/Berlin", "2025-10-26T02:00:00Z", ""},
		{"2025-03-30 03:00 Europe/Berlin", "2025-03-30T01:00:00Z", ""},
		{"2025-12-17 10:16:43.123456 Europe/Moscow", "2025-12-17T07:16:43.123456Z", ""},
		{"1890-01-01 00:53:28 Europe/Berlin", "1890-01-01T00:00:00Z", ""},
		{"2040-12-31 12:00 Europe/Berlin", "2040-12-31T11:00:00Z", ""},
		{"0001-01-01 00:00 UTC", "0001-01-01T00:00:00Z", ""},
		{"9999-12-31 12:00 Europe/Berlin", "9999-12-31T11:00:00Z", ""},
		{"2025-10-26 02:30 +02:00", "2025-10-26T00:30:00Z", ""},
		{"2025-10-26 02:30 +01:00", "2025-10-26T01:30:00Z", ""},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			b, _ := json.Marshal(tc.text)
			var decoded Timestamptz
			err := json.Unmarshal(b, &decoded)
			if tc.errorPart != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorPart) {
					t.Fatalf("error = %v, want %q", err, tc.errorPart)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want, err := time.Parse(time.RFC3339Nano, tc.utc)
			if err != nil {
				t.Fatal(err)
			}
			assertSameInstant(t, decoded.Time, want)
		})
	}
}
