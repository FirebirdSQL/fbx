package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/internal/fbio"
)

// TimestamptzScanner scans a Firebird TIMESTAMP WITH TIME ZONE value.
type TimestamptzScanner interface {
	// ScanTimestamptz scans v into the receiver.
	ScanTimestamptz(v Timestamptz) error
}

// TimestamptzValuer returns a Firebird TIMESTAMP WITH TIME ZONE value.
type TimestamptzValuer interface {
	// TimestamptzValue returns the Firebird TIMESTAMP WITH TIME ZONE value to encode.
	TimestamptzValue() (Timestamptz, error)
}

// Timestamptz represents the Firebird TIMESTAMP WITH TIME ZONE type.
//
// Time contains the timestamp adjusted to the stored Firebird time zone. Valid is false when the value represents SQL NULL.
type Timestamptz struct {
	Time  time.Time // Time contains the timestamp adjusted to the stored Firebird time zone.
	Valid bool      // Valid is false when the value represents SQL NULL.
	tzRaw uint16
}

// ScanTimestamptz implements [TimestamptzScanner].
func (tstz *Timestamptz) ScanTimestamptz(v Timestamptz) error {
	*tstz = v
	return nil
}

// TimestamptzValue implements [TimestamptzValuer].
func (tstz Timestamptz) TimestamptzValue() (Timestamptz, error) {
	return tstz, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (tstz *Timestamptz) Scan(src any) error {
	if src == nil {
		*tstz = Timestamptz{}
		return nil
	}

	switch src := src.(type) {
	case time.Time:
		*tstz = Timestamptz{Time: src, Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T into Timestamptz", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (tstz Timestamptz) Value() (driver.Value, error) {
	if !tstz.Valid {
		return nil, nil
	}

	return tstz.Time, nil
}

// MarshalJSON encodes tstz as a JSON Firebird TIMESTAMP WITH TIME ZONE string or null.
func (tstz Timestamptz) MarshalJSON() ([]byte, error) {
	if !tstz.Valid {
		return []byte("null"), nil
	}

	s, err := formatTimestamptzString(tstz)
	if err != nil {
		return nil, err
	}

	return json.Marshal(s)
}

// UnmarshalJSON decodes a JSON timestamp-with-time-zone string or null into tstz.
func (tstz *Timestamptz) UnmarshalJSON(b []byte) error {
	var s *string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	if s == nil {
		*tstz = Timestamptz{}
		return nil
	}

	v, err := parseTimestamptzString(*s)
	if err != nil {
		return err
	}

	*tstz = v

	return nil
}

// TimestamptzCodec encodes and decodes Firebird TIMESTAMP WITH TIME ZONE values.
type TimestamptzCodec struct{}

// PlanEncode returns an encode plan for supported Firebird TIMESTAMP WITH TIME ZONE input values.
func (*TimestamptzCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case TimestamptzValuer:
		return encodePlanTimestamptzCodecBinary{}
	case time.Time:
		return encodePlanTimestamptzCodecBinaryTime{}
	case string:
		return encodePlanTimestamptzCodecBinaryString{}
	}

	return nil
}

type encodePlanTimestamptzCodecBinary struct{}

func (encodePlanTimestamptzCodecBinary) Encode(value any, buf []byte) (newBuf []byte, err error) {
	ts, err := value.(TimestamptzValuer).TimestamptzValue()
	if err != nil {
		return nil, err
	}

	if !ts.Valid {
		return nil, nil
	}

	if ts.tzRaw == 0 {
		tzRaw, err := tzRawFromTime(ts.Time)
		if err != nil {
			return nil, err
		}
		ts.tzRaw = tzRaw
	}

	return encodeTimestamptzBinary(buf, ts)
}

type encodePlanTimestamptzCodecBinaryTime struct{}

func (encodePlanTimestamptzCodecBinaryTime) Encode(value any, buf []byte) ([]byte, error) {
	t := value.(time.Time)

	tzRaw, err := tzRawFromTime(t)
	if err != nil {
		return nil, err
	}

	ts := Timestamptz{
		Time:  t,
		tzRaw: tzRaw,
		Valid: true,
	}

	return encodeTimestamptzBinary(buf, ts)
}

type encodePlanTimestamptzCodecBinaryString struct{}

func (encodePlanTimestamptzCodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	ts, err := parseTimestamptzString(value.(string))
	if err != nil {
		return nil, err
	}

	return encodeTimestamptzBinary(buf, ts)
}

// PlanScan returns a scan plan for supported Firebird TIMESTAMP WITH TIME ZONE target values.
func (c *TimestamptzCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case TimestamptzScanner:
		return &scanPlanBinaryTimestamptzToTimestamptzScanner{}
	case *time.Time:
		return &scanPlanBinaryTimestamptzToTimeScanner{}
	case *string:
		return &scanPlanBinaryTimestamptzToStringScanner{}
	}

	return nil
}

type scanPlanBinaryTimestamptzToTimestamptzScanner struct{}

func (plan *scanPlanBinaryTimestamptzToTimestamptzScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TimestamptzScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanTimestamptz(Timestamptz{})
	}

	v, err := decodeTimestamptzBinary(src)
	if err != nil {
		return err
	}

	return scanner.ScanTimestamptz(v)
}

