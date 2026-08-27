package fbtype

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
)

var (
	int128TwoPow127 = new(big.Int).Lsh(big.NewInt(1), 127)
	int128TwoPow128 = new(big.Int).Lsh(big.NewInt(1), 128)

	int128Max = new(big.Int).Sub(new(big.Int).Set(int128TwoPow127), big.NewInt(1))
	int128Min = new(big.Int).Neg(new(big.Int).Set(int128TwoPow127))
)

// Int128Scanner scans a Firebird INT128 value.
type Int128Scanner interface {
	// ScanInt128 scans n into the receiver.
	ScanInt128(Int128) error
}

// Int128Valuer returns a Firebird INT128 value.
type Int128Valuer interface {
	// Int128Value returns the Firebird INT128 value to encode.
	Int128Value() (Int128, error)
}

// Int128 represents the Firebird INT128 type.
//
// Int contains the integer value. Valid is false when the value represents SQL
// NULL. Firebird INT128 supports the signed range -2^127 through 2^127-1.
type Int128 struct {
	Int   big.Int
	Valid bool
}

// ScanInt128 implements [Int128Scanner].
func (dst *Int128) ScanInt128(n Int128) error {
	if !n.Valid {
		*dst = Int128{}
		return nil
	}

	if err := checkInt128Range(&n.Int); err != nil {
		*dst = Int128{}
		return err
	}

	dst.Valid = true
	dst.Int.Set(&n.Int)

	return nil
}

// Int128Value implements [Int128Valuer].
func (n Int128) Int128Value() (Int128, error) {
	if !n.Valid {
		return Int128{}, nil
	}

	if err := checkInt128Range(&n.Int); err != nil {
		return Int128{}, err
	}

	var tmp big.Int
	tmp.Set(&n.Int)

	return Int128{Int: tmp, Valid: true}, nil
}

// Scan implements the [database/sql.Scanner] interface.
func (dst *Int128) Scan(src any) error {
	if src == nil {
		*dst = Int128{}
		return nil
	}

	switch v := src.(type) {
	case int64:
		dst.Valid = true
		dst.Int.SetInt64(v)
		return nil
	case string:
		n, err := ParseInt128(v)
		if err != nil {
			*dst = Int128{}
			return err
		}
		*dst = n
		return nil
	case []byte:
		n, err := ParseInt128(string(v))
		if err != nil {
			*dst = Int128{}
			return err
		}
		*dst = n
		return nil
	case big.Int:
		if err := checkInt128Range(&v); err != nil {
			*dst = Int128{}
			return err
		}
		dst.Valid = true
		dst.Int.Set(&v)
		return nil
	case *big.Int:
		if v == nil {
			*dst = Int128{}
			return nil
		}
		if err := checkInt128Range(v); err != nil {
			*dst = Int128{}
			return err
		}
		dst.Valid = true
		dst.Int.Set(v)
		return nil
	default:
		*dst = Int128{}
		return fmt.Errorf("cannot scan %T", src)
	}
}

// Value implements the [database/sql/driver.Valuer] interface.
func (src Int128) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}

	if err := checkInt128Range(&src.Int); err != nil {
		return nil, err
	}

	return src.Int.String(), nil
}

// MarshalJSON encodes src as a JSON number or null.
func (src Int128) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}

	if err := checkInt128Range(&src.Int); err != nil {
		return nil, err
	}

	return []byte(src.Int.String()), nil
}

// UnmarshalJSON decodes a JSON number, JSON string, or null into dst.
func (dst *Int128) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		*dst = Int128{}
		return nil
	}

	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			*dst = Int128{}
			return err
		}
		n, err := ParseInt128(s)
		if err != nil {
			*dst = Int128{}
			return err
		}
		*dst = n
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(b, &num); err != nil {
		*dst = Int128{}
		return err
	}

	n, err := ParseInt128(num.String())
	if err != nil {
		*dst = Int128{}
		return err
	}

	*dst = n

	return nil
}

