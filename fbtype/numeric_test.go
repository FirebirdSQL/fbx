package fbtype_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func isExpectedEqNumeric(a any) func(any) bool {
	return func(v any) bool {
		aa := a.(fbtype.Numeric)
		vv := v.(fbtype.Numeric)

		if aa.Valid != vv.Valid {
			return false
		}

		// If NULL doesn't matter what the rest of the values are.
		if !aa.Valid {
			return true
		}

		aaInt := (&big.Int{}).Set(aa.Int)
		vvInt := (&big.Int{}).Set(vv.Int)

		if aa.Exp < vv.Exp {
			mul := (&big.Int{}).Exp(big.NewInt(10), big.NewInt(int64(vv.Exp-aa.Exp)), nil)
			vvInt.Mul(vvInt, mul)
		} else if aa.Exp > vv.Exp {
			mul := (&big.Int{}).Exp(big.NewInt(10), big.NewInt(int64(aa.Exp-vv.Exp)), nil)
			aaInt.Mul(aaInt, mul)
		}

		return aaInt.Cmp(vvInt) == 0
	}
}

func mustParseNumeric(t *testing.T, src string) fbtype.Numeric {
	var n fbtype.Numeric
	err := n.Scan(src)
	require.NoError(t, err)
	return n
}

func TestNumericBinaryScanToFloat64_Local(t *testing.T) {
	n := fbtype.Numeric{
		Int:   big.NewInt(1234),
		Exp:   -2,
		Valid: true,
	}

	buf, err := fbtype.EncodeNumericBinaryFB(nil, n, -2, 4)
	require.NoError(t, err)

	var dst fbtype.Float8

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, -2)
	m := fbtype.NewMap()
	dataType, ok := m.TypeForTypeID(typeMetadata.TypeID, typeMetadata.SubType)
	require.True(t, ok)
	require.IsType(t, &fbtype.NumericCodec{}, dataType.Codec)

	plan := dataType.Codec.PlanScan(m, typeMetadata, &dst)
	require.NotNil(t, plan)

	err = plan.Scan(buf, &dst)
	require.NoError(t, err)
	require.True(t, dst.Valid)
	assert.InDelta(t, 12.34, dst.Float64, 1e-9)
}

func TestNumericValidNilIntIsZero_Local(t *testing.T) {
	n := fbtype.Numeric{Valid: true}

	value, err := n.Value()
	require.NoError(t, err)
	assert.Equal(t, "0", value)

	jsonBytes, err := n.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, []byte("0"), jsonBytes)

	intValue, err := n.Int64Value()
	require.NoError(t, err)
	assert.Equal(t, fbtype.Int8{Int64: 0, Valid: true}, intValue)

	buf, err := fbtype.EncodeNumericBinaryFB(nil, n, 0, 4)
	require.NoError(t, err)
	assert.Equal(t, []byte{0, 0, 0, 0}, buf)
}

func TestNumericBinaryScanToFloat64_DB(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		var f fbtype.Float8

		err := conn.QueryRow(
			ctx,
			"SELECT CAST(4.2 AS NUMERIC(4,2)) FROM rdb$database",
		).Scan(&f)
		require.NoError(tb, err)

		require.True(tb, f.Valid)
		assert.InDelta(tb, 4.2, f.Float64, 1e-9)
	})
}

func TestNumericBinaryScanToInt64_DB(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		var i fbtype.Int8

		err := conn.QueryRow(
			ctx,
			"SELECT CAST(42 AS NUMERIC(18,0)) FROM rdb$database",
		).Scan(&i)
		require.NoError(tb, err)

		require.True(tb, i.Valid)
		assert.Equal(tb, int64(42), i.Int64)
	})
}

