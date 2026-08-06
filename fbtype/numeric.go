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
)

var big10 *big.Int = big.NewInt(10)

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
	*n = cloneNumeric(v)
	return nil
}

// NumericValue implements [NumericValuer].
func (n Numeric) NumericValue() (Numeric, error) {
	return cloneNumeric(n), nil
}

// Float64Value implements [Float64Valuer].
func (n Numeric) Float64Value() (Float8, error) {
	if !n.Valid {
		return Float8{}, nil
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

// Int64Value implements [Int64Valuer].
func (n Numeric) Int64Value() (Int8, error) {
	if !n.Valid {
		return Int8{}, nil
	}

	bi, err := n.toBigInt()
	if err != nil {
		return Int8{}, err
	}

	if !bi.IsInt64() {
		return Int8{}, fmt.Errorf("cannot convert %v to int64", n)
	}

	return Int8{Int64: bi.Int64(), Valid: true}, nil
}

// String returns the decimal representation of n, or an empty string for SQL NULL.
func (n Numeric) String() string {
	if !n.Valid {
		return ""
	}

	buf, err := encodeNumericText(n, nil)
	if err != nil {
		return ""
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
	if n.Int == nil {
		return big.NewInt(0), nil
	}

	if n.Exp == 0 {
		return n.Int, nil
	}

	num := &big.Int{}
	num.Set(n.Int)
	if n.Exp > 0 {
		mul := &big.Int{}
		mul.Exp(big10, big.NewInt(int64(n.Exp)), nil)
		num.Mul(num, mul)
		return num, nil
	}

	div := &big.Int{}
	div.Exp(big10, big.NewInt(int64(-n.Exp)), nil)
	remainder := &big.Int{}
	num.DivMod(num, div, remainder)
	if remainder.Sign() != 0 {
		return nil, fmt.Errorf("cannot convert %v to integer", n)
	}
	return num, nil
}

func parseNumericString(str string) (n *big.Int, exp int32, err error) {
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
		if err != nil {
			*n = Numeric{}
			return err
		}
		*n = parsed
		return nil
	case big.Int:
		var z big.Int
		z.Set(&src)
		*n = Numeric{Int: &z, Valid: true}
		return nil
	case *big.Int:
		if src == nil {
			*n = Numeric{}
			return nil
		}
		var z big.Int
		z.Set(src)
		*n = Numeric{Int: &z, Valid: true}
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

	return n.numberTextBytes(), nil
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

	exp := int(n.Exp)
	if exp > 0 {
		buf.WriteString(intStr)
		for i := 0; i < exp; i++ {
			buf.WriteByte('0')
		}
	} else if exp < 0 {
		if len(intStr) <= -exp {
			buf.WriteString("0.")
			leadingZeros := -exp - len(intStr)
			for i := 0; i < leadingZeros; i++ {
				buf.WriteByte('0')
			}
			buf.WriteString(intStr)
		} else if len(intStr) > -exp {
			dpPos := len(intStr) + exp
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

// PlanEncode returns an encode plan for supported Firebird NUMERIC and DECIMAL input values.
func (NumericCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case NumericValuer:
		return encodePlanNumericCodecBinaryNumericValuer{scale: typeMetadata.Scale, length: typeMetadata.Length}
	case Float64Valuer:
		return encodePlanNumericCodecBinaryFloat64Valuer{scale: typeMetadata.Scale, length: typeMetadata.Length}
	case Int64Valuer:
		return encodePlanNumericCodecBinaryInt64Valuer{scale: typeMetadata.Scale, length: typeMetadata.Length}
	case string:
		return encodePlanNumericCodecTextString{scale: typeMetadata.Scale, length: typeMetadata.Length}
	}

	return nil
}

type encodePlanNumericCodecBinaryNumericValuer struct {
	scale  int32
	length uint32
}

func (p encodePlanNumericCodecBinaryNumericValuer) Encode(value any, buf []byte) ([]byte, error) {
	n, err := value.(NumericValuer).NumericValue()
	if err != nil {
		return nil, err
	}

	return EncodeNumericBinaryFB(buf, n, p.scale, p.length)
}

type encodePlanNumericCodecBinaryFloat64Valuer struct {
	scale  int32
	length uint32
}

func (p encodePlanNumericCodecBinaryFloat64Valuer) Encode(value any, buf []byte) ([]byte, error) {
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

	return EncodeNumericBinaryFBRoundHalfAwayFromZero(buf, n, p.scale, p.length)
}

type encodePlanNumericCodecBinaryInt64Valuer struct {
	scale  int32
	length uint32
}

func (p encodePlanNumericCodecBinaryInt64Valuer) Encode(value any, buf []byte) ([]byte, error) {
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

	return EncodeNumericBinaryFB(buf, n, p.scale, p.length)
}

type encodePlanNumericCodecTextString struct {
	scale  int32
	length uint32
}

func (p encodePlanNumericCodecTextString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)

	var n Numeric
	if err := n.ScanScientific(s); err != nil {
		return nil, err
	}
	n.Valid = true

	return EncodeNumericBinaryFB(buf, n, p.scale, p.length)
}

func encodeNumericText(n Numeric, buf []byte) (newBuf []byte, err error) {
	if !n.Valid {
		return nil, nil
	}

	buf = append(buf, n.numberTextBytes()...)

	return buf, nil
}

// PlanScan returns a scan plan for supported Firebird NUMERIC and DECIMAL target values.
func (NumericCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *Numeric:
		return scanPlanBinaryNumericToNumeric{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	case NumericScanner:
		return scanPlanBinaryNumericToNumericScanner{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	case Float64Scanner:
		return scanPlanBinaryNumericToFloat64Scanner{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	case Int64Scanner:
		return scanPlanBinaryNumericToInt64Scanner{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	case TextScanner:
		return scanPlanBinaryNumericToTextScanner{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	case *string:
		return scanPlanBinaryNumericToString{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	case *float64:
		return scanPlanBinaryNumericToFloat64Ptr{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	case *int64:
		return scanPlanBinaryNumericToInt64Ptr{
			scale:  typeMetadata.Scale,
			length: typeMetadata.Length,
		}
	}

	return nil
}

type scanPlanBinaryNumericToFloat64Ptr struct {
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
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
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
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
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
		*nPtr = Numeric{}
		return err
	}

	*nPtr = n
	return nil
}

type scanPlanBinaryNumericToNumericScanner struct {
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
		return err
	}

	return scanner.ScanNumeric(n)
}

type scanPlanBinaryNumericToFloat64Scanner struct {
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
		return err
	}

	f8, err := n.Float64Value()
	if err != nil {
		return err
	}

	return scanner.ScanFloat64(f8)
}

type scanPlanBinaryNumericToInt64Scanner struct {
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
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
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
		return err
	}

	sbuf, err := encodeNumericText(n, nil)
	if err != nil {
		return err
	}

	return scanner.ScanText(Text{String: string(sbuf), Valid: true})
}

type scanPlanBinaryNumericToString struct {
	scale  int32
	length uint32
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
	if err := DecodeNumericBinaryFB(&n, src, p.scale); err != nil {
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

	var bigInt *big.Int
	switch len(src) {
	case 4:
		s := int64(int32(binary.BigEndian.Uint32(src)))
		bigInt = big.NewInt(s)
	case 8:
		s := int64(binary.BigEndian.Uint64(src))
		bigInt = big.NewInt(s)
	case 16:
		bigInt = new(big.Int)
		if err := int128SetFromBinary(bigInt, src); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported numeric length: %d", len(src))
	}

	*dst = Numeric{
		Int:   bigInt,
		Exp:   scale,
		Valid: true,
	}
	return nil
}

// EncodeNumericBinaryFB appends the Firebird binary encoding of n to buf.
func EncodeNumericBinaryFB(buf []byte, n Numeric, scale int32, length uint32) ([]byte, error) {
	if !n.Valid {
		return nil, nil
	}

	expDiff := int64(n.Exp) - int64(scale)

	var scaled big.Int
	if n.Int != nil {
		scaled.Set(n.Int)
	}

	if expDiff > 0 {
		var pow big.Int
		pow.Exp(big.NewInt(10), big.NewInt(expDiff), nil)
		scaled.Mul(&scaled, &pow)
	} else if expDiff < 0 {
		var pow big.Int
		pow.Exp(big.NewInt(10), big.NewInt(-expDiff), nil)
		var rem big.Int
		var q big.Int
		q.QuoRem(&scaled, &pow, &rem)
		if rem.Sign() != 0 {
			return nil, fmt.Errorf("numeric value %v cannot be represented exactly with scale %d", &n, scale)
		}
		scaled.Set(&q)
	}

	if scaled.IsInt64() {
		val := scaled.Int64()
		return appendFBScaledIntByLength(buf, val, length)
	}

	if length == 16 {
		if err := checkInt128Range(&scaled); err != nil {
			return nil, fmt.Errorf("numeric value %v is out of range for int128: %w", &n, err)
		}
		return appendInt128BinaryFromBig(buf, &scaled)
	}

	return nil, fmt.Errorf("numeric value %v is out of range for length %d", &n, length)
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

	expDiff := int64(n.Exp) - int64(scale)

	var scaled big.Int
	if n.Int != nil {
		scaled.Set(n.Int)
	}

	if expDiff > 0 {
		var pow big.Int
		pow.Exp(big.NewInt(10), big.NewInt(expDiff), nil)
		scaled.Mul(&scaled, &pow)
	} else if expDiff < 0 {
		var pow big.Int
		pow.Exp(big.NewInt(10), big.NewInt(-expDiff), nil)

		var rem big.Int
		var q big.Int
		q.QuoRem(&scaled, &pow, &rem)

		if rem.Sign() != 0 {
			absRem := new(big.Int).Abs(&rem)
			twice := new(big.Int).Lsh(absRem, 1)
			cmp := twice.Cmp(&pow)
			if cmp >= 0 {
				if scaled.Sign() >= 0 {
					q.Add(&q, big.NewInt(1))
				} else {
					q.Sub(&q, big.NewInt(1))
				}
			}
		}
		scaled.Set(&q)
	}

	if scaled.IsInt64() {
		val := scaled.Int64()
		return appendFBScaledIntByLength(buf, val, length)
	}

	if length == 16 {
		if err := checkInt128Range(&scaled); err != nil {
			return nil, fmt.Errorf("numeric value %v is out of range for int128: %w", &n, err)
		}
		return appendInt128BinaryFromBig(buf, &scaled)
	}

	return nil, fmt.Errorf("numeric value %v is out of range for length %d", &n, length)
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
		num, exp, err := parseNumericString(s)
		if err != nil {
			return Numeric{}, err
		}
		return Numeric{Int: num, Exp: exp, Valid: true}, nil
	}

	if expIndex != strings.LastIndexAny(s, "eE") {
		return Numeric{}, fmt.Errorf("invalid scientific notation: %s", s)
	}
	if expIndex == 0 || expIndex == len(s)-1 {
		return Numeric{}, fmt.Errorf("invalid scientific notation: %s", s)
	}

	num, exp, err := parseNumericString(s[:expIndex])
	if err != nil {
		return Numeric{}, err
	}

	scientificExp, err := strconv.ParseInt(s[expIndex+1:], 10, 32)
	if err != nil {
		return Numeric{}, fmt.Errorf("invalid scientific notation exponent %q: %w", s[expIndex+1:], err)
	}

	finalExp := int64(exp) + scientificExp
	if finalExp < math.MinInt32 || finalExp > math.MaxInt32 {
		return Numeric{}, fmt.Errorf("numeric exponent %d is out of int32 range", finalExp)
	}

	return Numeric{Int: num, Exp: int32(finalExp), Valid: true}, nil
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
