package fbtype

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"rdb.red-soft.ru/fbx/fbtype/decfloat"
)

const (
	dec34MaxIntDigitsPlain    = 34
	dec34MaxLeadingZerosPlain = 6
)

// Dec34Scanner scans a Firebird DECFLOAT(34) value.
type Dec34Scanner interface {
	// ScanDec34 scans v into the receiver.
	ScanDec34(v DecFloat34) error
}

// Dec34Valuer returns a Firebird DECFLOAT(34) value.
type Dec34Valuer interface {
	// Dec34Value returns the DECFLOAT(34) value to encode.
	Dec34Value() (DecFloat34, error)
}

// DecFloat34 represents a Firebird DECFLOAT(34) value.
//
// For finite values, Int contains the signed coefficient and Exp contains the
// unbiased decimal exponent. For example, Int=1234 and Exp=-2 represents
// 12.34. A nil Int represents zero. Special selects a non-finite value and, in
// that case, Int and Exp are ignored. Valid is false for SQL NULL.
type DecFloat34 struct {
	Int     *big.Int
	Exp     int32
	Valid   bool
	Special SpecialValue
	// NegativeZero preserves the IEEE-754 negative sign of a finite zero.
	// It must be false for nonzero and special values.
	NegativeZero bool
}

// ScanDec34 implements [Dec34Scanner].
func (d *DecFloat34) ScanDec34(v DecFloat34) error {
	if err := v.validate(); err != nil {
		*d = DecFloat34{}
		return err
	}

	*d = cloneDecFloat34(v)
	return nil
}

// Dec34Value implements [Dec34Valuer].
func (d DecFloat34) Dec34Value() (DecFloat34, error) {
	if err := d.validate(); err != nil {
		return DecFloat34{}, err
	}
	return cloneDecFloat34(d), nil
}

// Scan implements the [database/sql.Scanner] interface.
func (d *DecFloat34) Scan(src any) error {
	*d = DecFloat34{}

	if src == nil {
		return nil
	}

	switch src := src.(type) {
	case string:
		return scanPlanTextAnyToDecFloat34Scanner{}.Scan([]byte(src), d)
	case []byte:
		return scanPlanTextAnyToDecFloat34Scanner{}.Scan(src, d)
	}

	return fmt.Errorf("cannot scan %T into DecFloat34", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (d DecFloat34) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	return d.String(), nil
}

// String returns the decimal representation of d, or an empty string for SQL
// NULL.
func (d DecFloat34) String() string {
	if !d.Valid {
		return ""
	}
	if d.Special != SpecialNone {
		return d.Special.String()
	}
	if d.IsNegativeZero() {
		return "-0"
	}
	return formatDecFloat34General(d.Int, d.Exp)
}

// MarshalJSON encodes d as a JSON string or null.
func (d DecFloat34) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON decodes a JSON string or null into d.
func (d *DecFloat34) UnmarshalJSON(b []byte) error {
	*d = DecFloat34{}

	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == nil {
		*d = DecFloat34{}
		return nil
	}
	return scanPlanTextAnyToDecFloat34Scanner{}.Scan([]byte(*s), d)
}

// IsSpecial reports whether d is a valid infinity or NaN value.
func (d DecFloat34) IsSpecial() bool {
	return d.Valid && (d.Special.isInfinity() || d.Special.isNaN())
}

// IsInfinity reports whether d is a valid positive or negative infinity.
func (d DecFloat34) IsInfinity() bool {
	return d.Valid && d.Special.isInfinity()
}

// IsNaN reports whether d is a valid quiet or signaling NaN value.
func (d DecFloat34) IsNaN() bool {
	return d.Valid && d.Special.isNaN()
}

// IsNegativeZero reports whether d is a finite IEEE-754 negative zero.
func (d DecFloat34) IsNegativeZero() bool {
	return d.Valid && d.Special == SpecialNone && d.NegativeZero &&
		(d.Int == nil || d.Int.Sign() == 0)
}

func (d DecFloat34) validate() error {
	return validateDecFloat("DecFloat34", d.Int, d.Valid, d.Special, d.NegativeZero)
}

func cloneDecFloat34(d DecFloat34) DecFloat34 {
	if !d.Valid {
		return DecFloat34{}
	}

	clone := d
	if d.Int != nil {
		clone.Int = new(big.Int).Set(d.Int)
	}
	return clone
}

// DecFloat34Codec encodes and decodes Firebird DECFLOAT(34) values.
type DecFloat34Codec struct{}

// PlanEncode returns an encode plan for supported Firebird DECFLOAT(34) input
// values.
func (DecFloat34Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case Dec34Valuer:
		return encodePlanDecFloat34CodecBinary{}
	case NumericValuer:
		return encodePlanDecFloat34CodecBinaryFromNumeric{}
	case Float64Valuer:
		return encodePlanDecFloat34CodecBinaryFromFloat64{}
	case string:
		return encodePlanDecFloat34CodecBinaryFromString{}
	}

	return nil
}

