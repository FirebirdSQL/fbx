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

type Float4 struct {
	Float32 float32
	Valid   bool
}

// ScanFloat64 implements the Float64Scanner interface.
func (f *Float4) ScanFloat64(n Float8) error {
	*f = Float4{Float32: float32(n.Float64), Valid: n.Valid}
	return nil
}

func (f Float4) Float64Value() (Float8, error) {
	return Float8{Float64: float64(f.Float32), Valid: f.Valid}, nil
}

func (f *Float4) ScanInt64(n Int8) error {
	*f = Float4{Float32: float32(n.Int64), Valid: n.Valid}
	return nil
}

func (f Float4) Int64Value() (Int8, error) {
	return Int8{Int64: int64(f.Float32), Valid: f.Valid}, nil
}

// Scan implements the database/sql Scanner interface.
func (f *Float4) Scan(src any) error {
	if src == nil {
		*f = Float4{}
		return nil
	}

	switch src := src.(type) {
	case float64:
		*f = Float4{Float32: float32(src), Valid: true}
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

// Value implements the database/sql/driver Valuer interface.
func (f Float4) Value() (driver.Value, error) {
	if !f.Valid {
		return nil, nil
	}
	return float64(f.Float32), nil
}

func (f Float4) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(f.Float32)
}

func (f *Float4) UnmarshalJSON(b []byte) error {
	var n *float32
	err := json.Unmarshal(b, &n)
	if err != nil {
		return err
	}

	if n == nil {
		*f = Float4{}
	} else {
		*f = Float4{Float32: *n, Valid: true}
	}

	return nil
}

type Float4Codec struct{}

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

type encodePlanTextFloat32 struct{}

func (encodePlanTextFloat32) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n := value.(float32)
	return append(buf, strconv.FormatFloat(float64(n), 'f', -1, 32)...), nil
}

type encodePlanTextString struct{}

func (encodePlanTextString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	return append(buf, s...), nil
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

	return fbio.AppendUint32BE(buf, math.Float32bits(float32(n.Float64))), nil
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	n := int32(binary.BigEndian.Uint32(src))
	f := (dst).(*float32)
	*f = math.Float32frombits(uint32(n))

	return nil
}

type scanPlanBinaryFloat4ToFloat64Scanner struct{}

func (scanPlanBinaryFloat4ToFloat64Scanner) Scan(src []byte, dst any) error {
	s := (dst).(Float64Scanner)

	if src == nil {
		return s.ScanFloat64(Float8{})
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	n := int32(binary.BigEndian.Uint32(src))
	return s.ScanFloat64(Float8{Float64: float64(math.Float32frombits(uint32(n))), Valid: true})
}

type scanPlanBinaryFloat4ToInt64Scanner struct{}

func (scanPlanBinaryFloat4ToInt64Scanner) Scan(src []byte, dst any) error {
	s := (dst).(Int64Scanner)

	if src == nil {
		return s.ScanInt64(Int8{})
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	ui32 := int32(binary.BigEndian.Uint32(src))
	f32 := math.Float32frombits(uint32(ui32))
	i64 := int64(f32)
	if f32 != float32(i64) {
		return fmt.Errorf("cannot losslessly convert %v to int64", f32)
	}

	return s.ScanInt64(Int8{Int64: i64, Valid: true})
}

type scanPlanBinaryFloat4ToTextScanner struct{}

func (scanPlanBinaryFloat4ToTextScanner) Scan(src []byte, dst any) error {
	s := (dst).(TextScanner)

	if src == nil {
		return s.ScanText(Text{})
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for float4: %v", len(src))
	}

	ui32 := int32(binary.BigEndian.Uint32(src))
	f32 := math.Float32frombits(uint32(ui32))

	return s.ScanText(Text{String: strconv.FormatFloat(float64(f32), 'f', -1, 32), Valid: true})
}

type scanPlanBinaryFloat4ToString struct{}

func (scanPlanBinaryFloat4ToString) Scan(src []byte, dst any) error {
	target := dst.(*string)

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
