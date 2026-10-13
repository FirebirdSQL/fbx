package fbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx/internal/fbio"
)

const (
	firebirdTimestampFormat = "2006-01-02 15:04:05.999999"
	jsonTimestampFormat     = "2006-01-02T15:04:05.999999"
)

// TimestampScanner scans a Firebird TIMESTAMP value.
type TimestampScanner interface {
	// ScanTimestamp scans v into the receiver.
	ScanTimestamp(v Timestamp) error
}

// TimestampValuer returns a Firebird TIMESTAMP value.
type TimestampValuer interface {
	// TimestampValue returns the Firebird TIMESTAMP value to encode.
	TimestampValue() (Timestamp, error)
}

// Timestamp represents the Firebird TIMESTAMP type.
//
// Firebird TIMESTAMP stores a date and time of day without time zone. Time is
// interpreted as wall-clock date/time when encoding; its location is not
// converted to UTC. Valid is false when the value represents SQL NULL.
type Timestamp struct {
	Time  time.Time
	Valid bool
}

// ScanTimestamp implements [TimestampScanner].
func (ts *Timestamp) ScanTimestamp(v Timestamp) error {
	*ts = v
	return nil
}

// TimestampValue implements [TimestampValuer].
func (ts Timestamp) TimestampValue() (Timestamp, error) {
	return ts, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (ts *Timestamp) Scan(src any) error {
	if src == nil {
		*ts = Timestamp{}
		return nil
	}

	switch src := src.(type) {
	case string:
		t, err := parseTimestampString(src, nil)
		if err != nil {
			*ts = Timestamp{}
			return err
		}
		*ts = Timestamp{Time: t, Valid: true}
		return nil
	case time.Time:
		*ts = Timestamp{Time: src, Valid: true}
		return nil
	}

	*ts = Timestamp{}
	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (ts Timestamp) Value() (driver.Value, error) {
	if !ts.Valid {
		return nil, nil
	}

	return ts.Time, nil
}

// MarshalJSON encodes ts as a JSON timestamp string or null.
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	if !ts.Valid {
		return []byte("null"), nil
	}

	s := ts.Time.Format(jsonTimestampFormat)
	return json.Marshal(s)
}

// UnmarshalJSON decodes a JSON timestamp string or null into ts.
func (ts *Timestamp) UnmarshalJSON(b []byte) error {
	var s *string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	if s == nil {
		*ts = Timestamp{}
		return nil
	}

	tss := *s

	tim, err := parseTimestampString(tss, nil)
	if err == nil {
		*ts = Timestamp{Time: tim, Valid: true}
		return nil
	}

	*ts = Timestamp{}
	return fmt.Errorf(
		"cannot unmarshal %s to timestamp: %w",
		*s, err,
	)
}

// TimestampCodec encodes and decodes Firebird TIMESTAMP values.
type TimestampCodec struct {
	// ScanLocation is the location assigned to decoded TIMESTAMP values. Because
	// Firebird TIMESTAMP has no time zone, a non-nil ScanLocation preserves the
	// stored wall-clock fields but changes the instant represented by time.Time.
	ScanLocation *time.Location
}

// PlanEncode returns an encode plan for supported Firebird TIMESTAMP input values.
func (*TimestampCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case TimestampValuer:
		return encodePlanTimestampCodecBinary{}
	case time.Time:
		return encodePlanTimestampCodecBinaryTime{}
	case string:
		return encodePlanTimestampCodecBinaryString{}
	}
	return nil
}

type encodePlanTimestampCodecBinary struct{}

func (encodePlanTimestampCodecBinary) Encode(value any, buf []byte) ([]byte, error) {
	ts, err := value.(TimestampValuer).TimestampValue()
	if err != nil {
		return nil, err
	}
	if !ts.Valid {
		return nil, nil
	}

	return encodeTimestampBinary(ts.Time, buf)
}

type encodePlanTimestampCodecBinaryTime struct{}

func (encodePlanTimestampCodecBinaryTime) Encode(value any, buf []byte) ([]byte, error) {
	t := value.(time.Time)

	return encodeTimestampBinary(t, buf)
}

type encodePlanTimestampCodecBinaryString struct{}

func (encodePlanTimestampCodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	t, err := parseTimestampString(value.(string), nil)
	if err != nil {
		return nil, err
	}

	return encodeTimestampBinary(t, buf)
}