type scanPlanBinaryTimestamptzToTimeScanner struct{}

func (plan *scanPlanBinaryTimestamptzToTimeScanner) Scan(src []byte, dst any) error {
	ptr, ok := dst.(*time.Time)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}

	tstz, err := decodeTimestamptzBinary(src)
	if err != nil {
		return err
	}
	if !tstz.Valid {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}

	t := tstz.Time

	*ptr = t
	return nil
}

type scanPlanBinaryTimestamptzToStringScanner struct{}

func (plan *scanPlanBinaryTimestamptzToStringScanner) Scan(src []byte, dst any) error {
	ptr, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	tstz, err := decodeTimestamptzBinary(src)
	if err != nil {
		return err
	}

	if !tstz.Valid {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	s, err := formatTimestamptzString(tstz)
	if err != nil {
		return err
	}

	*ptr = s
	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c *TimestamptzCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var tstz Timestamptz
	err := codecScan(c, m, typeMetadata, src, &tstz)
	if err != nil {
		return nil, err
	}

	return tstz.Time, nil
}

// DecodeValue decodes src into the default Go value for Firebird TIMESTAMP WITH TIME ZONE.
func (c *TimestamptzCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var tstz Timestamptz
	err := codecScan(c, m, typeMetadata, src, &tstz)
	if err != nil {
		return nil, err
	}

	return tstz.Time, nil
}

func encodeTimestamptzBinary(buf []byte, ts Timestamptz) ([]byte, error) {
	if !ts.Valid {
		return nil, nil
	}

	utc := ts.Time.In(time.UTC)

	tMidnightUTC := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)

	days := int32((tMidnightUTC.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)

	usec := timeOfDayMicroseconds(utc)
	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIMESTAMPTZ time-of-day out of range (µs since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)

	buf = fbio.AppendUint32BE(buf, uint32(days))
	buf = fbio.AppendUint32BE(buf, ticks)
	buf = fbio.AppendUint32BE(buf, uint32(ts.tzRaw))

	return buf, nil
}

func decodeTimestamptzBinary(src []byte) (Timestamptz, error) {
	if src == nil {
		return Timestamptz{}, nil
	}
	if len(src) != 12 {
		return Timestamptz{}, fmt.Errorf("invalid length for timestamptz: %v", len(src))
	}

	days := int32(binary.BigEndian.Uint32(src[0:4]))
	ticks := binary.BigEndian.Uint32(src[4:8])
	tzVal := binary.BigEndian.Uint32(src[8:12])

	tDate := firebirdDateEpoch.AddDate(0, 0, int(days))

	usec := int64(ticks) * 100
	if usec < 0 || usec >= microsecondsPerDay {
		return Timestamptz{}, fmt.Errorf("TIMESTAMPTZ time-of-day out of range in decode (µs since midnight: %d)", usec)
	}

	tzRaw := uint16(tzVal & 0xFFFF)
	loc := fbproto.ResolveLocationByTzRaw(tzRaw)

	t := timeFromMicroseconds(usec, tDate, time.UTC).In(loc)

	return Timestamptz{
		Time:  t,
		tzRaw: tzRaw,
		Valid: true,
	}, nil
}

func parseTimestamptzString(s string) (Timestamptz, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return timestamptzFromTime(t)
	}

	t, err = time.Parse(time.RFC3339, s)
	if err == nil {
		return timestamptzFromTime(t)
	}

	datePart, timePart, zonePart, err := splitTimestamptzString(s)
	if err != nil {
		return Timestamptz{}, fmt.Errorf("cannot parse %q as TIMESTAMPTZ: %w", s, err)
	}

	date, err := parseDateString(datePart)
	if err != nil {
		return Timestamptz{}, fmt.Errorf("cannot parse %q as TIMESTAMPTZ: %w", s, err)
	}

	usec, err := parseTimetzClock(timePart)
	if err != nil {
		return Timestamptz{}, fmt.Errorf("cannot parse %q as TIMESTAMPTZ: %w", s, err)
	}

	loc, tzRaw, err := parseTimetzZone(zonePart)
	if err != nil {
		return Timestamptz{}, fmt.Errorf("cannot parse %q as TIMESTAMPTZ: %w", s, err)
	}

	t = timeFromMicroseconds(usec, date, loc)
	return Timestamptz{Time: t, tzRaw: tzRaw, Valid: true}, nil
}