// String returns the decimal representation of n, or an empty string for SQL NULL.
func (n Int128) String() string {
	if !n.Valid {
		return ""
	}

	return n.Int.String()
}

// Int128Codec encodes and decodes Firebird INT128 values.
type Int128Codec struct{}

// PlanEncode returns an encode plan for supported Firebird INT128 input values.
func (Int128Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case int8:
		return encodePlanInt128CodecBinaryInt8{}
	case int16:
		return encodePlanInt128CodecBinaryInt16{}
	case int32:
		return encodePlanInt128CodecBinaryInt32{}
	case int64:
		return encodePlanInt128CodecBinaryInt64{}
	case int:
		return encodePlanInt128CodecBinaryInt{}
	case uint8:
		return encodePlanInt128CodecBinaryUint8{}
	case uint16:
		return encodePlanInt128CodecBinaryUint16{}
	case uint32:
		return encodePlanInt128CodecBinaryUint32{}
	case uint64:
		return encodePlanInt128CodecBinaryUint64{}
	case uint:
		return encodePlanInt128CodecBinaryUint{}
	case string:
		return encodePlanInt128CodecBinaryString{}
	case big.Int:
		return encodePlanInt128CodecBinaryBigInt{}
	case *big.Int:
		return encodePlanInt128CodecBinaryBigIntPtr{}
	case Int128Valuer:
		return encodePlanInt128CodecBinaryValuer{}
	case NumericValuer:
		return encodePlanInt128CodecBinaryNumericValuer{}
	case Int64Valuer:
		return encodePlanInt128CodecBinaryInt64Valuer{}
	}
	return nil
}

type encodePlanInt128CodecBinaryInt64 struct{}

func (encodePlanInt128CodecBinaryInt64) Encode(value any, buf []byte) ([]byte, error) {
	v := value.(int64)

	start := len(buf)
	buf = append(buf, make([]byte, 16)...)

	var hi uint64
	if v < 0 {
		hi = ^uint64(0)
	}

	binary.BigEndian.PutUint64(buf[start:start+8], hi)
	binary.BigEndian.PutUint64(buf[start+8:start+16], uint64(v))

	return buf, nil
}

type encodePlanInt128CodecBinaryInt8 struct{}

func (encodePlanInt128CodecBinaryInt8) Encode(value any, buf []byte) ([]byte, error) {
	return encodePlanInt128CodecBinaryInt64{}.Encode(int64(value.(int8)), buf)
}

type encodePlanInt128CodecBinaryString struct{}

func (encodePlanInt128CodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)

	z := new(big.Int)
	_, ok := z.SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("cannot parse %q as int128", s)
	}

	if err := checkInt128Range(z); err != nil {
		return nil, err
	}

	return appendInt128BinaryFromBig(buf, z)
}

type encodePlanInt128CodecBinaryInt32 struct{}

func (encodePlanInt128CodecBinaryInt32) Encode(value any, buf []byte) ([]byte, error) {
	return encodePlanInt128CodecBinaryInt64{}.Encode(int64(value.(int32)), buf)
}

type encodePlanInt128CodecBinaryInt16 struct{}

func (encodePlanInt128CodecBinaryInt16) Encode(value any, buf []byte) ([]byte, error) {
	return encodePlanInt128CodecBinaryInt64{}.Encode(int64(value.(int16)), buf)
}

type encodePlanInt128CodecBinaryInt struct{}

func (encodePlanInt128CodecBinaryInt) Encode(value any, buf []byte) ([]byte, error) {
	return encodePlanInt128CodecBinaryInt64{}.Encode(int64(value.(int)), buf)
}

type encodePlanInt128CodecBinaryUint64 struct{}

func (encodePlanInt128CodecBinaryUint64) Encode(value any, buf []byte) ([]byte, error) {
	return appendInt128BinaryFromUint64(buf, value.(uint64)), nil
}

type encodePlanInt128CodecBinaryUint32 struct{}

