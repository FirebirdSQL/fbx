package fbtype_test

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestInt128Transcode(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("int128 is not supported by server engine version <= 3")
		}
	})

	max := new(big.Int).Lsh(big.NewInt(1), 127)
	max.Sub(max, big.NewInt(1))

	min := new(big.Int).Lsh(big.NewInt(1), 127)
	min.Neg(min)

	pos := new(big.Int).Lsh(big.NewInt(1), 100)
	pos.Add(pos, big.NewInt(12345))

	neg := new(big.Int).Neg(pos)

	int64Max := big.NewInt(math.MaxInt64)
	int64Min := big.NewInt(math.MinInt64)
	uint64Max := new(big.Int).SetUint64(math.MaxUint64)

	int32Max := big.NewInt(math.MaxInt32)
	int32Min := big.NewInt(math.MinInt32)
	uint32Max := big.NewInt(math.MaxUint32)

	int16Max := big.NewInt(math.MaxInt16)
	int16Min := big.NewInt(math.MinInt16)
	uint16Max := big.NewInt(math.MaxUint16)

	int8Max := big.NewInt(math.MaxInt8)
	int8Min := big.NewInt(math.MinInt8)
	uint8Max := big.NewInt(math.MaxUint8)

	fbxtest.RunValueRoundTripTests(
		context.Background(),
		t,
		defaultConnTestRunner,
		nil,
		"int128",
		[]fbxtest.ValueRoundTripTest{
			{Param: int64(0), Result: new(int64), Test: isExpectedEq(int64(0))},
			{Param: int64(1), Result: new(int64), Test: isExpectedEq(int64(1))},
			{Param: int64(-1), Result: new(int64), Test: isExpectedEq(int64(-1))},
			{Param: int64(42), Result: new(uint64), Test: isExpectedEq(uint64(42))},
			{Param: int64(-42), Result: new(int64), Test: isExpectedEq(int64(-42))},
			{Param: int64(math.MaxInt64), Result: new(int64), Test: isExpectedEq(int64(math.MaxInt64))},
			{Param: int64(math.MinInt64), Result: new(int64), Test: isExpectedEq(int64(math.MinInt64))},
			{Param: makeInt128(int64Max), Result: new(int64), Test: isExpectedEq(int64(math.MaxInt64))},
			{Param: makeInt128(int64Min), Result: new(int64), Test: isExpectedEq(int64(math.MinInt64))},
			{Param: int64(0), Result: new(uint64), Test: isExpectedEq(uint64(0))},
			{Param: int64(42), Result: new(uint64), Test: isExpectedEq(uint64(42))},
			{Param: int64(math.MaxInt64), Result: new(uint64), Test: isExpectedEq(uint64(math.MaxInt64))},
			{Param: makeInt128(uint64Max), Result: new(uint64), Test: isExpectedEq(uint64(math.MaxUint64))},
			{Param: int32(0), Result: new(int32), Test: isExpectedEq(int32(0))},
			{Param: int32(math.MaxInt32), Result: new(int32), Test: isExpectedEq(int32(math.MaxInt32))},
			{Param: int32(math.MinInt32), Result: new(int32), Test: isExpectedEq(int32(math.MinInt32))},
			{Param: makeInt128(int32Max), Result: new(int32), Test: isExpectedEq(int32(math.MaxInt32))},
			{Param: makeInt128(int32Min), Result: new(int32), Test: isExpectedEq(int32(math.MinInt32))},
			{Param: int32(0), Result: new(uint32), Test: isExpectedEq(uint32(0))},
			{Param: int32(123), Result: new(uint32), Test: isExpectedEq(uint32(123))},
			{Param: makeInt128(uint32Max), Result: new(uint32), Test: isExpectedEq(uint32(math.MaxUint32))},
			{Param: int16(0), Result: new(int16), Test: isExpectedEq(int16(0))},
			{Param: int16(math.MaxInt16), Result: new(int16), Test: isExpectedEq(int16(math.MaxInt16))},
			{Param: int16(math.MinInt16), Result: new(int16), Test: isExpectedEq(int16(math.MinInt16))},
			{Param: makeInt128(int16Max), Result: new(int16), Test: isExpectedEq(int16(math.MaxInt16))},
			{Param: makeInt128(int16Min), Result: new(int16), Test: isExpectedEq(int16(math.MinInt16))},
			{Param: int16(0), Result: new(uint16), Test: isExpectedEq(uint16(0))},
			{Param: int16(255), Result: new(uint16), Test: isExpectedEq(uint16(255))},
			{Param: makeInt128(uint16Max), Result: new(uint16), Test: isExpectedEq(uint16(math.MaxUint16))},
			{Param: int16(0), Result: new(int8), Test: isExpectedEq(int8(0))},
			{Param: int16(math.MaxInt8), Result: new(int8), Test: isExpectedEq(int8(math.MaxInt8))},
			{Param: int16(math.MinInt8), Result: new(int8), Test: isExpectedEq(int8(math.MinInt8))},
			{Param: makeInt128(int8Max), Result: new(int8), Test: isExpectedEq(int8(math.MaxInt8))},
			{Param: makeInt128(int8Min), Result: new(int8), Test: isExpectedEq(int8(math.MinInt8))},
			{Param: int16(0), Result: new(uint8), Test: isExpectedEq(uint8(0))},
			{Param: int16(123), Result: new(uint8), Test: isExpectedEq(uint8(123))},
			{Param: makeInt128(uint8Max), Result: new(uint8), Test: isExpectedEq(uint8(math.MaxUint8))},
			{Param: int(0), Result: new(int), Test: isExpectedEq(int(0))},
			{Param: int(math.MaxInt), Result: new(int), Test: isExpectedEq(int(math.MaxInt))},
			{Param: int(math.MinInt), Result: new(int), Test: isExpectedEq(int(math.MinInt))},
			{Param: int(0), Result: new(uint), Test: isExpectedEq(uint(0))},
			{Param: int(12345), Result: new(uint), Test: isExpectedEq(uint(12345))},
			{Param: makeInt128(pos), Result: new(big.Int), Test: isExpectedInt128(pos, true)},
			{Param: makeInt128(neg), Result: new(big.Int), Test: isExpectedInt128(neg, true)},
			{Param: makeInt128(max), Result: new(big.Int), Test: isExpectedInt128(max, true)},
			{Param: makeInt128(min), Result: new(big.Int), Test: isExpectedInt128(min, true)},
			{Param: makeInt128(big.NewInt(0)), Result: new(fbtype.Int128), Test: isExpectedInt128(big.NewInt(0), true)},
			{Param: makeInt128(big.NewInt(1)), Result: new(fbtype.Int128), Test: isExpectedInt128(big.NewInt(1), true)},
			{Param: makeInt128(big.NewInt(-1)), Result: new(fbtype.Int128), Test: isExpectedInt128(big.NewInt(-1), true)},
			{Param: makeInt128(pos), Result: new(fbtype.Int128), Test: isExpectedInt128(pos, true)},
			{Param: makeInt128(neg), Result: new(fbtype.Int128), Test: isExpectedInt128(neg, true)},
			{Param: makeInt128(max), Result: new(fbtype.Int128), Test: isExpectedInt128(max, true)},
			{Param: makeInt128(min), Result: new(fbtype.Int128), Test: isExpectedInt128(min, true)},
			{Param: makeInt128(int64Max), Result: new(fbtype.Int128), Test: isExpectedInt128(int64Max, true)},
			{Param: makeInt128(int64Min), Result: new(fbtype.Int128), Test: isExpectedInt128(int64Min, true)},
			{Param: makeInt128(uint64Max), Result: new(fbtype.Int128), Test: isExpectedInt128(uint64Max, true)},
			{
				Param:  mustParseNumeric(t, "9223372036854775808.999"),
				Result: new(big.Int),
				Test:   isExpectedInt128(mustBigInt(t, "9223372036854775808"), true),
			},
			{
				Param:  mustParseNumeric(t, "-9223372036854775809.999"),
				Result: new(big.Int),
				Test:   isExpectedInt128(mustBigInt(t, "-9223372036854775809"), true),
			},
			{
				Param:  mustParseNumeric(t, "-0.999"),
				Result: new(big.Int),
				Test:   isExpectedInt128(big.NewInt(0), true),
			},
			{Param: fbtype.Numeric{}, Result: new(fbtype.Int128), Test: isExpectedInt128(nil, false)},
			{Param: nil, Result: new(fbtype.Int128), Test: isExpectedInt128(nil, false)},
		},
	)
}

