package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx/internal/pgio"
)

var firebirdDateEpoch = time.Date(1858, time.November, 17, 0, 0, 0, 0, time.UTC)

var supportedDateLayouts = []string{
	"2006-01-02",
	"2006-1-2",
	"02.01.2006",
	"2.1.2006",
	"2006.01.02",
	"2006.1.2",
	"2006/01/02",
	"2006/1/2",
}

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
		err := scanPlanTextAnyToDateScanner{}.Scan([]byte(src), dst)
		if err != nil {
			*dst = Date{}
		}
		return err
	case time.Time:
		t, err := normalizeFirebirdDate(src)
		if err != nil {
			*dst = Date{}
			return err
		}
		*dst = Date{Time: t, Valid: true}
		return nil
	}

	*dst = Date{}
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

	s, err := formatDateString(src.Time)
	if err != nil {
		return nil, err
	}

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

	t, err = normalizeFirebirdDate(t)
	if err != nil {
		return err
	}

	*dst = Date{Time: t, Valid: true}

	return nil
}

type DateCodec struct{}

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

	t, err := normalizeFirebirdDate(date.Time)
	if err != nil {
		return nil, err
	}

	days := int32((t.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)
	return pgio.AppendInt32(buf, days), nil
}

type encodePlanDateCodecBinaryTime struct{}

func (encodePlanDateCodecBinaryTime) Encode(value any, buf []byte) (newBuf []byte, err error) {
	t, err := normalizeFirebirdDate(value.(time.Time))
	if err != nil {
		return nil, err
	}

	days := int32((t.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)

	return pgio.AppendInt32(buf, days), nil
}

type encodePlanDateCodecBinaryString struct{}

func (encodePlanDateCodecBinaryString) Encode(value any, buf []byte) (newBuf []byte, err error) {
	t, err := parseDateString(value.(string))
	if err != nil {
		return nil, err
	}

	days := int32((t.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay)

	return pgio.AppendInt32(buf, days), nil
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

	formatted, err := formatDateString(date)
	if err != nil {
		return err
	}

	*s = formatted
	return nil
}

type scanPlanTextAnyToDateScanner struct{}

func (scanPlanTextAnyToDateScanner) Scan(src []byte, dst any) error {
	scanner := (dst).(DateScanner)

	if src == nil {
		return scanner.ScanDate(Date{})
	}

	t, err := parseDateString(string(src))
	if err != nil {
		return err
	}

	return scanner.ScanDate(Date{Time: t, Valid: true})
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

func normalizeFirebirdDate(t time.Time) (time.Time, error) {
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

	year := t.Year()
	if year < 1 || year > 9999 {
		return time.Time{}, fmt.Errorf(
			"date out of Firebird range: %04d-%02d-%02d (supported range is 0001-01-01..9999-12-31)",
			t.Year(), t.Month(), t.Day(),
		)
	}

	return t, nil
}

func formatDateString(t time.Time) (string, error) {
	t, err := normalizeFirebirdDate(t)
	if err != nil {
		return "", err
	}

	return t.Format("2006-01-02"), nil
}

func parseDateString(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("invalid date format: empty string")
	}

	for _, layout := range supportedDateLayouts {
		t, err := time.ParseInLocation(layout, s, time.UTC)
		if err == nil {
			return normalizeFirebirdDate(t)
		}
	}

	return time.Time{}, fmt.Errorf(
		"invalid date format: %q (supported: YYYY-MM-DD, YYYY-M-D, DD.MM.YYYY, D.M.YYYY, YYYY.MM.DD, YYYY.M.D, YYYY/MM/DD, YYYY/M/D)", s)
}
