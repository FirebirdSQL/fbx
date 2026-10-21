package fbtype_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/fbtype"
	"github.com/firebirdsql/fbx/fbxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNumericScanUsesPhysicalStorageLength(t *testing.T) {
	tests := []struct {
		name         string
		typeMetadata fbtype.TypeMetadata
		coefficient  string
		wireLength   uint32
	}{
		{
			name:         "smallint",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 1, 2, 0),
			coefficient:  "32767",
			wireLength:   4,
		},
		{
			name:         "integer",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, 0),
			coefficient:  "2147483647",
			wireLength:   4,
		},
		{
			name:         "bigint",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 1, 8, 0),
			coefficient:  "9223372036854775807",
			wireLength:   8,
		},
		{
			name:         "int128",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 1, 16, 0),
			coefficient:  "170141183460469231731687303715884105727",
			wireLength:   16,
		},
	}

	m := fbtype.NewMap()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := fbtype.EncodeNumericBinaryFB(nil, mustParseNumeric(t, tt.coefficient), 0, tt.wireLength)
			require.NoError(t, err)

			var got fbtype.Numeric
			err = m.Scan(tt.typeMetadata, src, &got)
			require.NoError(t, err)
			assert.Equal(t, tt.coefficient, got.Int.String())
		})
	}
}

func TestNumericScanChecksSmallintPhysicalRange(t *testing.T) {
	m := fbtype.NewMap()
	shortMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 1, 2, 0)
	longMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, 0)

	for _, coefficient := range []string{"32768", "-32769"} {
		t.Run(coefficient, func(t *testing.T) {
			src, err := fbtype.EncodeNumericBinaryFB(nil, mustParseNumeric(t, coefficient), 0, 4)
			require.NoError(t, err)

			got := mustParseNumeric(t, "42")
			err = m.Scan(shortMetadata, src, &got)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "out of range for SMALLINT storage")
			assert.Equal(t, fbtype.Numeric{}, got)

			err = m.Scan(longMetadata, src, &got)
			require.NoError(t, err)
			assert.Equal(t, coefficient, got.Int.String())
		})
	}
}

func TestNumericScanRejectsWireLengthMismatches(t *testing.T) {
	tests := []struct {
		name         string
		typeMetadata fbtype.TypeMetadata
		wireLength   uint32
	}{
		{
			name:         "smallint with int64 wire value",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 1, 2, 0),
			wireLength:   8,
		},
		{
			name:         "integer with int64 wire value",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, 0),
			wireLength:   8,
		},
		{
			name:         "bigint with int32 wire value",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 1, 8, 0),
			wireLength:   4,
		},
		{
			name:         "int128 with int64 wire value",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 1, 16, 0),
			wireLength:   8,
		},
	}

	m := fbtype.NewMap()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := fbtype.EncodeNumericBinaryFB(nil, mustParseNumeric(t, "1"), 0, tt.wireLength)
			require.NoError(t, err)

			got := mustParseNumeric(t, "42")
			err = m.Scan(tt.typeMetadata, src, &got)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid wire length")
			assert.Equal(t, fbtype.Numeric{}, got)
		})
	}
}

func TestNumericScanRejectsUnsupportedStorageLength(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 3, 0)
	src, err := fbtype.EncodeNumericBinaryFB(nil, mustParseNumeric(t, "1"), 0, 4)
	require.NoError(t, err)

	got := mustParseNumeric(t, "42")
	err = m.Scan(typeMetadata, src, &got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid storage length")
	assert.Equal(t, fbtype.Numeric{}, got)
}

