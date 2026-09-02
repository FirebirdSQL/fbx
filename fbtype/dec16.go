package fbtype

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"rdb.red-soft.ru/fbx/fbtype/decfloat"
)

// SpecialValue identifies whether an IEEE-754 decimal is finite, an infinity,
// or a NaN value.
type SpecialValue byte

const (
	SpecialNone SpecialValue = iota
	SpecialPosInf
	SpecialNegInf
	SpecialQuietNaN
	SpecialSignalingNaN
	SpecialNegQuietNaN
	SpecialNegSignalingNaN
)

const (
	decFloat16CoefficientDigits = 16
	decFloat34CoefficientDigits = 34
	maxDecFloatErrorInputBytes  = 128
)

var decFloat16BinaryCodec = decfloat.NewDecimalCodec(decfloat.Decimal64)

// String returns the canonical textual representation of s.
func (s SpecialValue) String() string {
	switch s {
	case SpecialNone:
		return "none"
	case SpecialPosInf:
		return "+Infinity"
	case SpecialNegInf:
		return "-Infinity"
	case SpecialQuietNaN:
		return "NaN"
	case SpecialSignalingNaN:
		return "sNaN"
	case SpecialNegQuietNaN:
		return "-NaN"
	case SpecialNegSignalingNaN:
		return "-sNaN"
	default:
		return "unknown"
	}
}

func (s SpecialValue) isKnown() bool {
	switch s {
	case SpecialNone, SpecialPosInf, SpecialNegInf, SpecialQuietNaN,
		SpecialSignalingNaN, SpecialNegQuietNaN, SpecialNegSignalingNaN:
		return true
	default:
		return false
	}
}

func (s SpecialValue) isInfinity() bool {
	return s == SpecialPosInf || s == SpecialNegInf
}

func (s SpecialValue) isNaN() bool {
	return s == SpecialQuietNaN || s == SpecialSignalingNaN ||
		s == SpecialNegQuietNaN || s == SpecialNegSignalingNaN
}

// Dec16Scanner scans a Firebird DECFLOAT(16) value.
type Dec16Scanner interface {
	// ScanDec16 scans v into the receiver.
	ScanDec16(v DecFloat16) error
}

// Dec16Valuer returns a Firebird DECFLOAT(16) value.
type Dec16Valuer interface {
	// Dec16Value returns the DECFLOAT(16) value to encode.
	Dec16Value() (DecFloat16, error)
}

// DecFloat16 represents a Firebird DECFLOAT(16) value.
//
// For finite values, Int contains the signed coefficient and Exp contains the
// unbiased decimal exponent. For example, Int=1234 and Exp=-2 represents
// 12.34. A nil Int represents zero. Special selects a non-finite value and, in
// that case, Int and Exp are ignored. Valid is false for SQL NULL.
type DecFloat16 struct {
	Int     *big.Int
	Exp     int32
	Valid   bool
	Special SpecialValue
	// NegativeZero preserves the IEEE-754 negative sign of a finite zero.
	// It must be false for nonzero and special values.
	NegativeZero bool
}

// ScanDec16 implements [Dec16Scanner].
func (d *DecFloat16) ScanDec16(v DecFloat16) error {
	if err := v.validate(); err != nil {
		*d = DecFloat16{}
		return err
	}

	*d = cloneDecFloat16(v)
	return nil
}

// Dec16Value implements [Dec16Valuer].
func (d DecFloat16) Dec16Value() (DecFloat16, error) {
	if err := d.validate(); err != nil {
		return DecFloat16{}, err
	}
	return cloneDecFloat16(d), nil
}

// Scan implements the [database/sql.Scanner] interface.
func (d *DecFloat16) Scan(src any) error {
	*d = DecFloat16{}

	if src == nil {
		return nil
	}

	switch src := src.(type) {
	case string:
		return scanPlanTextAnyToDecFloat16Scanner{}.Scan([]byte(src), d)
	case []byte:
		return scanPlanTextAnyToDecFloat16Scanner{}.Scan(src, d)
	}

	return fmt.Errorf("cannot scan %T into DecFloat16", src)
}

