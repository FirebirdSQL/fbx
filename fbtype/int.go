package fbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/internal/fbio"
)

// Int64Scanner scans a Firebird integer value represented as [Int8].
type Int64Scanner interface {
	// ScanInt64 scans n into the receiver.
	ScanInt64(Int8) error
}

// Int64Valuer returns a Firebird integer value represented as [Int8].
type Int64Valuer interface {
	// Int64Value returns the Firebird integer value to encode.
	Int64Value() (Int8, error)
}

// Int2 represents the Firebird SMALLINT type.
//
// Firebird sends SMALLINT values as 4-byte row values in DSQL, but Int2 keeps
// the Go-side value at the natural 16-bit width. Valid is false when the value
// represents SQL NULL.
type Int2 struct {
	Int16 int16
	Valid bool
}

// ScanInt64 implements [Int64Scanner].
func (dst *Int2) ScanInt64(n Int8) error {
	if !n.Valid {
		*dst = Int2{}
		return nil
	}

	if n.Int64 < math.MinInt16 {
		*dst = Int2{}
		return fmt.Errorf("%d is less than minimum value for Int2", n.Int64)
	}
	if n.Int64 > math.MaxInt16 {
		*dst = Int2{}
		return fmt.Errorf("%d is greater than maximum value for Int2", n.Int64)
	}
	*dst = Int2{Int16: int16(n.Int64), Valid: true}

	return nil
}

// Int64Value implements [Int64Valuer].
func (n Int2) Int64Value() (Int8, error) {
	return Int8{Int64: int64(n.Int16), Valid: n.Valid}, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (dst *Int2) Scan(src any) error {
	if src == nil {
		*dst = Int2{}
		return nil
	}

	var n int64

	switch src := src.(type) {
	case int64:
		n = src
	case string:
		var err error
		n, err = strconv.ParseInt(src, 10, 16)
		if err != nil {
			*dst = Int2{}
			return err
		}
	case []byte:
		var err error
		n, err = strconv.ParseInt(string(src), 10, 16)
		if err != nil {
			*dst = Int2{}
			return err
		}
	default:
		*dst = Int2{}
		return fmt.Errorf("cannot scan %T", src)
	}

	if n < math.MinInt16 {
		*dst = Int2{}
		return fmt.Errorf("%d is less than minimum value for Int2", n)
	}
	if n > math.MaxInt16 {
		*dst = Int2{}
		return fmt.Errorf("%d is greater than maximum value for Int2", n)
	}
	*dst = Int2{Int16: int16(n), Valid: true}

	return nil
}

// Value implements the [database/sql/driver.Valuer] interface.
func (src Int2) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}
	return int64(src.Int16), nil
}

// MarshalJSON encodes src as a JSON number or null.
func (src Int2) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(src.Int16), 10)), nil
}

// UnmarshalJSON decodes a JSON number or null into dst.
func (dst *Int2) UnmarshalJSON(b []byte) error {
	var n *int16
	err := json.Unmarshal(b, &n)
	if err != nil {
		*dst = Int2{}
		return err
	}

	if n == nil {
		*dst = Int2{}
	} else {
		*dst = Int2{Int16: *n, Valid: true}
	}

	return nil
}

// Int4 represents the Firebird INTEGER type.
//
// Valid is false when the value represents SQL NULL.
type Int4 struct {
	Int32 int32
	Valid bool
}

// ScanInt64 implements [Int64Scanner].
func (dst *Int4) ScanInt64(n Int8) error {
	if !n.Valid {
		*dst = Int4{}
		return nil
	}

	if n.Int64 < math.MinInt32 {
		*dst = Int4{}
		return fmt.Errorf("%d is less than minimum value for Int4", n.Int64)
	}
	if n.Int64 > math.MaxInt32 {
		*dst = Int4{}
		return fmt.Errorf("%d is greater than maximum value for Int4", n.Int64)
	}
	*dst = Int4{Int32: int32(n.Int64), Valid: true}

	return nil
}

