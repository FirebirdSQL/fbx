package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"rdb.red-soft.ru/fbx/internal/pgio"
)

var firebirdDateEpoch = time.Date(1858, time.November, 17, 0, 0, 0, 0, time.UTC)

type DateScanner interface {
	ScanDate(v Date) error
}

type DateValuer interface {
	DateValue() (Date, error)
}

type Date struct {
	Time  time.Time
	Valid bool
}

func (d *Date) ScanDate(v Date) error {
	*d = v
	return nil
}

func (d Date) DateValue() (Date, error) {
	return d, nil
}

// Scan implements the database/sql Scanner interface.
func (dst *Date) Scan(src any) error {
	if src == nil {
		*dst = Date{}
		return nil
	}

	switch src := src.(type) {
	case string:
		return scanPlanTextAnyToDateScanner{}.Scan([]byte(src), dst)
	case time.Time:
		*dst = Date{Time: src, Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (src Date) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}

	return src.Time, nil
}

func (src Date) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}

	s := src.Time.Format("2006-01-02")

	return json.Marshal(s)
}

func (dst *Date) UnmarshalJSON(b []byte) error {
	var s *string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	if s == nil {
		*dst = Date{}
		return nil
	}

	t, err := time.ParseInLocation("2006-01-02", *s, time.UTC)
	if err != nil {
		return err
	}

	*dst = Date{Time: t, Valid: true}

	return nil
}

type DateCodec struct{}

func (DateCodec) FormatSupported(format int16) bool {
	return format == BinaryFormatCode
}

func (DateCodec) PreferredFormat() int16 {
	return BinaryFormatCode
}

func (DateCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case DateValuer:
		return encodePlanDateCodecBinary{}
	case time.Time:
		return encodePlanDateCodecBinaryTime{}
	case string:
		return encodePlanDateCodecBinaryString{}
	}

	return nil
}

type encodePlanDateCodecBinary struct{}

