package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"rdb.red-soft.ru/fbx/internal/fbio"
)

// Float64Scanner scans a Firebird floating-point value.
type Float64Scanner interface {
	ScanFloat64(Float8) error
}

// Float64Valuer returns a Firebird floating-point value to encode.
type Float64Valuer interface {
	Float64Value() (Float8, error)
}

func float64ToInt64(n float64) (int64, error) {
	const int64UpperBound = 1 << 63
	if math.IsNaN(n) || math.IsInf(n, 0) || n < math.MinInt64 || n >= int64UpperBound {
		return 0, fmt.Errorf("cannot losslessly convert %v to int64", n)
	}

	i64 := int64(n)
	if n != float64(i64) {
		return 0, fmt.Errorf("cannot losslessly convert %v to int64", n)
	}

	return i64, nil
}

// Float8 represents the Firebird DOUBLE PRECISION type.
//
// Valid is false when the value represents SQL NULL.
type Float8 struct {
	Float64 float64
	Valid   bool
}

// ScanFloat64 implements [Float64Scanner].
func (f *Float8) ScanFloat64(n Float8) error {
	*f = n
	return nil
}

// Float64Value implements [Float64Valuer].
func (f Float8) Float64Value() (Float8, error) {
	return f, nil
}

// ScanInt64 implements [Int64Scanner].
func (f *Float8) ScanInt64(n Int8) error {
	*f = Float8{Float64: float64(n.Int64), Valid: n.Valid}
	return nil
}

// Int64Value implements [Int64Valuer].
func (f Float8) Int64Value() (Int8, error) {
	if !f.Valid {
		return Int8{}, nil
	}

	i64, err := float64ToInt64(f.Float64)
	if err != nil {
		return Int8{}, err
	}

	return Int8{Int64: i64, Valid: true}, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (f *Float8) Scan(src any) error {
	if src == nil {
		*f = Float8{}
		return nil
	}

	switch src := src.(type) {
	case float64:
		*f = Float8{Float64: src, Valid: true}
		return nil
	case string:
		n, err := strconv.ParseFloat(string(src), 64)
		if err != nil {
			return err
		}
		*f = Float8{Float64: n, Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (f Float8) Value() (driver.Value, error) {
	if !f.Valid {
		return nil, nil
	}
	return f.Float64, nil
}

// MarshalJSON encodes f as a JSON number or null.
func (f Float8) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(f.Float64)
}

// UnmarshalJSON decodes a JSON number or null into f.
func (f *Float8) UnmarshalJSON(b []byte) error {
	var n *float64
	err := json.Unmarshal(b, &n)
	if err != nil {
		return err
	}

	if n == nil {
		*f = Float8{}
	} else {
		*f = Float8{Float64: *n, Valid: true}
	}

	return nil
}

// Float8Codec encodes and decodes Firebird DOUBLE PRECISION values.
type Float8Codec struct{}

// PlanEncode returns an encode plan for supported Firebird DOUBLE PRECISION input values.
func (Float8Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case float64:
		return encodePlanFloat8CodecBinaryFloat64{}
	case string:
		return encodePlanFloat8CodecBinaryString{}
	case Float64Valuer:
		return encodePlanFloat8CodecBinaryFloat64Valuer{}
	case Int64Valuer:
		return encodePlanFloat8CodecBinaryInt64Valuer{}
	}

	return nil
}

type encodePlanFloat8CodecBinaryFloat64 struct{}

func (encodePlanFloat8CodecBinaryFloat64) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n := value.(float64)
	return fbio.AppendUint64BE(buf, math.Float64bits(n)), nil
}

type encodePlanFloat8CodecBinaryString struct{}

func (encodePlanFloat8CodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return fbio.AppendUint64BE(buf, math.Float64bits(n)), nil
}

type encodePlanFloat8CodecBinaryFloat64Valuer struct{}

func (encodePlanFloat8CodecBinaryFloat64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Float64Valuer).Float64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	return fbio.AppendUint64BE(buf, math.Float64bits(n.Float64)), nil
}

type encodePlanFloat8CodecBinaryInt64Valuer struct{}

func (encodePlanFloat8CodecBinaryInt64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	f := float64(n.Int64)
	return fbio.AppendUint64BE(buf, math.Float64bits(f)), nil
}

// PlanScan returns a scan plan for supported Firebird DOUBLE PRECISION target values.
func (Float8Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *float64:
		return scanPlanBinaryFloat8ToFloat64{}
	case Float64Scanner:
		return scanPlanBinaryFloat8ToFloat64Scanner{}
	case Int64Scanner:
		return scanPlanBinaryFloat8ToInt64Scanner{}
	case TextScanner:
		return scanPlanBinaryFloat8ToTextScanner{}
	case *string:
		return scanPlanBinaryFloat8ToString{}
	}

	return nil
}

type scanPlanBinaryFloat8ToFloat64 struct{}

func (scanPlanBinaryFloat8ToFloat64) Scan(src []byte, dst any) error {
	f, ok := (dst).(*float64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	*f = math.Float64frombits(binary.BigEndian.Uint64(src))

	return nil
}

type scanPlanBinaryFloat8ToFloat64Scanner struct{}

func (scanPlanBinaryFloat8ToFloat64Scanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(Float64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanFloat64(Float8{})
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	f64 := math.Float64frombits(binary.BigEndian.Uint64(src))
	return s.ScanFloat64(Float8{Float64: f64, Valid: true})
}

type scanPlanBinaryFloat8ToInt64Scanner struct{}

func (scanPlanBinaryFloat8ToInt64Scanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(Int64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanInt64(Int8{})
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	f64 := math.Float64frombits(binary.BigEndian.Uint64(src))
	i64, err := float64ToInt64(f64)
	if err != nil {
		return err
	}

	return s.ScanInt64(Int8{Int64: i64, Valid: true})
}

type scanPlanBinaryFloat8ToString struct{}

func (scanPlanBinaryFloat8ToString) Scan(src []byte, dst any) error {
	target, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	ui64 := binary.BigEndian.Uint64(src)
	f64 := math.Float64frombits(ui64)

	*target = strconv.FormatFloat(f64, 'f', -1, 64)
	return nil
}

type scanPlanBinaryFloat8ToTextScanner struct{}

func (scanPlanBinaryFloat8ToTextScanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanText(Text{})
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	f64 := math.Float64frombits(binary.BigEndian.Uint64(src))

	return s.ScanText(Text{String: strconv.FormatFloat(f64, 'f', -1, 64), Valid: true})
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c Float8Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	return c.DecodeValue(m, typeMetadata, src)
}

// DecodeValue decodes src into the default Go value for Firebird DOUBLE PRECISION.
func (c Float8Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var n float64
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
