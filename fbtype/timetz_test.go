package fbtype

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

func assertSameTimetzClockAndOffset(t *testing.T, got, want time.Time) {
	t.Helper()

	if got.Hour() != want.Hour() || got.Minute() != want.Minute() || got.Second() != want.Second() || got.Nanosecond() != want.Nanosecond() {
		t.Fatalf("time-of-day differs:\n  got  = %s\n  want = %s", got, want)
	}

	_, gotOffset := got.Zone()
	_, wantOffset := want.Zone()
	if gotOffset != wantOffset {
		t.Fatalf("offset differs: got %d sec, want %d sec", gotOffset, wantOffset)
	}
}

func TestTimetzCodecRoundTripOffsetLocation(t *testing.T) {
	var c TimetzCodec

	in := time.Date(2025, 12, 10, 18, 0, 0, 123400000, time.FixedZone("+03:30", (3*60+30)*60))
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	planEnc := c.PlanEncode(nil, typeMetadata, in)
	if planEnc == nil {
		t.Fatalf("PlanEncode returned nil for time.Time")
	}

	buf, err := planEnc.Encode(in, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(buf) != 8 {
		t.Fatalf("encoded length = %d, want 8", len(buf))
	}

	wantUTCUsec := int64(14)*microsecondsPerHour +
		int64(30)*microsecondsPerMinute +
		int64(0)*microsecondsPerSecond +
		123400
	if gotTicks := binary.BigEndian.Uint32(buf[0:4]); gotTicks != uint32(wantUTCUsec/100) {
		t.Fatalf("encoded ticks = %d, want %d", gotTicks, wantUTCUsec/100)
	}
	if gotTzRaw := binary.BigEndian.Uint32(buf[4:8]); gotTzRaw != uint32(1439+3*60+30) {
		t.Fatalf("encoded tz raw = %d, want %d", gotTzRaw, 1439+3*60+30)
	}

	var got time.Time
	planScan := c.PlanScan(nil, typeMetadata, &got)
	if planScan == nil {
		t.Fatalf("PlanScan returned nil for *time.Time")
	}
	if err := planScan.Scan(buf, &got); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	want := time.Date(2000, 1, 1, 18, 0, 0, 123400000, time.FixedZone("+03:30", (3*60+30)*60))
	assertSameTimetzClockAndOffset(t, got, want)
	if got.Location().String() != "+03:30" {
		t.Fatalf("location = %q, want +03:30", got.Location().String())
	}
}

func TestTimetzCodecRoundTripNamedLocation(t *testing.T) {
	var c TimetzCodec

	locMSK := mustLoadLocation(t, "Europe/Moscow")
	in := Timetz{
		Time:  time.Date(2025, 12, 17, 10, 16, 43, 0, locMSK),
		Valid: true,
	}
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	planEnc := c.PlanEncode(nil, typeMetadata, in)
	if planEnc == nil {
		t.Fatalf("PlanEncode returned nil for Timetz")
	}

	buf, err := planEnc.Encode(in, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var got Timetz
	planScan := c.PlanScan(nil, typeMetadata, &got)
	if planScan == nil {
		t.Fatalf("PlanScan returned nil for *Timetz")
	}
	if err := planScan.Scan(buf, &got); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if !got.Valid {
		t.Fatalf("got.Valid = false, want true")
	}
	if got.Time.Location().String() != "Europe/Moscow" {
		t.Fatalf("location = %q, want Europe/Moscow", got.Time.Location().String())
	}
	if got.Time.Hour() != 10 || got.Time.Minute() != 16 || got.Time.Second() != 43 {
		t.Fatalf("time-of-day = %s, want 10:16:43", got.Time)
	}
}

func TestTimetzCodecNamedLocationIgnoresInputDateForWallClock(t *testing.T) {
	var c TimetzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	tests := []struct {
		name       string
		locName    string
		inputMonth time.Month
		inputDay   int
		hour       int
	}{
		{
			name:       "Europe/London summer input uses winter anchor",
			locName:    "Europe/London",
			inputMonth: time.July,
			inputDay:   15,
			hour:       10,
		},
		{
			name:       "America/New_York summer input uses winter anchor",
			locName:    "America/New_York",
			inputMonth: time.July,
			inputDay:   15,
			hour:       10,
		},
		{
			name:       "Australia/Sydney winter input uses summer anchor",
			locName:    "Australia/Sydney",
			inputMonth: time.July,
			inputDay:   15,
			hour:       8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := mustLoadLocation(t, tt.locName)
			in := time.Date(2025, tt.inputMonth, tt.inputDay, tt.hour, 0, 0, 0, loc)

			buf, err := c.PlanEncode(nil, typeMetadata, in).Encode(in, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			var got Timetz
			if err := c.PlanScan(nil, typeMetadata, &got).Scan(buf, &got); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			if !got.Valid {
				t.Fatalf("got.Valid = false, want true")
			}
			if got.Time.Location().String() != tt.locName {
				t.Fatalf("location = %q, want %q", got.Time.Location().String(), tt.locName)
			}
			if got.Time.Hour() != tt.hour || got.Time.Minute() != 0 || got.Time.Second() != 0 || got.Time.Nanosecond() != 0 {
				t.Fatalf("time-of-day = %s, want %02d:00:00", got.Time, tt.hour)
			}
		})
	}
}

func TestTimetzCodecRoundTripString(t *testing.T) {
	var c TimetzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	tests := []struct {
		name        string
		input       string
		wantHour    int
		wantMinute  int
		wantSecond  int
		wantNanos   int
		wantLocName string
	}{
		{
			name:        "offset with space",
			input:       "11:31:12.1234 +03:30",
			wantHour:    11,
			wantMinute:  31,
			wantSecond:  12,
			wantNanos:   123400000,
			wantLocName: "+03:30",
		},
		{
			name:        "compact offset",
			input:       "04:05:06.789-08",
			wantHour:    4,
			wantMinute:  5,
			wantSecond:  6,
			wantNanos:   789000000,
			wantLocName: "-08:00",
		},
		{
			name:        "named zone without seconds",
			input:       "11:31 Europe/Moscow",
			wantHour:    11,
			wantMinute:  31,
			wantLocName: "Europe/Moscow",
		},
		{
			name:        "UTC suffix",
			input:       "12:00:00Z",
			wantHour:    12,
			wantLocName: "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planEnc := c.PlanEncode(nil, typeMetadata, tt.input)
			if planEnc == nil {
				t.Fatalf("PlanEncode returned nil for string")
			}

			buf, err := planEnc.Encode(tt.input, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			var got Timetz
			planScan := c.PlanScan(nil, typeMetadata, &got)
			if planScan == nil {
				t.Fatalf("PlanScan returned nil for *Timetz")
			}
			if err := planScan.Scan(buf, &got); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			if !got.Valid {
				t.Fatalf("got.Valid = false, want true")
			}
			if got.Time.Hour() != tt.wantHour || got.Time.Minute() != tt.wantMinute || got.Time.Second() != tt.wantSecond || got.Time.Nanosecond() != tt.wantNanos {
				t.Fatalf("time-of-day = %s, want %02d:%02d:%02d.%09d",
					got.Time, tt.wantHour, tt.wantMinute, tt.wantSecond, tt.wantNanos)
			}
			if got.Time.Location().String() != tt.wantLocName {
				t.Fatalf("location = %q, want %q", got.Time.Location().String(), tt.wantLocName)
			}
		})
	}
}

func TestTimetzCodecScansStringAndText(t *testing.T) {
	var c TimetzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)
	in := time.Date(2025, 12, 10, 18, 0, 0, 123400000, time.FixedZone("+03:00", 3*60*60))

	buf, err := c.PlanEncode(nil, typeMetadata, in).Encode(in, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var gotString string
	if err := c.PlanScan(nil, typeMetadata, &gotString).Scan(buf, &gotString); err != nil {
		t.Fatalf("Scan string failed: %v", err)
	}
	if gotString != "18:00:00.123400 +03:00" {
		t.Fatalf("string scan = %q, want %q", gotString, "18:00:00.123400 +03:00")
	}

	var gotText Text
	if err := c.PlanScan(nil, typeMetadata, &gotText).Scan(buf, &gotText); err != nil {
		t.Fatalf("Scan Text failed: %v", err)
	}
	if !gotText.Valid || gotText.String != gotString {
		t.Fatalf("Text scan = %#v, want valid %q", gotText, gotString)
	}
}

func TestTimetzCodecEncodeNullStruct(t *testing.T) {
	var c TimetzCodec

	in := Timetz{Valid: false}
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)
	planEnc := c.PlanEncode(nil, typeMetadata, in)
	if planEnc == nil {
		t.Fatalf("PlanEncode returned nil for TimetzValuer")
	}

	buf, err := planEnc.Encode(in, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if buf != nil {
		t.Fatalf("Encode for invalid Timetz must return nil buf, got %v", buf)
	}
}

func TestTimetzCodecEncodeInvalidString(t *testing.T) {
	var c TimetzCodec

	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)
	planEnc := c.PlanEncode(nil, typeMetadata, "not-a-time")
	if planEnc == nil {
		t.Fatalf("PlanEncode returned nil for string")
	}

	if _, err := planEnc.Encode("not-a-time", nil); err == nil {
		t.Fatalf("Encode expected error, got nil")
	}
}

func TestDecodeTimetzBinaryRejectsInvalidInput(t *testing.T) {
	if _, err := decodeTimetzBinary([]byte{1, 2, 3}); err == nil {
		t.Fatalf("decodeTimetzBinary expected length error, got nil")
	}

	src := make([]byte, 8)
	binary.BigEndian.PutUint32(src[0:4], uint32(microsecondsPerDay/100))
	if _, err := decodeTimetzBinary(src); err == nil {
		t.Fatalf("decodeTimetzBinary expected time-of-day range error, got nil")
	}
}

func TestTimetzScanValueAndJSON(t *testing.T) {
	src := "03:04:05.000006 -06:00"

	var ttz Timetz
	if err := ttz.Scan(src); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if !ttz.Valid {
		t.Fatalf("ttz.Valid = false, want true")
	}

	value, err := ttz.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	if _, ok := value.(time.Time); !ok {
		t.Fatalf("Value type = %T, want time.Time", value)
	}

	b, err := json.Marshal(ttz)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(b) != `"03:04:05.000006 -06:00"` {
		t.Fatalf("JSON = %s, want %s", b, `"03:04:05.000006 -06:00"`)
	}

	var decoded Timetz
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if !reflect.DeepEqual(ttz.Time, decoded.Time) {
		t.Fatalf("decoded time differs:\n got  = %#v\n want = %#v", decoded.Time, ttz.Time)
	}

	if err := ttz.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) failed: %v", err)
	}
	if ttz.Valid {
		t.Fatalf("ttz.Valid = true, want false after Scan(nil)")
	}
}

func TestTimeWrapperTimetz(t *testing.T) {
	in := time.Date(2025, 7, 1, 10, 30, 45, 123400000, time.FixedZone("+03:00", 3*60*60))

	v, err := timeWrapper(in).TimetzValue()
	if err != nil {
		t.Fatalf("TimetzValue failed: %v", err)
	}
	if !v.Valid || !reflect.DeepEqual(v.Time, in) {
		t.Fatalf("TimetzValue = %#v, want valid %s", v, in)
	}

	var out timeWrapper
	if err := (&out).ScanTimetz(Timetz{Time: in, Valid: true}); err != nil {
		t.Fatalf("ScanTimetz failed: %v", err)
	}
	if !reflect.DeepEqual(time.Time(out), in) {
		t.Fatalf("ScanTimetz time = %#v, want %#v", time.Time(out), in)
	}

	if err := (&out).ScanTimetz(Timetz{}); err == nil {
		t.Fatalf("ScanTimetz expected NULL error, got nil")
	}
}