func TestInt128_ScanValueJSON(t *testing.T) {
	var a fbtype.Int128
	require.NoError(t, a.Scan("123"))
	require.True(t, a.Valid)
	require.Equal(t, "123", a.String())

	require.NoError(t, a.Scan(nil))
	require.False(t, a.Valid)

	b, err := fbtype.ParseInt128("-999")
	require.NoError(t, err)

	v, err := b.Value()
	require.NoError(t, err)
	require.Equal(t, "-999", v)

	raw, err := json.Marshal(b)
	require.NoError(t, err)
	require.Equal(t, []byte("-999"), raw)

	var c fbtype.Int128
	require.NoError(t, json.Unmarshal([]byte("170141183460469231731687303715884105727"), &c))
	require.True(t, c.Valid)
	require.Equal(t, "170141183460469231731687303715884105727", c.String())

	var cMin fbtype.Int128
	require.NoError(t, json.Unmarshal([]byte("-170141183460469231731687303715884105728"), &cMin))
	require.True(t, cMin.Valid)
	require.Equal(t, "-170141183460469231731687303715884105728", cMin.String())

	var d fbtype.Int128
	require.NoError(t, json.Unmarshal([]byte("\"42\""), &d))
	require.True(t, d.Valid)
	require.Equal(t, "42", d.String())

	var e fbtype.Int128
	require.NoError(t, json.Unmarshal([]byte("null"), &e))
	require.False(t, e.Valid)
}

