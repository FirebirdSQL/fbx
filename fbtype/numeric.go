package fbtype

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

var big0 *big.Int = big.NewInt(0)
var big10 *big.Int = big.NewInt(10)

type NumericScanner interface {
	ScanNumeric(v Numeric) error
}

type NumericValuer interface {
	NumericValue() (Numeric, error)
}

type Numeric struct {
	Int   *big.Int
	Exp   int32
	Valid bool
}

func (n *Numeric) ScanNumeric(v Numeric) error {
	*n = v
	return nil
}

func (n Numeric) NumericValue() (Numeric, error) {
	return n, nil
}

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

func (n *Numeric) ScanInt64(v Int8) error {
	if !v.Valid {
		*n = Numeric{}
		return nil
	}

	*n = Numeric{Int: big.NewInt(v.Int64), Valid: true}
	return nil
}

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

func (n *Numeric) ScanScientific(src string) error {
	if !strings.ContainsAny(src, "eE") {
		return scanPlanTextAnyToNumericScanner{}.Scan([]byte(src), n)
	}

	if bigF, ok := new(big.Float).SetString(string(src)); ok {
		smallF, _ := bigF.Float64()
		src = strconv.FormatFloat(smallF, 'f', -1, 64)
	}

	num, exp, err := parseNumericString(src)
	if err != nil {
		return err
	}

	*n = Numeric{Int: num, Exp: exp, Valid: true}

	return nil
}

func (n *Numeric) toBigInt() (*big.Int, error) {
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
	if remainder.Cmp(big0) != 0 {
		return nil, fmt.Errorf("cannot convert %v to integer", n)
	}
	return num, nil
}

func parseNumericString(str string) (n *big.Int, exp int32, err error) {
	idx := strings.IndexByte(str, '.')

	if idx == -1 {
		for len(str) > 1 && str[len(str)-1] == '0' && str[len(str)-2] != '-' {
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
	case string:
		return scanPlanTextAnyToNumericScanner{}.Scan([]byte(src), n)
	}

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
	return string(buf), err
}

func (n Numeric) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}

	return n.numberTextBytes(), nil
}

func (n *Numeric) UnmarshalJSON(src []byte) error {
	if bytes.Equal(src, []byte(`null`)) {
		*n = Numeric{}
		return nil
	}

	return scanPlanTextAnyToNumericScanner{}.Scan(src, n)
}

func (n Numeric) numberTextBytes() []byte {
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

type NumericCodec struct{}

func (NumericCodec) FormatSupported(format int16) bool {
	return format == TextFormatCode || format == BinaryFormatCode
}

func (NumericCodec) PreferredFormat() int16 {
	return BinaryFormatCode
}

func (NumericCodec) PlanEncode(m *Map, oid uint32, subType int32, length uint32, scale int32, format int16, value any) EncodePlan {
	switch value.(type) {
	case NumericValuer:
		return encodePlanNumericCodecBinaryNumericValuer{scale: scale, length: length}
	case Float64Valuer:
		return encodePlanNumericCodecBinaryFloat64Valuer{scale: scale, length: length}
	case Int64Valuer:
		return encodePlanNumericCodecBinaryInt64Valuer{scale: scale, length: length}
	case string:
		return encodePlanNumericCodecTextString{scale: scale, length: length}
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

func (NumericCodec) PlanScan(m *Map, typeId uint32, subType int32, length uint32, scale int32, format int16, target any) ScanPlan {
	switch format {
	case BinaryFormatCode:
		switch target.(type) {
		case *Numeric:
			return scanPlanBinaryNumericToNumeric{
				scale:  scale,
				length: length,
			}
		case NumericScanner:
			return scanPlanBinaryNumericToNumericScanner{
				scale:  scale,
				length: length,
			}
		case Float64Scanner:
			return scanPlanBinaryNumericToFloat64Scanner{
				scale:  scale,
				length: length,
			}
		case Int64Scanner:
			return scanPlanBinaryNumericToInt64Scanner{
				scale:  scale,
				length: length,
			}
		case TextScanner:
			return scanPlanBinaryNumericToTextScanner{
				scale:  scale,
				length: length,
			}
		}
	case TextFormatCode:
		switch target.(type) {
		case NumericScanner:
			return scanPlanTextAnyToNumericScanner{}
		case Float64Scanner:
			return scanPlanTextAnyToFloat64Scanner{}
		case Int64Scanner:
			return scanPlanTextAnyToInt64Scanner{}
		}
	}

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

	return DecodeNumericBinaryFB(nPtr, src, p.scale)
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
		return scanner.ScanInt64(Int8{}) // NULL
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

type scanPlanTextAnyToNumericScanner struct{}

func (scanPlanTextAnyToNumericScanner) Scan(src []byte, dst any) error {
	scanner := (dst).(NumericScanner)

	if src == nil {
		return scanner.ScanNumeric(Numeric{})
	}

	num, exp, err := parseNumericString(string(src))
	if err != nil {
		return err
	}

	return scanner.ScanNumeric(Numeric{Int: num, Exp: exp, Valid: true})
}

func (c NumericCodec) DecodeDatabaseSQLValue(m *Map, oid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	if format == TextFormatCode {
		return string(src), nil
	}

	var n Numeric
	err := codecScan(c, m, oid, subType, length, scale, format, src, &n)
	if err != nil {
		return nil, err
	}

	buf, err := m.Encode(oid, subType, length, scale, TextFormatCode, n, nil)
	if err != nil {
		return nil, err
	}
	return string(buf), nil
}

func (c NumericCodec) DecodeValue(m *Map, oid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var n Numeric
	err := codecScan(c, m, oid, subType, length, scale, format, src, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func DecodeNumericBinaryFB(dst *Numeric, src []byte, scale int32) error {
	if src == nil {
		*dst = Numeric{}
		return nil
	}

	var s int64
	switch len(src) {
	case 4:
		s = int64(int32(binary.BigEndian.Uint32(src)))
	case 8:
		s = int64(binary.BigEndian.Uint64(src))
	default:
		return fmt.Errorf("unsupported numeric length: %d", len(src))
	}

	*dst = Numeric{
		Int:   big.NewInt(s),
		Exp:   scale,
		Valid: true,
	}
	return nil
}

func EncodeNumericBinaryFB(buf []byte, n Numeric, scale int32, length uint32) ([]byte, error) {
	if !n.Valid {
		return nil, nil
	}

	expDiff := int64(n.Exp - scale)

	var scaled big.Int
	scaled.Set(n.Int)

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

	if !scaled.IsInt64() {
		return nil, fmt.Errorf("numeric value %v is out of range for int64", &n)
	}
	val := scaled.Int64()

	return appendFBScaledIntByLength(buf, val, length)
}

func float64ToNumeric(f float64) (Numeric, error) {
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

func EncodeNumericBinaryFBRoundHalfAwayFromZero(buf []byte, n Numeric, scale int32, length uint32) ([]byte, error) {
	if !n.Valid {
		return nil, nil
	}

	expDiff := int64(n.Exp - scale)

	var scaled big.Int
	scaled.Set(n.Int)

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

	if !scaled.IsInt64() {
		return nil, fmt.Errorf("numeric value %v is out of range for int64", &n)
	}
	val := scaled.Int64()

	return appendFBScaledIntByLength(buf, val, length)
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

	default:
		return nil, fmt.Errorf("unsupported numeric length %d", length)
	}
}