// Int64Value implements [Int64Valuer].
func (n Int4) Int64Value() (Int8, error) {
	return Int8{Int64: int64(n.Int32), Valid: n.Valid}, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (dst *Int4) Scan(src any) error {
	if src == nil {
		*dst = Int4{}
		return nil
	}

	var n int64

	switch src := src.(type) {
	case int64:
		n = src
	case string:
		var err error
		n, err = strconv.ParseInt(src, 10, 32)
		if err != nil {
			*dst = Int4{}
			return err
		}
	case []byte:
		var err error
		n, err = strconv.ParseInt(string(src), 10, 32)
		if err != nil {
			*dst = Int4{}
			return err
		}
	default:
		*dst = Int4{}
		return fmt.Errorf("cannot scan %T", src)
	}

	if n < math.MinInt32 {
		*dst = Int4{}
		return fmt.Errorf("%d is less than minimum value for Int4", n)
	}
	if n > math.MaxInt32 {
		*dst = Int4{}
		return fmt.Errorf("%d is greater than maximum value for Int4", n)
	}
	*dst = Int4{Int32: int32(n), Valid: true}

	return nil
}

// Value implements the [database/sql/driver.Valuer] interface.
func (src Int4) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}
	return int64(src.Int32), nil
}

// MarshalJSON encodes src as a JSON number or null.
func (src Int4) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(src.Int32), 10)), nil
}

// UnmarshalJSON decodes a JSON number or null into dst.
func (dst *Int4) UnmarshalJSON(b []byte) error {
	var n *int32
	err := json.Unmarshal(b, &n)
	if err != nil {
		*dst = Int4{}
		return err
	}

	if n == nil {
		*dst = Int4{}
	} else {
		*dst = Int4{Int32: *n, Valid: true}
	}

	return nil
}

// Int4Codec encodes and decodes Firebird SMALLINT and INTEGER values.
//
// Firebird DSQL row data uses a 4-byte value for SMALLINT, so this codec also
// handles SMALLINT while preserving int16 as the default Go value for DecodeValue.
type Int4Codec struct{}

// PlanEncode returns an encode plan for supported Firebird SMALLINT and INTEGER input values.
func (Int4Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	small := typeMetadata.TypeID == fbproto.SQL_TYPE_SHORT
	switch value.(type) {
	case int32:
		return encodePlanInt4CodecBinaryInt32{small: small}
	case Int64Valuer:
		return encodePlanInt4CodecBinaryInt64Valuer{small: small}
	case string:
		return encodePlanInt4CodecBinaryFromString{small: small}
	}
	return nil
}

type encodePlanInt4CodecBinaryInt32 struct{ small bool }

func (p encodePlanInt4CodecBinaryInt32) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n := value.(int32)
	if p.small {
		if n > math.MaxInt16 {
			return nil, fmt.Errorf("%d is greater than maximum value for smallint", n)
		}
		if n < math.MinInt16 {
			return nil, fmt.Errorf("%d is less than minimum value for smallint", n)
		}
	}
	return fbio.AppendInt32BE(buf, n), nil
}

type encodePlanInt4CodecBinaryInt64Valuer struct{ small bool }

func (p encodePlanInt4CodecBinaryInt64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	maxv, minv, typeName := int64(math.MaxInt32), int64(math.MinInt32), "int4"
	if p.small {
		maxv, minv, typeName = math.MaxInt16, math.MinInt16, "smallint"
	}
	if n.Int64 > maxv {
		return nil, fmt.Errorf("%d is greater than maximum value for %s", n.Int64, typeName)
	}
	if n.Int64 < minv {
		return nil, fmt.Errorf("%d is less than minimum value for %s", n.Int64, typeName)
	}

	return fbio.AppendInt32BE(buf, int32(n.Int64)), nil
}

type encodePlanInt4CodecBinaryFromString struct{ small bool }

func (p encodePlanInt4CodecBinaryFromString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	bitSize, typeName := 32, "int"
	if p.small {
		bitSize, typeName = 16, "smallint"
	}
	n, err := strconv.ParseInt(s, 10, bitSize)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q as %s: %w", s, typeName, err)
	}
	return fbio.AppendInt32BE(buf, int32(n)), nil
}