// Value implements the [database/sql/driver.Valuer] interface.
func (d DecFloat16) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	if err := d.validate(); err != nil {
		return nil, err
	}

	return d.String(), nil
}

// String returns the canonical decimal representation of d, or an empty string
// for SQL NULL. For finite values, the representation preserves the coefficient,
// exponent, and sign of zero.
func (d DecFloat16) String() string {
	if !d.Valid {
		return ""
	}

	if d.Special != SpecialNone {
		return d.Special.String()
	}
	return formatDecFloat(d.Int, d.Exp, d.IsNegativeZero())
}

// MarshalJSON encodes d as a JSON string or null.
func (d DecFloat16) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}
	if err := d.validate(); err != nil {
		return nil, err
	}

	return json.Marshal(d.String())
}

// UnmarshalJSON decodes a JSON string or null into d.
func (d *DecFloat16) UnmarshalJSON(b []byte) error {
	*d = DecFloat16{}

	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s == nil {
		*d = DecFloat16{}
		return nil
	}

	return scanPlanTextAnyToDecFloat16Scanner{}.Scan([]byte(*s), d)
}

// IsSpecial reports whether d is a valid infinity or NaN value.
func (d DecFloat16) IsSpecial() bool {
	return d.Valid && (d.Special.isInfinity() || d.Special.isNaN())
}

// IsInfinity reports whether d is a valid positive or negative infinity.
func (d DecFloat16) IsInfinity() bool {
	return d.Valid && d.Special.isInfinity()
}

// IsNaN reports whether d is a valid quiet or signaling NaN value.
func (d DecFloat16) IsNaN() bool {
	return d.Valid && d.Special.isNaN()
}

// IsNegativeZero reports whether d is a finite IEEE-754 negative zero.
func (d DecFloat16) IsNegativeZero() bool {
	return d.Valid && d.Special == SpecialNone && d.NegativeZero &&
		(d.Int == nil || d.Int.Sign() == 0)
}

func (d DecFloat16) validate() error {
	if err := validateDecFloat("DecFloat16", d.Int, d.Valid, d.Special, d.NegativeZero); err != nil {
		return err
	}
	return validateFiniteDecFloat("DecFloat16", d.Int, d.Valid, d.Special, d.Exp, decfloat.Decimal64)
}

func validateDecFloat(typeName string, coefficient *big.Int, valid bool, special SpecialValue, negativeZero bool) error {
	if !valid {
		return nil
	}
	if !special.isKnown() {
		return fmt.Errorf("fbtype: %s has unknown special value %d", typeName, special)
	}
	if negativeZero && (special != SpecialNone || coefficient != nil && coefficient.Sign() != 0) {
		return fmt.Errorf("fbtype: %s NegativeZero requires a finite zero", typeName)
	}
	return nil
}

func validateFiniteDecFloat(typeName string, coefficient *big.Int, valid bool, special SpecialValue, exponent int32, format decfloat.DecimalFormat) error {
	if !valid || special != SpecialNone {
		return nil
	}
	if coefficient == nil {
		coefficient = new(big.Int)
	}
	if err := format.ValidateFiniteCoefficientAndExponent(coefficient, int(exponent)); err != nil {
		return fmt.Errorf("fbtype: invalid %s: %w", typeName, err)
	}
	return nil
}

func cloneDecFloat16(d DecFloat16) DecFloat16 {
	if !d.Valid {
		return DecFloat16{}
	}

	clone := d
	if d.Int != nil {
		clone.Int = new(big.Int).Set(d.Int)
	}
	return clone
}

// DecFloat16Codec encodes and decodes Firebird DECFLOAT(16) values.
type DecFloat16Codec struct{}

