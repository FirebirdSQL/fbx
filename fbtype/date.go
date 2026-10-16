package fbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx/internal/fbio"
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

// DateScanner scans a Firebird DATE value.
type DateScanner interface {
	// ScanDate scans v into the receiver.
	ScanDate(v Date) error
}

// DateValuer returns a Firebird DATE value.
type DateValuer interface {
	// DateValue returns the Firebird DATE value to encode.
	DateValue() (Date, error)
}

// Date represents the Firebird DATE type.
//
// Time contains the date normalized to midnight UTC. Firebird DATE stores only
// the calendar date, so time-of-day and location are discarded when values are
// encoded or scanned. Valid is false when the value represents SQL NULL.
type Date struct {
	Time  time.Time // Time contains the date normalized to midnight UTC.
	Valid bool      // Valid is false when the value represents SQL NULL.
}

// ScanDate implements [DateScanner].
func (d *Date) ScanDate(v Date) error {
	*d = v
	return nil
}

// DateValue implements [DateValuer].
func (d Date) DateValue() (Date, error) {
	return d, nil
}

// Scan implements the [database/sql.Scanner] interface.
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

// Value implements the [database/sql/driver.Valuer] interface.
func (src Date) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}

	return normalizeFirebirdDate(src.Time)
}

// MarshalJSON encodes src as a JSON string in YYYY-MM-DD format or null.
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

// UnmarshalJSON decodes a JSON string in YYYY-MM-DD format or null into dst.
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

// DateCodec encodes and decodes Firebird DATE values.
type DateCodec struct{}

// PlanEncode returns an encode plan for supported Firebird DATE input values.
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

	days, err := firebirdDateDayOffset(date.Time)
	if err != nil {
		return nil, err
	}

	return fbio.AppendInt32BE(buf, days), nil
}

type encodePlanDateCodecBinaryTime struct{}

func (encodePlanDateCodecBinaryTime) Encode(value any, buf []byte) (newBuf []byte, err error) {
	days, err := firebirdDateDayOffset(value.(time.Time))
	if err != nil {
		return nil, err
	}

	return fbio.AppendInt32BE(buf, days), nil
}

type encodePlanDateCodecBinaryString struct{}

func (encodePlanDateCodecBinaryString) Encode(value any, buf []byte) (newBuf []byte, err error) {
	t, err := parseDateString(value.(string))
	if err != nil {
		return nil, err
	}

	days, err := firebirdDateDayOffset(t)
	if err != nil {
		return nil, err
	}

	return fbio.AppendInt32BE(buf, days), nil
}

// PlanScan returns a scan plan for supported Firebird DATE target values.
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
	scanner, ok := dst.(DateScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanDate(Date{})
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("fbtype: Firebird DATE: %w", err)
	}

	dayOffset := int32(bits)
	t, err := firebirdDateFromDayOffset(dayOffset)
	if err != nil {
		return err
	}

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

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("fbtype: Firebird DATE: %w", err)
	}

	dayOffset := int32(bits)
	date, err := firebirdDateFromDayOffset(dayOffset)
	if err != nil {
		return err
	}

	*t = date

	return nil
}

type scanPlanBinaryDateToString struct{}

func (scanPlanBinaryDateToString) Scan(src []byte, dst any) error {
	s, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("fbtype: Firebird DATE: %w", err)
	}

	dayOffset := int32(bits)
	date, err := firebirdDateFromDayOffset(dayOffset)
	if err != nil {
		return err
	}

	formatted, err := formatDateString(date)
	if err != nil {
		return err
	}

	*s = formatted
	return nil
}

type scanPlanTextAnyToDateScanner struct{}

func (scanPlanTextAnyToDateScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(DateScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanDate(Date{})
	}

	t, err := parseDateString(string(src))
	if err != nil {
		return err
	}

	return scanner.ScanDate(Date{Time: t, Valid: true})
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
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

// DecodeValue decodes src into the default Go value for Firebird DATE.
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

func firebirdDateDayOffset(t time.Time) (int32, error) {
	t, err := normalizeFirebirdDate(t)
	if err != nil {
		return 0, err
	}

	days := (t.Unix() - firebirdDateEpoch.Unix()) / secondsPerDay
	if days < math.MinInt32 || days > math.MaxInt32 {
		return 0, fmt.Errorf(
			"date out of Firebird wire range: %04d-%02d-%02d",
			t.Year(), t.Month(), t.Day(),
		)
	}

	return int32(days), nil
}

func firebirdDateFromDayOffset(dayOffset int32) (time.Time, error) {
	return normalizeFirebirdDate(firebirdDateEpoch.AddDate(0, 0, int(dayOffset)))
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
