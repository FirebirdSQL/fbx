package fbtype

import (
	"database/sql/driver"
	"fmt"
	"strconv"
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

type TimeScanner interface {
	ScanTime(v Time) error
}

type TimeValuer interface {
	TimeValue() (Time, error)
}

// Time represents the Firebird time type. The Firebird time is a time of day without time zone.
//
// Time is represented as the number of 100 microseconds since midnight in the same way that Firebird does. Other time and
// date types in fbtype can use time.Time as the underlying representation.
//
// The time with time zone type is not supported. Use of time with time zone is discouraged by the Firebird documentation.
type Time struct {
	Microseconds int64 // Number of 100 microseconds since midnight
	Valid        bool
}

func (t *Time) ScanTime(v Time) error {
	*t = v
	return nil
}

func (t Time) TimeValue() (Time, error) {
	return t, nil
}

// Scan implements the database/sql Scanner interface.
func (t *Time) Scan(src any) error {
	if src == nil {
		*t = Time{}
		return nil
	}

	switch src := src.(type) {
	case string:
		usec, err := parseTimeString(src)
		if err != nil {
			t.Microseconds = 0
			t.Valid = false
			return err
		}
		t.Microseconds = usec
		t.Valid = true
		return nil
	case time.Time:
		t.Microseconds = int64(src.Hour())*microsecondsPerHour +
			int64(src.Minute())*microsecondsPerMinute +
			int64(src.Second())*microsecondsPerSecond +
			int64(src.Nanosecond())/1000
		t.Valid = true
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (t Time) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}

	typeMetadata := NewTypeMetadata(0, 0, 0, 0)
	buf, err := TimeCodec{}.PlanEncode(nil, typeMetadata, t).Encode(t, nil)
	if err != nil {
		return nil, err
	}
	return string(buf), err
}

type TimeCodec struct{}

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
	str := value.(string)

	if len(str) < 8 || str[2] != ':' || str[5] != ':' {
		return nil, fmt.Errorf("invalid TIME format: %q (expected HH:MM:SS or HH:MM:SS.ffffff)", str)
	}

	hours, err := strconv.ParseInt(str[0:2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse hours from %q: %w", str, err)
	}
	if hours < 0 || hours > 23 {
		return nil, fmt.Errorf("hours out of range (0-23): %d", hours)
	}
	usec := hours * microsecondsPerHour

	minutes, err := strconv.ParseInt(str[3:5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse minutes from %q: %w", str, err)
	}
	if minutes < 0 || minutes > 59 {
		return nil, fmt.Errorf("minutes out of range (0-59): %d", minutes)
	}
	usec += minutes * microsecondsPerMinute

	seconds, err := strconv.ParseInt(str[6:8], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse seconds from %q: %w", str, err)
	}
	if seconds < 0 || seconds > 59 {
		return nil, fmt.Errorf("seconds out of range (0-59): %d", seconds)
	}
	usec += seconds * microsecondsPerSecond

	if len(str) > 9 {
		if str[8] != '.' || len(str) > 15 {
			return nil, fmt.Errorf("invalid TIME format: %q (expected HH:MM:SS.ffffff)", str)
		}

		fraction := str[9:]
		n, err := strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse microseconds from %q: %w", str, err)
		}

		for i := len(fraction); i < 6; i++ {
			n *= 10
		}

		usec += n
	}

	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIME value out of range for Firebird (µs since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)
	buf = fbio.AppendInt32BE(buf, int32(ticks))

	return buf, nil
}

type encodePlanTimeCodecText struct{}

func (encodePlanTimeCodecText) Encode(value any, buf []byte) (newBuf []byte, err error) {
	t, err := value.(TimeValuer).TimeValue()
	if err != nil {
		return nil, err
	}

	if !t.Valid {
		return nil, nil
	}

	usec := t.Microseconds
	hours := usec / microsecondsPerHour
	usec -= hours * microsecondsPerHour
	minutes := usec / microsecondsPerMinute
	usec -= minutes * microsecondsPerMinute
	seconds := usec / microsecondsPerSecond
	usec -= seconds * microsecondsPerSecond

	s := fmt.Sprintf("%02d:%02d:%02d.%06d", hours, minutes, seconds, usec)

	return append(buf, s...), nil
}

type encodePlanTimeCodecBinaryTime struct{}

func (encodePlanTimeCodecBinaryTime) Encode(value any, buf []byte) ([]byte, error) {
	t := value.(time.Time)

	usec := int64(t.Hour())*microsecondsPerHour +
		int64(t.Minute())*microsecondsPerMinute +
		int64(t.Second())*microsecondsPerSecond +
		int64(t.Nanosecond())/1000

	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIME value out of range (µs since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)

	buf = fbio.AppendInt32BE(buf, int32(ticks))

	return buf, nil
}

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
	target := dst.(*time.Time)

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}

	ticks := fbio.ReadInt32BE(src)
	usec := int64(ticks) * 100

	if usec < 0 || usec >= microsecondsPerDay {
		return fmt.Errorf("TIME value out of range: %d microseconds", usec)
	}

	hours := usec / microsecondsPerHour
	usec %= microsecondsPerHour

	minutes := usec / microsecondsPerMinute
	usec %= microsecondsPerMinute

	seconds := usec / microsecondsPerSecond
	usec %= microsecondsPerSecond

	nanos := usec * 1000

	*target = time.Date(2000, 1, 1, int(hours), int(minutes), int(seconds), int(nanos), time.UTC)

	return nil
}

