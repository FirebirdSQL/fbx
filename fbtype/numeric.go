package fbtype

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/internal/fbio"
)

var big10 *big.Int = big.NewInt(10)

const maxNumericScale int32 = 38

// NumericScanner scans a Firebird NUMERIC or DECIMAL value.
type NumericScanner interface {
	// ScanNumeric scans v into the receiver.
	ScanNumeric(v Numeric) error
}

// NumericValuer returns a Firebird NUMERIC or DECIMAL value.
type NumericValuer interface {
	// NumericValue returns the Firebird NUMERIC or DECIMAL value to encode.
	NumericValue() (Numeric, error)
}

// Numeric represents a Firebird NUMERIC or DECIMAL value.
//
// Int contains the unscaled integer value and Exp contains the decimal exponent.
// For example, Int=1234 and Exp=-2 represents 12.34. Valid is false when the
// value represents SQL NULL. A valid Numeric with nil Int is treated as zero.
type Numeric struct {
	Int   *big.Int
	Exp   int32
	Valid bool
}

// ScanNumeric implements [NumericScanner].
func (n *Numeric) ScanNumeric(v Numeric) error {
	if err := validateNumericRepresentable(v); err != nil {
		*n = Numeric{}
		return err
	}

	*n = cloneNumeric(v)
	return nil
}

// NumericValue implements [NumericValuer].
func (n Numeric) NumericValue() (Numeric, error) {
	if err := validateNumericRepresentable(n); err != nil {
		return Numeric{}, err
	}

	return cloneNumeric(n), nil
}

// Float64Value implements [Float64Valuer].
func (n Numeric) Float64Value() (Float8, error) {
	if !n.Valid {
		return Float8{}, nil
	}
	if err := validateNumericRepresentable(n); err != nil {
		return Float8{}, err
	}

	buf := make([]byte, 0, 32)

	if n.Int == nil {
		buf = append(buf, '0')
	} else {
		buf = append(buf, n.Int.String()...)
	}
	buf = append(buf, 'e')
	buf = append(buf, strconv.FormatInt(int64(n.Exp), 10)...)

	f, err := strconv.ParseFloat(string(buf), 64)
	if err != nil {
		return Float8{}, err
	}

	return Float8{Float64: f, Valid: true}, nil
}

// ScanInt64 implements [Int64Scanner].
func (n *Numeric) ScanInt64(v Int8) error {
	if !v.Valid {
		*n = Numeric{}
		return nil
	}

	*n = Numeric{Int: big.NewInt(v.Int64), Valid: true}
	return nil
}

// Int64Value implements [Int64Valuer]. It discards any fractional part toward zero.
func (n Numeric) Int64Value() (Int8, error) {
	if !n.Valid {
		return Int8{}, nil
	}

	bi, err := n.toTruncatedBigInt()
	if err != nil {
		return Int8{}, err
	}

	if !bi.IsInt64() {
		return Int8{}, fmt.Errorf("cannot convert %v to int64", n)
	}

	return Int8{Int64: bi.Int64(), Valid: true}, nil
}

// String returns the decimal representation of n, an empty string for SQL NULL,
// or a diagnostic string when n cannot represent a Firebird NUMERIC value.
func (n Numeric) String() string {
	if !n.Valid {
		return ""
	}

	buf, err := encodeNumericText(n, nil)
	if err != nil {
		return fmt.Sprintf("<invalid Numeric: %v>", err)
	}

	return string(buf)
}

// ScanScientific parses src as a decimal number, including scientific notation.
func (n *Numeric) ScanScientific(src string) error {
	parsed, err := parseNumericText(src)
	if err != nil {
		*n = Numeric{}
		return err
	}
	*n = parsed
	return nil
}

func (n *Numeric) toBigInt() (*big.Int, error) {
	if err := validateNumericRepresentable(*n); err != nil {
		return nil, err
	}

	return rescaleNumeric(*n, 0, numericExact)
}

func (n *Numeric) toTruncatedBigInt() (*big.Int, error) {
	if err := validateNumericRepresentable(*n); err != nil {
		return nil, err
	}

	return rescaleNumeric(*n, 0, numericTruncate)
}