// PlanEncode returns an encode plan for supported Firebird DECFLOAT(16) input
// values.
func (DecFloat16Codec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case Dec16Valuer:
		return encodePlanDecFloat16CodecBinary{}
	case NumericValuer:
		return encodePlanDecFloat16CodecBinaryFromNumeric{}
	case Float64Valuer:
		return encodePlanDecFloat16CodecBinaryFromFloat64{}
	case string:
		return encodePlanDecFloat16CodecBinaryFromString{}
	}

	return nil
}

type encodePlanDecFloat16CodecBinary struct{}

func (encodePlanDecFloat16CodecBinary) Encode(value any, buf []byte) ([]byte, error) {
	d, err := value.(Dec16Valuer).Dec16Value()
	if err != nil {
		return nil, err
	}

	if !d.Valid {
		return nil, nil
	}

	return encodeDecFloat16Binary(buf, d)
}

type encodePlanDecFloat16CodecBinaryFromNumeric struct{}

func (encodePlanDecFloat16CodecBinaryFromNumeric) Encode(value any, buf []byte) ([]byte, error) {
	n, err := value.(NumericValuer).NumericValue()
	if err != nil {
		return nil, err
	}

	if !n.Valid {
		return nil, nil
	}

	d := DecFloat16{
		Int:     n.Int,
		Exp:     n.Exp,
		Valid:   true,
		Special: SpecialNone,
	}

	return encodeDecFloat16Binary(buf, d)
}

type encodePlanDecFloat16CodecBinaryFromFloat64 struct{}

func (encodePlanDecFloat16CodecBinaryFromFloat64) Encode(value any, buf []byte) ([]byte, error) {
	f, err := value.(Float64Valuer).Float64Value()
	if err != nil {
		return nil, err
	}
	if !f.Valid {
		return nil, nil
	}

	d, err := Float64ToDecFloat16(f.Float64)
	if err != nil {
		return nil, err
	}

	return encodeDecFloat16Binary(buf, d)
}

type encodePlanDecFloat16CodecBinaryFromString struct{}

func (encodePlanDecFloat16CodecBinaryFromString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)

	var d DecFloat16
	if err := parseDecFloat16String(s, &d); err != nil {
		return nil, err
	}

	return encodeDecFloat16Binary(buf, d)
}

// PlanScan returns a scan plan for supported Firebird DECFLOAT(16) target
// values.
func (DecFloat16Codec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *DecFloat16:
		return scanPlanBinaryDecFloat16ToDecFloat16{}
	case Dec16Scanner:
		return scanPlanBinaryDecFloat16ToDecFloat16Scanner{}
	case *string:
		return scanPlanBinaryDecFloat16ToString{}
	case TextScanner:
		return scanPlanBinaryDecFloat16ToTextScanner{}
	case Float64Scanner:
		return scanPlanBinaryDecFloat16ToFloat64Scanner{}
	}

	return nil
}

type scanPlanBinaryDecFloat16ToDecFloat16 struct{}

func (scanPlanBinaryDecFloat16ToDecFloat16) Scan(src []byte, dst any) error {
	dPtr, ok := dst.(*DecFloat16)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		*dPtr = DecFloat16{}
		return nil
	}

	return decodeDecFloat16Binary(src, dPtr)
}

type scanPlanBinaryDecFloat16ToDecFloat16Scanner struct{}

func (scanPlanBinaryDecFloat16ToDecFloat16Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Dec16Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanDec16(DecFloat16{})
	}

	var d DecFloat16
	if err := decodeDecFloat16Binary(src, &d); err != nil {
		return err
	}

	return scanner.ScanDec16(d)
}

type scanPlanBinaryDecFloat16ToString struct{}

func (scanPlanBinaryDecFloat16ToString) Scan(src []byte, dst any) error {
	strPtr, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into *string")
	}

	var d DecFloat16
	if err := decodeDecFloat16Binary(src, &d); err != nil {
		return err
	}

	*strPtr = d.String()
	return nil
}

