package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"rdb.red-soft.ru/fbx/internal/pgio"
)

const (
	firebirdTimestampFormat = "2006-01-02 15:04:05.999999"
	jsonTimestampFormat     = "2006-01-02T15:04:05.999999"
)

type TimestampScanner interface {
	ScanTimestamp(v Timestamp) error
}

type TimestampValuer interface {
	TimestampValue() (Timestamp, error)
}

// Timestamp represents the Firebird timestamp type.
type Timestamp struct {
	Time  time.Time // Time zone will be ignored when encoding to Firebird.
	Valid bool
}

func (ts *Timestamp) ScanTimestamp(v Timestamp) error {
	*ts = v
	return nil
}

func (ts Timestamp) TimestampValue() (Timestamp, error) {
	return ts, nil
}

// Scan implements the database/sql Scanner interface.
func (ts *Timestamp) Scan(src any) error {
	if src == nil {
		*ts = Timestamp{}
		return nil
	}

	switch src := src.(type) {
	case string:
		return (&scanPlanTextTimestampToTimestampScanner{}).Scan([]byte(src), ts)
	case time.Time:
		*ts = Timestamp{Time: src, Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (ts Timestamp) Value() (driver.Value, error) {
	if !ts.Valid {
		return nil, nil
	}

	return ts.Time, nil
}

func (ts Timestamp) MarshalJSON() ([]byte, error) {
	if !ts.Valid {
		return []byte("null"), nil
	}

	s := ts.Time.Format(jsonTimestampFormat)
	return json.Marshal(s)
}

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

	if tim, err := time.Parse(time.RFC3339Nano, tss); err == nil {
		*ts = Timestamp{Time: tim, Valid: true}
		return nil
	}

	if tim, err := time.ParseInLocation(jsonTimestampFormat, tss, time.UTC); err == nil {
		*ts = Timestamp{Time: tim, Valid: true}
		return nil
	}

	if tim, err := time.ParseInLocation(firebirdTimestampFormat, tss, time.UTC); err == nil {
		*ts = Timestamp{Time: tim, Valid: true}
		return nil
	}

	ts.Valid = false
	return fmt.Errorf(
		"cannot unmarshal %s to timestamp with layout %s or %s or %s",
		*s, time.RFC3339Nano, jsonTimestampFormat, firebirdTimestampFormat,
	)
}

type TimestampCodec struct {
	// ScanLocation is the location that the time is assumed to be in for scanning. This is different from
	// TimestamptzCodec.ScanLocation in that this setting does change the instant in time that the timestamp represents.
	ScanLocation *time.Location
}

func (*TimestampCodec) FormatSupported(format int16) bool {
	return format == TextFormatCode || format == BinaryFormatCode
}

func (*TimestampCodec) PreferredFormat() int16 {
	return BinaryFormatCode
}

func (*TimestampCodec) PlanEncode(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, value any) EncodePlan {
	if _, ok := value.(TimestampValuer); !ok {
		return nil
	}

	switch format {
	case BinaryFormatCode:
		return encodePlanTimestampCodecBinary{}
	case TextFormatCode:
		return encodePlanTimestampCodecText{}
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

	t := ts.Time

	tMidnightUTC := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

	days := int32((tMidnightUTC.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)

	usec := int64(t.Hour())*microsecondsPerHour +
		int64(t.Minute())*microsecondsPerMinute +
		int64(t.Second())*microsecondsPerSecond +
		int64(t.Nanosecond())/1000

	if usec < 0 || usec >= microsecondsPerDay {
		return nil, fmt.Errorf("fbtype: TIMESTAMP time-of-day out of range (µs since midnight: %d)", usec)
	}

	ticks := uint32(usec / 100)

	buf = pgio.AppendInt32(buf, days)
	buf = pgio.AppendInt32(buf, int32(ticks))

	return buf, nil
}

type encodePlanTimestampCodecText struct{}

func (encodePlanTimestampCodecText) Encode(value any, buf []byte) (newBuf []byte, err error) {
	ts, err := value.(TimestampValuer).TimestampValue()
	if err != nil {
		return nil, err
	}

	if !ts.Valid {
		return nil, nil
	}

	t := ts.Time.In(time.UTC).Truncate(time.Microsecond)
	s := t.Format(firebirdTimestampFormat)

	buf = append(buf, s...)

	return buf, nil
}

func (c *TimestampCodec) PlanScan(m *Map, typeId uint32, subType int32, length uint32, scale int32, format int16, target any) ScanPlan {
	switch format {
	case BinaryFormatCode:
		switch target.(type) {
		case TimestampScanner:
			return &scanPlanBinaryTimestampToTimestampScanner{location: c.ScanLocation}
		}
	case TextFormatCode:
		switch target.(type) {
		case TimestampScanner:
			return &scanPlanTextTimestampToTimestampScanner{location: c.ScanLocation}
		}
	}

	return nil
}

type scanPlanBinaryTimestampToTimestampScanner struct{ location *time.Location }

func (plan *scanPlanBinaryTimestampToTimestampScanner) Scan(src []byte, dst any) error {
	scanner := dst.(TimestampScanner)

	if src == nil {
		return scanner.ScanTimestamp(Timestamp{})
	}
	if len(src) != 8 {
		return fmt.Errorf("invalid length for TIMESTAMP: %d", len(src))
	}

	dayOffset := int32(binary.BigEndian.Uint32(src[0:4]))
	ticks := binary.BigEndian.Uint32(src[4:8])

	tDate := firebirdDateEpoch.AddDate(0, 0, int(dayOffset))

	usec := int64(ticks) * 100
	if usec < 0 || usec >= microsecondsPerDay {
		return fmt.Errorf("TIMESTAMP time-of-day out of range in decode (µs since midnight: %d)", usec)
	}

	hour := usec / microsecondsPerHour
	usec -= hour * microsecondsPerHour
	min := usec / microsecondsPerMinute
	usec -= min * microsecondsPerMinute
	sec := usec / microsecondsPerSecond
	usec -= sec * microsecondsPerSecond

	loc := time.UTC
	if plan.location != nil {
		loc = plan.location
	}

	t := time.Date(
		tDate.Year(), tDate.Month(), tDate.Day(),
		int(hour), int(min), int(sec), int(usec)*1000,
		loc,
	)

	return scanner.ScanTimestamp(Timestamp{Time: t, Valid: true})
}

type scanPlanTextTimestampToTimestampScanner struct{ location *time.Location }

func (plan *scanPlanTextTimestampToTimestampScanner) Scan(src []byte, dst any) error {
	scanner := dst.(TimestampScanner)

	if src == nil {
		return scanner.ScanTimestamp(Timestamp{})
	}

	s := string(src)
	loc := time.UTC
	if plan.location != nil {
		loc = plan.location
	}

	t, err := time.ParseInLocation("2006-01-02 15:04:05.999999", s, loc)
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02 15:04:05", s, loc)
		if err != nil {
			return err
		}
	}

	return scanner.ScanTimestamp(Timestamp{Time: t, Valid: true})
}

func (c *TimestampCodec) DecodeDatabaseSQLValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var ts Timestamp
	err := codecScan(c, m, typeid, subType, length, scale, format, src, &ts)
	if err != nil {
		return nil, err
	}

	return ts.Time, nil
}

func (c *TimestampCodec) DecodeValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var ts Timestamp
	err := codecScan(c, m, typeid, subType, length, scale, format, src, &ts)
	if err != nil {
		return nil, err
	}

	return ts.Time, nil
}