// PlanScan returns a scan plan for supported Firebird SMALLINT and INTEGER target values.
func (Int4Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *int8:
		return scanPlanBinaryInt4ToInt8{}
	case *int16:
		return scanPlanBinaryInt4ToInt16{}
	case *int32:
		return scanPlanBinaryInt4ToInt32{}
	case *int64:
		return scanPlanBinaryInt4ToInt64{}
	case *int:
		return scanPlanBinaryInt4ToInt{}
	case *uint8:
		return scanPlanBinaryInt4ToUint8{}
	case *uint16:
		return scanPlanBinaryInt4ToUint16{}
	case *uint32:
		return scanPlanBinaryInt4ToUint32{}
	case *uint64:
		return scanPlanBinaryInt4ToUint64{}
	case *uint:
		return scanPlanBinaryInt4ToUint{}
	case Int64Scanner:
		return scanPlanBinaryInt4ToInt64Scanner{}
	case *string:
		return scanPlanBinaryInt4ToString{}
	case TextScanner:
		return scanPlanBinaryInt4ToTextScanner{}
	}

	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c Int4Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	if typeMetadata.TypeID == fbproto.SQL_TYPE_SHORT {
		var n int16
		if err := codecScan(c, m, typeMetadata, src, &n); err != nil {
			return nil, err
		}
		return int64(n), nil
	}

	var n int64
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// DecodeValue decodes src into the default Go value for Firebird SMALLINT or INTEGER.
func (c Int4Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	if typeMetadata.TypeID == fbproto.SQL_TYPE_SHORT {
		var n int16
		err := codecScan(c, m, typeMetadata, src, &n)
		if err != nil {
			return nil, err
		}
		return n, nil
	}

	var n int32
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}

type scanPlanBinaryInt4ToInt8 struct{}

func (scanPlanBinaryInt4ToInt8) Scan(src []byte, dst any) error {
	p, ok := dst.(*int8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int32(bits)
	if n < math.MinInt8 {
		return fmt.Errorf("%d is less than minimum value for int8", n)
	} else if n > math.MaxInt8 {
		return fmt.Errorf("%d is greater than maximum value for int8", n)
	}

	*p = int8(n)

	return nil
}

type scanPlanBinaryInt4ToUint8 struct{}

func (scanPlanBinaryInt4ToUint8) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int32(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint8", n)
	}

	if n > math.MaxUint8 {
		return fmt.Errorf("%d is greater than maximum value for uint8", n)
	}

	*p = uint8(n)

	return nil
}

type scanPlanBinaryInt4ToInt16 struct{}

func (scanPlanBinaryInt4ToInt16) Scan(src []byte, dst any) error {
	p, ok := dst.(*int16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int32(bits)
	if n < math.MinInt16 {
		return fmt.Errorf("%d is less than minimum value for int16", n)
	} else if n > math.MaxInt16 {
		return fmt.Errorf("%d is greater than maximum value for int16", n)
	}

	*p = int16(n)

	return nil
}

type scanPlanBinaryInt4ToUint16 struct{}

func (scanPlanBinaryInt4ToUint16) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int32(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint16", n)
	}

	if n > math.MaxUint16 {
		return fmt.Errorf("%d is greater than maximum value for uint16", n)
	}

	*p = uint16(n)

	return nil
}

type scanPlanBinaryInt4ToInt32 struct{}

func (scanPlanBinaryInt4ToInt32) Scan(src []byte, dst any) error {
	p, ok := dst.(*int32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	*p = int32(bits)

	return nil
}

type scanPlanBinaryInt4ToUint32 struct{}

func (scanPlanBinaryInt4ToUint32) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int32(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint32", n)
	}

	*p = uint32(n)

	return nil
}

type scanPlanBinaryInt4ToInt64 struct{}

func (scanPlanBinaryInt4ToInt64) Scan(src []byte, dst any) error {
	p, ok := dst.(*int64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	*p = int64(int32(bits))

	return nil
}

type scanPlanBinaryInt4ToUint64 struct{}

func (scanPlanBinaryInt4ToUint64) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int32(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint64", n)
	}

	*p = uint64(n)

	return nil
}

type scanPlanBinaryInt4ToInt struct{}

func (scanPlanBinaryInt4ToInt) Scan(src []byte, dst any) error {
	p, ok := dst.(*int)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	*p = int(int32(bits))

	return nil
}

type scanPlanBinaryInt4ToUint struct{}

func (scanPlanBinaryInt4ToUint) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int64(int32(bits))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint", n)
	}

	*p = uint(n)

	return nil
}

type scanPlanBinaryInt4ToInt64Scanner struct{}

func (scanPlanBinaryInt4ToInt64Scanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(Int64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanInt64(Int8{})
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int64(int32(bits))

	return s.ScanInt64(Int8{Int64: n, Valid: true})
}

type scanPlanBinaryInt4ToString struct{}

func (scanPlanBinaryInt4ToString) Scan(src []byte, dst any) error {
	target, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int32(bits)
	*target = strconv.FormatInt(int64(n), 10)
	return nil
}

type scanPlanBinaryInt4ToTextScanner struct{}

func (scanPlanBinaryInt4ToTextScanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanText(Text{})
	}

	bits, err := fbio.Uint32BEExact(src)
	if err != nil {
		return fmt.Errorf("int4: %w", err)
	}

	n := int64(int32(bits))

	return s.ScanText(Text{String: strconv.FormatInt(n, 10), Valid: true})
}