func TestNumericBinaryScanToInt64_Local(t *testing.T) {
	n := fbtype.Numeric{
		Int:   big.NewInt(123456),
		Exp:   0,
		Valid: true,
	}

	buf, err := fbtype.EncodeNumericBinaryFB(nil, n, 0, 8)
	require.NoError(t, err)

	var dst fbtype.Int8

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 0, 8, 0)
	plan := fbtype.NumericCodec{}.PlanScan(
		nil,
		typeMetadata,
		&dst,
	)
	require.NotNil(t, plan)

	err = plan.Scan(buf, &dst)
	require.NoError(t, err)
	require.True(t, dst.Valid)
	assert.Equal(t, int64(123456), dst.Int64)
}

func TestNumericCodec(t *testing.T) {

	eqNum := func(s string) func(any) bool {
		return isExpectedEqNumeric(mustParseNumeric(t, s))
	}

	type rtCase struct {
		typeName string
		tests    []fbxtest.ValueRoundTripTest
	}

	cases := []rtCase{
		{
			typeName: "numeric(38,0)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "0"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("0"),
				},
				{
					Param:  mustParseNumeric(t, "1234567890123456789"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1234567890123456789"),
				},
				{
					Param:  mustParseNumeric(t, "170141183460469231731687303715884105727"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("170141183460469231731687303715884105727"),
				},
				{
					Param:  mustParseNumeric(t, "-170141183460469231731687303715884105728"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-170141183460469231731687303715884105728"),
				},
			},
		},
		{
			typeName: "numeric(38,2)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "12.12"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("12.12"),
				},
				{
					Param:  mustParseNumeric(t, "1234567890123456789012345678.90"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1234567890123456789012345678.90"),
				},
				{
					Param:  mustParseNumeric(t, "1701411834604692317316873037158841057.27"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1701411834604692317316873037158841057.27"),
				},
			},
		},
		{
			typeName: "numeric(38,30)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "0.123"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("0.123"),
				},
				{
					Param:  mustParseNumeric(t, "123456.789012345678"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("123456.789012345678"),
				},
				{
					Param:  mustParseNumeric(t, "1.2345678901234567890123456789"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1.2345678901234567890123456789"),
				},
				{
					Param:  mustParseNumeric(t, "0.999999999999999999999999999999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("0.999999999999999999999999999999"),
				},
				{
					Param:  1.23456789012345678901234567895,
					Result: new(fbtype.Numeric),
					Test:   eqNum("1.2345678901234567"),
				},
			},
		},
		{
			typeName: "numeric(18,5)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "0"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("0"),
				},
				{
					Param:  mustParseNumeric(t, "1"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1"),
				},
				{
					Param:  mustParseNumeric(t, "-1"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-1"),
				},
				{
					Param:  mustParseNumeric(t, "3.14159"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("3.14159"),
				},
				{
					Param:  mustParseNumeric(t, "-123.45"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-123.45"),
				},
				{
					Param:  mustParseNumeric(t, "1.23"),
					Result: new(string),
					Test:   isExpectedEq("1.23000"),
				},
				{
					Param:  mustParseNumeric(t, "1.23000"),
					Result: new(string),
					Test:   isExpectedEq("1.23000"),
				},
				{
					Param:  mustParseNumeric(t, "1234567890.12345"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1234567890.12345"),
				},
				{
					Param:  mustParseNumeric(t, "-9876543210.00001"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-9876543210.00001"),
				},
				{
					Param:  fbtype.Numeric{},
					Result: new(fbtype.Numeric),
					Test:   isExpectedEq(fbtype.Numeric{}),
				},
				{
					Param:  nil,
					Result: new(fbtype.Numeric),
					Test:   isExpectedEq(fbtype.Numeric{}),
				},
			},
		},
		{
			typeName: "numeric(18,0)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "0"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("0"),
				},
				{
					Param:  mustParseNumeric(t, "1"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1"),
				},
				{
					Param:  mustParseNumeric(t, "-1"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-1"),
				},
				{
					Param:  mustParseNumeric(t, "9223372036854775807"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("9223372036854775807"),
				},
				{
					Param:  mustParseNumeric(t, "-9223372036854775807"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-9223372036854775807"),
				},
			},
		},
		{
			typeName: "numeric(9,2)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "0"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("0"),
				},
				{
					Param:  mustParseNumeric(t, "1.23"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1.23"),
				},
				{
					Param:  mustParseNumeric(t, "-9999999.99"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-9999999.99"),
				},
				{
					Param:  mustParseNumeric(t, "9999999.99"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("9999999.99"),
				},
			},
		},
		{
			typeName: "numeric(4,2)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "4.2"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("4.2"),
				},
			},
		},
		{
			typeName: "numeric(4,0)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "0"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("0"),
				},
				{
					Param:  mustParseNumeric(t, "1"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1"),
				},
				{
					Param:  mustParseNumeric(t, "-1"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-1"),
				},
				{
					Param:  mustParseNumeric(t, "-9999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-9999"),
				},
				{
					Param:  mustParseNumeric(t, "9999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("9999"),
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.typeName, func(t *testing.T) {
			if strings.HasPrefix(c.typeName, "numeric(38,") {
				defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
					if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
						t.Skip(c.typeName + " is not supported by server engine version <= 3")
					}
				})
			}

			fbxtest.RunValueRoundTripTests(
				context.Background(),
				t,
				defaultConnTestRunner,
				nil,
				c.typeName,
				c.tests,
			)
		})
	}
}

func TestNumericFloat64Valuer(t *testing.T) {
	for i, tt := range []struct {
		n fbtype.Numeric
		f fbtype.Float8
	}{
		{mustParseNumeric(t, "170141183460469231731687303715884105727"), fbtype.Float8{Float64: 1.7014118346046923173e+38, Valid: true}},
		{mustParseNumeric(t, "-170141183460469231731687303715884105728"), fbtype.Float8{Float64: -1.7014118346046923173e+38, Valid: true}},
		{mustParseNumeric(t, "1"), fbtype.Float8{Float64: 1, Valid: true}},
		{mustParseNumeric(t, "0.0000000000000000001"), fbtype.Float8{Float64: 0.0000000000000000001, Valid: true}},
		{mustParseNumeric(t, "-99999999999"), fbtype.Float8{Float64: -99999999999, Valid: true}},
		{fbtype.Numeric{Valid: true}, fbtype.Float8{Valid: true}},
		{fbtype.Numeric{}, fbtype.Float8{}},
	} {
		f, err := tt.n.Float64Value()
		assert.NoErrorf(t, err, "%d", i)
		assert.Equalf(t, tt.f, f, "%d", i)
	}
}

func TestNumericScanScientificExact(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "large positive exponent",
			src:  "1.234567890123456789e18",
			want: "1234567890123456789",
		},
		{
			name: "large negative exponent",
			src:  "1.234567890123456789e-3",
			want: "0.001234567890123456789",
		},
		{
			name: "trailing zeros preserved",
			src:  "1.2300e2",
			want: "123.00",
		},
		{
			name: "uppercase exponent",
			src:  "-9.99E3",
			want: "-9990",
		},
		{
			name: "positive sign and exponent",
			src:  "+1.20e+2",
			want: "120",
		},
		{
			name: "positive signed zero",
			src:  "+000",
			want: "0",
		},
		{
			name: "negative signed zero",
			src:  "-000",
			want: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got fbtype.Numeric
			require.NoError(t, got.ScanScientific(tt.src))
			require.True(t, got.Valid)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestNumericScanSupportsDatabaseSQLValuesAndClearsOnError(t *testing.T) {
	var got fbtype.Numeric

	require.NoError(t, got.Scan([]byte("12.34")))
	assert.Equal(t, "12.34", got.String())

	require.NoError(t, got.Scan(int64(-42)))
	assert.Equal(t, "-42", got.String())

	require.NoError(t, got.Scan(float64(12.5)))
	assert.Equal(t, "12.5", got.String())

	bigValue := big.NewInt(99)
	require.NoError(t, got.Scan(bigValue))
	bigValue.SetInt64(100)
	assert.Equal(t, "99", got.String())

	bigPtrValue := big.NewInt(123)
	require.NoError(t, got.Scan(bigPtrValue))
	bigPtrValue.SetInt64(321)
	assert.Equal(t, "123", got.String())

	stale := mustParseNumeric(t, "42")
	require.Error(t, stale.Scan("not-a-number"))
	assert.False(t, stale.Valid)

	stale = mustParseNumeric(t, "42")
	require.Error(t, stale.Scan(math.Inf(1)))
	assert.False(t, stale.Valid)

	stale = mustParseNumeric(t, "42")
	require.Error(t, stale.Scan(struct{}{}))
	assert.False(t, stale.Valid)
}

func TestNumericScanNumericAndValueCopyBigInt(t *testing.T) {
	src := fbtype.Numeric{Int: big.NewInt(42), Exp: -1, Valid: true}

	var dst fbtype.Numeric
	require.NoError(t, dst.ScanNumeric(src))
	src.Int.SetInt64(99)
	assert.Equal(t, "4.2", dst.String())

	value, err := dst.NumericValue()
	require.NoError(t, err)
	value.Int.SetInt64(77)
	assert.Equal(t, "4.2", dst.String())
}

func TestNumericUnmarshalJSONWhitespaceNullQuotedScientificAndClears(t *testing.T) {
	stale := mustParseNumeric(t, "42")
	require.NoError(t, json.Unmarshal([]byte(" null "), &stale))
	assert.False(t, stale.Valid)

	var got fbtype.Numeric
	require.NoError(t, json.Unmarshal([]byte(`"1.234567890123456789e18"`), &got))
	assert.Equal(t, "1234567890123456789", got.String())

	require.NoError(t, json.Unmarshal([]byte(`1e-3`), &got))
	assert.Equal(t, "0.001", got.String())

	stale = mustParseNumeric(t, "42")
	require.Error(t, json.Unmarshal([]byte(`"bad"`), &stale))
	assert.False(t, stale.Valid)
}

func TestNumericDecodeDatabaseSQLValueReturnsString(t *testing.T) {
	n := mustParseNumeric(t, "12.34")
	buf, err := fbtype.EncodeNumericBinaryFB(nil, n, -2, 4)
	require.NoError(t, err)

	codec := fbtype.NumericCodec{}
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, -2)

	value, err := codec.DecodeDatabaseSQLValue(fbtype.NewMap(), typeMetadata, buf)
	require.NoError(t, err)
	require.IsType(t, "", value)
	assert.Equal(t, "12.34", value)

	value, err = codec.DecodeDatabaseSQLValue(fbtype.NewMap(), typeMetadata, nil)
	require.NoError(t, err)
	assert.Nil(t, value)
}

func TestNumericPrimitiveScanNullErrors(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, -2)

	var f float64
	err := m.Scan(typeMetadata, nil, &f)
	require.ErrorContains(t, err, "cannot scan NULL into *float64")

	var i int64
	err = m.Scan(typeMetadata, nil, &i)
	require.ErrorContains(t, err, "cannot scan NULL into *int64")

	var s string
	err = m.Scan(typeMetadata, nil, &s)
	require.ErrorContains(t, err, "cannot scan NULL into *string")
}

func TestNumericInt128_DB(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("numeric(38,*) is not supported by server engine version <= 3")
		}

		var n fbtype.Numeric

		err := conn.QueryRow(ctx,
			"SELECT CAST(170141183460469231731687303715884105727 AS NUMERIC(38,0)) FROM rdb$database",
		).Scan(&n)
		require.NoError(tb, err)
		require.True(tb, n.Valid)
		assert.Equal(tb, "170141183460469231731687303715884105727", n.Int.String())
		assert.Equal(tb, int32(0), n.Exp)

		err = conn.QueryRow(ctx,
			"SELECT CAST(1234567890123456789012345678.90 AS NUMERIC(38,2)) FROM rdb$database",
		).Scan(&n)
		require.NoError(tb, err)
		require.True(tb, n.Valid)
		assert.Equal(tb, "123456789012345678901234567890", n.Int.String())
		assert.Equal(tb, int32(-2), n.Exp)

		err = conn.QueryRow(ctx,
			"SELECT CAST(0.999999999999999999999999999999 AS NUMERIC(38,30)) FROM rdb$database",
		).Scan(&n)
		require.NoError(tb, err)
		require.True(tb, n.Valid)
		assert.Equal(tb, "999999999999999999999999999999", n.Int.String())
		assert.Equal(tb, int32(-30), n.Exp)
	})
}

func TestNumericRounding_DB(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("numeric(38,*) is not supported by server engine version <= 3")
		}

		var s string

		err := conn.QueryRow(ctx,
			"SELECT CAST(1.234567890123456789 AS NUMERIC(38,9)) FROM rdb$database",
		).Scan(&s)
		require.NoError(tb, err)
		assert.Equal(tb, "1.234567890", s)

		err = conn.QueryRow(ctx,
			"SELECT CAST(ROUND(1.234567890123456789 * 1e9, 0) / 1e9 AS NUMERIC(38,9)) FROM rdb$database",
		).Scan(&s)
		require.NoError(tb, err)
		assert.Equal(tb, "1.234567890", s)

		err = conn.QueryRow(ctx,
			"SELECT CAST(170141183460469231731687303715884105727 AS NUMERIC(38,0)) FROM rdb$database",
		).Scan(&s)
		require.NoError(tb, err)
		assert.Equal(tb, "170141183460469231731687303715884105727", s)

		err = conn.QueryRow(ctx,
			"SELECT CAST(1234567890123456789.01 AS NUMERIC(38,2)) FROM rdb$database",
		).Scan(&s)
		require.NoError(tb, err)
		assert.Equal(tb, "1234567890123456789.01", s)
	})
}

