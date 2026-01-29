package fbtype_test

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestInt128Transcode(t *testing.T) {
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