// Int8 represents the Firebird BIGINT type.
//
// Valid is false when the value represents SQL NULL.
type Int8 struct {
	Int64 int64
	Valid bool
}

// ScanInt64 implements [Int64Scanner].
func (dst *Int8) ScanInt64(n Int8) error {
	if !n.Valid {
		*dst = Int8{}
		return nil
	}

	*dst = Int8{Int64: n.Int64, Valid: true}

	return nil
}

// Int64Value implements [Int64Valuer].
func (n Int8) Int64Value() (Int8, error) {
	return Int8{Int64: n.Int64, Valid: n.Valid}, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (dst *Int8) Scan(src any) error {
	if src == nil {
		*dst = Int8{}
		return nil
	}

	var n int64

	switch src := src.(type) {
	case int64:
		n = src
	case string:
		var err error
		n, err = strconv.ParseInt(src, 10, 64)
		if err != nil {
			*dst = Int8{}
			return err
		}
	case []byte:
		var err error
		n, err = strconv.ParseInt(string(src), 10, 64)
		if err != nil {
			*dst = Int8{}
			return err
		}
	default:
		*dst = Int8{}
		return fmt.Errorf("cannot scan %T", src)
	}

	*dst = Int8{Int64: n, Valid: true}

	return nil
}

// Value implements the [database/sql/driver.Valuer] interface.
func (src Int8) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}
	return src.Int64, nil
}

// MarshalJSON encodes src as a JSON number or null.
func (src Int8) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(src.Int64, 10)), nil
}

// UnmarshalJSON decodes a JSON number or null into dst.
func (dst *Int8) UnmarshalJSON(b []byte) error {
	var n *int64
	err := json.Unmarshal(b, &n)
	if err != nil {
		*dst = Int8{}
		return err
	}

	if n == nil {
		*dst = Int8{}
	} else {
		*dst = Int8{Int64: *n, Valid: true}
	}

	return nil
}

// Int8Codec encodes and decodes Firebird BIGINT values.
type Int8Codec struct{}

// PlanEncode returns an encode plan for supported Firebird BIGINT input values.
func (Int8Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case int64:
		return encodePlanInt8CodecBinaryInt64{}
	case Int64Valuer:
		return encodePlanInt8CodecBinaryInt64Valuer{}
	case string:
		return encodePlanInt8CodecBinaryFromString{}
	}

	return nil
}

type encodePlanInt8CodecBinaryInt64 struct{}

func (encodePlanInt8CodecBinaryInt64) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n := value.(int64)
	return fbio.AppendInt64BE(buf, n), nil
}

type encodePlanInt8CodecBinaryInt64Valuer struct{}

func (encodePlanInt8CodecBinaryInt64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	return fbio.AppendInt64BE(buf, n.Int64), nil
}

type encodePlanInt8CodecBinaryFromString struct{}

func (encodePlanInt8CodecBinaryFromString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q as bigint: %w", s, err)
	}
	return fbio.AppendInt64BE(buf, n), nil
}

// PlanScan returns a scan plan for supported Firebird BIGINT target values.
func (Int8Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *int8:
		return scanPlanBinaryInt8ToInt8{}
	case *int16:
		return scanPlanBinaryInt8ToInt16{}
	case *int32:
		return scanPlanBinaryInt8ToInt32{}
	case *int64:
		return scanPlanBinaryInt8ToInt64{}
	case *int:
		return scanPlanBinaryInt8ToInt{}
	case *uint8:
		return scanPlanBinaryInt8ToUint8{}
	case *uint16:
		return scanPlanBinaryInt8ToUint16{}
	case *uint32:
		return scanPlanBinaryInt8ToUint32{}
	case *uint64:
		return scanPlanBinaryInt8ToUint64{}
	case *uint:
		return scanPlanBinaryInt8ToUint{}
	case Int64Scanner:
		return scanPlanBinaryInt8ToInt64Scanner{}
	case *string:
		return scanPlanBinaryInt8ToString{}
	case TextScanner:
		return scanPlanBinaryInt8ToTextScanner{}
	}

	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c Int8Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var n int64
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// DecodeValue decodes src into the default Go value for Firebird BIGINT.
func (c Int8Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var n int64
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}