func mustBigInt(t *testing.T, s string) *big.Int {
	t.Helper()
	z := new(big.Int)
	_, ok := z.SetString(s, 10)
	require.True(t, ok, "invalid big.Int: %s", s)
	return z
}

func TestInt128Overflow(t *testing.T) {
	maxInt128 := "170141183460469231731687303715884105727"
	minInt128 := "-170141183460469231731687303715884105728"

	okCases := []fbtype.Numeric{
		{Int: mustBigInt(t, maxInt128), Exp: 0, Valid: true},
		{Int: mustBigInt(t, minInt128), Exp: 0, Valid: true},
	}

	for _, n := range okCases {
		_, err := fbtype.EncodeNumericBinaryFB(nil, n, 0, 16)
		require.NoError(t, err)
	}

	overflowCases := []fbtype.Numeric{
		{Int: mustBigInt(t, "170141183460469231731687303715884105728"), Exp: 0, Valid: true},
		{Int: mustBigInt(t, "-170141183460469231731687303715884105729"), Exp: 0, Valid: true},
		{Int: mustBigInt(t, "9999999999999999999999999999999999999999"), Exp: 0, Valid: true},
		{Int: mustBigInt(t, maxInt128), Exp: 0, Valid: true},
	}

	for i, n := range overflowCases {
		scale := int32(0)
		if i == 3 {
			scale = -30
		}

		_, err := fbtype.EncodeNumericBinaryFB(nil, n, scale, 16)
		require.Errorf(t, err, "overflow case %d", i)
		assert.Containsf(t, err.Error(), "out of range for int128", "case %d: %v", i, n)
	}
}

