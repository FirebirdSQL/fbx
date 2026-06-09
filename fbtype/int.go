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

type Int64Scanner interface {
	ScanInt64(Int8) error
}

type Int64Valuer interface {
	Int64Value() (Int8, error)
}

type Int2 struct {
	Int16 int16
	Valid bool
}

// ScanInt64 implements the Int64Scanner interface.
func (dst *Int2) ScanInt64(n Int8) error {
	if !n.Valid {
		*dst = Int2{}
		return nil
	}

	if n.Int64 < math.MinInt16 {
		return fmt.Errorf("%d is less than minimum value for Int2", n.Int64)
	}
	if n.Int64 > math.MaxInt16 {
		return fmt.Errorf("%d is greater than maximum value for Int2", n.Int64)
	}
	*dst = Int2{Int16: int16(n.Int64), Valid: true}

	return nil
}

func (n Int2) Int64Value() (Int8, error) {
	return Int8{Int64: int64(n.Int16), Valid: n.Valid}, nil
}

// Scan implements the database/sql Scanner interface.
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
			return err
		}
	case []byte:
		var err error
		n, err = strconv.ParseInt(string(src), 10, 16)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot scan %T", src)
	}

	if n < math.MinInt16 {
		return fmt.Errorf("%d is less than minimum value for Int2", n)
	}
	if n > math.MaxInt16 {
		return fmt.Errorf("%d is greater than maximum value for Int2", n)
	}
	*dst = Int2{Int16: int16(n), Valid: true}

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (src Int2) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}
	return int64(src.Int16), nil
}

func (src Int2) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(src.Int16), 10)), nil
}

func (dst *Int2) UnmarshalJSON(b []byte) error {
	var n *int16
	err := json.Unmarshal(b, &n)
	if err != nil {
		return err
	}

	if n == nil {
		*dst = Int2{}
	} else {
		*dst = Int2{Int16: *n, Valid: true}
	}

	return nil
}

type Int2Codec struct{}

func (Int2Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case int16:
		return encodePlanInt2CodecBinaryInt16{}
	case Int64Valuer:
		return encodePlanInt2CodecBinaryInt64Valuer{}
	case string:
		return encodePlanInt2CodecBinaryFromString{}
	}
	return nil
}

type encodePlanInt2CodecBinaryInt16 struct{}

func (encodePlanInt2CodecBinaryInt16) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n := value.(int16)
	return fbio.AppendInt16BE(buf, int16(n)), nil
}

type encodePlanInt2CodecBinaryInt64Valuer struct{}

func (encodePlanInt2CodecBinaryInt64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	if n.Int64 > math.MaxInt16 {
		return nil, fmt.Errorf("%d is greater than maximum value for int2", n.Int64)
	}
	if n.Int64 < math.MinInt16 {
		return nil, fmt.Errorf("%d is less than minimum value for int2", n.Int64)
	}

	return fbio.AppendInt16BE(buf, int16(n.Int64)), nil
}

type encodePlanInt2CodecBinaryFromString struct{}

func (encodePlanInt2CodecBinaryFromString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)

	n, err := strconv.ParseInt(s, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q as smallint: %w", s, err)
	}

	return fbio.AppendInt16BE(buf, int16(n)), nil
}

func (Int2Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *int8:
		return scanPlanBinaryInt2ToInt8{}
	case *int16:
		return scanPlanBinaryInt2ToInt16{}
	case *int32:
		return scanPlanBinaryInt2ToInt32{}
	case *int64:
		return scanPlanBinaryInt2ToInt64{}
	case *int:
		return scanPlanBinaryInt2ToInt{}
	case *uint8:
		return scanPlanBinaryInt2ToUint8{}
	case *uint16:
		return scanPlanBinaryInt2ToUint16{}
	case *uint32:
		return scanPlanBinaryInt2ToUint32{}
	case *uint64:
		return scanPlanBinaryInt2ToUint64{}
	case *uint:
		return scanPlanBinaryInt2ToUint{}
	case Int64Scanner:
		return scanPlanBinaryInt2ToInt64Scanner{}
	case *string:
		return scanPlanBinaryInt2ToString{}
	case TextScanner:
		return scanPlanBinaryInt2ToTextScanner{}
	}

	return nil
}