// PlanScan returns a scan plan for supported Firebird TIMESTAMP target values.
func (c *TimestampCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case TimestampScanner:
		return &scanPlanBinaryTimestampToTimestampScanner{location: c.ScanLocation}
	case *time.Time:
		return &scanPlanBinaryTimestampToTime{location: c.ScanLocation}
	case *string:
		return &scanPlanBinaryTimestampToString{location: c.ScanLocation}
	}

	return nil
}

type scanPlanBinaryTimestampToTimestampScanner struct{ location *time.Location }

func (plan *scanPlanBinaryTimestampToTimestampScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TimestampScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanTimestamp(Timestamp{})
	}
	t, err := decodeTimestampBinary(src, plan.location)
	if err != nil {
		return err
	}

	return scanner.ScanTimestamp(Timestamp{Time: t, Valid: true})
}

type scanPlanBinaryTimestampToTime struct{ location *time.Location }

func (plan *scanPlanBinaryTimestampToTime) Scan(src []byte, dst any) error {
	target, ok := dst.(*time.Time)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}
	t, err := decodeTimestampBinary(src, plan.location)
	if err != nil {
		return err
	}

	*target = t
	return nil
}

type scanPlanBinaryTimestampToString struct{ location *time.Location }

func (plan *scanPlanBinaryTimestampToString) Scan(src []byte, dst any) error {
	target, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	t, err := decodeTimestampBinary(src, plan.location)
	if err != nil {
		return err
	}

	*target = formatTimestampString(t)

	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c *TimestampCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var ts Timestamp
	err := codecScan(c, m, typeMetadata, src, &ts)
	if err != nil {
		return nil, err
	}

	return ts.Time, nil
}

// DecodeValue decodes src into the default Go value for Firebird TIMESTAMP.
func (c *TimestampCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var ts Timestamp
	err := codecScan(c, m, typeMetadata, src, &ts)
	if err != nil {
		return nil, err
	}

	return ts.Time, nil
}

func encodeTimestampBinary(t time.Time, buf []byte) ([]byte, error) {
	days, err := firebirdDateDayOffset(t)
	if err != nil {
		return nil, err
	}

	usec := timeOfDayMicroseconds(t)
	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIMESTAMP time-of-day out of range (µs since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)

	buf = fbio.AppendInt32BE(buf, days)
	buf = fbio.AppendInt32BE(buf, int32(ticks))

	return buf, nil
}

func decodeTimestampBinary(src []byte, location *time.Location) (time.Time, error) {
	r := fbio.NewReader(src)
	dayOffset := r.Int32BE()
	ticks := r.Uint32BE()
	if err := r.Finish(); err != nil {
		return time.Time{}, fmt.Errorf("TIMESTAMP: %w", err)
	}

	tDate, err := firebirdDateFromDayOffset(dayOffset)
	if err != nil {
		return time.Time{}, err
	}

	usec := int64(ticks) * 100
	if usec < 0 || usec >= microsecondsPerDay {
		return time.Time{}, fmt.Errorf("TIMESTAMP time-of-day out of range in decode (µs since midnight: %d)", usec)
	}

	loc := time.UTC
	if location != nil {
		loc = location
	}

	return timeFromMicroseconds(usec, tDate, loc), nil
}

func parseTimestampString(s string, location *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("invalid TIMESTAMP format: empty string")
	}

	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	datePart, timePart, err := splitTimestampString(s)
	if err != nil {
		return time.Time{}, err
	}

	date, err := parseDateString(datePart)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot parse TIMESTAMP date from %q: %w", s, err)
	}

	var usec int64
	if timePart != "" {
		usec, err = parseTimeString(timePart)
		if err != nil {
			return time.Time{}, fmt.Errorf("cannot parse TIMESTAMP time from %q: %w", s, err)
		}
	}

	loc := time.UTC
	if location != nil {
		loc = location
	}

	return timeFromMicroseconds(usec, date, loc), nil
}

func splitTimestampString(s string) (datePart string, timePart string, err error) {
	if strings.Count(s, "T") == 1 && !strings.ContainsAny(s, " \t\r\n") {
		parts := strings.SplitN(s, "T", 2)
		if parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
	}

	fields := strings.Fields(s)
	switch len(fields) {
	case 1:
		return fields[0], "", nil
	case 2:
		return fields[0], fields[1], nil
	default:
		return "", "", fmt.Errorf("invalid TIMESTAMP format: %q (expected <date> [<time>], e.g. YYYY-MM-DD [HH:MM[:SS[.ffffff]]])", s)
	}
}

func formatTimestampString(t time.Time) string {
	return t.Format(firebirdTimestampFormat)
}