func (encodePlanInt128CodecBinaryUint32) Encode(value any, buf []byte) ([]byte, error) {
	return appendInt128BinaryFromUint64(buf, uint64(value.(uint32))), nil
}

type encodePlanInt128CodecBinaryUint16 struct{}

func (encodePlanInt128CodecBinaryUint16) Encode(value any, buf []byte) ([]byte, error) {
	return appendInt128BinaryFromUint64(buf, uint64(value.(uint16))), nil
}

type encodePlanInt128CodecBinaryUint8 struct{}

func (encodePlanInt128CodecBinaryUint8) Encode(value any, buf []byte) ([]byte, error) {
	return appendInt128BinaryFromUint64(buf, uint64(value.(uint8))), nil
}

type encodePlanInt128CodecBinaryUint struct{}

func (encodePlanInt128CodecBinaryUint) Encode(value any, buf []byte) ([]byte, error) {
	return appendInt128BinaryFromUint64(buf, uint64(value.(uint))), nil
}

type encodePlanInt128CodecBinaryBigInt struct{}

func (encodePlanInt128CodecBinaryBigInt) Encode(value any, buf []byte) ([]byte, error) {
	v := value.(big.Int)
	return appendInt128BinaryFromBig(buf, &v)
}

type encodePlanInt128CodecBinaryBigIntPtr struct{}

func (encodePlanInt128CodecBinaryBigIntPtr) Encode(value any, buf []byte) ([]byte, error) {
	v := value.(*big.Int)
	if v == nil {
		return nil, nil
	}
	return appendInt128BinaryFromBig(buf, v)
}

type encodePlanInt128CodecBinaryValuer struct{}

func (encodePlanInt128CodecBinaryValuer) Encode(value any, buf []byte) ([]byte, error) {
	n, err := value.(Int128Valuer).Int128Value()
	if err != nil {
		return nil, err
	}
	if !n.Valid {
		return nil, nil
	}
	return appendInt128BinaryFromBig(buf, &n.Int)
}

type encodePlanInt128CodecBinaryInt64Valuer struct{}

func (encodePlanInt128CodecBinaryInt64Valuer) Encode(value any, buf []byte) ([]byte, error) {
	n, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}
	if !n.Valid {
		return nil, nil
	}
	return encodePlanInt128CodecBinaryInt64{}.Encode(n.Int64, buf)
}

type encodePlanInt128CodecBinaryNumericValuer struct{}

func (encodePlanInt128CodecBinaryNumericValuer) Encode(value any, buf []byte) ([]byte, error) {
	n, err := value.(NumericValuer).NumericValue()
	if err != nil {
		return nil, err
	}
	if !n.Valid {
		return nil, nil
	}

	integer, err := n.toTruncatedBigInt()
	if err != nil {
		return nil, err
	}

	return appendInt128BinaryFromBig(buf, integer)
}

// PlanScan returns a scan plan for supported Firebird INT128 target values.
func (Int128Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *int8:
		return scanPlanBinaryInt128ToInt8{}
	case *int16:
		return scanPlanBinaryInt128ToInt16{}
	case *int32:
		return scanPlanBinaryInt128ToInt32{}
	case *int64:
		return scanPlanBinaryInt128ToInt64{}
	case *int:
		return scanPlanBinaryInt128ToInt{}
	case *uint8:
		return scanPlanBinaryInt128ToUint8{}
	case *uint16:
		return scanPlanBinaryInt128ToUint16{}
	case *uint32:
		return scanPlanBinaryInt128ToUint32{}
	case *uint64:
		return scanPlanBinaryInt128ToUint64{}
	case *uint:
		return scanPlanBinaryInt128ToUint{}
	case *string:
		return scanPlanBinaryInt128ToString{}
	case *big.Int:
		return scanPlanBinaryInt128ToBigInt{}
	case Int128Scanner:
		return scanPlanBinaryInt128ToInt128Scanner{}
	case TextScanner:
		return scanPlanBinaryInt128ToTextScanner{}
	}

	return nil
}

