package fbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx/internal/fbio"
)

const (
	secondsPerDay         = int64(24 * 60 * 60)
	microsecondsPerSecond = 1000000
	microsecondsPerMinute = 60 * microsecondsPerSecond
	microsecondsPerHour   = 60 * microsecondsPerMinute
	microsecondsPerDay    = 24 * microsecondsPerHour
	microsecondsPerMonth  = 30 * microsecondsPerDay
)

var (
	// timeAnchorDate is the arbitrary date used when Firebird TIME values must
	// be represented as time.Time.
	timeAnchorDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	// timetzBaseDate is the Firebird base date used to resolve named time zones
	// for TIME WITH TIME ZONE.
	timetzBaseDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

func timeOfDayMicroseconds(t time.Time) int64 {
	return int64(t.Hour())*microsecondsPerHour +
		int64(t.Minute())*microsecondsPerMinute +
		int64(t.Second())*microsecondsPerSecond +
		int64(t.Nanosecond())/1000
}

func timeFromMicroseconds(usec int64, date time.Time, loc *time.Location) time.Time {
	hours := usec / microsecondsPerHour
	usec -= hours * microsecondsPerHour
	minutes := usec / microsecondsPerMinute
	usec -= minutes * microsecondsPerMinute
	seconds := usec / microsecondsPerSecond
	usec -= seconds * microsecondsPerSecond

	return time.Date(
		date.Year(), date.Month(), date.Day(),
		int(hours), int(minutes), int(seconds), int(usec)*1000,
		loc,
	)
}

func timeWithDate(t time.Time, date time.Time) time.Time {
	return time.Date(
		date.Year(), date.Month(), date.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
		t.Location(),
	)
}

// TimeScanner scans a Firebird TIME value.
type TimeScanner interface {
	// ScanTime scans v into the receiver.
	ScanTime(v Time) error
}

// TimeValuer returns a Firebird TIME value.
type TimeValuer interface {
	// TimeValue returns the Firebird TIME value to encode.
	TimeValue() (Time, error)
}

// Time represents the Firebird TIME type.
//
// Microseconds contains the time of day as microseconds since midnight.
// Firebird stores TIME on the wire as the number of 100-microsecond ticks since
// midnight, so values are truncated to that precision when encoded. Valid is
// false when the value represents SQL NULL.
type Time struct {
	Microseconds int64
	Valid        bool
}

// ScanTime implements [TimeScanner].
func (t *Time) ScanTime(v Time) error {
	*t = v
	return nil
}

// TimeValue implements [TimeValuer].
func (t Time) TimeValue() (Time, error) {
	return t, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (t *Time) Scan(src any) error {
	if src == nil {
		*t = Time{}
		return nil
	}

	switch src := src.(type) {
	case string:
		usec, err := parseTimeString(src)
		if err != nil {
			*t = Time{}
			return err
		}
		t.Microseconds = usec
		t.Valid = true
		return nil
	case time.Time:
		t.Microseconds = timeOfDayMicroseconds(src)
		t.Valid = true
		return nil
	}

	*t = Time{}
	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (t Time) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}

	return timeFromTimeValue(t)
}

// MarshalJSON encodes t as a JSON string in HH:MM:SS.ffffff format or null.
func (t Time) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}

	s, err := formatTimeString(t.Microseconds)
	if err != nil {
		return nil, err
	}

	return json.Marshal(s)
}

// UnmarshalJSON decodes a JSON time string or null into t.
func (t *Time) UnmarshalJSON(b []byte) error {
	var s *string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	if s == nil {
		*t = Time{}
		return nil
	}

	usec, err := parseTimeString(*s)
	if err != nil {
		*t = Time{}
		return err
	}

	*t = Time{Microseconds: usec, Valid: true}
	return nil
}

// TimeCodec encodes and decodes Firebird TIME values.
type TimeCodec struct{}

// PlanEncode returns an encode plan for supported Firebird TIME input values.
func (TimeCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case TimeValuer:
		return encodePlanTimeCodecBinary{}
	case string:
		return encodePlanTimeCodecBinaryString{}
	case time.Time:
		return encodePlanTimeCodecBinaryTime{}
	}

	return nil
}