func parseDecimalMantissa(str string) (n *big.Int, exp int32, err error) {
	idx := strings.IndexByte(str, '.')

	if idx == -1 {
		for len(str) > 1 && str[len(str)-1] == '0' && str[len(str)-2] != '-' && str[len(str)-2] != '+' {
			str = str[:len(str)-1]
			exp++
		}
	} else {
		exp = int32(-(len(str) - idx - 1))
		str = str[:idx] + str[idx+1:]
	}

	accum := &big.Int{}
	if _, ok := accum.SetString(str, 10); !ok {
		return nil, 0, fmt.Errorf("%s is not a number", str)
	}
	if idx == -1 && accum.Sign() == 0 {
		exp = 0
	}

	return accum, exp, nil
}

// Scan implements the database/sql Scanner interface.
func (n *Numeric) Scan(src any) error {
	if src == nil {
		*n = Numeric{}
		return nil
	}

	switch src := src.(type) {
	case Numeric:
		return n.ScanNumeric(src)
	case string:
		parsed, err := parseNumericText(src)
		if err != nil {
			*n = Numeric{}
			return err
		}
		*n = parsed
		return nil
	case []byte:
		parsed, err := parseNumericText(string(src))
		if err != nil {
			*n = Numeric{}
			return err
		}
		*n = parsed
		return nil
	case int64:
		*n = Numeric{Int: big.NewInt(src), Valid: true}
		return nil
	case float64:
		parsed, err := float64ToNumeric(src)
		if err == nil {
			err = validateNumericRepresentable(parsed)
		}
		if err != nil {
			*n = Numeric{}
			return err
		}
		*n = parsed
		return nil
	case big.Int:
		var z big.Int
		z.Set(&src)
		parsed := Numeric{Int: &z, Valid: true}
		if err := validateNumericRepresentable(parsed); err != nil {
			*n = Numeric{}
			return err
		}
		*n = parsed
		return nil
	case *big.Int:
		if src == nil {
			*n = Numeric{}
			return nil
		}
		var z big.Int
		z.Set(src)
		parsed := Numeric{Int: &z, Valid: true}
		if err := validateNumericRepresentable(parsed); err != nil {
			*n = Numeric{}
			return err
		}
		*n = parsed
		return nil
	}

	*n = Numeric{}
	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (n Numeric) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	buf, err := encodeNumericText(n, nil)
	if err != nil {
		return nil, err
	}
	return string(buf), nil
}

// MarshalJSON encodes n as a JSON number or null.
func (n Numeric) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}

	return encodeNumericText(n, nil)
}

// UnmarshalJSON decodes a JSON number, JSON string, or null into n.
func (n *Numeric) UnmarshalJSON(src []byte) error {
	src = bytes.TrimSpace(src)
	if bytes.Equal(src, []byte(`null`)) {
		*n = Numeric{}
		return nil
	}

	var numericText string
	if len(src) > 0 && src[0] == '"' {
		if err := json.Unmarshal(src, &numericText); err != nil {
			*n = Numeric{}
			return err
		}
	} else {
		var num json.Number
		if err := json.Unmarshal(src, &num); err != nil {
			*n = Numeric{}
			return err
		}
		numericText = num.String()
	}

	parsed, err := parseNumericText(numericText)
	if err != nil {
		*n = Numeric{}
		return err
	}
	*n = parsed

	return nil
}

func (n Numeric) numberTextBytes() []byte {
	if n.Int == nil {
		return []byte("0")
	}
	if n.Int.Sign() == 0 && n.Exp >= 0 {
		return []byte("0")
	}

	intStr := n.Int.String()

	buf := &bytes.Buffer{}

	if len(intStr) > 0 && intStr[:1] == "-" {
		intStr = intStr[1:]
		buf.WriteByte('-')
	}

	exp := int64(n.Exp)
	if exp > 0 {
		buf.WriteString(intStr)
		for i := int64(0); i < exp; i++ {
			buf.WriteByte('0')
		}
	} else if exp < 0 {
		if int64(len(intStr)) <= -exp {
			buf.WriteString("0.")
			leadingZeros := -exp - int64(len(intStr))
			for i := int64(0); i < leadingZeros; i++ {
				buf.WriteByte('0')
			}
			buf.WriteString(intStr)
		} else if int64(len(intStr)) > -exp {
			dpPos := int64(len(intStr)) + exp
			buf.WriteString(intStr[:dpPos])
			buf.WriteByte('.')
			buf.WriteString(intStr[dpPos:])
		}
	} else {
		buf.WriteString(intStr)
	}

	return buf.Bytes()
}