type scanPlanBinaryTimeToTimeScanner struct{}

func (scanPlanBinaryTimeToTimeScanner) Scan(src []byte, dst any) error {
	scanner := dst.(TimeScanner)

	if src == nil {
		return scanner.ScanTime(Time{})
	}

	if len(src) != 4 {
		return fmt.Errorf("fbtype: invalid length for Firebird TIME: %d", len(src))
	}

	ticks := uint32(fbio.ReadInt32BE(src))
	usec := int64(ticks) * 100

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

// Scan TIME to *string
type scanPlanBinaryTimeToString struct{}

func (scanPlanBinaryTimeToString) Scan(src []byte, dst any) error {
	target := dst.(*string)

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

func (c TimeCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	_, formatted, err := decodeTimeBinary(src)
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

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
	if len(src) != 4 {
		return 0, "", fmt.Errorf("fbtype: invalid length for Firebird TIME: %d", len(src))
	}

	ticks := uint32(fbio.ReadInt32BE(src))
	usec := int64(ticks) * 100

	hours := usec / microsecondsPerHour
	usec -= hours * microsecondsPerHour
	minutes := usec / microsecondsPerMinute
	usec -= minutes * microsecondsPerMinute
	seconds := usec / microsecondsPerSecond
	usec -= seconds * microsecondsPerSecond

	totalUsec := hours*microsecondsPerHour + minutes*microsecondsPerMinute + seconds*microsecondsPerSecond + usec

	formatted := fmt.Sprintf("%02d:%02d:%02d.%06d", hours, minutes, seconds, usec)

	return totalUsec, formatted, nil
}

func parseTimeString(s string) (usec int64, err error) {
	if len(s) < 8 || s[2] != ':' || s[5] != ':' {
		return 0, fmt.Errorf("cannot decode %v into Time", s)
	}

	hours, err := strconv.ParseInt(s[0:2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot decode %v into Time", s)
	}
	usec = hours * microsecondsPerHour

	minutes, err := strconv.ParseInt(s[3:5], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot decode %v into Time", s)
	}
	usec += minutes * microsecondsPerMinute

	seconds, err := strconv.ParseInt(s[6:8], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot decode %v into Time", s)
	}
	usec += seconds * microsecondsPerSecond

	if len(s) > 9 {
		if s[8] != '.' || len(s) > 15 {
			return 0, fmt.Errorf("cannot decode %v into Time", s)
		}

		fraction := s[9:]
		n, err := strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot decode %v into Time", s)
		}

		for i := len(fraction); i < 6; i++ {
			n *= 10
		}

		usec += n
	}

	return usec, nil
}