func (encodePlanDateCodecBinary) Encode(value any, buf []byte) (newBuf []byte, err error) {
	date, err := value.(DateValuer).DateValue()
	if err != nil {
		return nil, err
	}

	if !date.Valid {
		return nil, nil
	}

	tUnix := time.Date(date.Time.Year(), date.Time.Month(), date.Time.Day(), 0, 0, 0, 0, time.UTC)

	days := int32((tUnix.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)

	return pgio.AppendInt32(buf, days), nil
}

type encodePlanDateCodecBinaryTime struct{}

func (encodePlanDateCodecBinaryTime) Encode(value any, buf []byte) (newBuf []byte, err error) {
	t := value.(time.Time)

	tUnix := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	days := int32((tUnix.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)

	return pgio.AppendInt32(buf, days), nil
}

type encodePlanDateCodecBinaryString struct{}

func (encodePlanDateCodecBinaryString) Encode(value any, buf []byte) (newBuf []byte, err error) {
	str := value.(string)

	match := dateRegexp.FindStringSubmatch(str)
	if match == nil {
		return nil, fmt.Errorf("invalid date format: %q (expected YYYY-MM-DD)", str)
	}

	year, err := strconv.ParseInt(match[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot parse year: %w", err)
	}

	month, err := strconv.ParseInt(match[2], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot parse month: %w", err)
	}

	day, err := strconv.ParseInt(match[3], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot parse day: %w", err)
	}

	if len(match[4]) > 0 {
		year = -year + 1
	}

	t := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
	days := int32((t.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)

	return pgio.AppendInt32(buf, days), nil
}

type encodePlanDateCodecText struct{}

func (encodePlanDateCodecText) Encode(value any, buf []byte) (newBuf []byte, err error) {
	date, err := value.(DateValuer).DateValue()
	if err != nil {
		return nil, err
	}

	if !date.Valid {
		return nil, nil
	}

	year := date.Time.Year()

	yearBytes := strconv.AppendInt(make([]byte, 0, 6), int64(year), 10)
	for i := len(yearBytes); i < 4; i++ {
		buf = append(buf, '0')
	}
	buf = append(buf, yearBytes...)
	buf = append(buf, '-')
	if date.Time.Month() < 10 {
		buf = append(buf, '0')
	}
	buf = strconv.AppendInt(buf, int64(date.Time.Month()), 10)
	buf = append(buf, '-')
	if date.Time.Day() < 10 {
		buf = append(buf, '0')
	}
	buf = strconv.AppendInt(buf, int64(date.Time.Day()), 10)

	return buf, nil
}

func (DateCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case DateScanner:
		return scanPlanBinaryDateToDateScanner{}
	case *time.Time:
		return scanPlanBinaryDateToTime{}
	case *string:
		return scanPlanBinaryDateToString{}
	}

	return nil
}

type scanPlanBinaryDateToDateScanner struct{}

func (scanPlanBinaryDateToDateScanner) Scan(src []byte, dst any) error {
	scanner := (dst).(DateScanner)

	if src == nil {
		return scanner.ScanDate(Date{})
	}

	if len(src) != 4 {
		return fmt.Errorf("fbtype: invalid length for Firebird DATE: %d", len(src))
	}

	dayOffset := int32(binary.BigEndian.Uint32(src))

	t := firebirdDateEpoch.AddDate(0, 0, int(dayOffset))

	return scanner.ScanDate(Date{Time: t, Valid: true})
}

type scanPlanBinaryDateToTime struct{}

func (scanPlanBinaryDateToTime) Scan(src []byte, dst any) error {
	t, ok := dst.(*time.Time)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *time.Time")
	}

	if len(src) != 4 {
		return fmt.Errorf("fbtype: invalid length for Firebird DATE: %d", len(src))
	}

	dayOffset := int32(binary.BigEndian.Uint32(src))
	*t = firebirdDateEpoch.AddDate(0, 0, int(dayOffset))

	return nil
}

type scanPlanBinaryDateToString struct{}

func (scanPlanBinaryDateToString) Scan(src []byte, dst any) error {
	s, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		*s = ""
		return nil
	}

	if len(src) != 4 {
		return fmt.Errorf("fbtype: invalid length for Firebird DATE: %d", len(src))
	}

	dayOffset := int32(binary.BigEndian.Uint32(src))
	date := firebirdDateEpoch.AddDate(0, 0, int(dayOffset))

	*s = date.Format("2006-01-02")

	return nil
}

type scanPlanTextAnyToDateScanner struct{}

var dateRegexp = regexp.MustCompile(`^(\d{4,})-(\d\d)-(\d\d)( BC)?$`)

func (scanPlanTextAnyToDateScanner) Scan(src []byte, dst any) error {
	scanner := (dst).(DateScanner)

	if src == nil {
		return scanner.ScanDate(Date{})
	}

	sbuf := string(src)
	match := dateRegexp.FindStringSubmatch(sbuf)
	if match != nil {
		year, err := strconv.ParseInt(match[1], 10, 32)
		if err != nil {
			return fmt.Errorf("BUG: cannot parse date that regexp matched (year): %w", err)
		}

		month, err := strconv.ParseInt(match[2], 10, 32)
		if err != nil {
			return fmt.Errorf("BUG: cannot parse date that regexp matched (month): %w", err)
		}

		day, err := strconv.ParseInt(match[3], 10, 32)
		if err != nil {
			return fmt.Errorf("BUG: cannot parse date that regexp matched (month): %w", err)
		}

		// BC matched
		if len(match[4]) > 0 {
			year = -year + 1
		}

		t := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
		return scanner.ScanDate(Date{Time: t, Valid: true})
	}

	return fmt.Errorf("invalid date format")
}

func (c DateCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var date Date
	err := codecScan(c, m, typeMetadata, src, &date)
	if err != nil {
		return nil, err
	}

	return date.Time, nil
}

func (c DateCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var date Date
	err := codecScan(c, m, typeMetadata, src, &date)
	if err != nil {
		return nil, err
	}

	return date.Time, nil
}