// NumericCodec encodes and decodes Firebird NUMERIC and DECIMAL values.
type NumericCodec struct{}

func validateNumericTypeMetadata(typeMetadata TypeMetadata) error {
	var expectedLength uint32
	switch typeMetadata.TypeID {
	case fbproto.SQL_TYPE_SHORT:
		expectedLength = 2
	case fbproto.SQL_TYPE_LONG:
		expectedLength = 4
	case fbproto.SQL_TYPE_INT64:
		expectedLength = 8
	case fbproto.SQL_TYPE_INT128:
		expectedLength = 16
	default:
		return fmt.Errorf("unsupported numeric type ID: %d", typeMetadata.TypeID)
	}

	if typeMetadata.Length != expectedLength {
		return fmt.Errorf(
			"invalid storage length %d for numeric type ID %d: expected %d",
			typeMetadata.Length, typeMetadata.TypeID, expectedLength,
		)
	}

	return nil
}

// PlanEncode returns an encode plan for supported Firebird NUMERIC and DECIMAL input values.
func (NumericCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case NumericValuer:
		return encodePlanNumericCodecBinaryNumericValuer{typeMetadata: typeMetadata}
	case Float64Valuer:
		return encodePlanNumericCodecBinaryFloat64Valuer{typeMetadata: typeMetadata}
	case Int64Valuer:
		return encodePlanNumericCodecBinaryInt64Valuer{typeMetadata: typeMetadata}
	case string:
		return encodePlanNumericCodecTextString{typeMetadata: typeMetadata}
	}

	return nil
}

type encodePlanNumericCodecBinaryNumericValuer struct {
	typeMetadata TypeMetadata
}

func (p encodePlanNumericCodecBinaryNumericValuer) Encode(value any, buf []byte) ([]byte, error) {
	if err := validateNumericTypeMetadata(p.typeMetadata); err != nil {
		return nil, err
	}

	n, err := value.(NumericValuer).NumericValue()
	if err != nil {
		return nil, err
	}

	return EncodeNumericBinaryFB(buf, n, p.typeMetadata.Scale, p.typeMetadata.Length)
}

type encodePlanNumericCodecBinaryFloat64Valuer struct {
	typeMetadata TypeMetadata
}

func (p encodePlanNumericCodecBinaryFloat64Valuer) Encode(value any, buf []byte) ([]byte, error) {
	if err := validateNumericTypeMetadata(p.typeMetadata); err != nil {
		return nil, err
	}

	v, err := value.(Float64Valuer).Float64Value()
	if err != nil {
		return nil, err
	}
	if !v.Valid {
		return nil, nil
	}

	n, err := float64ToNumeric(v.Float64)
	if err != nil {
		return nil, err
	}
	n.Valid = true

	return encodeNumericBinaryFBRoundHalfAwayFromZero(buf, n, p.typeMetadata.Scale, p.typeMetadata.Length)
}

type encodePlanNumericCodecBinaryInt64Valuer struct {
	typeMetadata TypeMetadata
}

func (p encodePlanNumericCodecBinaryInt64Valuer) Encode(value any, buf []byte) ([]byte, error) {
	if err := validateNumericTypeMetadata(p.typeMetadata); err != nil {
		return nil, err
	}

	v, err := value.(Int64Valuer).Int64Value()
	if err != nil {
		return nil, err
	}
	if !v.Valid {
		return nil, nil
	}

	n := Numeric{
		Int:   big.NewInt(v.Int64),
		Exp:   0,
		Valid: true,
	}

	return EncodeNumericBinaryFB(buf, n, p.typeMetadata.Scale, p.typeMetadata.Length)
}

type encodePlanNumericCodecTextString struct {
	typeMetadata TypeMetadata
}

func (p encodePlanNumericCodecTextString) Encode(value any, buf []byte) ([]byte, error) {
	if err := validateNumericTypeMetadata(p.typeMetadata); err != nil {
		return nil, err
	}

	s := value.(string)

	var n Numeric
	if err := n.ScanScientific(s); err != nil {
		return nil, err
	}
	n.Valid = true

	return EncodeNumericBinaryFB(buf, n, p.typeMetadata.Scale, p.typeMetadata.Length)
}