type encodePlanDecFloat34CodecBinary struct{}

func (encodePlanDecFloat34CodecBinary) Encode(value any, buf []byte) ([]byte, error) {
	d, err := value.(Dec34Valuer).Dec34Value()
	if err != nil {
		return nil, err
	}
	if !d.Valid {
		return nil, nil
	}
	return encodeDecFloat34Binary(buf, d)
}

type encodePlanDecFloat34CodecBinaryFromNumeric struct{}

func (encodePlanDecFloat34CodecBinaryFromNumeric) Encode(value any, buf []byte) ([]byte, error) {
	n, err := value.(NumericValuer).NumericValue()
	if err != nil {
		return nil, err
	}
	if !n.Valid {
		return nil, nil
	}

	d := DecFloat34{
		Int:     n.Int,
		Exp:     n.Exp,
		Valid:   true,
		Special: SpecialNone,
	}
	return encodeDecFloat34Binary(buf, d)
}

type encodePlanDecFloat34CodecBinaryFromFloat64 struct{}

func (encodePlanDecFloat34CodecBinaryFromFloat64) Encode(value any, buf []byte) ([]byte, error) {
	f, err := value.(Float64Valuer).Float64Value()
	if err != nil {
		return nil, err
	}
	if !f.Valid {
		return nil, nil
	}

	d, err := Float64ToDecFloat34(f.Float64)
	if err != nil {
		return nil, err
	}
	return encodeDecFloat34Binary(buf, d)
}

type encodePlanDecFloat34CodecBinaryFromString struct{}

func (encodePlanDecFloat34CodecBinaryFromString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)

	var d DecFloat34
	if err := parseDecFloat34String(s, &d); err != nil {
		return nil, err
	}
	return encodeDecFloat34Binary(buf, d)
}

// PlanScan returns a scan plan for supported Firebird DECFLOAT(34) target
// values.
func (DecFloat34Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *DecFloat34:
		return scanPlanBinaryDecFloat34ToDecFloat34{}
	case Dec34Scanner:
		return scanPlanBinaryDecFloat34ToDecFloat34Scanner{}
	case *string:
		return scanPlanBinaryDecFloat34ToString{}
	case TextScanner:
		return scanPlanBinaryDecFloat34ToTextScanner{}
	case Float64Scanner:
		return scanPlanBinaryDecFloat34ToFloat64Scanner{}
	}

	return nil
}

type scanPlanBinaryDecFloat34ToDecFloat34 struct{}

func (scanPlanBinaryDecFloat34ToDecFloat34) Scan(src []byte, dst any) error {
	dPtr, ok := dst.(*DecFloat34)
	if !ok {
		return ErrScanTargetTypeChanged
	}
	if src == nil {
		*dPtr = DecFloat34{}
		return nil
	}
	return decodeDecFloat34Binary(src, dPtr)
}

type scanPlanBinaryDecFloat34ToDecFloat34Scanner struct{}

func (scanPlanBinaryDecFloat34ToDecFloat34Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Dec34Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}
	if src == nil {
		return scanner.ScanDec34(DecFloat34{})
	}

	var d DecFloat34
	if err := decodeDecFloat34Binary(src, &d); err != nil {
		return err
	}
	return scanner.ScanDec34(d)
}

type scanPlanBinaryDecFloat34ToString struct{}