func (c Int2Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
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

func (c Int2Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var n int16
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}

type scanPlanBinaryInt2ToInt8 struct{}

func (scanPlanBinaryInt2ToInt8) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	p, ok := (dst).(*int8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int16(binary.BigEndian.Uint16(src))
	if n < math.MinInt8 {
		return fmt.Errorf("%d is less than minimum value for int8", n)
	} else if n > math.MaxInt8 {
		return fmt.Errorf("%d is greater than maximum value for int8", n)
	}

	*p = int8(n)

	return nil
}

type scanPlanBinaryInt2ToUint8 struct{}

func (scanPlanBinaryInt2ToUint8) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for uint2: %v", len(src))
	}

	p, ok := (dst).(*uint8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int16(binary.BigEndian.Uint16(src))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint8", n)
	}

	if n > math.MaxUint8 {
		return fmt.Errorf("%d is greater than maximum value for uint8", n)
	}

	*p = uint8(n)

	return nil
}

type scanPlanBinaryInt2ToInt16 struct{}

func (scanPlanBinaryInt2ToInt16) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	p, ok := (dst).(*int16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int16(binary.BigEndian.Uint16(src))

	return nil
}

type scanPlanBinaryInt2ToUint16 struct{}

func (scanPlanBinaryInt2ToUint16) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for uint2: %v", len(src))
	}

	p, ok := (dst).(*uint16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int16(binary.BigEndian.Uint16(src))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint16", n)
	}

	*p = uint16(n)

	return nil
}

type scanPlanBinaryInt2ToInt32 struct{}

func (scanPlanBinaryInt2ToInt32) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	p, ok := (dst).(*int32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int32(int16(binary.BigEndian.Uint16(src)))

	return nil
}

type scanPlanBinaryInt2ToUint32 struct{}

func (scanPlanBinaryInt2ToUint32) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for uint2: %v", len(src))
	}

	p, ok := (dst).(*uint32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int16(binary.BigEndian.Uint16(src))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint32", n)
	}

	*p = uint32(n)

	return nil
}

type scanPlanBinaryInt2ToInt64 struct{}

func (scanPlanBinaryInt2ToInt64) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	p, ok := (dst).(*int64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int64(int16(binary.BigEndian.Uint16(src)))

	return nil
}

type scanPlanBinaryInt2ToUint64 struct{}

func (scanPlanBinaryInt2ToUint64) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for uint2: %v", len(src))
	}

	p, ok := (dst).(*uint64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int16(binary.BigEndian.Uint16(src))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint64", n)
	}

	*p = uint64(n)

	return nil
}

type scanPlanBinaryInt2ToInt struct{}

func (scanPlanBinaryInt2ToInt) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	p, ok := (dst).(*int)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int(int16(binary.BigEndian.Uint16(src)))

	return nil
}

type scanPlanBinaryInt2ToUint struct{}

func (scanPlanBinaryInt2ToUint) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for uint2: %v", len(src))
	}

	p, ok := (dst).(*uint)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(int16(binary.BigEndian.Uint16(src)))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint", n)
	}

	*p = uint(n)

	return nil
}

type scanPlanBinaryInt2ToInt64Scanner struct{}

func (scanPlanBinaryInt2ToInt64Scanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(Int64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanInt64(Int8{})
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	n := int64(int16(binary.BigEndian.Uint16(src)))

	return s.ScanInt64(Int8{Int64: n, Valid: true})
}

type scanPlanBinaryInt2ToString struct{}

func (scanPlanBinaryInt2ToString) Scan(src []byte, dst any) error {
	target := dst.(*string)

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	n := int16(binary.BigEndian.Uint16(src))
	*target = strconv.FormatInt(int64(n), 10)
	return nil
}

type scanPlanBinaryInt2ToTextScanner struct{}

func (scanPlanBinaryInt2ToTextScanner) Scan(src []byte, dst any) error {
	s, ok := (dst).(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanText(Text{})
	}

	if len(src) != 2 {
		return fmt.Errorf("invalid length for int2: %v", len(src))
	}

	n := int64(int16(binary.BigEndian.Uint16(src)))

	return s.ScanText(Text{String: strconv.FormatInt(n, 10), Valid: true})
}

type Int4 struct {
	Int32 int32
	Valid bool
}

// ScanInt64 implements the Int64Scanner interface.
func (dst *Int4) ScanInt64(n Int8) error {
	if !n.Valid {
		*dst = Int4{}
		return nil
	}

	if n.Int64 < math.MinInt32 {
		return fmt.Errorf("%d is less than minimum value for Int4", n.Int64)
	}
	if n.Int64 > math.MaxInt32 {
		return fmt.Errorf("%d is greater than maximum value for Int4", n.Int64)
	}
	*dst = Int4{Int32: int32(n.Int64), Valid: true}

	return nil
}

func (n Int4) Int64Value() (Int8, error) {
	return Int8{Int64: int64(n.Int32), Valid: n.Valid}, nil
}

// Scan implements the database/sql Scanner interface.
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
			return err
		}
	case []byte:
		var err error
		n, err = strconv.ParseInt(string(src), 10, 32)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot scan %T", src)
	}

	if n < math.MinInt32 {
		return fmt.Errorf("%d is less than minimum value for Int4", n)
	}
	if n > math.MaxInt32 {
		return fmt.Errorf("%d is greater than maximum value for Int4", n)
	}
	*dst = Int4{Int32: int32(n), Valid: true}

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (src Int4) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}
	return int64(src.Int32), nil
}