func TestNumericCodecRejectsTypeLengthMismatch(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 16, -2)

	buf, err := m.Encode(typeMetadata, "1.23", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid storage length 16")
	assert.Nil(t, buf)

	src, err := fbtype.EncodeNumericBinaryFB(nil, mustParseNumeric(t, "1.23"), -2, 16)
	require.NoError(t, err)

	got := mustParseNumeric(t, "42")
	err = m.Scan(typeMetadata, src, &got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid storage length 16")
	assert.Equal(t, fbtype.Numeric{}, got)
}

func TestNumericCheckedReadPreservesSignAndScale(t *testing.T) {
	m := fbtype.NewMap()
	codec := fbtype.NumericCodec{}
	for _, tt := range []struct {
		name          string
		typeID        uint32
		storageLength uint32
		wireLength    int
	}{
		{"smallint", fbproto.SQL_TYPE_SHORT, 2, 4},
		{"integer", fbproto.SQL_TYPE_LONG, 4, 4},
		{"bigint", fbproto.SQL_TYPE_INT64, 8, 8},
		{"int128", fbproto.SQL_TYPE_INT128, 16, 16},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := bytes.Repeat([]byte{0xff}, tt.wireLength)
			src[len(src)-2], src[len(src)-1] = 0xcf, 0xc7
			want := fbtype.Numeric{Int: big.NewInt(-12345), Exp: -2, Valid: true}

			var direct fbtype.Numeric
			require.NoError(t, fbtype.DecodeNumericBinaryFB(&direct, src, -2))
			require.Equal(t, want, direct)

			for _, subtype := range []int32{1, 2} {
				t.Run(fmt.Sprintf("subtype_%d", subtype), func(t *testing.T) {
					metadata := fbtype.NewTypeMetadata(tt.typeID, subtype, tt.storageLength, -2)
					var scanned fbtype.Numeric
					require.NoError(t, m.Scan(metadata, src, &scanned))
					require.Equal(t, want, scanned)

					value, err := codec.DecodeValue(m, metadata, src)
					require.NoError(t, err)
					require.Equal(t, want, value)
					sqlValue, err := codec.DecodeDatabaseSQLValue(m, metadata, src)
					require.NoError(t, err)
					require.Equal(t, "-123.45", sqlValue)
				})
			}
		})
	}
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
					Param:  mustParseNumeric(t, "99999999999999999999999999999999999999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("99999999999999999999999999999999999999"),
				},
				{
					Param:  mustParseNumeric(t, "-99999999999999999999999999999999999999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-99999999999999999999999999999999999999"),
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
			typeName: "numeric(23,5)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "999999999999999999.99999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("999999999999999999.99999"),
				},
				{
					Param:  mustParseNumeric(t, "-999999999999999999.99999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-999999999999999999.99999"),
				},
				{
					Param:  mustParseNumeric(t, "1701411834604692317316873037158841.05727"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("1701411834604692317316873037158841.05727"),
				},
				{
					Param:  mustParseNumeric(t, "-1701411834604692317316873037158841.05728"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-1701411834604692317316873037158841.05728"),
				},
			},
		},
		{
			typeName: "numeric(12,11)",
			tests: []fbxtest.ValueRoundTripTest{
				{
					Param:  mustParseNumeric(t, "9.99999999999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("9.99999999999"),
				},
				{
					Param:  mustParseNumeric(t, "-9.99999999999"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-9.99999999999"),
				},
				{
					Param:  mustParseNumeric(t, "92233720.36854775807"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("92233720.36854775807"),
				},
				{
					Param:  mustParseNumeric(t, "-92233720.36854775808"),
					Result: new(fbtype.Numeric),
					Test:   eqNum("-92233720.36854775808"),
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
			if strings.HasPrefix(c.typeName, "numeric(38,") || strings.HasPrefix(c.typeName, "numeric(23,") {
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

func TestParseNumericRejectsUnsupportedExponent(t *testing.T) {
	for _, src := range []string{
		"0e-39",
		"0e-2147483648",
		"000e-40",
		"1e-39",
		"1e-2147483648",
		"1e2147483647",
		"2e38",
		"1e9223372036854775807",
		"1e9223372036854775808",
	} {
		t.Run(src, func(t *testing.T) {
			_, err := fbtype.ParseNumeric(src)
			require.Error(t, err)
		})
	}
}

func TestParseNumericCombinesExponentsBeforeValidation(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			src:  "0.000000000001e50",
			want: "1" + strings.Repeat("0", 38),
		},
		{
			src:  "100e-40",
			want: "0." + strings.Repeat("0", 37) + "1",
		},
		{
			src:  "0.00",
			want: "0.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			n, err := fbtype.ParseNumeric(tt.src)
			require.NoError(t, err)
			assert.Equal(t, tt.want, n.String())
		})
	}
}

func TestNumericManualUnsupportedExponentReturnsErrors(t *testing.T) {
	for _, exp := range []int32{math.MinInt32, math.MaxInt32} {
		t.Run(strconv.FormatInt(int64(exp), 10), func(t *testing.T) {
			n := fbtype.Numeric{Int: big.NewInt(1), Exp: exp, Valid: true}

			assert.Contains(t, n.String(), "<invalid Numeric:")

			_, err := n.Value()
			require.Error(t, err)

			_, err = n.NumericValue()
			require.Error(t, err)

			_, err = n.Float64Value()
			require.Error(t, err)

			_, err = n.Int64Value()
			require.Error(t, err)

			_, err = json.Marshal(n)
			require.Error(t, err)

			_, err = fbtype.EncodeNumericBinaryFB(nil, n, 0, 16)
			require.Error(t, err)
		})
	}
}

func TestNumericInt64ValueRescalesBeforeRangeCheck(t *testing.T) {
	n := fbtype.Numeric{
		Int:   mustBigInt(t, "92233720368547758070"),
		Exp:   -1,
		Valid: true,
	}

	got, err := n.Int64Value()
	require.NoError(t, err)
	assert.Equal(t, fbtype.Int8{Int64: math.MaxInt64, Valid: true}, got)
}

func TestNumericInt64ValueTruncatesFractionTowardZero(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int64
	}{
		{name: "positive", src: "123.987", want: 123},
		{name: "negative", src: "-123.987", want: -123},
		{name: "positive below one", src: "0.999", want: 0},
		{name: "negative above minus one", src: "-0.999", want: 0},
		{name: "positive int64 maximum", src: "9223372036854775807.9", want: math.MaxInt64},
		{name: "negative int64 minimum", src: "-9223372036854775808.9", want: math.MinInt64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := mustParseNumeric(t, tt.src)

			got, err := n.Int64Value()
			require.NoError(t, err)
			assert.Equal(t, fbtype.Int8{Int64: tt.want, Valid: true}, got)
		})
	}
}

func TestNumericInt64ValueChecksRangeAfterTruncation(t *testing.T) {
	for _, src := range []string{
		"9223372036854775808.1",
		"-9223372036854775809.1",
	} {
		t.Run(src, func(t *testing.T) {
			n := mustParseNumeric(t, src)

			_, err := n.Int64Value()
			require.Error(t, err)
		})
	}
}

func TestNumericEncodesToIntegerCodecsWithTruncation(t *testing.T) {
	tests := []struct {
		name         string
		typeMetadata fbtype.TypeMetadata
		src          string
		want         int64
	}{
		{
			name:         "smallint",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 2, 0),
			src:          "32767.9",
			want:         math.MaxInt16,
		},
		{
			name:         "integer",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0),
			src:          "-2147483648.9",
			want:         math.MinInt32,
		},
		{
			name:         "bigint",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 0, 8, 0),
			src:          "9223372036854775807.9",
			want:         math.MaxInt64,
		},
	}

	m := fbtype.NewMap()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := m.Encode(tt.typeMetadata, mustParseNumeric(t, tt.src), nil)
			require.NoError(t, err)

			var got int64
			err = m.Scan(tt.typeMetadata, buf, &got)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNumericEncodeToIntegerCodecChecksRangeAfterTruncation(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 2, 0)

	_, err := m.Encode(typeMetadata, mustParseNumeric(t, "32768.1"), nil)
	require.Error(t, err)
}

func TestNumericScanToInt64StillRequiresExactValue(t *testing.T) {
	n := mustParseNumeric(t, "123.45")
	buf, err := fbtype.EncodeNumericBinaryFB(nil, n, -2, 4)
	require.NoError(t, err)

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 1, 4, -2)
	var dst int64
	plan := fbtype.NumericCodec{}.PlanScan(nil, typeMetadata, &dst)
	require.NotNil(t, plan)

	err = plan.Scan(buf, &dst)
	require.Error(t, err)
}

func TestNumericRoundingRejectsManualExtremeExponent(t *testing.T) {
	n := fbtype.Numeric{Int: big.NewInt(1), Exp: math.MinInt32, Valid: true}

	_, err := fbtype.EncodeNumericBinaryFBRoundHalfAwayFromZero(nil, n, -38, 16)
	require.Error(t, err)
}

func TestNumericFloatRoundingHandlesExtremeNegativeExponent(t *testing.T) {
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT128, 1, 16, -38)

	buf, err := m.Encode(typeMetadata, math.SmallestNonzeroFloat64, nil)
	require.NoError(t, err)
	assert.Equal(t, make([]byte, 16), buf)
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
