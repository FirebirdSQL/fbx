package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/internal/fbio"
)

type TimestamptzScanner interface {
	ScanTimestamptz(v Timestamptz) error
}

type TimestamptzValuer interface {
	TimestamptzValue() (Timestamptz, error)
}

type Timestamptz struct {
	Time  time.Time
	Valid bool
	tzRaw uint16
}

func (tstz *Timestamptz) ScanTimestamptz(v Timestamptz) error {
	*tstz = v
	return nil
}

func (tstz Timestamptz) TimestamptzValue() (Timestamptz, error) {
	return tstz, nil
}

// Scan implements the database/sql Scanner interface.
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

// Value implements the database/sql/driver Valuer interface.
func (tstz Timestamptz) Value() (driver.Value, error) {
	if !tstz.Valid {
		return nil, nil
	}

	return tstz.Time, nil
}

func (tstz Timestamptz) MarshalJSON() ([]byte, error) {
	if !tstz.Valid {
		return []byte("null"), nil
	}

	s := tstz.Time.Format(time.RFC3339Nano)

	return json.Marshal(s)
}

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

	tim, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		return err
	}

	*tstz = Timestamptz{Time: tim, Valid: true}

	return nil
}

type TimestamptzCodec struct{}

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
	s := value.(string)

	var (
		t   time.Time
		err error
	)

	t, err = time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("timestamptz: cannot parse %q as RFC3339: %w", s, err)
		}
	}

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
		*ptr = time.Time{}
		return nil
	}

	tstz, err := decodeTimestamptzBinary(src)
	if err != nil {
		return err
	}
	if !tstz.Valid {
		*ptr = time.Time{}
		return nil
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
		*ptr = ""
		return nil
	}

	tstz, err := decodeTimestamptzBinary(src)
	if err != nil {
		return err
	}

	if !tstz.Valid {
		*ptr = ""
		return nil
	}

	t := tstz.Time

	*ptr = t.Format(time.RFC3339Nano)
	return nil
}

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

	usec := int64(utc.Hour())*microsecondsPerHour +
		int64(utc.Minute())*microsecondsPerMinute +
		int64(utc.Second())*microsecondsPerSecond +
		int64(utc.Nanosecond())/1000

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

	hour := usec / microsecondsPerHour
	usec -= hour * microsecondsPerHour
	min := usec / microsecondsPerMinute
	usec -= min * microsecondsPerMinute
	sec := usec / microsecondsPerSecond
	usec -= sec * microsecondsPerSecond

	utc := time.Date(
		tDate.Year(), tDate.Month(), tDate.Day(),
		int(hour), int(min), int(sec), int(usec)*1000,
		time.UTC,
	)

	tzRaw := uint16(tzVal & 0xFFFF)
	loc := fbproto.ResolveLocationByTzRaw(tzRaw)

	t := utc.In(loc)

	return Timestamptz{
		Time:  t,
		tzRaw: tzRaw,
		Valid: true,
	}, nil
}

func tzRawFromTime(t time.Time) (uint16, error) {
	_, offsetSeconds := t.Zone()
	offsetMinutes := offsetSeconds / 60

	if offsetMinutes < -1439 || offsetMinutes > 1438 {
		return 0, fmt.Errorf("timestamptz: UTC offset %d minutes out of range", offsetMinutes)
	}

	return uint16(offsetMinutes + 1439), nil
}
