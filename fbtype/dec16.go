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
)

const (
	dec16MaxIntDigitsPlain    = 16
	dec16MaxLeadingZerosPlain = 6
)

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
	default:
		return "unknown"
	}
}

func (s SpecialValue) isKnown() bool {
	switch s {
	case SpecialNone, SpecialPosInf, SpecialNegInf, SpecialQuietNaN, SpecialSignalingNaN:
		return true
	default:
		return false
	}
}

func (s SpecialValue) isInfinity() bool {
	return s == SpecialPosInf || s == SpecialNegInf
}

func (s SpecialValue) isNaN() bool {
	return s == SpecialQuietNaN || s == SpecialSignalingNaN
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

// String returns the decimal representation of d, or an empty string for SQL
// NULL.
func (d DecFloat16) String() string {
	if !d.Valid {
		return ""
	}

	if d.Special != SpecialNone {
		return d.Special.String()
	}

	return formatDecFloat16General(d.Int, d.Exp)
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

func (d DecFloat16) validate() error {
	return validateDecFloat("DecFloat16", d.Valid, d.Special)
}

func validateDecFloat(typeName string, valid bool, special SpecialValue) error {
	if !valid {
		return nil
	}
	if !special.isKnown() {
		return fmt.Errorf("fbtype: %s has unknown special value %d", typeName, special)
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

// Float64ToDecFloat16 converts f to a DecFloat16. It preserves infinities. A
// finite value is converted through its shortest round-trip decimal
// representation and returns an error if that representation does not fit
// DECFLOAT(16).
func Float64ToDecFloat16(f float64) (DecFloat16, error) {
	if math.IsNaN(f) {
		return DecFloat16{Valid: true, Special: SpecialQuietNaN}, nil
	}
	if math.IsInf(f, 1) {
		return DecFloat16{Valid: true, Special: SpecialPosInf}, nil
	}
	if math.IsInf(f, -1) {
		return DecFloat16{Valid: true, Special: SpecialNegInf}, nil
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)

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
	}

	s := formatDecFloatScientific(d.Int, d.Exp)

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
	parsed, err := parseDecFloatString(s, "DecFloat16")
	if err != nil {
		*dst = DecFloat16{}
		return err
	}

	*dst = DecFloat16{
		Int:     parsed.coefficient,
		Exp:     parsed.exponent,
		Valid:   true,
		Special: parsed.special,
	}
	return nil
}

type parsedDecFloat struct {
	coefficient *big.Int
	exponent    int32
	special     SpecialValue
}

func parseDecFloatString(s string, typeName string) (parsedDecFloat, error) {
	original := strings.TrimSpace(s)
	if special, ok := parseSpecialValue(original); ok {
		return parsedDecFloat{special: special}, nil
	}

	mantissa := original
	var scientificExponent int64
	if exponentIndex := strings.IndexAny(original, "eE"); exponentIndex >= 0 {
		if exponentIndex != strings.LastIndexAny(original, "eE") {
			return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %q: multiple exponents", typeName, original)
		}

		mantissa = strings.TrimSpace(original[:exponentIndex])
		exponentText := strings.TrimSpace(original[exponentIndex+1:])
		if exponentText == "" {
			return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %q: missing exponent", typeName, original)
		}

		var err error
		scientificExponent, err = strconv.ParseInt(exponentText, 10, 32)
		if err != nil {
			return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %q: bad exponent %q: %w", typeName, original, exponentText, err)
		}
	}

	coefficient, exponent, err := parseNumericString(mantissa)
	if err != nil {
		return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %q: %w", typeName, original, err)
	}

	if coefficient.Sign() == 0 {
		return parsedDecFloat{coefficient: coefficient}, nil
	}

	combinedExponent := int64(exponent) + scientificExponent
	if combinedExponent < math.MinInt32 || combinedExponent > math.MaxInt32 {
		return parsedDecFloat{}, fmt.Errorf("cannot parse %s from %q: exponent %d is out of int32 range", typeName, original, combinedExponent)
	}

	return parsedDecFloat{
		coefficient: coefficient,
		exponent:    int32(combinedExponent),
	}, nil
}

func parseSpecialValue(s string) (SpecialValue, bool) {
	switch strings.ToLower(s) {
	case "+infinity", "+inf", "infinity", "inf":
		return SpecialPosInf, true
	case "-infinity", "-inf":
		return SpecialNegInf, true
	case "nan", "+nan":
		return SpecialQuietNaN, true
	case "snan", "+snan":
		return SpecialSignalingNaN, true
	default:
		return SpecialNone, false
	}
}

// formatDecimal renders as a plain decimal string without scientific notation.
func formatDecimal(coef *big.Int, exp int32) string {
	if coef == nil {
		return "0"
	}

	intStr := coef.String()
	negative := false
	if strings.HasPrefix(intStr, "-") {
		negative = true
		intStr = intStr[1:]
	}

	var result strings.Builder
	if negative {
		result.WriteByte('-')
	}

	if exp == 0 {
		result.WriteString(intStr)
		return result.String()
	}

	if exp > 0 {
		result.WriteString(intStr)
		for i := int32(0); i < exp; i++ {
			result.WriteByte('0')
		}
		return result.String()
	}

	absExp := -int64(exp)
	if int64(len(intStr)) <= absExp {
		result.WriteString("0.")
		for i := int64(0); i < absExp-int64(len(intStr)); i++ {
			result.WriteByte('0')
		}
		result.WriteString(intStr)
	} else {
		dpPos := len(intStr) - int(absExp)
		result.WriteString(intStr[:dpPos])
		result.WriteByte('.')
		result.WriteString(intStr[dpPos:])
	}

	return result.String()
}

// formatDecFloat16General formats a finite DecFloat16 using general rules:
// plain form when short, otherwise scientific.
func formatDecFloat16General(coef *big.Int, exp int32) string {
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
	if digitsToLeft > 0 && digitsToLeft <= int64(dec16MaxIntDigitsPlain) {
		return formatDecimal(coef, exp)
	}
	if digitsToLeft <= 0 {
		leadingZeros := -digitsToLeft
		if leadingZeros <= int64(dec16MaxLeadingZerosPlain) {
			return formatDecimal(coef, exp)
		}
	}

	return formatDecFloatScientific(coef, exp)
}

// formatDecFloatScientific formats as normalized scientific notation with one digit before the dot.
func formatDecFloatScientific(coef *big.Int, exp int32) string {
	if coef == nil || coef.Sign() == 0 {
		return "0"
	}

	neg := coef.Sign() < 0
	abs := new(big.Int).Abs(coef)
	digits := abs.String()
	dc := len(digits)

	k := int64(exp) + int64(dc-1)

	var mant string
	if dc == 1 {
		mant = digits
	} else {
		mant = digits[:1] + "." + digits[1:]
	}
	if neg {
		mant = "-" + mant
	}

	return fmt.Sprintf("%sE%+d", mant, k)
}

// encodeDecFloat16Binary encodes DecFloat16 into IEEE-754 decimal64.
func encodeDecFloat16Binary(buf []byte, d DecFloat16) ([]byte, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	if !d.Valid {
		return nil, nil
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal64)

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
	if coef.Sign() < 0 {
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

// decodeDecFloat16Binary decodes IEEE-754 decimal64 to DecFloat16.
func decodeDecFloat16Binary(src []byte, dst *DecFloat16) error {
	*dst = DecFloat16{}

	if len(src) != 8 {
		return fmt.Errorf("invalid length for DECFLOAT(16): expected 8 bytes, got %d", len(src))
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal64)

	dd, err := codec.ParseBytes(src)
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
		Int:     coef,
		Exp:     dd.Exponent,
		Valid:   true,
		Special: SpecialNone,
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
		return SpecialQuietNaN, nil
	case decfloat.DecimalSignalingNaN:
		return SpecialSignalingNaN, nil
	default:
		return SpecialNone, fmt.Errorf("fbtype: decimal type %d is not special", d.Type)
	}
}
