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

type Dec16Scanner interface {
	ScanDec16(v DecFloat16) error
}

type Dec16Valuer interface {
	Dec16Value() (DecFloat16, error)
}

type DecFloat16 struct {
	Int     *big.Int
	Exp     int32
	Valid   bool
	Special SpecialValue
}

func (d *DecFloat16) ScanDec16(v DecFloat16) error {
	*d = v
	return nil
}

func (d DecFloat16) Dec16Value() (DecFloat16, error) {
	return d, nil
}

// Scan implements the database/sql Scanner interface
func (d *DecFloat16) Scan(src any) error {
	if src == nil {
		*d = DecFloat16{}
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

// Value implements the database/sql/driver Valuer interface.
func (d DecFloat16) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}

	return d.String(), nil
}

func (d DecFloat16) String() string {
	if !d.Valid {
		return ""
	}

	if d.Special != SpecialNone {
		return d.Special.String()
	}

	return formatDecFloat16General(d.Int, d.Exp)
}

func (d DecFloat16) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(d.String())
}

func (d *DecFloat16) UnmarshalJSON(b []byte) error {
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

func (d DecFloat16) IsSpecial() bool {
	return d.Special != SpecialNone
}

func (d DecFloat16) IsInfinity() bool {
	return d.Special == SpecialPosInf || d.Special == SpecialNegInf
}

func (d DecFloat16) IsNaN() bool {
	return d.Special == SpecialQuietNaN || d.Special == SpecialSignalingNaN
}

type DecFloat16Codec struct{}

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
		*strPtr = ""
		return nil
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

// Float64ToDecFloat16 converts float64 to DecFloat16, handling NaN/Inf explicitly and parsing finite values via decimal text.
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
		return DecFloat16{}, fmt.Errorf("cannot convert float64 %f to DecFloat16: %w", f, err)
	}

	return d, nil
}

// decFloat16ToFloat64 converts DecFloat16 to float64 (may lose precision) and returns an error on finite overflow.
func decFloat16ToFloat64(d DecFloat16) (float64, error) {
	if !d.Valid {
		return 0, fmt.Errorf("invalid DecFloat16")
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

// parseDecFloat16String parses a textual DECFLOAT(16) (including scientific notation) into dst.
func parseDecFloat16String(s string, dst *DecFloat16) error {
	original := strings.TrimSpace(s)
	s = original

	switch strings.ToLower(s) {
	case "+infinity", "+inf", "infinity", "inf":
		*dst = DecFloat16{Valid: true, Special: SpecialPosInf}
		return nil
	case "-infinity", "-inf":
		*dst = DecFloat16{Valid: true, Special: SpecialNegInf}
		return nil
	case "nan":
		*dst = DecFloat16{Valid: true, Special: SpecialQuietNaN}
		return nil
	case "snan":
		*dst = DecFloat16{Valid: true, Special: SpecialSignalingNaN}
		return nil
	}

	var exp10 int32
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mant := strings.TrimSpace(s[:i])
		expPart := strings.TrimSpace(s[i+1:])
		if expPart == "" {
			return fmt.Errorf("cannot parse DecFloat16 from %q: missing exponent", original)
		}
		e, err := strconv.ParseInt(expPart, 10, 32)
		if err != nil {
			return fmt.Errorf("cannot parse DecFloat16 from %q: bad exponent %q: %w", original, expPart, err)
		}
		exp10 = int32(e)
		s = mant
	}

	num, exp, err := parseDecimalMantissa(s)
	if err != nil {
		return fmt.Errorf("cannot parse DecFloat16 from %q: %w", original, err)
	}

	exp += exp10
	if num.Sign() == 0 {
		exp = 0
	}

	*dst = DecFloat16{
		Int:     num,
		Exp:     exp,
		Valid:   true,
		Special: SpecialNone,
	}

	return nil
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

	absExp := int(-exp)
	if len(intStr) <= absExp {
		result.WriteString("0.")
		for i := 0; i < absExp-len(intStr); i++ {
			result.WriteByte('0')
		}
		result.WriteString(intStr)
	} else {
		dpPos := len(intStr) - absExp
		result.WriteString(intStr[:dpPos])
		result.WriteByte('.')
		result.WriteString(intStr[dpPos:])
	}

	return result.String()
}

// formatDecFloat16General formats a finite DecFloat16 using “general” rules: plain form when short, otherwise scientific.
func formatDecFloat16General(coef *big.Int, exp int32) string {
	if coef == nil || coef.Sign() == 0 {
		return "0"
	}

	abs := new(big.Int).Abs(coef)
	digits := abs.String()
	digitsCount := len(digits)

	digitsToLeft := digitsCount + int(exp)

	if exp == 0 {
		return formatDecimal(coef, exp)
	}
	if digitsToLeft > 0 && digitsToLeft <= dec16MaxIntDigitsPlain {
		return formatDecimal(coef, exp)
	}
	if digitsToLeft <= 0 {
		leadingZeros := -digitsToLeft
		if leadingZeros <= dec16MaxLeadingZerosPlain {
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

// encodeDecFloat16Binary encodes DecFloat16 into IEEE-754 decimal64
func encodeDecFloat16Binary(buf []byte, d DecFloat16) ([]byte, error) {
	if !d.Valid {
		return nil, nil
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal64)

	if d.Special != SpecialNone {
		dd := decfloat.DecodedDecimal{
			Signum: decfloat.SignumPositive,
			Type:   decfloat.DecimalFinite,
		}

		switch d.Special {
		case SpecialPosInf:
			dd.Type = decfloat.DecimalInfinity
			dd.Signum = decfloat.SignumPositive
		case SpecialNegInf:
			dd.Type = decfloat.DecimalInfinity
			dd.Signum = decfloat.SignumNegative
		case SpecialQuietNaN:
			dd.Type = decfloat.DecimalNaN
			dd.Signum = decfloat.SignumPositive
		case SpecialSignalingNaN:
			dd.Type = decfloat.DecimalSignalingNaN
			dd.Signum = decfloat.SignumPositive
		default:
			return nil, fmt.Errorf("encodeDecFloat16Binary: unknown special %v", d.Special)
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
		coef = new(big.Int).Abs(coef)
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

// decodeDecFloat16Binary decodes IEEE-754 decimal64 to DecFloat16
func decodeDecFloat16Binary(src []byte, dst *DecFloat16) error {
	if len(src) != 8 {
		return fmt.Errorf("invalid length for DECFLOAT(16): expected 8 bytes, got %d", len(src))
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal64)

	dd, err := codec.ParseBytes(src)
	if err != nil {
		return err
	}

	if dd.Type != decfloat.DecimalFinite {
		*dst = DecFloat16{
			Valid: true,
		}

		switch dd.Type {
		case decfloat.DecimalInfinity:
			if dd.Signum == decfloat.SignumNegative {
				dst.Special = SpecialNegInf
			} else {
				dst.Special = SpecialPosInf
			}
		case decfloat.DecimalNaN:
			dst.Special = SpecialQuietNaN
		case decfloat.DecimalSignalingNaN:
			dst.Special = SpecialSignalingNaN
		default:
			return fmt.Errorf("decodeDecFloat16Binary: unknown decimal type %v", dd.Type)
		}

		return nil
	}

	coef := dd.Coefficient
	if coef == nil {
		coef = big.NewInt(0)
	}

	if dd.Signum == decfloat.SignumNegative && coef.Sign() > 0 {
		coef = new(big.Int).Neg(coef)
	}

	*dst = DecFloat16{
		Int:     coef,
		Exp:     dd.Exponent,
		Valid:   true,
		Special: SpecialNone,
	}
	return nil
}