type scanPlanBinaryDecFloat16ToTextScanner struct{}

func (scanPlanBinaryDecFloat16ToTextScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanText(Text{})
	}

	var d DecFloat16
	if err := decodeDecFloat16Binary(src, &d); err != nil {
		return err
	}

	return scanner.ScanText(Text{String: d.String(), Valid: true})
}

type scanPlanBinaryDecFloat16ToFloat64Scanner struct{}

func (scanPlanBinaryDecFloat16ToFloat64Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Float64Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanFloat64(Float8{})
	}

	var d DecFloat16
	if err := decodeDecFloat16Binary(src, &d); err != nil {
		return err
	}

	f64, err := decFloat16ToFloat64(d)
	if err != nil {
		return err
	}

	return scanner.ScanFloat64(Float8{Float64: f64, Valid: true})
}

type scanPlanTextAnyToDecFloat16Scanner struct{}

func (scanPlanTextAnyToDecFloat16Scanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(Dec16Scanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanDec16(DecFloat16{})
	}

	var d DecFloat16
	if err := parseDecFloat16String(string(src), &d); err != nil {
		return err
	}

	return scanner.ScanDec16(d)
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c DecFloat16Codec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var d DecFloat16
	err := codecScan(c, m, typeMetadata, src, &d)
	if err != nil {
		return nil, err
	}

	return d.String(), nil
}