type scanPlanBinaryInt128ToString struct{}

func (scanPlanBinaryInt128ToString) Scan(src []byte, dst any) error {
	p, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	var z big.Int
	if err := int128SetFromBinary(&z, src); err != nil {
		return err
	}

	*p = z.String()
	return nil
}

type scanPlanBinaryInt128ToBigInt struct{}

func (scanPlanBinaryInt128ToBigInt) Scan(src []byte, dst any) error {
	p, ok := dst.(*big.Int)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	return int128SetFromBinary(p, src)
}

type scanPlanBinaryInt128ToInt128Scanner struct{}

func (scanPlanBinaryInt128ToInt128Scanner) Scan(src []byte, dst any) error {
	s, ok := dst.(Int128Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanInt128(Int128{})
	}

	var z big.Int
	if err := int128SetFromBinary(&z, src); err != nil {
		return err
	}

	var v big.Int
	v.Set(&z)

	return s.ScanInt128(Int128{Int: v, Valid: true})
}

type scanPlanBinaryInt128ToTextScanner struct{}

func (scanPlanBinaryInt128ToTextScanner) Scan(src []byte, dst any) error {
	s, ok := dst.(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return s.ScanText(Text{})
	}

	var z big.Int
	if err := int128SetFromBinary(&z, src); err != nil {
		return err
	}

	return s.ScanText(Text{String: z.String(), Valid: true})
}

type scanPlanBinaryInt128ToInt64 struct{}

func (scanPlanBinaryInt128ToInt64) Scan(src []byte, dst any) error {
	p, ok := dst.(*int64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsInt64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("value is out of range for int64")
	}

	*p = n
	return nil
}

type scanPlanBinaryInt128ToInt32 struct{}

func (scanPlanBinaryInt128ToInt32) Scan(src []byte, dst any) error {
	p, ok := dst.(*int32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsInt64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n < math.MinInt32 || n > math.MaxInt32 {
		return fmt.Errorf("value is out of range for int32")
	}

	*p = int32(n)
	return nil
}

type scanPlanBinaryInt128ToInt16 struct{}

func (scanPlanBinaryInt128ToInt16) Scan(src []byte, dst any) error {
	p, ok := dst.(*int16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsInt64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n < math.MinInt16 || n > math.MaxInt16 {
		return fmt.Errorf("value is out of range for int16")
	}

	*p = int16(n)
	return nil
}

type scanPlanBinaryInt128ToInt8 struct{}

func (scanPlanBinaryInt128ToInt8) Scan(src []byte, dst any) error {
	p, ok := dst.(*int8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsInt64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n < math.MinInt8 || n > math.MaxInt8 {
		return fmt.Errorf("value is out of range for int8")
	}

	*p = int8(n)

	return nil
}

type scanPlanBinaryInt128ToInt struct{}

func (scanPlanBinaryInt128ToInt) Scan(src []byte, dst any) error {
	p, ok := dst.(*int)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsInt64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n < math.MinInt || n > math.MaxInt {
		return fmt.Errorf("value is out of range for int")
	}

	*p = int(n)

	return nil
}

type scanPlanBinaryInt128ToUint64 struct{}

func (scanPlanBinaryInt128ToUint64) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsUint64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("value is out of range for uint64")
	}
	*p = n

	return nil
}

type scanPlanBinaryInt128ToUint32 struct{}

func (scanPlanBinaryInt128ToUint32) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint32)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsUint64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n > math.MaxUint32 {
		return fmt.Errorf("value is out of range for uint32")
	}

	*p = uint32(n)

	return nil
}

type scanPlanBinaryInt128ToUint16 struct{}

func (scanPlanBinaryInt128ToUint16) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsUint64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n > math.MaxUint16 {
		return fmt.Errorf("value is out of range for uint16")
	}

	*p = uint16(n)

	return nil
}