func encodeNumericText(n Numeric, buf []byte) (newBuf []byte, err error) {
	if !n.Valid {
		return nil, nil
	}
	if err := validateNumericRepresentable(n); err != nil {
		return nil, err
	}

	buf = append(buf, n.numberTextBytes()...)

	return buf, nil
}

// PlanScan returns a scan plan for supported Firebird NUMERIC and DECIMAL target values.
func (NumericCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *Numeric:
		return scanPlanBinaryNumericToNumeric{typeMetadata: typeMetadata}
	case NumericScanner:
		return scanPlanBinaryNumericToNumericScanner{typeMetadata: typeMetadata}
	case Float64Scanner:
		return scanPlanBinaryNumericToFloat64Scanner{typeMetadata: typeMetadata}
	case Int64Scanner:
		return scanPlanBinaryNumericToInt64Scanner{typeMetadata: typeMetadata}
	case TextScanner:
		return scanPlanBinaryNumericToTextScanner{typeMetadata: typeMetadata}
	case *string:
		return scanPlanBinaryNumericToString{typeMetadata: typeMetadata}
	case *float64:
		return scanPlanBinaryNumericToFloat64Ptr{typeMetadata: typeMetadata}
	case *int64:
		return scanPlanBinaryNumericToInt64Ptr{typeMetadata: typeMetadata}
	}

	return nil
}

type scanPlanBinaryNumericToFloat64Ptr struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToFloat64Ptr) Scan(src []byte, dst any) error {
	fPtr, ok := dst.(*float64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		return err
	}

	f8, err := n.Float64Value()
	if err != nil {
		return err
	}

	*fPtr = f8.Float64
	return nil
}

type scanPlanBinaryNumericToInt64Ptr struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToInt64Ptr) Scan(src []byte, dst any) error {
	iPtr, ok := dst.(*int64)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		return err
	}

	bigInt, err := n.toBigInt()
	if err != nil {
		return err
	}

	if !bigInt.IsInt64() {
		return fmt.Errorf("%v is out of range for int64", bigInt)
	}

	*iPtr = bigInt.Int64()
	return nil
}

type scanPlanBinaryNumericToNumeric struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToNumeric) Scan(src []byte, dst any) error {
	nPtr, ok := dst.(*Numeric)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		*nPtr = Numeric{}
		return nil
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		*nPtr = Numeric{}
		return err
	}

	*nPtr = n
	return nil
}

type scanPlanBinaryNumericToNumericScanner struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToNumericScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(NumericScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanNumeric(Numeric{})
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		return err
	}

	return scanner.ScanNumeric(n)
}

type scanPlanBinaryNumericToFloat64Scanner struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToFloat64Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Float64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanFloat64(Float8{})
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		return err
	}

	f8, err := n.Float64Value()
	if err != nil {
		return err
	}

	return scanner.ScanFloat64(f8)
}

type scanPlanBinaryNumericToInt64Scanner struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToInt64Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Int64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanInt64(Int8{})
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		return err
	}

	bigInt, err := n.toBigInt()
	if err != nil {
		return err
	}

	if !bigInt.IsInt64() {
		return fmt.Errorf("%v is out of range for int64", bigInt)
	}

	return scanner.ScanInt64(Int8{Int64: bigInt.Int64(), Valid: true})
}

type scanPlanBinaryNumericToTextScanner struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToTextScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanText(Text{})
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		return err
	}

	sbuf, err := encodeNumericText(n, nil)
	if err != nil {
		return err
	}

	return scanner.ScanText(Text{String: string(sbuf), Valid: true})
}

type scanPlanBinaryNumericToString struct {
	typeMetadata TypeMetadata
}

func (p scanPlanBinaryNumericToString) Scan(src []byte, dst any) error {
	s, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	var n Numeric
	if err := decodeNumericBinaryFBForScan(&n, src, p.typeMetadata); err != nil {
		return err
	}

	*s = n.String()
	return nil
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c NumericCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var n Numeric
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	buf, err := encodeNumericText(n, nil)
	if err != nil {
		return nil, err
	}

	return string(buf), nil
}