func TestNumericEncodeFromFloat64_DB(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		typeName string
		arg      float64

		wantText string
		wantInt  string
		wantExp  int32
	}{
		{
			name:     "numeric(9,2) keeps trailing zeros (12.10)",
			typeName: "numeric(9,2)",
			arg:      12.10,
			wantText: "12.10",
			wantInt:  "1210",
			wantExp:  -2,
		},
		{
			name:     "numeric(9,2) rounds half away from zero (12.129 -> 12.13)",
			typeName: "numeric(9,2)",
			arg:      12.129,
			wantText: "12.13",
			wantInt:  "1213",
			wantExp:  -2,
		},
		{
			name:     "numeric(9,2) rounds half away from zero (negative)",
			typeName: "numeric(9,2)",
			arg:      -12.125,
			wantText: "-12.13",
			wantInt:  "-1213",
			wantExp:  -2,
		},
		{
			name:     "numeric(10,2) dirty float (4323.989... -> 4323.99)",
			typeName: "numeric(10,2)",
			arg:      4323.9898923948928,
			wantText: "4323.99",
			wantInt:  "432399",
			wantExp:  -2,
		},
		{
			name:     "numeric(9,0) integer-ish (123456789)",
			typeName: "numeric(9,0)",
			arg:      123456789.0,
			wantText: "123456789",
			wantInt:  "123456789",
			wantExp:  0,
		},
		{
			name:     "numeric(18,0) bigint-ish (9e17)",
			typeName: "numeric(18,0)",
			arg:      9_000_000_000_000_000_00.0,
			wantText: "900000000000000000",
			wantInt:  "900000000000000000",
			wantExp:  0,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
				var n fbtype.Numeric
				q := fmt.Sprintf("select cast(? as %s) from rdb$database", tc.typeName)
				err := conn.QueryRow(ctx, q, tc.arg).Scan(&n)
				require.NoError(tb, err)
				require.True(tb, n.Valid)
				require.NotNil(tb, n.Int)
				assert.Equal(tb, tc.wantInt, n.Int.String())
				assert.Equal(tb, tc.wantExp, n.Exp)

				var s string
				q = fmt.Sprintf("select cast(cast(? as %s) as varchar(50)) from rdb$database", tc.typeName)
				err = conn.QueryRow(ctx, q, tc.arg).Scan(&s)
				require.NoError(tb, err)
				assert.Equal(tb, tc.wantText, s)
			})
		})
	}
}