func TestInt128_ScanErrors(t *testing.T) {
	var a fbtype.Int128

	err := a.Scan("170141183460469231731687303715884105728")
	require.Error(t, err)

	err = a.Scan("-170141183460469231731687303715884105729")
	require.Error(t, err)

	err = a.Scan("not-a-number")
	require.Error(t, err)

	err = a.Scan(3.14)
	require.Error(t, err)
}

func TestInt128_BigIntScan(t *testing.T) {
	max := new(big.Int).Lsh(big.NewInt(1), 127)
	max.Sub(max, big.NewInt(1))

	min := new(big.Int).Lsh(big.NewInt(1), 127)
	min.Neg(min)

	var a fbtype.Int128

	require.NoError(t, a.Scan(max))
	require.True(t, a.Valid)
	require.Equal(t, 0, a.Int.Cmp(max))

	var minCopy big.Int
	minCopy.Set(min)
	require.NoError(t, a.Scan(minCopy))
	require.True(t, a.Valid)
	require.Equal(t, 0, a.Int.Cmp(min))

	require.NoError(t, a.Scan((*big.Int)(nil)))
	require.False(t, a.Valid)

	tooLarge := new(big.Int).Lsh(big.NewInt(1), 128)
	err := a.Scan(tooLarge)
	require.Error(t, err)
}

func TestInt128_Int64Scan(t *testing.T) {
	var a fbtype.Int128

	require.NoError(t, a.Scan(int64(123)))
	require.True(t, a.Valid)
	require.Equal(t, "123", a.String())

	require.NoError(t, a.Scan(int64(-456)))
	require.True(t, a.Valid)
	require.Equal(t, "-456", a.String())

	require.NoError(t, a.Scan(int64(math.MaxInt64)))
	require.True(t, a.Valid)
	require.Equal(t, "9223372036854775807", a.String())

	require.NoError(t, a.Scan(int64(math.MinInt64)))
	require.True(t, a.Valid)
	require.Equal(t, "-9223372036854775808", a.String())
}