func (scanPlanBinaryDecFloat34ToString) Scan(src []byte, dst any) error {
	strPtr, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}
	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	var d DecFloat34
	if err := decodeDecFloat34Binary(src, &d); err != nil {
		return err
	}
	*strPtr = d.String()
	return nil
}

type scanPlanBinaryDecFloat34ToTextScanner struct{}

func (scanPlanBinaryDecFloat34ToTextScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}
	if src == nil {
		return scanner.ScanText(Text{})
	}

	var d DecFloat34
	if err := decodeDecFloat34Binary(src, &d); err != nil {
		return err
	}
	return scanner.ScanText(Text{String: d.String(), Valid: true})
}

type scanPlanBinaryDecFloat34ToFloat64Scanner struct{}

func (scanPlanBinaryDecFloat34ToFloat64Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Float64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}
	if src == nil {
		return scanner.ScanFloat64(Float8{})
	}

	var d DecFloat34
	if err := decodeDecFloat34Binary(src, &d); err != nil {
		return err
	}

	f64, err := decFloat34ToFloat64(d)
	if err != nil {
		return err
	}
	return scanner.ScanFloat64(Float8{Float64: f64, Valid: true})
}

type scanPlanTextAnyToDecFloat34Scanner struct{}

func (scanPlanTextAnyToDecFloat34Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Dec34Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}
	if src == nil {
		return scanner.ScanDec34(DecFloat34{})
	}

	var d DecFloat34
	if err := parseDecFloat34String(string(src), &d); err != nil {
		return err
	}
	return scanner.ScanDec34(d)
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c DecFloat34Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var d DecFloat34
	if err := codecScan(c, m, typeMetadata, src, &d); err != nil {
		return nil, err
	}
	return d.String(), nil
}

// DecodeValue decodes src into the default Go value for Firebird DECFLOAT(34).
func (c DecFloat34Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var d DecFloat34
	if err := codecScan(c, m, typeMetadata, src, &d); err != nil {
		return nil, err
	}
	return d, nil
}

// Float64ToDecFloat34 converts f to a DecFloat34. It preserves signed zero,
// infinities, and the sign of NaN. A finite value is converted through its
// shortest round-trip decimal representation and returns an error if that
// representation does not fit DECFLOAT(34).
func Float64ToDecFloat34(f float64) (DecFloat34, error) {
	if math.IsNaN(f) {
		special := SpecialQuietNaN
		if math.Signbit(f) {
			special = SpecialNegQuietNaN
		}
		return DecFloat34{Valid: true, Special: special}, nil
	}
	if math.IsInf(f, 1) {
		return DecFloat34{Valid: true, Special: SpecialPosInf}, nil
	}
	if math.IsInf(f, -1) {
		return DecFloat34{Valid: true, Special: SpecialNegInf}, nil
	}
	if f == 0 {
		return DecFloat34{
			Int:          big.NewInt(0),
			Valid:        true,
			NegativeZero: math.Signbit(f),
		}, nil
	}

	s := strconv.FormatFloat(f, 'f', -1, 64)

	var d DecFloat34
	if err := parseDecFloat34String(s, &d); err != nil {
		return DecFloat34{}, fmt.Errorf("cannot convert float64 %g to DecFloat34: %w", f, err)
	}
	if err := decfloat.Decimal128.ValidateFiniteCoefficientAndExponent(d.Int, int(d.Exp)); err != nil {
		return DecFloat34{}, fmt.Errorf("cannot convert float64 %g to DecFloat34: %w", f, err)
	}
	return d, nil
}

// decFloat34ToFloat64 converts DecFloat34 to float64 (may lose precision) and returns an error on finite overflow.
func decFloat34ToFloat64(d DecFloat34) (float64, error) {
	if !d.Valid {
		return 0, fmt.Errorf("invalid DecFloat34")
	}
	if err := d.validate(); err != nil {
		return 0, err
	}

	switch d.Special {
	case SpecialPosInf:
		return math.Inf(1), nil
	case SpecialNegInf:
		return math.Inf(-1), nil
	case SpecialQuietNaN, SpecialSignalingNaN:
		return math.NaN(), nil
	case SpecialNegQuietNaN, SpecialNegSignalingNaN:
		return math.Copysign(math.NaN(), -1), nil
	}
	if d.IsNegativeZero() {
		return math.Copysign(0, -1), nil
	}

	s := formatDecFloatScientific(d.Int, d.Exp)

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		var ne *strconv.NumError
		if errors.As(err, &ne) && ne.Err == strconv.ErrRange {
			if math.IsInf(f, 0) {
				return 0, fmt.Errorf("decfloat34: overflow converting %s to float64", s)
			}
			return f, nil
		}
		return 0, err
	}
	return f, nil
}

