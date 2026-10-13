package fbtype

import (
	"database/sql/driver"
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
// Time contains the timestamp adjusted to the stored Firebird time zone. Valid
// is false when the value represents SQL NULL.
type Timestamptz struct {
	Time  time.Time
	Valid bool
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

// MarshalJSON encodes tstz as a JSON timestamp with its UTC offset and optional zone name, or null.
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

// UnmarshalJSON decodes a supported timestamp-with-time-zone string or null into tstz.
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

	days, err := firebirdDateDayOffset(utc)
	if err != nil {
		return nil, err
	}

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
	r := fbio.NewReader(src)
	days := r.Int32BE()
	ticks := r.Uint32BE()
	tzVal := r.Uint32BE()
	if err := r.Finish(); err != nil {
		return Timestamptz{}, fmt.Errorf("timestamptz: %w", err)
	}

	tDate, err := firebirdDateFromDayOffset(days)
	if err != nil {
		return Timestamptz{}, err
	}

	usec := int64(ticks) * 100
	if usec < 0 || usec >= microsecondsPerDay {
		return Timestamptz{}, fmt.Errorf("TIMESTAMPTZ time-of-day out of range in decode (µs since midnight: %d)", usec)
	}

	tzRaw := uint16(tzVal & 0xFFFF)
	loc, err := fbproto.ResolveLocationByTzRaw(tzRaw)
	if err != nil {
		return Timestamptz{}, fmt.Errorf("cannot decode TIMESTAMPTZ: %w", err)
	}

	t := timeFromMicroseconds(usec, tDate, time.UTC).In(loc)

	return Timestamptz{
		Time:  t,
		tzRaw: tzRaw,
		Valid: true,
	}, nil
}

func parseTimestamptzString(s string) (Timestamptz, error) {
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, "[]") {
		return parseTimestamptzNamedZone(s)
	}

	t, err := parseTimestamptzOffset(s)
	if err == nil {
		if !strings.HasSuffix(s, "Z") {
			// Keep numeric offsets independent of time.Local, including +00:00.
			_, offset := t.Zone()
			t = t.In(time.FixedZone(formatTimetzOffsetName(offset/60), offset))
		}
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

	wall := timeFromMicroseconds(usec, date, time.UTC)
	t, err = resolveTimestamptzLocalTime(wall, loc)
	if err != nil {
		return Timestamptz{}, err
	}
	return Timestamptz{Time: t, tzRaw: tzRaw, Valid: true}, nil
}

// parseTimestamptzOffset parses an ISO timestamp with an explicit offset.
func parseTimestamptzOffset(s string) (time.Time, error) {
	zoneStart := strings.LastIndexAny(s, "Z+-")
	clockStart := strings.IndexByte(s, 'T')
	if clockStart < 0 || zoneStart <= clockStart {
		return time.Time{}, fmt.Errorf("TIMESTAMPTZ requires an ISO timestamp with a UTC offset: %q", s)
	}
	zone := s[zoneStart:]
	if zone != "Z" {
		if (len(zone) != 6 && len(zone) != 9) || zone[3] != ':' ||
			zone[1:3] > "23" || zone[4:6] > "59" ||
			(len(zone) == 9 && (zone[6] != ':' || zone[7:9] > "59")) {
			return time.Time{}, fmt.Errorf("invalid TIMESTAMPTZ UTC offset: %q", zone)
		}
	}
	var err error
	for _, layout := range [...]string{
		time.RFC3339Nano,
		"2006-01-02T15:04Z07:00",
		"2006-01-02T15:04:05Z07:00:00",
		"2006-01-02T15:04Z07:00:00",
	} {
		var t time.Time
		t, err = time.ParseInLocation(layout, s, time.UTC)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid TIMESTAMPTZ with UTC offset %q: %w", s, err)
}

func parseTimestamptzNamedZone(s string) (Timestamptz, error) {
	timestamp, suffix, ok := strings.Cut(s, "[")
	if !ok || !strings.HasSuffix(suffix, "]") || len(suffix) < 2 ||
		strings.ContainsAny(timestamp, "]") || strings.ContainsAny(suffix[:len(suffix)-1], "[]") {
		return Timestamptz{}, fmt.Errorf("invalid TIMESTAMPTZ zone suffix: %q", s)
	}
	zoneName := suffix[:len(suffix)-1]
	t, err := parseTimestamptzOffset(timestamp)
	if err != nil {
		return Timestamptz{}, err
	}
	loc, tzRaw, err := parseTimetzZone(zoneName)
	if err != nil {
		return Timestamptz{}, fmt.Errorf("cannot parse %q as TIMESTAMPTZ: %w", s, err)
	}
	localized := t.In(loc)
	_, offset := t.Zone()
	_, zoneOffset := localized.Zone()
	if !strings.HasSuffix(timestamp, "Z") && offset != zoneOffset {
		return Timestamptz{}, fmt.Errorf("TIMESTAMPTZ UTC offset does not match time zone %q: %q", zoneName, s)
	}
	return Timestamptz{Time: localized, tzRaw: tzRaw, Valid: true}, nil
}

// resolveTimestamptzLocalTime resolves legacy local fields and rejects gaps and overlaps.
func resolveTimestamptzLocalTime(wall time.Time, loc *time.Location) (time.Time, error) {
	const offsetWindow = 24 * time.Hour
	limit := wall.Add(offsetWindow)
	var result time.Time
	found := false
	for cursor := wall.Add(-offsetWindow); !cursor.After(limit); {
		local := cursor.In(loc)
		_, offset := local.Zone()
		candidate := wall.Add(-time.Duration(offset) * time.Second).In(loc)
		_, candidateOffset := candidate.Zone()
		if candidateOffset == offset {
			if found && !candidate.Equal(result) {
				return time.Time{}, fmt.Errorf("ambiguous TIMESTAMPTZ local time %s in %s; specify a UTC offset",
					wall.Format("2006-01-02 15:04:05.999999"), loc)
			}
			result, found = candidate, true
		}
		_, end := local.ZoneBounds()
		if end.IsZero() || end.After(limit) {
			break
		}
		if !end.After(cursor) {
			end = time.Date(cursor.UTC().Year()+1, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		cursor = end
	}
	if !found {
		return time.Time{}, fmt.Errorf("nonexistent TIMESTAMPTZ local time %s in %s",
			wall.Format("2006-01-02 15:04:05.999999"), loc)
	}
	return result, nil
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

	loc, err := fbproto.ResolveLocationByTzRaw(tzRaw)
	if err != nil {
		return Timestamptz{}, fmt.Errorf("cannot resolve TIMESTAMPTZ time zone: %w", err)
	}
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

	loc, err := fbproto.ResolveLocationByTzRaw(tzRaw)
	if err != nil {
		return "", fmt.Errorf("cannot format TIMESTAMPTZ: %w", err)
	}
	t := tstz.Time.In(loc)
	layout := "2006-01-02T15:04:05.000000-07:00"
	if _, offset := t.Zone(); offset%60 != 0 {
		layout += ":00"
	}
	s := t.Format(layout)
	if tzRaw > 2878 {
		s += "[" + formatTimetzLocationName(tzRaw, loc) + "]"
	}
	return s, nil
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

	resolvedLoc, err := fbproto.ResolveLocationByTzRaw(tzRaw)
	if err != nil {
		return 0, false
	}

	_, originalOffset := t.Zone()
	_, resolvedOffset := t.In(resolvedLoc).Zone()
	if originalOffset != resolvedOffset {
		return 0, false
	}

	return tzRaw, true
}
