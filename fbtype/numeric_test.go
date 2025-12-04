package fbtype_test

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func mustParseBigInt(t *testing.T, src string) *big.Int {
	i := &big.Int{}
	if _, ok := i.SetString(src, 10); !ok {
		t.Fatalf("could not parse big.Int: %s", src)
	}
	return i
}

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
	plan := fbtype.NumericCodec{}.PlanScan(nil, fbtype.NumericOID, 0, 0, 0, fbtype.TextFormatCode, &n)
	require.NotNil(t, plan)
	err := plan.Scan([]byte(src), &n)
	require.NoError(t, err)
	return n
}

func TestNumericCodec(t *testing.T) {
	skipCockroachDB(t, "server formats numeric text format differently")

	eqNum := func(s string) func(any) bool {
		return isExpectedEqNumeric(mustParseNumeric(t, s))
	}

	type rtCase struct {
		typeName string
		tests    []fbxtest.ValueRoundTripTest
	}

	cases := []rtCase{
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
	}

	for _, c := range cases {
		t.Run(c.typeName, func(t *testing.T) {
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

func TestNumericCodecInfinity(t *testing.T) {
	t.Skip()
	skipPostgreSQLVersionLessThan(t, 14)

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "numeric", []fbxtest.ValueRoundTripTest{
		{Param: math.Inf(1), Result: new(float64), Test: isExpectedEq(math.Inf(1))},
		{Param: float32(math.Inf(1)), Result: new(float32), Test: isExpectedEq(float32(math.Inf(1)))},
		{Param: math.Inf(-1), Result: new(float64), Test: isExpectedEq(math.Inf(-1))},
		{Param: float32(math.Inf(-1)), Result: new(float32), Test: isExpectedEq(float32(math.Inf(-1)))},
	})
}

func TestNumericFloat64Valuer(t *testing.T) {
	for i, tt := range []struct {
		n fbtype.Numeric
		f fbtype.Float8
	}{
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

func TestNumericCodecFuzz(t *testing.T) {
	t.Skip()

	r := rand.New(rand.NewSource(0))
	max := &big.Int{}
	max.SetString("9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", 10)

	tests := make([]fbxtest.ValueRoundTripTest, 0, 2000)
	for i := 0; i < 10; i++ {
		for j := -50; j < 50; j++ {
			num := (&big.Int{}).Rand(r, max)

			n := fbtype.Numeric{Int: num, Exp: int32(j), Valid: true}
			tests = append(tests, fbxtest.ValueRoundTripTest{Param: n, Result: new(fbtype.Numeric), Test: isExpectedEqNumeric(n)})

			negNum := &big.Int{}
			negNum.Neg(num)
			n = fbtype.Numeric{Int: negNum, Exp: int32(j), Valid: true}
			tests = append(tests, fbxtest.ValueRoundTripTest{Param: n, Result: new(fbtype.Numeric), Test: isExpectedEqNumeric(n)})
		}
	}

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "numeric", tests)
}

func TestNumericMarshalJSON(t *testing.T) {
	t.Skip()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {

		for i, tt := range []struct {
			decString string
		}{
			{"0"},
			{"1"},
			{"-1"},
			{"1000000000000000000"},
			{"1234.56789"},
			{"1.56789"},
			{"0.00000000000056789"},
			{"0.00123000"},
			{"123e-3"},
			{"243723409723490243842378942378901237502734019231380123e23790"},
			{"3409823409243892349028349023482934092340892390101e-14021"},
			{"-1.1"},
			{"-1.0231"},
			{"-10.0231"},
			{"-0.1"},   // failed with "invalid character '.' in numeric literal"
			{"-0.01"},  // failed with "invalid character '-' after decimal point in numeric literal"
			{"-0.001"}, // failed with "invalid character '-' after top-level value"
		} {
			var num fbtype.Numeric
			var pgJSON string
			err := conn.QueryRow(ctx, `select $1::numeric, to_json($1::numeric)`, tt.decString).Scan(&num, &pgJSON)
			require.NoErrorf(t, err, "%d", i)

			goJSON, err := json.Marshal(num)
			require.NoErrorf(t, err, "%d", i)

			require.Equal(t, pgJSON, string(goJSON))
		}
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