func TestInt128_ParseErrors(t *testing.T) {
	_, err := fbtype.ParseInt128("abc")
	require.Error(t, err)

	_, err = fbtype.ParseInt128("999999999999999999999999999999999999999")
	require.Error(t, err)

	_, err = fbtype.ParseInt128("")
	require.Error(t, err)
}

func TestInt128CodecEncodeIntegerInputs(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 0, 16, 0)

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "int8", value: int8(-5), want: "-5"},
		{name: "uint8", value: uint8(math.MaxUint8), want: "255"},
		{name: "uint16", value: uint16(math.MaxUint16), want: "65535"},
		{name: "uint32", value: uint32(math.MaxUint32), want: "4294967295"},
		{name: "uint64", value: uint64(math.MaxUint64), want: "18446744073709551615"},
		{name: "uint", value: uint(math.MaxUint), want: new(big.Int).SetUint64(uint64(math.MaxUint)).String()},
		{name: "Int64Valuer", value: fbtype.Int8{Int64: math.MinInt64, Valid: true}, want: "-9223372036854775808"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := m.Encode(typeMetadata, tt.value, nil)
			require.NoError(t, err)
			require.Len(t, buf, 16)

			var got big.Int
			err = m.Scan(typeMetadata, buf, &got)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.String())
		})
	}

	buf, err := m.Encode(typeMetadata, fbtype.Int8{}, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

func TestInt128CodecEncodesNumericWithTruncation(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 0, 16, 0)

	tests := []struct {
		name  string
		value fbtype.Numeric
		want  string
	}{
		{name: "above int64", value: mustParseNumeric(t, "9223372036854775808.999"), want: "9223372036854775808"},
		{name: "negative above int64", value: mustParseNumeric(t, "-9223372036854775809.999"), want: "-9223372036854775809"},
		{name: "positive below one", value: mustParseNumeric(t, "0.999"), want: "0"},
		{name: "negative above minus one", value: mustParseNumeric(t, "-0.999"), want: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := m.Encode(typeMetadata, tt.value, nil)
			require.NoError(t, err)

			var got big.Int
			err = m.Scan(typeMetadata, buf, &got)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.String())
		})
	}

	buf, err := m.Encode(typeMetadata, fbtype.Numeric{}, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

func TestInt128ScanClearsDestinationOnError(t *testing.T) {
	tooLarge := new(big.Int).Lsh(big.NewInt(1), 127)

	tests := []struct {
		name string
		scan func(*fbtype.Int128) error
	}{
		{name: "invalid string", scan: func(dst *fbtype.Int128) error {
			return dst.Scan("not-a-number")
		}},
		{name: "string out of range", scan: func(dst *fbtype.Int128) error {
			return dst.Scan("170141183460469231731687303715884105728")
		}},
		{name: "big.Int out of range", scan: func(dst *fbtype.Int128) error {
			return dst.Scan(tooLarge)
		}},
		{name: "*big.Int out of range", scan: func(dst *fbtype.Int128) error {
			return dst.Scan(new(big.Int).Set(tooLarge))
		}},
		{name: "Int128 out of range", scan: func(dst *fbtype.Int128) error {
			return dst.ScanInt128(makeInt128(tooLarge))
		}},
		{name: "unsupported source type", scan: func(dst *fbtype.Int128) error {
			return dst.Scan(3.14)
		}},
		{name: "invalid JSON", scan: func(dst *fbtype.Int128) error {
			return json.Unmarshal([]byte(`"not-a-number"`), dst)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeInt128(big.NewInt(42))
			err := tt.scan(&got)
			require.Error(t, err)
			require.False(t, got.Valid)
			require.Equal(t, "0", got.Int.String())
		})
	}
}

func TestInt128RejectsOutOfRangeStructValues(t *testing.T) {
	tooLarge := makeInt128(new(big.Int).Lsh(big.NewInt(1), 127))

	_, err := tooLarge.Int128Value()
	require.Error(t, err)

	_, err = tooLarge.Value()
	require.Error(t, err)

	_, err = tooLarge.MarshalJSON()
	require.Error(t, err)

	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 0, 16, 0)
	_, err = m.Encode(typeMetadata, tooLarge, nil)
	require.Error(t, err)
}