func TestNumericEncodePlanMemoization_Regression_DB(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		var s string

		err := conn.QueryRow(ctx,
			"select cast(cast(? as numeric(9,2)) as varchar(20)) from rdb$database",
			12.12,
		).Scan(&s)
		require.NoError(tb, err)
		require.Equal(tb, "12.12", s)

		err = conn.QueryRow(ctx,
			"select cast(cast(? as numeric(9,0)) as varchar(20)) from rdb$database",
			123456789.0,
		).Scan(&s)
		require.NoError(tb, err)
		require.Equal(tb, "123456789", s)
	})
}

func TestNumericUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		want    *fbtype.Numeric
		src     []byte
		wantErr bool
	}{
		{
			name:    "null",
			want:    &fbtype.Numeric{},
			src:     []byte(`null`),
			wantErr: false,
		},
		{
			name:    "0",
			want:    &fbtype.Numeric{Valid: true, Int: big.NewInt(0)},
			src:     []byte("0"),
			wantErr: false,
		},
		{
			name:    "1",
			want:    &fbtype.Numeric{Valid: true, Int: big.NewInt(1)},
			src:     []byte("1"),
			wantErr: false,
		},
		{
			name:    "-1",
			want:    &fbtype.Numeric{Valid: true, Int: big.NewInt(-1)},
			src:     []byte("-1"),
			wantErr: false,
		},
		{
			name:    "bigInt",
			want:    &fbtype.Numeric{Valid: true, Int: big.NewInt(1), Exp: 30},
			src:     []byte("1000000000000000000000000000000"),
			wantErr: false,
		},
		{
			name:    "float: 1234.56789",
			want:    &fbtype.Numeric{Valid: true, Int: big.NewInt(123456789), Exp: -5},
			src:     []byte("1234.56789"),
			wantErr: false,
		},
		{
			name:    "invalid value",
			want:    &fbtype.Numeric{},
			src:     []byte("0xffff"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &fbtype.Numeric{}
			if err := got.UnmarshalJSON(tt.src); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}