// DecodeValue decodes src into the default Go value for Firebird DECFLOAT(16).
func (c DecFloat16Codec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var d DecFloat16
	err := codecScan(c, m, typeMetadata, src, &d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Float64ToDecFloat16 converts f to a DecFloat16. It preserves signed zero,
// infinities, and the sign of NaN. A finite value is converted through its
// shortest round-trip decimal representation and returns an error if that
// representation does not fit DECFLOAT(16).
func Float64ToDecFloat16(f float64) (DecFloat16, error) {
	if math.IsNaN(f) {
		special := SpecialQuietNaN
		if math.Signbit(f) {
			special = SpecialNegQuietNaN
		}
		return DecFloat16{Valid: true, Special: special}, nil
	}
	if math.IsInf(f, 1) {
		return DecFloat16{Valid: true, Special: SpecialPosInf}, nil
	}
	if math.IsInf(f, -1) {
		return DecFloat16{Valid: true, Special: SpecialNegInf}, nil
	}
	if f == 0 {
		return DecFloat16{
			Int:          big.NewInt(0),
			Valid:        true,
			NegativeZero: math.Signbit(f),
		}, nil
	}

	s := strconv.FormatFloat(f, 'g', -1, 64)

	var d DecFloat16
	if err := parseDecFloat16String(s, &d); err != nil {
		return DecFloat16{}, fmt.Errorf("cannot convert float64 %g to DecFloat16: %w", f, err)
	}
	if err := decfloat.Decimal64.ValidateFiniteCoefficientAndExponent(d.Int, int(d.Exp)); err != nil {
		return DecFloat16{}, fmt.Errorf("cannot convert float64 %g to DecFloat16: %w", f, err)
	}

	return d, nil
}

// decFloat16ToFloat64 converts DecFloat16 to float64 (may lose precision) and returns an error on finite overflow.
func decFloat16ToFloat64(d DecFloat16) (float64, error) {
	if !d.Valid {
		return 0, fmt.Errorf("invalid DecFloat16")
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

	s := formatDecFloat(d.Int, d.Exp, false)

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		var ne *strconv.NumError
		if errors.As(err, &ne) && ne.Err == strconv.ErrRange {
			if math.IsInf(f, 0) {
				return 0, fmt.Errorf("decfloat16: overflow converting %s to float64", s)
			}
			return f, nil
		}
		return 0, err
	}
	return f, nil
}

// parseDecFloat16String parses a textual DECFLOAT(16), including scientific
// notation, into dst.
func parseDecFloat16String(s string, dst *DecFloat16) error {
	parsed, err := parseDecFloatString(s, "DecFloat16", decFloat16CoefficientDigits, decfloat.Decimal64)
	if err != nil {
		*dst = DecFloat16{}
		return err
	}

	*dst = DecFloat16{
		Int:          parsed.coefficient,
		Exp:          parsed.exponent,
		Valid:        true,
		Special:      parsed.special,
		NegativeZero: parsed.negativeZero,
	}
	return nil
}

type parsedDecFloat struct {
	coefficient  *big.Int
	exponent     int32
	special      SpecialValue
	negativeZero bool
}

func parseDecFloatString(s string, typeName string, coefficientDigits int, format decfloat.DecimalFormat) (parsedDecFloat, error) {
	original := strings.TrimSpace(s)
	if special, ok := parseSpecialValue(original); ok {
		return parsedDecFloat{special: special}, nil
	}
	quotedOriginal := quoteDecFloatInput(original)

	mantissa := original
	var scientificExponent int64
	if exponentIndex := strings.IndexAny(original, "eE"); exponentIndex >= 0 {
		if exponentIndex != strings.LastIndexAny(original, "eE") {
			return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: multiple exponents", typeName, quotedOriginal)
		}

		mantissa = strings.TrimSpace(original[:exponentIndex])
		exponentText := strings.TrimSpace(original[exponentIndex+1:])
		if exponentText == "" {
			return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: missing exponent", typeName, quotedOriginal)
		}

		var err error
		scientificExponent, err = strconv.ParseInt(exponentText, 10, 64)
		if err != nil {
			if errors.Is(err, strconv.ErrRange) {
				return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: exponent %s is out of int64 range",
					typeName, quotedOriginal, quoteDecFloatInput(exponentText))
			}
			return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: bad exponent %s",
				typeName, quotedOriginal, quoteDecFloatInput(exponentText))
		}
	}

	coefficient, exponent, err := parseDecFloatMantissa(mantissa, coefficientDigits)
	if err != nil {
		return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: %w", typeName, quotedOriginal, err)
	}

	negativeZero := coefficient.Sign() == 0 && strings.HasPrefix(mantissa, "-")
	if scientificExponent > 0 && exponent > math.MaxInt64-scientificExponent ||
		scientificExponent < 0 && exponent < math.MinInt64-scientificExponent {
		return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: exponent is out of int64 range", typeName, quotedOriginal)
	}
	combinedExponent := exponent + scientificExponent
	if combinedExponent < math.MinInt32 || combinedExponent > math.MaxInt32 {
		return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: exponent %d is out of int32 range", typeName, quotedOriginal, combinedExponent)
	}
	if err := format.ValidateFiniteCoefficientAndExponent(coefficient, int(combinedExponent)); err != nil {
		return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %s: %w", typeName, quotedOriginal, err)
	}

	return parsedDecFloat{
		coefficient:  coefficient,
		exponent:     int32(combinedExponent),
		negativeZero: negativeZero,
	}, nil
}

