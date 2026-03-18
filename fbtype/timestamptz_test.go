package fbtype

import (
	"encoding/json"
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

func TestTimestamptzCodec_RoundTrip_Time(t *testing.T) {
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
		})
	}
}

func TestTimestamptzCodec_RoundTrip_String(t *testing.T) {
	var c TimestamptzCodec

	tests := []struct {
		name string
		str  string
	}{
		{
			name: "plus 14",
			str:  "2025-12-10T10:00:00+14:00",
		},
		{
			name: "minus 12",
			str:  "2025-12-10T23:00:00-12:00",
		},
		{
			name: "with fraction",
			str:  "2025-12-10T18:00:00.1234+03:00",
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

			want, err := time.Parse(time.RFC3339Nano, tt.str)
			if err != nil {
				t.Fatalf("time.Parse failed for %q: %v", tt.str, err)
			}

			assertSameInstant(t, got, want)
			assertSameOffset(t, got, want)
		})
	}
}

func TestTimestamptzCodec_RoundTrip_TimestamptzStruct(t *testing.T) {
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
		})
	}
}

func TestTimestamptzCodec_Encode_NullStruct(t *testing.T) {
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

func TestTimestamptz_MarshalJSON(t *testing.T) {
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
			want: `"1970-01-01T00:00:00Z"`,
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

func TestTimestamptz_UnmarshalJSON(t *testing.T) {
	type tc struct {
		name      string
		json      string
		wantValid bool
		wantTime  string
		wantErr   bool
	}

	tests := []tc{
		{
			name:      "valid with offset -06:00",
			json:      `"2012-01-02T03:04:05.000006-06:00"`,
			wantValid: true,
			wantTime:  "2012-01-02T03:04:05.000006-06:00",
		},
		{
			name:      "valid UTC",
			json:      `"1970-01-01T00:00:00Z"`,
			wantValid: true,
			wantTime:  "1970-01-01T00:00:00Z",
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
				want, err := time.Parse(time.RFC3339Nano, tt.wantTime)
				if err != nil {
					t.Fatalf("time.Parse(%q) failed: %v", tt.wantTime, err)
				}
				assertSameInstant(t, ts.Time, want)
				assertSameOffset(t, ts.Time, want)
			}
		})
	}
}

func TestTimestamptz_ScanAndValue(t *testing.T) {
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

func TestTimestamptz_Scan_UnsupportedType(t *testing.T) {
	var ts Timestamptz
	if err := ts.Scan("not-a-time"); err == nil {
		t.Fatalf("Scan(string) expected error, got nil")
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
			_, offResolved := time.Date(2025, 1, 1, 0, 0, 0, 0, resolvedLoc).Zone()

			if offOrig != offResolved {
				t.Fatalf("%s: offset mismatch after tzRaw roundtrip: orig=%d, resolved=%d",
					tt.name, offOrig, offResolved)
			}
		})
	}
}