type encodePlanTimeCodecBinary struct{}

func (encodePlanTimeCodecBinary) Encode(value any, buf []byte) ([]byte, error) {
	t, err := value.(TimeValuer).TimeValue()
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, nil
	}

	usec := t.Microseconds
	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIME value out of range for Firebird (µs since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)

	buf = fbio.AppendInt32BE(buf, int32(ticks))

	return buf, nil
}

type encodePlanTimeCodecBinaryString struct{}

func (encodePlanTimeCodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	usec, err := parseTimeString(value.(string))
	if err != nil {
		return nil, err
	}

	ticks := uint32(usec / 100)
	buf = fbio.AppendInt32BE(buf, int32(ticks))

	return buf, nil
}

type encodePlanTimeCodecBinaryTime struct{}

func (encodePlanTimeCodecBinaryTime) Encode(value any, buf []byte) ([]byte, error) {
	t := value.(time.Time)

	usec := timeOfDayMicroseconds(t)
	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIME value out of range (µs since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)

	buf = fbio.AppendInt32BE(buf, int32(ticks))

	return buf, nil
}

// PlanScan returns a scan plan for supported Firebird TIME target values.
func (TimeCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case TimeScanner:
		return scanPlanBinaryTimeToTimeScanner{}
	case TextScanner:
		return scanPlanBinaryTimeToTextScanner{}
	case *string:
		return scanPlanBinaryTimeToString{}
	case *time.Time:
		return scanPlanBinaryTimeToTime{}
	}

	return nil
}

type scanPlanBinaryTimeToTime struct{}

func (scanPlanBinaryTimeToTime) Scan(src []byte, dst any) error {
	target, ok := dst.(*time.Time)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}

	usec, _, err := decodeTimeBinary(src)
	if err != nil {
		return err
	}

	*target = timeFromMicroseconds(usec, timeAnchorDate, time.UTC)

	return nil
}

type scanPlanBinaryTimeToTimeScanner struct{}

func (scanPlanBinaryTimeToTimeScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TimeScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanTime(Time{})
	}

	usec, _, err := decodeTimeBinary(src)
	if err != nil {
		return err
	}

	return scanner.ScanTime(Time{
		Microseconds: usec,
		Valid:        true,
	})
}

type scanPlanBinaryTimeToTextScanner struct{}

func (scanPlanBinaryTimeToTextScanner) Scan(src []byte, dst any) error {
	ts, ok := (dst).(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return ts.ScanText(Text{})
	}

	_, formatted, err := decodeTimeBinary(src)
	if err != nil {
		return err
	}

	return ts.ScanText(Text{String: formatted, Valid: true})
}

// scanPlanBinaryTimeToString scans Firebird TIME into *string.
type scanPlanBinaryTimeToString struct{}

func (scanPlanBinaryTimeToString) Scan(src []byte, dst any) error {
	target, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	_, formatted, err := decodeTimeBinary(src)
	if err != nil {
		return err
	}

	*target = formatted
	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c TimeCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	usec, _, err := decodeTimeBinary(src)
	if err != nil {
		return nil, err
	}

	return timeFromMicroseconds(usec, timeAnchorDate, time.UTC), nil
}