type scanPlanBinaryInt8ToInt8 struct{}

func (scanPlanBinaryInt8ToInt8) Scan(src []byte, dst any) error {
	p, ok := dst.(*int8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < math.MinInt8 {
		return fmt.Errorf("%d is less than minimum value for int8", n)
	} else if n > math.MaxInt8 {
		return fmt.Errorf("%d is greater than maximum value for int8", n)
	}

	*p = int8(n)

	return nil
}

type scanPlanBinaryInt8ToUint8 struct{}

func (scanPlanBinaryInt8ToUint8) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint8", n)
	}

	if n > math.MaxUint8 {
		return fmt.Errorf("%d is greater than maximum value for uint8", n)
	}

	*p = uint8(n)

	return nil
}

type scanPlanBinaryInt8ToInt16 struct{}

func (scanPlanBinaryInt8ToInt16) Scan(src []byte, dst any) error {
	p, ok := dst.(*int16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < math.MinInt16 {
		return fmt.Errorf("%d is less than minimum value for int16", n)
	} else if n > math.MaxInt16 {
		return fmt.Errorf("%d is greater than maximum value for int16", n)
	}

	*p = int16(n)

	return nil
}

type scanPlanBinaryInt8ToUint16 struct{}

func (scanPlanBinaryInt8ToUint16) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint16", n)
	}

	if n > math.MaxUint16 {
		return fmt.Errorf("%d is greater than maximum value for uint16", n)
	}

	*p = uint16(n)

	return nil
}

type scanPlanBinaryInt8ToInt32 struct{}

func (scanPlanBinaryInt8ToInt32) Scan(src []byte, dst any) error {
	p, ok := dst.(*int32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < math.MinInt32 {
		return fmt.Errorf("%d is less than minimum value for int32", n)
	} else if n > math.MaxInt32 {
		return fmt.Errorf("%d is greater than maximum value for int32", n)
	}

	*p = int32(n)

	return nil
}

type scanPlanBinaryInt8ToUint32 struct{}

func (scanPlanBinaryInt8ToUint32) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint32", n)
	}

	if n > math.MaxUint32 {
		return fmt.Errorf("%d is greater than maximum value for uint32", n)
	}

	*p = uint32(n)

	return nil
}

type scanPlanBinaryInt8ToInt64 struct{}

func (scanPlanBinaryInt8ToInt64) Scan(src []byte, dst any) error {
	p, ok := dst.(*int64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	*p = int64(bits)

	return nil
}

type scanPlanBinaryInt8ToUint64 struct{}

func (scanPlanBinaryInt8ToUint64) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint64", n)
	}

	*p = uint64(n)

	return nil
}

type scanPlanBinaryInt8ToInt struct{}

func (scanPlanBinaryInt8ToInt) Scan(src []byte, dst any) error {
	p, ok := dst.(*int)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < math.MinInt {
		return fmt.Errorf("%d is less than minimum value for int", n)
	} else if n > math.MaxInt {
		return fmt.Errorf("%d is greater than maximum value for int", n)
	}

	*p = int(n)

	return nil
}

type scanPlanBinaryInt8ToUint struct{}

func (scanPlanBinaryInt8ToUint) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint", n)
	}

	if uint64(n) > math.MaxUint {
		return fmt.Errorf("%d is greater than maximum value for uint", n)
	}

	*p = uint(n)

	return nil
}

type scanPlanBinaryInt8ToInt64Scanner struct{}

func (scanPlanBinaryInt8ToInt64Scanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(Int64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanInt64(Int8{})
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)

	return s.ScanInt64(Int8{Int64: n, Valid: true})
}

type scanPlanBinaryInt8ToString struct{}

func (scanPlanBinaryInt8ToString) Scan(src []byte, dst any) error {
	target, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)
	*target = strconv.FormatInt(n, 10)
	return nil
}

type scanPlanBinaryInt8ToTextScanner struct{}

func (scanPlanBinaryInt8ToTextScanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanText(Text{})
	}

	bits, err := fbio.Uint64BEExact(src)
	if err != nil {
		return fmt.Errorf("int8: %w", err)
	}

	n := int64(bits)

	return s.ScanText(Text{String: strconv.FormatInt(n, 10), Valid: true})
}
