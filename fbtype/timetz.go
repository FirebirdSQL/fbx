package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/internal/fbio"
)

// TimetzScanner scans a Firebird TIME WITH TIME ZONE value.
type TimetzScanner interface {
	// ScanTimetz scans v into the receiver.
	ScanTimetz(v Timetz) error
}

// TimetzValuer returns a Firebird TIME WITH TIME ZONE value.
type TimetzValuer interface {
	// TimetzValue returns the Firebird TIME WITH TIME ZONE value to encode.
	TimetzValue() (Timetz, error)
}

// Timetz represents the Firebird TIME WITH TIME ZONE type.
//
// Time contains the decoded time of day and time zone on Firebird's TIME WITH
// TIME ZONE base date, 2020-01-01. The date is fixed only so Go can represent
// the value as time.Time; Firebird TIME WITH TIME ZONE itself does not store a
// date. Valid is false when the value represents SQL NULL.
type Timetz struct {
	Time  time.Time
	Valid bool
	tzRaw uint16
}

// ScanTimetz implements [TimetzScanner].
func (ttz *Timetz) ScanTimetz(v Timetz) error {
	*ttz = v
	return nil
}

// TimetzValue implements [TimetzValuer].
func (ttz Timetz) TimetzValue() (Timetz, error) {
	return ttz, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (ttz *Timetz) Scan(src any) error {
	if src == nil {
		*ttz = Timetz{}
		return nil
	}

	switch src := src.(type) {
	case string:
		v, err := parseTimetzString(src)
		if err != nil {
			*ttz = Timetz{}
			return err
		}
		*ttz = v
		return nil
	case time.Time:
		*ttz = Timetz{Time: src, Valid: true}
		return nil
	}

	*ttz = Timetz{}
	return fmt.Errorf("cannot scan %T into Timetz", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (ttz Timetz) Value() (driver.Value, error) {
	if !ttz.Valid {
		return nil, nil
	}

	return ttz.Time, nil
}

// MarshalJSON encodes ttz as a JSON Firebird TIME WITH TIME ZONE string or null.
func (ttz Timetz) MarshalJSON() ([]byte, error) {
	if !ttz.Valid {
		return []byte("null"), nil
	}

	s, err := formatTimetzString(ttz)
	if err != nil {
		return nil, err
	}

	return json.Marshal(s)
}

// UnmarshalJSON decodes a JSON time-with-time-zone string or null into ttz.
func (ttz *Timetz) UnmarshalJSON(b []byte) error {
	var s *string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	if s == nil {
		*ttz = Timetz{}
		return nil
	}

	v, err := parseTimetzString(*s)
	if err != nil {
		return err
	}

	*ttz = v
	return nil
}

// TimetzCodec encodes and decodes Firebird TIME WITH TIME ZONE values.
type TimetzCodec struct{}

// PlanEncode returns an encode plan for supported Firebird TIME WITH TIME ZONE input values.
func (*TimetzCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case TimetzValuer:
		return encodePlanTimetzCodecBinary{}
	case time.Time:
		return encodePlanTimetzCodecBinaryTime{}
	case string:
		return encodePlanTimetzCodecBinaryString{}
	}

	return nil
}

type encodePlanTimetzCodecBinary struct{}

func (encodePlanTimetzCodecBinary) Encode(value any, buf []byte) ([]byte, error) {
	ttz, err := value.(TimetzValuer).TimetzValue()
	if err != nil {
		return nil, err
	}

	if !ttz.Valid {
		return nil, nil
	}

	if ttz.tzRaw == 0 {
		tzRaw, err := tzRawFromTime(ttz.Time)
		if err != nil {
			return nil, err
		}
		ttz.tzRaw = tzRaw
	}

	return encodeTimetzBinary(buf, ttz)
}

type encodePlanTimetzCodecBinaryTime struct{}

func (encodePlanTimetzCodecBinaryTime) Encode(value any, buf []byte) ([]byte, error) {
	t := value.(time.Time)

	tzRaw, err := tzRawFromTime(t)
	if err != nil {
		return nil, err
	}

	return encodeTimetzBinary(buf, Timetz{Time: t, tzRaw: tzRaw, Valid: true})
}

type encodePlanTimetzCodecBinaryString struct{}

func (encodePlanTimetzCodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	ttz, err := parseTimetzString(value.(string))
	if err != nil {
		return nil, err
	}

	return encodeTimetzBinary(buf, ttz)
}

// PlanScan returns a scan plan for supported Firebird TIME WITH TIME ZONE target values.
func (c *TimetzCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case TimetzScanner:
		return &scanPlanBinaryTimetzToTimetzScanner{}
	case *time.Time:
		return &scanPlanBinaryTimetzToTimeScanner{}
	case *string:
		return &scanPlanBinaryTimetzToStringScanner{}
	case TextScanner:
		return &scanPlanBinaryTimetzToTextScanner{}
	}

	return nil
}

type scanPlanBinaryTimetzToTimetzScanner struct{}

func (plan *scanPlanBinaryTimetzToTimetzScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TimetzScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanTimetz(Timetz{})
	}

	v, err := decodeTimetzBinary(src)
	if err != nil {
		return err
	}

	return scanner.ScanTimetz(v)
}