// DecodeValue decodes src into the default Go value for Firebird TIME.
func (c TimeCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var t Time
	err := codecScan(c, m, typeMetadata, src, &t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func decodeTimeBinary(src []byte) (int64, string, error) {
	ticks, err := fbio.Uint32BEExact(src)
	if err != nil {
		return 0, "", fmt.Errorf("fbtype: Firebird TIME: %w", err)
	}

	usec := int64(ticks) * 100
	if err := validateTimeMicroseconds(usec); err != nil {
		return 0, "", err
	}

	formatted, err := formatTimeString(usec)
	if err != nil {
		return 0, "", err
	}

	return usec, formatted, nil
}

func timeFromTimeValue(t Time) (time.Time, error) {
	if err := validateTimeMicroseconds(t.Microseconds); err != nil {
		return time.Time{}, err
	}

	return timeFromMicroseconds(t.Microseconds, timeAnchorDate, time.UTC), nil
}

func formatTimeString(usec int64) (string, error) {
	if err := validateTimeMicroseconds(usec); err != nil {
		return "", err
	}

	hours := usec / microsecondsPerHour
	usec -= hours * microsecondsPerHour
	minutes := usec / microsecondsPerMinute
	usec -= minutes * microsecondsPerMinute
	seconds := usec / microsecondsPerSecond
	usec -= seconds * microsecondsPerSecond

	formatted := fmt.Sprintf("%02d:%02d:%02d.%06d", hours, minutes, seconds, usec)

	return formatted, nil
}

func validateTimeMicroseconds(usec int64) error {
	if usec < 0 || usec >= microsecondsPerDay {
		return fmt.Errorf("fbtype: TIME value out of range for Firebird (µs since midnight: %d)", usec)
	}

	return nil
}

func parseTimeString(s string) (usec int64, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("invalid TIME format: empty string")
	}

	fields := strings.Split(s, ":")
	if len(fields) < 2 || len(fields) > 3 {
		return 0, fmt.Errorf("invalid TIME format: %q (expected HH:MM[:SS[.ffffff]])", s)
	}

	hours, err := parseTimeNumber(fields[0], "hours", s)
	if err != nil {
		return 0, err
	}
	if hours < 0 || hours > 23 {
		return 0, fmt.Errorf("hours out of range (0-23): %d", hours)
	}

	minutes := int64(0)
	if len(fields) >= 2 {
		minutes, err = parseTimeNumber(fields[1], "minutes", s)
		if err != nil {
			return 0, err
		}
		if minutes < 0 || minutes > 59 {
			return 0, fmt.Errorf("minutes out of range (0-59): %d", minutes)
		}
	}

	seconds := int64(0)
	fractionUsec := int64(0)
	if len(fields) == 3 {
		secondsPart := fields[2]
		if secondsPart == "" {
			return 0, fmt.Errorf("invalid TIME format: %q (empty seconds)", s)
		}

		if strings.Count(secondsPart, ".") > 1 {
			return 0, fmt.Errorf("invalid TIME format: %q", s)
		}

		if secondsText, fraction, found := strings.Cut(secondsPart, "."); found {
			seconds, err = parseTimeNumber(secondsText, "seconds", s)
			if err != nil {
				return 0, err
			}
			if len(fraction) == 0 || len(fraction) > 6 {
				return 0, fmt.Errorf("invalid TIME fractional seconds: %q", s)
			}
			fractionUsec, err = parseTimeFraction(fraction, s)
			if err != nil {
				return 0, err
			}
		} else {
			seconds, err = parseTimeNumber(secondsPart, "seconds", s)
			if err != nil {
				return 0, err
			}
		}

		if seconds < 0 || seconds > 59 {
			return 0, fmt.Errorf("seconds out of range (0-59): %d", seconds)
		}
	} else if strings.Contains(fields[len(fields)-1], ".") {
		return 0, fmt.Errorf("invalid TIME format: %q (fractional seconds require seconds)", s)
	}

	usec = hours*microsecondsPerHour +
		minutes*microsecondsPerMinute +
		seconds*microsecondsPerSecond +
		fractionUsec

	if err := validateTimeMicroseconds(usec); err != nil {
		return 0, err
	}

	return usec, nil
}

func parseTimeNumber(s string, field string, input string) (int64, error) {
	if len(s) < 1 || len(s) > 2 {
		return 0, fmt.Errorf("invalid TIME %s in %q", field, input)
	}

	for i := range s {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("cannot parse %s from %q", field, input)
		}
	}

	return strconv.ParseInt(s, 10, 64)
}

func parseTimeFraction(s string, input string) (int64, error) {
	for i := range s {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("cannot parse microseconds from %q", input)
		}
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse microseconds from %q: %w", input, err)
	}

	for i := len(s); i < 6; i++ {
		n *= 10
	}

	return n, nil
}
