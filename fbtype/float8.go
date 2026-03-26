package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"rdb.red-soft.ru/fbx/internal/pgio"
)

type Float64Scanner interface {
	ScanFloat64(Float8) error
}

type Float64Valuer interface {
	Float64Value() (Float8, error)
}

type Float8 struct {
	Float64 float64
	Valid   bool
}

// ScanFloat64 implements the Float64Scanner interface.
func (f *Float8) ScanFloat64(n Float8) error {
	*f = n
	return nil
}

func (f Float8) Float64Value() (Float8, error) {
	return f, nil
}

func (f *Float8) ScanInt64(n Int8) error {
	*f = Float8{Float64: float64(n.Int64), Valid: n.Valid}
	return nil
}

func (f Float8) Int64Value() (Int8, error) {
	return Int8{Int64: int64(f.Float64), Valid: f.Valid}, nil
}

// Scan implements the database/sql Scanner interface.
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

// Value implements the database/sql/driver Valuer interface.
func (f Float8) Value() (driver.Value, error) {
	if !f.Valid {
		return nil, nil
	}
	return f.Float64, nil
}

func (f Float8) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(f.Float64)
}

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

type Float8Codec struct{}

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
	return pgio.AppendUint64(buf, math.Float64bits(n)), nil
}

type encodePlanFloat8CodecBinaryString struct{}

func (encodePlanFloat8CodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return pgio.AppendUint64(buf, math.Float64bits(n)), nil
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

	return pgio.AppendUint64(buf, math.Float64bits(n.Float64)), nil
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
	return pgio.AppendUint64(buf, math.Float64bits(f)), nil
}

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
	case string:
		return scanPlanBinaryFloat8ToString{}
	}

	return nil
}

type scanPlanBinaryFloat8ToFloat64 struct{}

func (scanPlanBinaryFloat8ToFloat64) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	n := int64(binary.BigEndian.Uint64(src))
	f := (dst).(*float64)
	*f = math.Float64frombits(uint64(n))

	return nil
}

type scanPlanBinaryFloat8ToFloat64Scanner struct{}

func (scanPlanBinaryFloat8ToFloat64Scanner) Scan(src []byte, dst any) error {
	s := (dst).(Float64Scanner)

	if src == nil {
		return s.ScanFloat64(Float8{})
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	n := int64(binary.BigEndian.Uint64(src))
	return s.ScanFloat64(Float8{Float64: math.Float64frombits(uint64(n)), Valid: true})
}

type scanPlanBinaryFloat8ToInt64Scanner struct{}

func (scanPlanBinaryFloat8ToInt64Scanner) Scan(src []byte, dst any) error {
	s := (dst).(Int64Scanner)

	if src == nil {
		return s.ScanInt64(Int8{})
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	ui64 := int64(binary.BigEndian.Uint64(src))
	f64 := math.Float64frombits(uint64(ui64))
	i64 := int64(f64)
	if f64 != float64(i64) {
		return fmt.Errorf("cannot losslessly convert %v to int64", f64)
	}

	return s.ScanInt64(Int8{Int64: i64, Valid: true})
}

type scanPlanBinaryFloat8ToString struct{}

func (scanPlanBinaryFloat8ToString) Scan(src []byte, dst any) error {
	target := dst.(*string)

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
	s := (dst).(TextScanner)

	if src == nil {
		return s.ScanText(Text{})
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for float8: %v", len(src))
	}

	ui64 := int64(binary.BigEndian.Uint64(src))
	f64 := math.Float64frombits(uint64(ui64))

	return s.ScanText(Text{String: strconv.FormatFloat(f64, 'f', -1, 64), Valid: true})
}

func (c Float8Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	return c.DecodeValue(m, typeMetadata, src)
}

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