func TestInt128UnmarshalJSONWhitespaceNull(t *testing.T) {
	got := makeInt128(big.NewInt(42))
	err := json.Unmarshal([]byte(" null "), &got)
	require.NoError(t, err)
	require.False(t, got.Valid)
}

func TestInt128ScanPlansReportTargetTypeChanged(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 0, 16, 0)
	src, err := m.Encode(typeMetadata, int64(42), nil)
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		target any
	}{
		{name: "int8", target: new(int8)},
		{name: "int16", target: new(int16)},
		{name: "int32", target: new(int32)},
		{name: "int64", target: new(int64)},
		{name: "int", target: new(int)},
		{name: "uint8", target: new(uint8)},
		{name: "uint16", target: new(uint16)},
		{name: "uint32", target: new(uint32)},
		{name: "uint64", target: new(uint64)},
		{name: "uint", target: new(uint)},
		{name: "string", target: new(string)},
		{name: "big.Int", target: new(big.Int)},
		{name: "Int128Scanner", target: new(fbtype.Int128)},
		{name: "TextScanner", target: new(fbtype.Text)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := m.PlanScan(typeMetadata, tt.target)
			require.NotNil(t, plan)

			var wrongTarget float64
			err := plan.Scan(src, &wrongTarget)
			require.ErrorIs(t, err, fbtype.ErrScanTargetTypeChanged)
		})
	}
}

func TestInt128ScanNullErrorsUseTargetType(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 0, 16, 0)

	var i32 int32
	err := m.PlanScan(typeMetadata, &i32).Scan(nil, &i32)
	require.ErrorContains(t, err, "cannot scan NULL into *int32")
	require.NotContains(t, err.Error(), "*int64")

	var u32 uint32
	err = m.PlanScan(typeMetadata, &u32).Scan(nil, &u32)
	require.ErrorContains(t, err, "cannot scan NULL into *uint32")
	require.NotContains(t, err.Error(), "*uint64")
}

func TestInt128ScanRangeErrorsUseTargetType(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 0, 16, 0)

	huge := new(big.Int).Lsh(big.NewInt(1), 100)
	src, err := m.Encode(typeMetadata, makeInt128(huge), nil)
	require.NoError(t, err)

	var i32 int32
	err = m.Scan(typeMetadata, src, &i32)
	require.ErrorContains(t, err, "value is out of range for int32")
	require.NotContains(t, err.Error(), "int64")

	negativeOne := big.NewInt(-1)
	src, err = m.Encode(typeMetadata, makeInt128(negativeOne), nil)
	require.NoError(t, err)

	var u32 uint32
	err = m.Scan(typeMetadata, src, &u32)
	require.ErrorContains(t, err, "value is out of range for uint32")
	require.NotContains(t, err.Error(), "uint64")
}

func isExpectedInt128(expected *big.Int, valid bool) func(any) bool {
	return func(v any) bool {
		switch x := v.(type) {
		case fbtype.Int128:
			if x.Valid != valid {
				return false
			}
			if !valid {
				return true
			}
			return x.Int.Cmp(expected) == 0
		case big.Int:
			if !valid {
				return false
			}
			return x.Cmp(expected) == 0
		default:
			return false
		}
	}
}

func makeInt128(v *big.Int) fbtype.Int128 {
	var z big.Int
	z.Set(v)

	return fbtype.Int128{Int: z, Valid: true}
}
