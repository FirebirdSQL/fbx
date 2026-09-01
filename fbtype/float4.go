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

// Float4 represents the Firebird FLOAT type.
//
// Valid is false when the value represents SQL NULL.
type Float4 struct {
	Float32 float32
	Valid   bool
}

func float64ToFloat32(n float64) (float32, error) {
	if !math.IsNaN(n) && !math.IsInf(n, 0) && (n > math.MaxFloat32 || n < -math.MaxFloat32) {
		return 0, fmt.Errorf("%v overflows float32", n)
	}
	return float32(n), nil
}

// ScanFloat64 implements [Float64Scanner].
func (f *Float4) ScanFloat64(n Float8) error {
	if !n.Valid {
		*f = Float4{}
		return nil
	}

	f32, err := float64ToFloat32(n.Float64)
	if err != nil {
		*f = Float4{}
		return err
	}

	*f = Float4{Float32: f32, Valid: true}
	return nil
}

// Float64Value implements [Float64Valuer].
func (f Float4) Float64Value() (Float8, error) {
	return Float8{Float64: float64(f.Float32), Valid: f.Valid}, nil
}

// ScanInt64 implements [Int64Scanner].
func (f *Float4) ScanInt64(n Int8) error {
	*f = Float4{Float32: float32(n.Int64), Valid: n.Valid}
	return nil
}

// Int64Value implements [Int64Valuer].
func (f Float4) Int64Value() (Int8, error) {
	if !f.Valid {
		return Int8{}, nil
	}

	i64, err := float64ToInt64(float64(f.Float32))
	if err != nil {
		return Int8{}, err
	}

	return Int8{Int64: i64, Valid: true}, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (f *Float4) Scan(src any) error {
	*f = Float4{}

	if src == nil {
		return nil
	}

	switch src := src.(type) {
	case float64:
		f32, err := float64ToFloat32(src)
		if err != nil {
			return err
		}
		*f = Float4{Float32: f32, Valid: true}
		return nil
	case string:
		n, err := strconv.ParseFloat(string(src), 32)
		if err != nil {
			return err
		}
		*f = Float4{Float32: float32(n), Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (f Float4) Value() (driver.Value, error) {
	if !f.Valid {
		return nil, nil
	}
	return float64(f.Float32), nil
}

// MarshalJSON encodes f as a JSON number or null.
func (f Float4) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(f.Float32)
}

// UnmarshalJSON decodes a JSON number or null into f.
func (f *Float4) UnmarshalJSON(b []byte) error {
	*f = Float4{}

	var n *float32
	err := json.Unmarshal(b, &n)
	if err != nil {
		return err
	}

	if n == nil {
		return nil
	}
	*f = Float4{Float32: *n, Valid: true}

	return nil
}

// Float4Codec encodes and decodes Firebird FLOAT values.
type Float4Codec struct{}

// PlanEncode returns an encode plan for supported Firebird FLOAT input values.
func (Float4Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case float32:
		return encodePlanFloat4CodecBinaryFloat32{}
	case string:
		return encodePlanFloat4CodecBinaryString{}
	case Float64Valuer:
		return encodePlanFloat4CodecBinaryFloat64Valuer{}
	case Int64Valuer:
		return encodePlanFloat4CodecBinaryInt64Valuer{}
	}

	return nil
}

type encodePlanFloat4CodecBinaryFloat32 struct{}

func (encodePlanFloat4CodecBinaryFloat32) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n := value.(float32)
	return fbio.AppendUint32BE(buf, math.Float32bits(n)), nil
}

type encodePlanFloat4CodecBinaryString struct{}

func (encodePlanFloat4CodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	n, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return nil, err
	}
	return fbio.AppendUint32BE(buf, math.Float32bits(float32(n))), nil
}

type encodePlanFloat4CodecBinaryFloat64Valuer struct{}

func (encodePlanFloat4CodecBinaryFloat64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Float64Valuer).Float64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	f32, err := float64ToFloat32(n.Float64)
	if err != nil {
		return nil, err
	}

	return fbio.AppendUint32BE(buf, math.Float32bits(f32)), nil
}

type encodePlanFloat4CodecBinaryInt64Valuer struct{}

func (encodePlanFloat4CodecBinaryInt64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	f := float32(n.Int64)
	return fbio.AppendUint32BE(buf, math.Float32bits(f)), nil
}

// PlanScan returns a scan plan for supported Firebird FLOAT target values.
func (Float4Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *float32:
		return scanPlanBinaryFloat4ToFloat32{}
	case Float64Scanner:
		return scanPlanBinaryFloat4ToFloat64Scanner{}
	case Int64Scanner:
		return scanPlanBinaryFloat4ToInt64Scanner{}
	case TextScanner:
		return scanPlanBinaryFloat4ToTextScanner{}
	case *string:
		return scanPlanBinaryFloat4ToString{}
	}

	return nil
}

type scanPlanBinaryFloat4ToFloat32 struct{}

func (scanPlanBinaryFloat4ToFloat32) Scan(src []byte, dst any) error {
	f, ok := (dst).(*float32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	*f = math.Float32frombits(binary.BigEndian.Uint32(src))

	return nil
}

type scanPlanBinaryFloat4ToFloat64Scanner struct{}

func (scanPlanBinaryFloat4ToFloat64Scanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(Float64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanFloat64(Float8{})
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	f32 := math.Float32frombits(binary.BigEndian.Uint32(src))
	return s.ScanFloat64(Float8{Float64: float64(f32), Valid: true})
}

type scanPlanBinaryFloat4ToInt64Scanner struct{}

func (scanPlanBinaryFloat4ToInt64Scanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(Int64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanInt64(Int8{})
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	f32 := math.Float32frombits(binary.BigEndian.Uint32(src))
	i64, err := float64ToInt64(float64(f32))
	if err != nil {
		return err
	}

	return s.ScanInt64(Int8{Int64: i64, Valid: true})
}

type scanPlanBinaryFloat4ToTextScanner struct{}

func (scanPlanBinaryFloat4ToTextScanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanText(Text{})
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	f32 := math.Float32frombits(binary.BigEndian.Uint32(src))

	return s.ScanText(Text{String: strconv.FormatFloat(float64(f32), 'f', -1, 32), Valid: true})
}

type scanPlanBinaryFloat4ToString struct{}

func (scanPlanBinaryFloat4ToString) Scan(src []byte, dst any) error {
	target, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	ui32 := binary.BigEndian.Uint32(src)
	f32 := math.Float32frombits(ui32)

	*target = strconv.FormatFloat(float64(f32), 'f', -1, 32)
	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c Float4Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var n float32
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return float64(n), nil
}

// DecodeValue decodes src into the default Go value for Firebird FLOAT.
func (c Float4Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var n float32
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