// DecodeValue decodes src into the default Go value for Firebird NUMERIC and DECIMAL.
func (c NumericCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var n Numeric
	err := codecScan(c, m, typeMetadata, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// DecodeNumericBinaryFB decodes a Firebird binary NUMERIC or DECIMAL value into dst.
func DecodeNumericBinaryFB(dst *Numeric, src []byte, scale int32) error {
	if src == nil {
		*dst = Numeric{}
		return nil
	}
	if err := validateNumericScale(scale); err != nil {
		*dst = Numeric{}
		return err
	}

	var bigInt *big.Int
	switch len(src) {
	case 4:
		bits, err := fbio.Uint32BEExact(src)
		if err != nil {
			*dst = Numeric{}
			return fmt.Errorf("numeric: %w", err)
		}
		bigInt = big.NewInt(int64(int32(bits)))
	case 8:
		bits, err := fbio.Uint64BEExact(src)
		if err != nil {
			*dst = Numeric{}
			return fmt.Errorf("numeric: %w", err)
		}
		bigInt = big.NewInt(int64(bits))
	case 16:
		bigInt = new(big.Int)
		if err := int128SetFromBinary(bigInt, src); err != nil {
			*dst = Numeric{}
			return err
		}
	default:
		*dst = Numeric{}
		return fmt.Errorf("unsupported numeric length: %d", len(src))
	}

	*dst = Numeric{
		Int:   bigInt,
		Exp:   scale,
		Valid: true,
	}
	return nil
}

func decodeNumericBinaryFBForScan(dst *Numeric, src []byte, typeMetadata TypeMetadata) error {
	if err := validateNumericTypeMetadata(typeMetadata); err != nil {
		*dst = Numeric{}
		return err
	}

	storageLength := typeMetadata.Length
	expectedWireLength := int(storageLength)
	switch storageLength {
	case 2:
		expectedWireLength = 4
	case 4, 8, 16:
	default:
		*dst = Numeric{}
		return fmt.Errorf("unsupported numeric storage length: %d", storageLength)
	}

	if len(src) != expectedWireLength {
		*dst = Numeric{}
		return fmt.Errorf(
			"invalid wire length %d for numeric storage length %d: expected %d",
			len(src), storageLength, expectedWireLength,
		)
	}

	if err := DecodeNumericBinaryFB(dst, src, typeMetadata.Scale); err != nil {
		return err
	}

	if storageLength == 2 {
		coefficient := dst.Int.Int64()
		if coefficient < math.MinInt16 || coefficient > math.MaxInt16 {
			*dst = Numeric{}
			return fmt.Errorf("numeric coefficient %d is out of range for SMALLINT storage", coefficient)
		}
	}

	return nil
}

// EncodeNumericBinaryFB appends the Firebird binary encoding of n to buf.
func EncodeNumericBinaryFB(buf []byte, n Numeric, scale int32, length uint32) ([]byte, error) {
	if !n.Valid {
		return nil, nil
	}
	if err := validateNumericRepresentable(n); err != nil {
		return nil, err
	}

	scaled, err := rescaleNumeric(n, scale, numericExact)
	if err != nil {
		return nil, err
	}

	return appendScaledNumeric(buf, scaled, length)
}

func float64ToNumeric(f float64) (Numeric, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Numeric{}, fmt.Errorf("cannot convert non-finite float %v to numeric", f)
	}

	if f == 0 {
		return Numeric{
			Int:   big.NewInt(0),
			Exp:   0,
			Valid: true,
		}, nil
	}

	s := strconv.FormatFloat(f, 'f', -1, 64)

	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		intStr := parts[0] + parts[1]
		scale := len(parts[1])

		i := new(big.Int)
		_, ok := i.SetString(intStr, 10)
		if !ok {
			return Numeric{}, fmt.Errorf("cannot convert float %f to numeric", f)
		}

		return Numeric{
			Int:   i,
			Exp:   int32(-scale),
			Valid: true,
		}, nil
	}

	i := new(big.Int)
	_, ok := i.SetString(s, 10)
	if !ok {
		return Numeric{}, fmt.Errorf("cannot convert float %f to numeric", f)
	}

	return Numeric{
		Int:   i,
		Exp:   0,
		Valid: true,
	}, nil
}