func (src Int4) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(src.Int32), 10)), nil
}

func (dst *Int4) UnmarshalJSON(b []byte) error {
	var n *int32
	err := json.Unmarshal(b, &n)
	if err != nil {
		return err
	}

	if n == nil {
		*dst = Int4{}
	} else {
		*dst = Int4{Int32: *n, Valid: true}
	}

	return nil
}

type Int4Codec struct{}

func (Int4Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case int32:
		return encodePlanInt4CodecBinaryInt32{}
	case Int64Valuer:
		return encodePlanInt4CodecBinaryInt64Valuer{}
	case string:
		return encodePlanInt4CodecBinaryFromString{}
	}
	return nil
}

type encodePlanInt4CodecBinaryInt32 struct{}

func (encodePlanInt4CodecBinaryInt32) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n := value.(int32)
	return fbio.AppendInt32BE(buf, int32(n)), nil
}

type encodePlanInt4CodecBinaryInt64Valuer struct{}

func (encodePlanInt4CodecBinaryInt64Valuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	n, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	if n.Int64 > math.MaxInt32 {
		return nil, fmt.Errorf("%d is greater than maximum value for int4", n.Int64)
	}
	if n.Int64 < math.MinInt32 {
		return nil, fmt.Errorf("%d is less than minimum value for int4", n.Int64)
	}

	return fbio.AppendInt32BE(buf, int32(n.Int64)), nil
}

type encodePlanInt4CodecBinaryFromString struct{}

func (encodePlanInt4CodecBinaryFromString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q as int: %w", s, err)
	}
	return fbio.AppendInt32BE(buf, int32(n)), nil
}

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

func (c Int4Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
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

func (c Int4Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	p, ok := (dst).(*int8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int32(binary.BigEndian.Uint32(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for uint4: %v", len(src))
	}

	p, ok := (dst).(*uint8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int32(binary.BigEndian.Uint32(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	p, ok := (dst).(*int16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int32(binary.BigEndian.Uint32(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for uint4: %v", len(src))
	}

	p, ok := (dst).(*uint16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int32(binary.BigEndian.Uint32(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	p, ok := (dst).(*int32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int32(binary.BigEndian.Uint32(src))

	return nil
}

type scanPlanBinaryInt4ToUint32 struct{}

func (scanPlanBinaryInt4ToUint32) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for uint4: %v", len(src))
	}

	p, ok := (dst).(*uint32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int32(binary.BigEndian.Uint32(src))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint32", n)
	}

	*p = uint32(n)

	return nil
}

type scanPlanBinaryInt4ToInt64 struct{}

func (scanPlanBinaryInt4ToInt64) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	p, ok := (dst).(*int64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int64(int32(binary.BigEndian.Uint32(src)))

	return nil
}

type scanPlanBinaryInt4ToUint64 struct{}

func (scanPlanBinaryInt4ToUint64) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for uint4: %v", len(src))
	}

	p, ok := (dst).(*uint64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int32(binary.BigEndian.Uint32(src))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint64", n)
	}

	*p = uint64(n)

	return nil
}

type scanPlanBinaryInt4ToInt struct{}

func (scanPlanBinaryInt4ToInt) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	p, ok := (dst).(*int)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int(int32(binary.BigEndian.Uint32(src)))

	return nil
}

type scanPlanBinaryInt4ToUint struct{}

func (scanPlanBinaryInt4ToUint) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for uint4: %v", len(src))
	}

	p, ok := (dst).(*uint)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(int32(binary.BigEndian.Uint32(src)))
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

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	n := int64(int32(binary.BigEndian.Uint32(src)))

	return s.ScanInt64(Int8{Int64: n, Valid: true})
}

type scanPlanBinaryInt4ToString struct{}

func (scanPlanBinaryInt4ToString) Scan(src []byte, dst any) error {
	target := dst.(*string)

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	n := int32(binary.BigEndian.Uint32(src))
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

	if len(src) != 4 {
		return fmt.Errorf("invalid length for int4: %v", len(src))
	}

	n := int64(int32(binary.BigEndian.Uint32(src)))

	return s.ScanText(Text{String: strconv.FormatInt(n, 10), Valid: true})
}

type Int8 struct {
	Int64 int64
	Valid bool
}

// ScanInt64 implements the Int64Scanner interface.
func (dst *Int8) ScanInt64(n Int8) error {
	if !n.Valid {
		*dst = Int8{}
		return nil
	}

	if n.Int64 < math.MinInt64 {
		return fmt.Errorf("%d is less than minimum value for Int8", n.Int64)
	}
	if n.Int64 > math.MaxInt64 {
		return fmt.Errorf("%d is greater than maximum value for Int8", n.Int64)
	}
	*dst = Int8{Int64: int64(n.Int64), Valid: true}

	return nil
}

func (n Int8) Int64Value() (Int8, error) {
	return Int8{Int64: int64(n.Int64), Valid: n.Valid}, nil
}

// Scan implements the database/sql Scanner interface.
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
			return err
		}
	case []byte:
		var err error
		n, err = strconv.ParseInt(string(src), 10, 64)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot scan %T", src)
	}

	if n < math.MinInt64 {
		return fmt.Errorf("%d is greater than maximum value for Int8", n)
	}
	if n > math.MaxInt64 {
		return fmt.Errorf("%d is greater than maximum value for Int8", n)
	}
	*dst = Int8{Int64: int64(n), Valid: true}

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (src Int8) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}
	return int64(src.Int64), nil
}

func (src Int8) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(src.Int64), 10)), nil
}