func parseDecFloatMantissa(s string, maxCoefficientDigits int) (*big.Int, int64, error) {
	if s == "" {
		return nil, 0, fmt.Errorf("mantissa is empty")
	}

	negative := false
	start := 0
	if s[0] == '+' || s[0] == '-' {
		negative = s[0] == '-'
		start++
	}
	if start == len(s) {
		return nil, 0, fmt.Errorf("mantissa has no digits")
	}

	digits := make([]byte, 0, maxCoefficientDigits)
	decimalPointSeen := false
	digitSeen := false
	nonzeroSeen := false
	var fractionalDigits int64
	for i := start; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '.':
			if decimalPointSeen {
				return nil, 0, fmt.Errorf("mantissa contains multiple decimal points")
			}
			decimalPointSeen = true
		case ch >= '0' && ch <= '9':
			digitSeen = true
			if decimalPointSeen {
				fractionalDigits++
			}
			if ch != '0' || nonzeroSeen {
				nonzeroSeen = true
				if len(digits) == maxCoefficientDigits {
					return nil, 0, fmt.Errorf("coefficient out of range: more than %d decimal digits", maxCoefficientDigits)
				}
				digits = append(digits, ch)
			}
		default:
			return nil, 0, fmt.Errorf("mantissa contains invalid character %q", ch)
		}
	}
	if !digitSeen {
		return nil, 0, fmt.Errorf("mantissa has no digits")
	}
	if !nonzeroSeen {
		return new(big.Int), -fractionalDigits, nil
	}

	coefficient := new(big.Int)
	if _, ok := coefficient.SetString(string(digits), 10); !ok {
		return nil, 0, fmt.Errorf("cannot parse decimal coefficient")
	}
	if negative {
		coefficient.Neg(coefficient)
	}
	return coefficient, -fractionalDigits, nil
}

func parseSpecialValue(s string) (SpecialValue, bool) {
	switch {
	case strings.EqualFold(s, "+infinity"), strings.EqualFold(s, "+inf"),
		strings.EqualFold(s, "infinity"), strings.EqualFold(s, "inf"):
		return SpecialPosInf, true
	case strings.EqualFold(s, "-infinity"), strings.EqualFold(s, "-inf"):
		return SpecialNegInf, true
	case strings.EqualFold(s, "nan"), strings.EqualFold(s, "+nan"):
		return SpecialQuietNaN, true
	case strings.EqualFold(s, "-nan"):
		return SpecialNegQuietNaN, true
	case strings.EqualFold(s, "snan"), strings.EqualFold(s, "+snan"):
		return SpecialSignalingNaN, true
	case strings.EqualFold(s, "-snan"):
		return SpecialNegSignalingNaN, true
	default:
		return SpecialNone, false
	}
}

func quoteDecFloatInput(s string) string {
	if len(s) <= maxDecFloatErrorInputBytes {
		return strconv.Quote(s)
	}
	return fmt.Sprintf("%s... (%d bytes)", strconv.Quote(s[:maxDecFloatErrorInputBytes]), len(s))
}

// formatDecFloat returns a one-to-one textual representation of a finite
// decimal coefficient and exponent, following the canonical BigDecimal form.
func formatDecFloat(coefficient *big.Int, exponent int32, negativeZero bool) string {
	negative := negativeZero
	digits := "0"
	if coefficient != nil && coefficient.Sign() != 0 {
		negative = coefficient.Sign() < 0
		digits = new(big.Int).Abs(coefficient).String()
	}

	adjustedExponent := int64(exponent) + int64(len(digits)-1)
	var result strings.Builder
	if negative {
		result.WriteByte('-')
	}

	if exponent <= 0 && adjustedExponent >= -6 {
		scale := -int64(exponent)
		if scale == 0 {
			result.WriteString(digits)
			return result.String()
		}
		if int64(len(digits)) <= scale {
			result.WriteString("0.")
			for i := int64(0); i < scale-int64(len(digits)); i++ {
				result.WriteByte('0')
			}
			result.WriteString(digits)
			return result.String()
		}

		decimalPoint := len(digits) - int(scale)
		result.WriteString(digits[:decimalPoint])
		result.WriteByte('.')
		result.WriteString(digits[decimalPoint:])
		return result.String()
	}

	result.WriteByte(digits[0])
	if len(digits) > 1 {
		result.WriteByte('.')
		result.WriteString(digits[1:])
	}
	result.WriteByte('E')
	if adjustedExponent >= 0 {
		result.WriteByte('+')
	}
	result.WriteString(strconv.FormatInt(adjustedExponent, 10))
	return result.String()
}