// EncodeNumericBinaryFBRoundHalfAwayFromZero appends the Firebird binary
// encoding of n to buf, rounding to scale when exact encoding is impossible.
func EncodeNumericBinaryFBRoundHalfAwayFromZero(buf []byte, n Numeric, scale int32, length uint32) ([]byte, error) {
	if !n.Valid {
		return nil, nil
	}
	if err := validateNumericRepresentable(n); err != nil {
		return nil, err
	}

	return encodeNumericBinaryFBRoundHalfAwayFromZero(buf, n, scale, length)
}

func encodeNumericBinaryFBRoundHalfAwayFromZero(buf []byte, n Numeric, scale int32, length uint32) ([]byte, error) {
	if !n.Valid {
		return nil, nil
	}

	scaled, err := rescaleNumeric(n, scale, numericRoundHalfAwayFromZero)
	if err != nil {
		return nil, err
	}

	return appendScaledNumeric(buf, scaled, length)
}

type numericRoundingMode uint8

const (
	numericExact numericRoundingMode = iota
	numericTruncate
	numericRoundHalfAwayFromZero
)

func rescaleNumeric(n Numeric, scale int32, mode numericRoundingMode) (*big.Int, error) {
	if err := validateNumericScale(scale); err != nil {
		return nil, err
	}

	scaled := new(big.Int)
	if n.Int == nil || n.Int.Sign() == 0 {
		return scaled, nil
	}
	scaled.Set(n.Int)

	expDiff := int64(n.Exp) - int64(scale)
	if expDiff > 0 {
		if expDiff > int64(maxNumericScale) {
			return nil, fmt.Errorf("numeric value is out of range at scale %d", scale)
		}

		pow := new(big.Int).Exp(big10, big.NewInt(expDiff), nil)
		return scaled.Mul(scaled, pow), nil
	}
	if expDiff == 0 {
		return scaled, nil
	}

	divisorExp := -expDiff
	if divisorExp > int64(decimalDigitCount(scaled)) {
		if mode == numericExact {
			return nil, fmt.Errorf("numeric value cannot be represented exactly with scale %d", scale)
		}
		return new(big.Int), nil
	}

	pow := new(big.Int).Exp(big10, big.NewInt(divisorExp), nil)
	q := new(big.Int)
	rem := new(big.Int)
	q.QuoRem(scaled, pow, rem)
	if rem.Sign() == 0 {
		return q, nil
	}
	if mode == numericExact {
		return nil, fmt.Errorf("numeric value cannot be represented exactly with scale %d", scale)
	}
	if mode == numericTruncate {
		return q, nil
	}

	twiceRemainder := new(big.Int).Lsh(new(big.Int).Abs(rem), 1)
	if twiceRemainder.Cmp(pow) >= 0 {
		if scaled.Sign() > 0 {
			q.Add(q, big.NewInt(1))
		} else {
			q.Sub(q, big.NewInt(1))
		}
	}
	return q, nil
}

func appendScaledNumeric(buf []byte, scaled *big.Int, length uint32) ([]byte, error) {
	if scaled.IsInt64() {
		return appendFBScaledIntByLength(buf, scaled.Int64(), length)
	}

	if length != 16 {
		return nil, fmt.Errorf("numeric value is out of range for length %d", length)
	}
	if scaled.Cmp(int128Min) < 0 || scaled.Cmp(int128Max) > 0 {
		return nil, fmt.Errorf("numeric value is out of range for int128")
	}

	return appendInt128BinaryFromBig(buf, scaled)
}

func appendFBScaledIntByLength(buf []byte, val int64, length uint32) ([]byte, error) {
	switch length {
	case 2:
		if val < math.MinInt16 || val > math.MaxInt16 {
			return nil, fmt.Errorf("numeric out of range for SMALLINT")
		}
		v := int32(val)
		var tmp [4]byte
		binary.BigEndian.PutUint32(tmp[:], uint32(v))
		return append(buf, tmp[:]...), nil

	case 4:
		if val < math.MinInt32 || val > math.MaxInt32 {
			return nil, fmt.Errorf("numeric out of range for INTEGER")
		}
		v := int32(val)
		var tmp [4]byte
		binary.BigEndian.PutUint32(tmp[:], uint32(v))
		return append(buf, tmp[:]...), nil

	case 8:
		var tmp [8]byte
		binary.BigEndian.PutUint64(tmp[:], uint64(val))
		return append(buf, tmp[:]...), nil

	case 16:
		z := big.NewInt(val)
		return appendInt128BinaryFromBig(buf, z)

	default:
		return nil, fmt.Errorf("unsupported numeric length %d", length)
	}
}