func (dst *Int8) UnmarshalJSON(b []byte) error {
	var n *int64
	err := json.Unmarshal(b, &n)
	if err != nil {
		return err
	}

	if n == nil {
		*dst = Int8{}
	} else {
		*dst = Int8{Int64: *n, Valid: true}
	}

	return nil
}

type Int8Codec struct{}

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
	return fbio.AppendInt64BE(buf, int64(n)), nil
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

	if n.Int64 > math.MaxInt64 {
		return nil, fmt.Errorf("%d is greater than maximum value for int8", n.Int64)
	}
	if n.Int64 < math.MinInt64 {
		return nil, fmt.Errorf("%d is less than minimum value for int8", n.Int64)
	}

	return fbio.AppendInt64BE(buf, int64(n.Int64)), nil
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	p, ok := (dst).(*int8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for uint8: %v", len(src))
	}

	p, ok := (dst).(*uint8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	p, ok := (dst).(*int16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for uint8: %v", len(src))
	}

	p, ok := (dst).(*uint16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	p, ok := (dst).(*int32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for uint8: %v", len(src))
	}

	p, ok := (dst).(*uint32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	p, ok := (dst).(*int64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	*p = int64(binary.BigEndian.Uint64(src))

	return nil
}

type scanPlanBinaryInt8ToUint64 struct{}

func (scanPlanBinaryInt8ToUint64) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for uint8: %v", len(src))
	}

	p, ok := (dst).(*uint64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
	if n < 0 {
		return fmt.Errorf("%d is less than minimum value for uint64", n)
	}

	*p = uint64(n)

	return nil
}

type scanPlanBinaryInt8ToInt struct{}

func (scanPlanBinaryInt8ToInt) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	p, ok := (dst).(*int)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(binary.BigEndian.Uint64(src))
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
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for uint8: %v", len(src))
	}

	p, ok := (dst).(*uint)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	n := int64(int64(binary.BigEndian.Uint64(src)))
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

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	n := int64(int64(binary.BigEndian.Uint64(src)))

	return s.ScanInt64(Int8{Int64: n, Valid: true})
}

type scanPlanBinaryInt8ToString struct{}

func (scanPlanBinaryInt8ToString) Scan(src []byte, dst any) error {
	target := dst.(*string)

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	n := int64(binary.BigEndian.Uint64(src))
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

	if len(src) != 8 {
		return fmt.Errorf("invalid length for int8: %v", len(src))
	}

	n := int64(int64(binary.BigEndian.Uint64(src)))

	return s.ScanText(Text{String: strconv.FormatInt(n, 10), Valid: true})
}
