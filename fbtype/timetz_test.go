package fbtype

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/internal/fbio"
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

	want := time.Date(2020, 1, 1, 18, 0, 0, 123400000, time.FixedZone("+03:30", (3*60+30)*60))
	assertSameTimetzClockAndOffset(t, got, want)
	if got.Location().String() != "+03:30" {
		t.Fatalf("location = %q, want +03:30", got.Location().String())
	}
}

func TestTimetzCodecRejectsSubMinuteOffset(t *testing.T) {
	var c TimetzCodec

	in := time.Date(2025, 12, 10, 12, 0, 0, 0, time.FixedZone("+00:00:30", 30))
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	planEnc := c.PlanEncode(nil, typeMetadata, in)
	if planEnc == nil {
		t.Fatalf("PlanEncode returned nil for time.Time")
	}

	if _, err := planEnc.Encode(in, nil); err == nil {
		t.Fatalf("Encode expected sub-minute offset error, got nil")
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

func TestTimetzCodecNamedLocationUsesFirebirdBaseDate(t *testing.T) {
	var c TimetzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	locSaoPaulo := mustLoadLocation(t, "America/Sao_Paulo")
	tzRaw, ok := fbproto.ResolveTzRawByLocationName("America/Sao_Paulo")
	if !ok {
		t.Fatalf("America/Sao_Paulo timezone ID is not registered")
	}

	in := time.Date(2025, 1, 1, 10, 0, 0, 123400000, locSaoPaulo)
	buf, err := c.PlanEncode(nil, typeMetadata, in).Encode(in, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	wantUTCUsec := int64(13)*microsecondsPerHour + 123400
	if gotTicks := binary.BigEndian.Uint32(buf[0:4]); gotTicks != uint32(wantUTCUsec/100) {
		t.Fatalf("encoded ticks = %d, want %d", gotTicks, wantUTCUsec/100)
	}
	if gotTzRaw := binary.BigEndian.Uint32(buf[4:8]); gotTzRaw != uint32(tzRaw) {
		t.Fatalf("encoded tz raw = %d, want %d", gotTzRaw, tzRaw)
	}

	serverBytes := make([]byte, 8)
	binary.BigEndian.PutUint32(serverBytes[0:4], uint32(wantUTCUsec/100))
	binary.BigEndian.PutUint32(serverBytes[4:8], uint32(tzRaw))

	var got Timetz
	if err := c.PlanScan(nil, typeMetadata, &got).Scan(serverBytes, &got); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if !got.Valid {
		t.Fatalf("got.Valid = false, want true")
	}

	want := time.Date(2020, 1, 1, 10, 0, 0, 123400000, locSaoPaulo)
	assertSameTimetzClockAndOffset(t, got.Time, want)
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
			name:        "single digit fields and named zone",
			input:       "1:2:3 Europe/Moscow",
			wantHour:    1,
			wantMinute:  2,
			wantSecond:  3,
			wantLocName: "Europe/Moscow",
		},
		{
			name:        "single digit fields and compact offset",
			input:       "1:2:3+03:00",
			wantHour:    1,
			wantMinute:  2,
			wantSecond:  3,
			wantLocName: "+03:00",
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

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{
			name:  "offset zone",
			input: time.Date(2025, 12, 10, 18, 0, 0, 123400000, time.FixedZone("+03:00", 3*60*60)),
			want:  "18:00:00.123400 +03:00",
		},
		{
			name:  "named zone",
			input: "11:31 Europe/Moscow",
			want:  "11:31:00.000000 Europe/Moscow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := c.PlanEncode(nil, typeMetadata, tt.input).Encode(tt.input, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			var gotString string
			if err := c.PlanScan(nil, typeMetadata, &gotString).Scan(buf, &gotString); err != nil {
				t.Fatalf("Scan string failed: %v", err)
			}
			if gotString != tt.want {
				t.Fatalf("string scan = %q, want %q", gotString, tt.want)
			}

			var gotText Text
			if err := c.PlanScan(nil, typeMetadata, &gotText).Scan(buf, &gotText); err != nil {
				t.Fatalf("Scan Text failed: %v", err)
			}
			if !gotText.Valid || gotText.String != gotString {
				t.Fatalf("Text scan = %#v, want valid %q", gotText, gotString)
			}
		})
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

func TestTimetzCodecScanNullToTimeReturnsError(t *testing.T) {
	var c TimetzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	var got time.Time
	planScan := c.PlanScan(nil, typeMetadata, &got)
	if planScan == nil {
		t.Fatalf("PlanScan returned nil for *time.Time")
	}

	if err := planScan.Scan(nil, &got); err == nil {
		t.Fatalf("Scan expected error for NULL into *time.Time, got nil")
	}
}

func TestTimetzCodecScanNullToStringReturnsError(t *testing.T) {
	var c TimetzCodec
	typeMetadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)

	var got string
	planScan := c.PlanScan(nil, typeMetadata, &got)
	if planScan == nil {
		t.Fatalf("PlanScan returned nil for *string")
	}

	if err := planScan.Scan(nil, &got); err == nil {
		t.Fatalf("Scan expected error for NULL into *string, got nil")
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
	for n := 0; n <= 9; n++ {
		if n == 8 {
			continue
		}
		t.Run(fmt.Sprintf("%d_bytes", n), func(t *testing.T) {
			got, err := decodeTimetzBinary(bytes.Repeat([]byte{0xff}, n))
			if n < 8 {
				if !errors.Is(err, fbio.ErrInsufficientBytes) {
					t.Fatalf("error = %v, want ErrInsufficientBytes", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "unexpected trailing bytes") {
				t.Fatalf("error = %v, want trailing bytes error", err)
			}
			if got != (Timetz{}) {
				t.Fatalf("result = %+v, want zero value", got)
			}
		})
	}
	if got, err := decodeTimetzBinary(nil); err != nil || got != (Timetz{}) {
		t.Fatalf("decodeTimetzBinary(nil) = %+v, %v, want zero value, nil", got, err)
	}

	src := make([]byte, 8)
	binary.BigEndian.PutUint32(src[0:4], uint32(microsecondsPerDay/100))
	if _, err := decodeTimetzBinary(src); err == nil {
		t.Fatalf("decodeTimetzBinary expected time-of-day range error, got nil")
	}
}

func TestDecodeTimetzBinaryCheckedReadPreservesZoneField(t *testing.T) {
	const tzRaw = 1439 + 3*60
	src := binary.BigEndian.AppendUint32(nil, 1)
	src = binary.BigEndian.AppendUint32(src, 0xabcd0000|tzRaw)
	got, err := decodeTimetzBinary(src)
	if err != nil {
		t.Fatalf("decodeTimetzBinary failed: %v", err)
	}
	if !got.Valid || got.tzRaw != tzRaw {
		t.Fatalf("result = %+v, want valid value with tzRaw %d", got, tzRaw)
	}
	const want = "2020-01-01T03:00:00.0001+03:00"
	if formatted := got.Time.Format(time.RFC3339Nano); formatted != want {
		t.Fatalf("time = %s, want %s", formatted, want)
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

	if err := json.Unmarshal([]byte(`"1:2:3 Europe/Moscow"`), &decoded); err != nil {
		t.Fatalf("UnmarshalJSON single-digit fields failed: %v", err)
	}
	if decoded.Time.Hour() != 1 || decoded.Time.Minute() != 2 || decoded.Time.Second() != 3 || decoded.Time.Location().String() != "Europe/Moscow" {
		t.Fatalf("decoded single-digit time = %s, want 01:02:03 Europe/Moscow", decoded.Time)
	}

	if err := ttz.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) failed: %v", err)
	}
	if ttz.Valid {
		t.Fatalf("ttz.Valid = true, want false after Scan(nil)")
	}
}

func TestTimetzMarshalJSONPreservesNamedZone(t *testing.T) {
	var parsed Timetz
	if err := parsed.Scan("11:31 Europe/Moscow"); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	b, err := json.Marshal(parsed)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(b) != `"11:31:00.000000 Europe/Moscow"` {
		t.Fatalf("JSON = %s, want %s", b, `"11:31:00.000000 Europe/Moscow"`)
	}

	locMoscow := mustLoadLocation(t, "Europe/Moscow")
	constructed := Timetz{
		Time:  time.Date(2025, 12, 17, 10, 16, 43, 123400000, locMoscow),
		Valid: true,
	}

	b, err = json.Marshal(constructed)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(b) != `"10:16:43.123400 Europe/Moscow"` {
		t.Fatalf("JSON = %s, want %s", b, `"10:16:43.123400 Europe/Moscow"`)
	}

	var decoded Timetz
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if !decoded.Valid {
		t.Fatalf("decoded.Valid = false, want true")
	}
	if decoded.Time.Location().String() != "Europe/Moscow" {
		t.Fatalf("decoded location = %q, want Europe/Moscow", decoded.Time.Location().String())
	}
}

func TestTimetzLegacyZoneRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		want         string
		wantLocation string
		wantOffset   int
	}{
		{
			name:         "ICU alias",
			input:        "10:16:43 CST",
			want:         "10:16:43.000000 CST",
			wantLocation: "America/Chicago",
			wantOffset:   -6 * 60 * 60,
		},
		{
			name:         "fixed SystemV zone",
			input:        "10:16:43 SystemV/CST6",
			want:         "10:16:43.000000 SystemV/CST6",
			wantLocation: "SystemV/CST6",
			wantOffset:   -6 * 60 * 60,
		},
	}

	m := NewMap()
	metadata := NewTypeMetadata(fbproto.SQL_TYPE_TIME_TZ, 0, 0, 0)
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

			var decoded Timetz
			if err := m.Scan(metadata, wire, &decoded); err != nil {
				t.Fatalf("Scan Timetz failed: %v", err)
			}
			if decoded.Time.Location().String() != tt.wantLocation {
				t.Fatalf("location = %q, want %q", decoded.Time.Location(), tt.wantLocation)
			}
			_, offset := decoded.Time.Zone()
			if offset != tt.wantOffset {
				t.Fatalf("offset = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

func TestTimetzRejectsUnsupportedSystemVDSTZones(t *testing.T) {
	zones := []string{
		"SystemV/AST4ADT",
		"SystemV/CST6CDT",
		"SystemV/EST5EDT",
		"SystemV/MST7MDT",
		"SystemV/PST8PDT",
		"SystemV/YST9YDT",
	}

	for _, zone := range zones {
		t.Run(zone, func(t *testing.T) {
			if _, _, err := parseTimetzZone(zone); err == nil || !strings.Contains(err.Error(), zone) {
				t.Fatalf("parseTimetzZone(%q) error = %v, want error containing zone name", zone, err)
			}

			tzRaw, ok := fbproto.ResolveTzRawByLocationName(zone)
			if !ok {
				t.Fatalf("time zone %q is not registered", zone)
			}
			src := make([]byte, 8)
			binary.BigEndian.PutUint32(src[4:8], uint32(tzRaw))
			if _, err := decodeTimetzBinary(src); err == nil || !strings.Contains(err.Error(), zone) {
				t.Fatalf("decodeTimetzBinary(%q) error = %v, want error containing zone name", zone, err)
			}
		})
	}
}

func TestTimetzMarshalJSONRejectsSubMinuteOffset(t *testing.T) {
	ttz := Timetz{
		Time:  time.Date(2025, 12, 10, 12, 0, 0, 0, time.FixedZone("+00:00:30", 30)),
		Valid: true,
	}

	if _, err := json.Marshal(ttz); err == nil {
		t.Fatalf("MarshalJSON expected sub-minute offset error, got nil")
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