// ParseNumeric parses s as a Firebird NUMERIC or DECIMAL value.
func ParseNumeric(s string) (Numeric, error) {
	return parseNumericText(s)
}

func parseNumericText(s string) (Numeric, error) {
	s = strings.TrimSpace(s)
	expIndex := strings.IndexAny(s, "eE")
	if expIndex == -1 {
		num, exp, err := parseDecimalMantissa(s)
		if err != nil {
			return Numeric{}, err
		}
		return newParsedNumeric(num, int64(exp))
	}

	if expIndex != strings.LastIndexAny(s, "eE") {
		return Numeric{}, fmt.Errorf("invalid scientific notation: %s", s)
	}
	if expIndex == 0 || expIndex == len(s)-1 {
		return Numeric{}, fmt.Errorf("invalid scientific notation: %s", s)
	}

	num, exp, err := parseDecimalMantissa(s[:expIndex])
	if err != nil {
		return Numeric{}, err
	}

	scientificExp, err := strconv.ParseInt(s[expIndex+1:], 10, 64)
	if err != nil {
		return Numeric{}, fmt.Errorf("invalid scientific notation exponent %q: %w", s[expIndex+1:], err)
	}

	mantissaExp := int64(exp)
	if scientificExp > 0 && mantissaExp > math.MaxInt64-scientificExp ||
		scientificExp < 0 && mantissaExp < math.MinInt64-scientificExp {
		return Numeric{}, fmt.Errorf("numeric exponent is out of range")
	}
	return newParsedNumeric(num, mantissaExp+scientificExp)
}

func newParsedNumeric(num *big.Int, exp int64) (Numeric, error) {
	if exp < math.MinInt32 || exp > math.MaxInt32 {
		return Numeric{}, fmt.Errorf("numeric exponent %d is out of int32 range", exp)
	}

	n := Numeric{Int: num, Exp: int32(exp), Valid: true}
	if err := validateNumericRepresentable(n); err != nil {
		return Numeric{}, err
	}
	return n, nil
}

func validateNumericScale(scale int32) error {
	if scale < -maxNumericScale || scale > 0 {
		return fmt.Errorf("numeric scale %d is outside the Firebird range [-%d, 0]", scale, maxNumericScale)
	}
	return nil
}

func validateNumericRepresentable(n Numeric) error {
	if !n.Valid {
		return nil
	}
	if n.Int == nil || n.Int.Sign() == 0 {
		if n.Exp < -maxNumericScale || n.Exp > maxNumericScale {
			return fmt.Errorf("numeric exponent %d is outside the supported Firebird range", n.Exp)
		}
		return nil
	}

	coefficient, exp := normalizeNumeric(n.Int, int64(n.Exp))
	if exp < -int64(maxNumericScale) || exp > int64(maxNumericScale) {
		return fmt.Errorf("numeric exponent %d is outside the supported Firebird range", exp)
	}

	if exp > 0 {
		pow := new(big.Int).Exp(big10, big.NewInt(exp), nil)
		coefficient.Mul(coefficient, pow)
	}
	if coefficient.Cmp(int128Min) < 0 || coefficient.Cmp(int128Max) > 0 {
		return fmt.Errorf("numeric value is out of range for int128")
	}
	return nil
}

func normalizeNumeric(coefficient *big.Int, exp int64) (*big.Int, int64) {
	normalized := new(big.Int).Set(coefficient)
	text := normalized.String()
	trimmed := strings.TrimRight(text, "0")
	trailingZeros := len(text) - len(trimmed)
	if trailingZeros == 0 {
		return normalized, exp
	}

	normalized.SetString(trimmed, 10)
	return normalized, exp + int64(trailingZeros)
}

func decimalDigitCount(n *big.Int) int {
	if n == nil || n.Sign() == 0 {
		return 1
	}

	return len(new(big.Int).Abs(n).String())
}

func cloneNumeric(n Numeric) Numeric {
	if !n.Valid {
		return Numeric{}
	}
	if n.Int == nil {
		return Numeric{Exp: n.Exp, Valid: true}
	}

	var z big.Int
	z.Set(n.Int)
	return Numeric{Int: &z, Exp: n.Exp, Valid: true}
}