type scanPlanBinaryTimetzToTimeScanner struct{}

func (plan *scanPlanBinaryTimetzToTimeScanner) Scan(src []byte, dst any) error {
	ptr, ok := dst.(*time.Time)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}

	ttz, err := decodeTimetzBinary(src)
	if err != nil {
		return err
	}
	if !ttz.Valid {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}

	*ptr = ttz.Time
	return nil
}

type scanPlanBinaryTimetzToStringScanner struct{}

func (plan *scanPlanBinaryTimetzToStringScanner) Scan(src []byte, dst any) error {
	ptr, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	ttz, err := decodeTimetzBinary(src)
	if err != nil {
		return err
	}
	if !ttz.Valid {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	s, err := formatTimetzString(ttz)
	if err != nil {
		return err
	}

	*ptr = s
	return nil
}

type scanPlanBinaryTimetzToTextScanner struct{}

func (plan *scanPlanBinaryTimetzToTextScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanText(Text{})
	}

	ttz, err := decodeTimetzBinary(src)
	if err != nil {
		return err
	}

	s, err := formatTimetzString(ttz)
	if err != nil {
		return err
	}

	return scanner.ScanText(Text{String: s, Valid: ttz.Valid})
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c *TimetzCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var ttz Timetz
	err := codecScan(c, m, typeMetadata, src, &ttz)
	if err != nil {
		return nil, err
	}

	return ttz.Time, nil
}

// DecodeValue decodes src into the default Go value for Firebird TIME WITH TIME ZONE.
func (c *TimetzCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var ttz Timetz
	err := codecScan(c, m, typeMetadata, src, &ttz)
	if err != nil {
		return nil, err
	}

	return ttz.Time, nil
}

func encodeTimetzBinary(buf []byte, ttz Timetz) ([]byte, error) {
	if !ttz.Valid {
		return nil, nil
	}

	loc := fbproto.ResolveLocationByTzRaw(ttz.tzRaw)
	local := timeWithDate(ttz.Time.In(loc), timetzBaseDate)
	utc := local.In(time.UTC)
	usec := timeOfDayMicroseconds(utc)
	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIMETZ time-of-day out of range (microseconds since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)

	buf = fbio.AppendUint32BE(buf, ticks)
	buf = fbio.AppendUint32BE(buf, uint32(ttz.tzRaw))

	return buf, nil
}

func decodeTimetzBinary(src []byte) (Timetz, error) {
	if src == nil {
		return Timetz{}, nil
	}
	if len(src) != 8 {
		return Timetz{}, fmt.Errorf("invalid length for timetz: %v", len(src))
	}

	ticks := binary.BigEndian.Uint32(src[0:4])
	tzVal := binary.BigEndian.Uint32(src[4:8])

	usec := int64(ticks) * 100
	if usec < 0 || usec >= microsecondsPerDay {
		return Timetz{}, fmt.Errorf("TIMETZ time-of-day out of range in decode (microseconds since midnight: %d)", usec)
	}

	tzRaw := uint16(tzVal & 0xFFFF)
	loc := fbproto.ResolveLocationByTzRaw(tzRaw)

	utcTime := timeFromMicroseconds(usec, timetzBaseDate, time.UTC)
	local := timeWithDate(utcTime.In(loc), timetzBaseDate)

	return Timetz{
		Time:  local,
		tzRaw: tzRaw,
		Valid: true,
	}, nil
}

func parseTimetzString(s string) (Timetz, error) {
	timePart, zonePart, err := splitTimetzString(s)
	if err != nil {
		return Timetz{}, err
	}

	usec, err := parseTimetzClock(timePart)
	if err != nil {
		return Timetz{}, err
	}

	loc, tzRaw, err := parseTimetzZone(zonePart)
	if err != nil {
		return Timetz{}, err
	}

	t := timeFromMicroseconds(usec, timetzBaseDate, loc)

	return Timetz{Time: t, tzRaw: tzRaw, Valid: true}, nil
}

func splitTimetzString(s string) (timePart string, zonePart string, err error) {
	fields := strings.Fields(s)
	switch len(fields) {
	case 1:
		return splitCompactTimetzString(fields[0])
	case 2:
		return fields[0], fields[1], nil
	default:
		return "", "", fmt.Errorf("invalid TIMETZ format: %q (expected HH:MM[:SS][.ffffff] zone)", s)
	}
}

func splitCompactTimetzString(s string) (timePart string, zonePart string, err error) {
	if s == "" {
		return "", "", fmt.Errorf("invalid TIMETZ format: %q", s)
	}

	if strings.HasSuffix(s, "Z") && len(s) > 1 {
		return s[:len(s)-1], "Z", nil
	}

	for i := len(s) - 1; i >= 5; i-- {
		if s[i] == '+' || s[i] == '-' {
			return s[:i], s[i:], nil
		}
	}

	return "", "", fmt.Errorf("invalid TIMETZ format: %q (missing time zone)", s)
}