// parseDecFloat34String parses a textual DECFLOAT(34), including scientific
// notation, into dst.
func parseDecFloat34String(s string, dst *DecFloat34) error {
	parsed, err := parseDecFloatString(s, "DecFloat34")
	if err != nil {
		*dst = DecFloat34{}
		return err
	}

	*dst = DecFloat34{
		Int:          parsed.coefficient,
		Exp:          parsed.exponent,
		Valid:        true,
		Special:      parsed.special,
		NegativeZero: parsed.negativeZero,
	}
	return nil
}

// formatDecFloat34General formats a finite DecFloat34 using general rules:
// plain form when short, otherwise scientific.
func formatDecFloat34General(coef *big.Int, exp int32) string {
	if coef == nil || coef.Sign() == 0 {
		return "0"
	}

	abs := new(big.Int).Abs(coef)
	digits := abs.String()
	digitsCount := len(digits)
	digitsToLeft := int64(digitsCount) + int64(exp)

	if exp == 0 {
		return formatDecimal(coef, exp)
	}
	if digitsToLeft > 0 && digitsToLeft <= int64(dec34MaxIntDigitsPlain) {
		return formatDecimal(coef, exp)
	}
	if digitsToLeft <= 0 {
		leadingZeros := -digitsToLeft
		if leadingZeros <= int64(dec34MaxLeadingZerosPlain) {
			return formatDecimal(coef, exp)
		}
	}
	return formatDecFloatScientific(coef, exp)
}

// encodeDecFloat34Binary encodes DecFloat34 into IEEE-754 decimal128.
func encodeDecFloat34Binary(buf []byte, d DecFloat34) ([]byte, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	if !d.Valid {
		return nil, nil
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal128)

	if d.Special != SpecialNone {
		dd, err := decodedDecimalFromSpecial(d.Special)
		if err != nil {
			return nil, err
		}
		b, err := codec.Encode(dd)
		if err != nil {
			return nil, err
		}
		return append(buf, b...), nil
	}

	coef := d.Int
	if coef == nil {
		coef = big.NewInt(0)
	}

	signum := decfloat.SignumPositive
	if coef.Sign() < 0 || d.NegativeZero {
		signum = decfloat.SignumNegative
	}

	dd := decfloat.DecodedDecimal{
		Signum:      signum,
		Type:        decfloat.DecimalFinite,
		Exponent:    d.Exp,
		Coefficient: coef,
	}

	b, err := codec.Encode(dd)
	if err != nil {
		return nil, err
	}
	return append(buf, b...), nil
}

// decodeDecFloat34Binary decodes IEEE-754 decimal128 to DecFloat34.
func decodeDecFloat34Binary(src []byte, dst *DecFloat34) error {
	*dst = DecFloat34{}

	if len(src) != 16 {
		return fmt.Errorf("invalid length for DECFLOAT(34): expected 16 bytes, got %d", len(src))
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal128)

	dd, err := codec.ParseBytes(src)
	if err != nil {
		return err
	}

	if dd.Type != decfloat.DecimalFinite {
		special, err := specialValueFromDecodedDecimal(dd)
		if err != nil {
			return err
		}
		*dst = DecFloat34{Valid: true, Special: special}
		return nil
	}

	coef := dd.Coefficient
	if coef == nil {
		coef = big.NewInt(0)
	}
	*dst = DecFloat34{
		Int:          coef,
		Exp:          dd.Exponent,
		Valid:        true,
		Special:      SpecialNone,
		NegativeZero: coef.Sign() == 0 && dd.Signum == decfloat.SignumNegative,
	}
	return nil
}