// encodeDecFloat16Binary encodes DecFloat16 into IEEE-754 decimal64.
func encodeDecFloat16Binary(buf []byte, d DecFloat16) ([]byte, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	if !d.Valid {
		return nil, nil
	}

	if d.Special != SpecialNone {
		dd, err := decodedDecimalFromSpecial(d.Special)
		if err != nil {
			return nil, err
		}
		b, err := decFloat16BinaryCodec.Encode(dd)
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

	b, err := decFloat16BinaryCodec.Encode(dd)
	if err != nil {
		return nil, err
	}
	return append(buf, b...), nil
}

// decodeDecFloat16Binary decodes IEEE-754 decimal64 to DecFloat16.
func decodeDecFloat16Binary(src []byte, dst *DecFloat16) error {
	*dst = DecFloat16{}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for DECFLOAT(16): expected 8 bytes, got %d", len(src))
	}

	dd, err := decFloat16BinaryCodec.ParseBytes(src)
	if err != nil {
		return err
	}

	if dd.Type != decfloat.DecimalFinite {
		special, err := specialValueFromDecodedDecimal(dd)
		if err != nil {
			return err
		}
		*dst = DecFloat16{Valid: true, Special: special}
		return nil
	}

	coef := dd.Coefficient
	if coef == nil {
		coef = big.NewInt(0)
	}

	*dst = DecFloat16{
		Int:          coef,
		Exp:          dd.Exponent,
		Valid:        true,
		Special:      SpecialNone,
		NegativeZero: coef.Sign() == 0 && dd.Signum == decfloat.SignumNegative,
	}
	return nil
}

func decodedDecimalFromSpecial(s SpecialValue) (decfloat.DecodedDecimal, error) {
	switch s {
	case SpecialPosInf:
		return decfloat.DecodedDecimal{Signum: decfloat.SignumPositive, Type: decfloat.DecimalInfinity}, nil
	case SpecialNegInf:
		return decfloat.DecodedDecimal{Signum: decfloat.SignumNegative, Type: decfloat.DecimalInfinity}, nil
	case SpecialQuietNaN:
		return decfloat.DecodedDecimal{Signum: decfloat.SignumPositive, Type: decfloat.DecimalNaN}, nil
	case SpecialSignalingNaN:
		return decfloat.DecodedDecimal{Signum: decfloat.SignumPositive, Type: decfloat.DecimalSignalingNaN}, nil
	case SpecialNegQuietNaN:
		return decfloat.DecodedDecimal{Signum: decfloat.SignumNegative, Type: decfloat.DecimalNaN}, nil
	case SpecialNegSignalingNaN:
		return decfloat.DecodedDecimal{Signum: decfloat.SignumNegative, Type: decfloat.DecimalSignalingNaN}, nil
	default:
		return decfloat.DecodedDecimal{}, fmt.Errorf("fbtype: unknown special value %d", s)
	}
}

func specialValueFromDecodedDecimal(d decfloat.DecodedDecimal) (SpecialValue, error) {
	var negative bool
	switch d.Signum {
	case decfloat.SignumPositive:
	case decfloat.SignumNegative:
		negative = true
	default:
		return SpecialNone, fmt.Errorf("fbtype: unknown decimal signum %d", d.Signum)
	}

	switch d.Type {
	case decfloat.DecimalInfinity:
		if negative {
			return SpecialNegInf, nil
		}
		return SpecialPosInf, nil
	case decfloat.DecimalNaN:
		if negative {
			return SpecialNegQuietNaN, nil
		}
		return SpecialQuietNaN, nil
	case decfloat.DecimalSignalingNaN:
		if negative {
			return SpecialNegSignalingNaN, nil
		}
		return SpecialSignalingNaN, nil
	default:
		return SpecialNone, fmt.Errorf("fbtype: decimal type %d is not special", d.Type)
	}
}
