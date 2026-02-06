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

const (
	dec34MaxIntDigitsPlain    = 34
	dec34MaxLeadingZerosPlain = 6
)

type Dec34Scanner interface {
	ScanDec34(v DecFloat34) error
}

type Dec34Valuer interface {
	Dec34Value() (DecFloat34, error)
}

type DecFloat34 struct {
	Int     *big.Int
	Exp     int32
	Valid   bool
	Special SpecialValue
}

func (d *DecFloat34) ScanDec34(v DecFloat34) error {
	*d = v
	return nil
}

func (d DecFloat34) Dec34Value() (DecFloat34, error) {
	return d, nil
}

// Scan implements the database/sql Scanner interface.
func (d *DecFloat34) Scan(src any) error {
	if src == nil {
		*d = DecFloat34{}
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

// Value implements the database/sql/driver Valuer interface.
func (d DecFloat34) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	return d.String(), nil
}

func (d DecFloat34) String() string {
	if !d.Valid {
		return ""
	}
	if d.Special != SpecialNone {
		return d.Special.String()
	}
	return formatDecFloat34General(d.Int, d.Exp)
}

func (d DecFloat34) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

func (d *DecFloat34) UnmarshalJSON(b []byte) error {
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

func (d DecFloat34) IsSpecial() bool  { return d.Special != SpecialNone }
func (d DecFloat34) IsInfinity() bool { return d.Special == SpecialPosInf || d.Special == SpecialNegInf }
func (d DecFloat34) IsNaN() bool      { return d.Special == SpecialQuietNaN || d.Special == SpecialSignalingNaN }

type DecFloat34Codec struct{}

func (DecFloat34Codec) FormatSupported(format int16) bool { return format == BinaryFormatCode }
func (DecFloat34Codec) PreferredFormat() int16            { return BinaryFormatCode }

func (DecFloat34Codec) PlanEncode(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, value any) EncodePlan {
	if format != BinaryFormatCode {
		return nil
	}

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

func (DecFloat34Codec) PlanScan(m *Map, typeId uint32, subType int32, length uint32, scale int32, format int16, target any) ScanPlan {
	if format != BinaryFormatCode {
		return nil
	}

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
		*strPtr = ""
		return nil
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

func (c DecFloat34Codec) DecodeDatabaseSQLValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var d DecFloat34
	if err := codecScan(c, m, typeid, subType, length, scale, format, src, &d); err != nil {
		return nil, err
	}
	return d.String(), nil
}

func (c DecFloat34Codec) DecodeValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var d DecFloat34
	if err := codecScan(c, m, typeid, subType, length, scale, format, src, &d); err != nil {
		return nil, err
	}
	return d, nil
}

// Float64ToDecFloat34 converts float64 to DecFloat34, handling NaN/Inf explicitly and parsing finite values via decimal text.
func Float64ToDecFloat34(f float64) (DecFloat34, error) {
	if math.IsNaN(f) {
		return DecFloat34{Valid: true, Special: SpecialQuietNaN}, nil
	}
	if math.IsInf(f, 1) {
		return DecFloat34{Valid: true, Special: SpecialPosInf}, nil
	}
	if math.IsInf(f, -1) {
		return DecFloat34{Valid: true, Special: SpecialNegInf}, nil
	}

	s := strconv.FormatFloat(f, 'f', -1, 64)

	var d DecFloat34
	if err := parseDecFloat34String(s, &d); err != nil {
		return DecFloat34{}, fmt.Errorf("cannot convert float64 %f to DecFloat34: %w", f, err)
	}
	return d, nil
}

// decFloat34ToFloat64 converts DecFloat34 to float64 (may lose precision) and returns an error on finite overflow.
func decFloat34ToFloat64(d DecFloat34) (float64, error) {
	if !d.Valid {
		return 0, fmt.Errorf("invalid DecFloat34")
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
				return 0, fmt.Errorf("decfloat34: overflow converting %s to float64", s)
			}
			return f, nil
		}
		return 0, err
	}
	return f, nil
}

// parseDecFloat34String parses a textual DECFLOAT(34) (including scientific notation) into dst.
func parseDecFloat34String(s string, dst *DecFloat34) error {
	original := strings.TrimSpace(s)
	s = original

	switch strings.ToLower(s) {
	case "+infinity", "+inf", "infinity", "inf":
		*dst = DecFloat34{Valid: true, Special: SpecialPosInf}
		return nil
	case "-infinity", "-inf":
		*dst = DecFloat34{Valid: true, Special: SpecialNegInf}
		return nil
	case "nan":
		*dst = DecFloat34{Valid: true, Special: SpecialQuietNaN}
		return nil
	case "snan":
		*dst = DecFloat34{Valid: true, Special: SpecialSignalingNaN}
		return nil
	}

	var exp10 int32
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mant := strings.TrimSpace(s[:i])
		expPart := strings.TrimSpace(s[i+1:])
		if expPart == "" {
			return fmt.Errorf("cannot parse DecFloat34 from %q: missing exponent", original)
		}
		e, err := strconv.ParseInt(expPart, 10, 32)
		if err != nil {
			return fmt.Errorf("cannot parse DecFloat34 from %q: bad exponent %q: %w", original, expPart, err)
		}
		exp10 = int32(e)
		s = mant
	}

	num, exp, err := parseNumericString(s)
	if err != nil {
		return fmt.Errorf("cannot parse DecFloat34 from %q: %w", original, err)
	}

	exp += exp10
	if num.Sign() == 0 {
		exp = 0
	}

	*dst = DecFloat34{
		Int:     num,
		Exp:     exp,
		Valid:   true,
		Special: SpecialNone,
	}
	return nil
}

// formatDecFloat34General formats a finite DecFloat34 using “general” rules: plain form when short, otherwise scientific.
func formatDecFloat34General(coef *big.Int, exp int32) string {
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
	if digitsToLeft > 0 && digitsToLeft <= dec34MaxIntDigitsPlain {
		return formatDecimal(coef, exp)
	}
	if digitsToLeft <= 0 {
		leadingZeros := -digitsToLeft
		if leadingZeros <= dec34MaxLeadingZerosPlain {
			return formatDecimal(coef, exp)
		}
	}
	return formatDecFloatScientific(coef, exp)
}

// encodeDecFloat34Binary encodes DecFloat34 into IEEE-754 decimal128.
func encodeDecFloat34Binary(buf []byte, d DecFloat34) ([]byte, error) {
	if !d.Valid {
		return nil, nil
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal128)

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
		case SpecialSignalingNaN:
			dd.Type = decfloat.DecimalSignalingNaN
		default:
			return nil, fmt.Errorf("encodeDecFloat34Binary: unknown special %v", d.Special)
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

// decodeDecFloat34Binary decodes IEEE-754 decimal128 to DecFloat34.
func decodeDecFloat34Binary(src []byte, dst *DecFloat34) error {
	if len(src) != 16 {
		return fmt.Errorf("invalid length for DECFLOAT(34): expected 16 bytes, got %d", len(src))
	}

	codec := decfloat.NewDecimalCodec(decfloat.Decimal128)

	dd, err := codec.ParseBytes(src)
	if err != nil {
		return err
	}

	if dd.Type != decfloat.DecimalFinite {
		*dst = DecFloat34{Valid: true}

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
			return fmt.Errorf("decodeDecFloat34Binary: unknown decimal type %v", dd.Type)
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

	*dst = DecFloat34{
		Int:     coef,
		Exp:     dd.Exponent,
		Valid:   true,
		Special: SpecialNone,
	}
	return nil
}