type scanPlanBinaryInt128ToUint8 struct{}

func (scanPlanBinaryInt128ToUint8) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint8)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsUint64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n > math.MaxUint8 {
		return fmt.Errorf("value is out of range for uint8")
	}

	*p = uint8(n)

	return nil
}

type scanPlanBinaryInt128ToUint struct{}

func (scanPlanBinaryInt128ToUint) Scan(src []byte, dst any) error {
	p, ok := dst.(*uint)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	n, ok, err := int128FitsUint64FromBinary(src)
	if err != nil {
		return err
	}

	if !ok || n > math.MaxUint {
		return fmt.Errorf("value is out of range for uint")
	}

	*p = uint(n)

	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c Int128Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var z big.Int
	if err := int128SetFromBinary(&z, src); err != nil {
		return nil, err
	}
	return z.String(), nil
}

// DecodeValue decodes src into the default Go value for Firebird INT128.
func (c Int128Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var z big.Int
	if err := int128SetFromBinary(&z, src); err != nil {
		return nil, err
	}
	var v big.Int
	v.Set(&z)
	return Int128{Int: v, Valid: true}, nil
}

func checkInt128Range(z *big.Int) error {
	if z.Cmp(int128Min) < 0 || z.Cmp(int128Max) > 0 {
		return fmt.Errorf("value out of range for int128: %s", z.String())
	}

	return nil
}

// ParseInt128 parses s as a Firebird INT128 value.
func ParseInt128(s string) (Int128, error) {
	z, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return Int128{}, fmt.Errorf("invalid int128: %q", s)
	}

	if err := checkInt128Range(z); err != nil {
		return Int128{}, err
	}

	var v big.Int
	v.Set(z)

	return Int128{Int: v, Valid: true}, nil
}

func int128SetFromBinary(dst *big.Int, src []byte) error {
	if len(src) != 16 {
		return fmt.Errorf("invalid length for int128: %v", len(src))
	}

	dst.SetBytes(src)

	if src[0]&0x80 != 0 {
		dst.Sub(dst, int128TwoPow128)
	}

	return nil
}

func appendInt128BinaryFromBig(buf []byte, z *big.Int) ([]byte, error) {
	if err := checkInt128Range(z); err != nil {
		return nil, err
	}

	var u big.Int

	if z.Sign() < 0 {
		u.Add(z, int128TwoPow128)
	} else {
		u.Set(z)
	}

	ub := u.Bytes()
	if len(ub) > 16 {
		return nil, fmt.Errorf("value out of range for int128: %s", z.String())
	}

	start := len(buf)
	buf = append(buf, make([]byte, 16)...)
	copy(buf[start+16-len(ub):start+16], ub)

	return buf, nil
}

func appendInt128BinaryFromUint64(buf []byte, v uint64) []byte {
	start := len(buf)
	buf = append(buf, make([]byte, 16)...)
	binary.BigEndian.PutUint64(buf[start+8:start+16], v)

	return buf
}

func int128FitsInt64FromBinary(src []byte) (int64, bool, error) {
	if len(src) != 16 {
		return 0, false, fmt.Errorf("invalid length for int128: %v", len(src))
	}

	hiU := binary.BigEndian.Uint64(src[:8])
	loU := binary.BigEndian.Uint64(src[8:])

	signBit := loU >> 63

	if signBit == 0 {
		if hiU == 0 {
			return int64(loU), true, nil
		}
		return 0, false, nil
	}

	if hiU == ^uint64(0) {
		return int64(loU), true, nil
	}

	return 0, false, nil
}

func int128FitsUint64FromBinary(src []byte) (uint64, bool, error) {
	if len(src) != 16 {
		return 0, false, fmt.Errorf("invalid length for int128: %v", len(src))
	}

	hiU := binary.BigEndian.Uint64(src[:8])
	if hiU != 0 {
		return 0, false, nil
	}

	loU := binary.BigEndian.Uint64(src[8:])

	return loU, true, nil
}