func splitTimestamptzString(s string) (datePart string, timePart string, zonePart string, err error) {
	fields := strings.Fields(s)
	switch len(fields) {
	case 2:
		if datePart, timePart, ok := strings.Cut(fields[0], "T"); ok {
			if datePart == "" || timePart == "" {
				return "", "", "", fmt.Errorf("invalid TIMESTAMPTZ format: %q", s)
			}
			return datePart, timePart, fields[1], nil
		}

		timePart, zonePart, err = splitCompactTimetzString(fields[1])
		if err != nil {
			return "", "", "", fmt.Errorf("invalid TIMESTAMPTZ format: %q (expected YYYY-MM-DD HH:MM[:SS][.ffffff] zone)", s)
		}
		return fields[0], timePart, zonePart, nil
	case 3:
		return fields[0], fields[1], fields[2], nil
	default:
		return "", "", "", fmt.Errorf("invalid TIMESTAMPTZ format: %q (expected RFC3339 or YYYY-MM-DD HH:MM[:SS][.ffffff] zone)", s)
	}
}

func timestamptzFromTime(t time.Time) (Timestamptz, error) {
	tzRaw, err := tzRawFromTime(t)
	if err != nil {
		return Timestamptz{}, err
	}

	loc := fbproto.ResolveLocationByTzRaw(tzRaw)
	return Timestamptz{Time: t.In(loc), tzRaw: tzRaw, Valid: true}, nil
}

func formatTimestamptzString(tstz Timestamptz) (string, error) {
	tzRaw := tstz.tzRaw
	if tzRaw == 0 {
		var err error
		tzRaw, err = tzRawFromTime(tstz.Time)
		if err != nil {
			return "", err
		}
	}

	loc := fbproto.ResolveLocationByTzRaw(tzRaw)
	t := tstz.Time.In(loc)
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%06d %s",
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond()/1000,
		loc.String(),
	), nil
}

func tzRawFromTime(t time.Time) (uint16, error) {
	if tzRaw, ok := tzRawFromLocation(t); ok {
		return tzRaw, nil
	}

	_, offsetSeconds := t.Zone()
	if offsetSeconds%60 != 0 {
		return 0, fmt.Errorf("UTC offset %d seconds is not representable by Firebird time zone", offsetSeconds)
	}

	offsetMinutes := offsetSeconds / 60

	if offsetMinutes < -1439 || offsetMinutes > 1439 {
		return 0, fmt.Errorf("UTC offset %d minutes out of range", offsetMinutes)
	}

	return uint16(offsetMinutes + 1439), nil
}

func tzRawFromLocation(t time.Time) (uint16, bool) {
	loc := t.Location()
	tzRaw, ok := fbproto.ResolveTzRawByLocationName(loc.String())
	if !ok {
		return 0, false
	}

	_, originalOffset := t.Zone()
	_, resolvedOffset := t.In(fbproto.ResolveLocationByTzRaw(tzRaw)).Zone()
	if originalOffset != resolvedOffset {
		return 0, false
	}

	return tzRaw, true
}