func parseTimetzClock(s string) (int64, error) {
	if len(s) < 5 || s[2] != ':' {
		return 0, fmt.Errorf("invalid TIMETZ time: %q", s)
	}

	hours, err := strconv.ParseInt(s[0:2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse hours from %q: %w", s, err)
	}
	if hours < 0 || hours > 23 {
		return 0, fmt.Errorf("hours out of range (0-23): %d", hours)
	}

	minutes, err := strconv.ParseInt(s[3:5], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse minutes from %q: %w", s, err)
	}
	if minutes < 0 || minutes > 59 {
		return 0, fmt.Errorf("minutes out of range (0-59): %d", minutes)
	}

	usec := hours*microsecondsPerHour + minutes*microsecondsPerMinute
	if len(s) == 5 {
		return usec, nil
	}

	if len(s) < 8 || s[5] != ':' {
		return 0, fmt.Errorf("invalid TIMETZ time: %q", s)
	}

	seconds, err := strconv.ParseInt(s[6:8], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse seconds from %q: %w", s, err)
	}
	if seconds < 0 || seconds > 59 {
		return 0, fmt.Errorf("seconds out of range (0-59): %d", seconds)
	}
	usec += seconds * microsecondsPerSecond

	if len(s) == 8 {
		return usec, nil
	}

	if s[8] != '.' {
		return 0, fmt.Errorf("invalid TIMETZ time: %q", s)
	}

	fraction := s[9:]
	if len(fraction) == 0 || len(fraction) > 6 {
		return 0, fmt.Errorf("invalid TIMETZ fractional seconds: %q", s)
	}

	n, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse microseconds from %q: %w", s, err)
	}

	for i := len(fraction); i < 6; i++ {
		n *= 10
	}

	usec += n
	return usec, nil
}

func parseTimetzZone(s string) (*time.Location, uint16, error) {
	if s == "" {
		return nil, 0, fmt.Errorf("missing TIMETZ time zone")
	}

	if s == "Z" {
		tzRaw, ok := fbproto.ResolveTzRawByLocationName("UTC")
		if !ok {
			return nil, 0, fmt.Errorf("timetz: UTC timezone ID is not registered")
		}
		return time.UTC, tzRaw, nil
	}

	if offsetMinutes, ok, err := parseTimetzOffset(s); ok || err != nil {
		if err != nil {
			return nil, 0, err
		}
		return time.FixedZone(formatTimetzOffsetName(offsetMinutes), offsetMinutes*60), uint16(offsetMinutes + 1439), nil
	}

	tzRaw, ok := fbproto.ResolveTzRawByLocationName(s)
	if !ok {
		return nil, 0, fmt.Errorf("timetz: unknown Firebird time zone %q", s)
	}

	loc, err := time.LoadLocation(s)
	if err != nil {
		return nil, 0, fmt.Errorf("timetz: cannot load time zone %q: %w", s, err)
	}

	return loc, tzRaw, nil
}

func parseTimetzOffset(s string) (offsetMinutes int, ok bool, err error) {
	if len(s) < 2 || (s[0] != '+' && s[0] != '-') {
		return 0, false, nil
	}

	sign := 1
	if s[0] == '-' {
		sign = -1
	}

	rest := s[1:]
	var hoursText, minutesText string
	if colon := strings.IndexByte(rest, ':'); colon >= 0 {
		hoursText = rest[:colon]
		minutesText = rest[colon+1:]
	} else {
		hoursText = rest
		minutesText = "0"
	}

	if hoursText == "" || minutesText == "" {
		return 0, true, fmt.Errorf("invalid TIMETZ time zone offset: %q", s)
	}

	hours, err := strconv.ParseInt(hoursText, 10, 64)
	if err != nil {
		return 0, true, fmt.Errorf("cannot parse time zone hours from %q: %w", s, err)
	}

	minutes, err := strconv.ParseInt(minutesText, 10, 64)
	if err != nil {
		return 0, true, fmt.Errorf("cannot parse time zone minutes from %q: %w", s, err)
	}

	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return 0, true, fmt.Errorf("TIMETZ time zone offset out of range: %q", s)
	}

	total := int(hours*60 + minutes)
	if total > 1439 {
		return 0, true, fmt.Errorf("TIMETZ time zone offset out of range: %q", s)
	}

	return sign * total, true, nil
}

func formatTimetzString(ttz Timetz) (string, error) {
	tzRaw := ttz.tzRaw
	if tzRaw == 0 {
		var err error
		tzRaw, err = tzRawFromTime(ttz.Time)
		if err != nil {
			return "", err
		}
	}

	loc := fbproto.ResolveLocationByTzRaw(tzRaw)
	t := ttz.Time.In(loc)
	return fmt.Sprintf("%02d:%02d:%02d.%06d %s",
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond()/1000,
		loc.String(),
	), nil
}

func formatTimetzOffsetName(offsetMinutes int) string {
	sign := "+"
	if offsetMinutes < 0 {
		sign = "-"
		offsetMinutes = -offsetMinutes
	}

	return fmt.Sprintf("%s%02d:%02d", sign, offsetMinutes/60, offsetMinutes%60)
}
