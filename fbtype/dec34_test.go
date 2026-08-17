package fbtype_test

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func mustDec34(t *testing.T, s string) fbtype.DecFloat34 {
	t.Helper()
	var d fbtype.DecFloat34
	require.NoError(t, d.Scan(s))
	return d
}

func isExpectedEqDec34(want fbtype.DecFloat34) func(any) bool {
	return func(got any) bool {
		var d fbtype.DecFloat34
		switch v := got.(type) {
		case *fbtype.DecFloat34:
			if v == nil {
				return !want.Valid
			}
			d = *v
		case fbtype.DecFloat34:
			d = v
		default:
			return false
		}

		if d.Valid != want.Valid {
			return false
		}
		if !want.Valid {
			return true
		}

		if d.Special != want.Special {
			return false
		}
		if d.NegativeZero != want.NegativeZero {
			return false
		}
		if want.Special != fbtype.SpecialNone {
			return true
		}

		if d.Exp != want.Exp {
			return false
		}
		if (d.Int == nil) != (want.Int == nil) {
			return false
		}
		if want.Int == nil {
			return true
		}
		return d.Int.Cmp(want.Int) == 0
	}
}

func isExpectedEqFloat64(want float64) func(any) bool {
	return func(got any) bool {
		var f float64
		switch v := got.(type) {
		case *float64:
			if v == nil {
				return false
			}
			f = *v
		case float64:
			f = v
		default:
			return false
		}

		if math.IsNaN(want) {
			return math.IsNaN(f) && math.Signbit(f) == math.Signbit(want)
		}
		if math.IsInf(want, 1) {
			return math.IsInf(f, 1)
		}
		if math.IsInf(want, -1) {
			return math.IsInf(f, -1)
		}
		if want == 0 {
			return f == 0 && math.Signbit(f) == math.Signbit(want)
		}

		epsilon := 1e-14
		if math.Abs(want) > 1 {
			return math.Abs((f-want)/want) <= epsilon
		}
		return math.Abs(f-want) <= epsilon
	}
}

func TestDecFloat34Codec_RoundTrip_DB(t *testing.T) {
	t.Parallel()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("decfloat(34) is not supported by server engine version <= 3")
		}
	})

	type tc struct {
		label     string
		val       fbtype.DecFloat34
		wantStr   string
		wantF64   float64
		skipFloat bool
	}

	testsData := []tc{
		{"zero", fbtype.DecFloat34{Int: big.NewInt(0), Exp: 0, Valid: true, Special: fbtype.SpecialNone}, "0", 0, false},
		{"one", fbtype.DecFloat34{Int: big.NewInt(1), Exp: 0, Valid: true, Special: fbtype.SpecialNone}, "1", 1, false},
		{"minus_one", fbtype.DecFloat34{Int: big.NewInt(-1), Exp: 0, Valid: true, Special: fbtype.SpecialNone}, "-1", -1, false},

		{"tiny_frac", fbtype.DecFloat34{Int: big.NewInt(23), Exp: -7, Valid: true, Special: fbtype.SpecialNone}, "0.0000023", 2.3e-06, false},
		{"half", fbtype.DecFloat34{Int: big.NewInt(5), Exp: -1, Valid: true, Special: fbtype.SpecialNone}, "0.5", 0.5, false},

		{"pi_34",
			fbtype.DecFloat34{
				Int:     mustBigInt(t, "3141592653589793238462643383279502"),
				Exp:     -33,
				Valid:   true,
				Special: fbtype.SpecialNone,
			},
			"3.141592653589793238462643383279502",
			3.1415926535897931,
			false,
		},

		{"max_34_digits",
			fbtype.DecFloat34{
				Int:     mustBigInt(t, "9999999999999999999999999999999999"),
				Exp:     0,
				Valid:   true,
				Special: fbtype.SpecialNone,
			},
			"9999999999999999999999999999999999",
			9.999999999999999e+33,
			false,
		},

		{"power_pos_300",
			fbtype.DecFloat34{Int: big.NewInt(1), Exp: 300, Valid: true, Special: fbtype.SpecialNone},
			"1E+300",
			1e+300,
			false,
		},

		{"power_neg_300",
			fbtype.DecFloat34{Int: big.NewInt(1), Exp: -300, Valid: true, Special: fbtype.SpecialNone},
			"1E-300",
			1e-300,
			false,
		},

		{"min_exponent",
			fbtype.DecFloat34{Int: big.NewInt(1), Exp: -6176, Valid: true, Special: fbtype.SpecialNone},
			"1E-6176",
			0,
			true,
		},
		{"near_max_exponent",
			fbtype.DecFloat34{Int: big.NewInt(1), Exp: 300, Valid: true, Special: fbtype.SpecialNone},
			"1E+300",
			1e+300,
			false,
		},
		{"max_exponent",
			fbtype.DecFloat34{
				Int:     mustBigInt(t, "9999999999999999999999999999999999"),
				Exp:     6111,
				Valid:   true,
				Special: fbtype.SpecialNone,
			},
			"9.999999999999999999999999999999999E+6144",
			0,
			true,
		},

		{"dpd_999",
			fbtype.DecFloat34{Int: big.NewInt(999), Exp: 0, Valid: true, Special: fbtype.SpecialNone},
			"999",
			999,
			false,
		},

		{"dpd_1000",
			fbtype.DecFloat34{Int: big.NewInt(1000), Exp: 0, Valid: true, Special: fbtype.SpecialNone},
			"1000",
			1000,
			false,
		},

		{"trailing_zeros_1",
			fbtype.DecFloat34{Int: big.NewInt(10), Exp: -1, Valid: true, Special: fbtype.SpecialNone},
			"1.0",
			1.0,
			false,
		},

		{"trailing_zeros_2",
			fbtype.DecFloat34{Int: big.NewInt(1000), Exp: -2, Valid: true, Special: fbtype.SpecialNone},
			"10.00",
			10.0,
			false,
		},

		{"negative_zero",
			fbtype.DecFloat34{Int: big.NewInt(0), Exp: 0, Valid: true, Special: fbtype.SpecialNone, NegativeZero: true},
			"-0",
			math.Copysign(0, -1),
			false,
		},

		{"pos_inf", fbtype.DecFloat34{Valid: true, Special: fbtype.SpecialPosInf}, "+Infinity", math.Inf(1), false},
		{"neg_inf", fbtype.DecFloat34{Valid: true, Special: fbtype.SpecialNegInf}, "-Infinity", math.Inf(-1), false},
		{"qnan", fbtype.DecFloat34{Valid: true, Special: fbtype.SpecialQuietNaN}, "NaN", math.NaN(), false},
		{"snan", fbtype.DecFloat34{Valid: true, Special: fbtype.SpecialSignalingNaN}, "sNaN", math.NaN(), false},
		{"neg_qnan", fbtype.DecFloat34{Valid: true, Special: fbtype.SpecialNegQuietNaN}, "-NaN", math.Copysign(math.NaN(), -1), false},
		{"neg_snan", fbtype.DecFloat34{Valid: true, Special: fbtype.SpecialNegSignalingNaN}, "-sNaN", math.Copysign(math.NaN(), -1), false},
	}

	var tests []fbxtest.ValueRoundTripTest
	for _, tc := range testsData {
		tests = append(tests, fbxtest.ValueRoundTripTest{
			Param:  tc.val,
			Result: new(fbtype.DecFloat34),
			Test:   isExpectedEqDec34(tc.val),
		})

		tests = append(tests, fbxtest.ValueRoundTripTest{
			Param:  tc.val,
			Result: new(string),
			Test:   isExpectedEqString(tc.wantStr),
		})

		if !tc.skipFloat {
			tests = append(tests, fbxtest.ValueRoundTripTest{
				Param:  tc.val,
				Result: new(float64),
				Test:   isExpectedEqFloat64(tc.wantF64),
			})
		}
	}

	tests = append(tests,
		fbxtest.ValueRoundTripTest{
			Param:  fbtype.DecFloat34{},
			Result: new(fbtype.DecFloat34),
			Test:   isExpectedEqDec34(fbtype.DecFloat34{}),
		},
		fbxtest.ValueRoundTripTest{
			Param:  nil,
			Result: new(fbtype.DecFloat34),
			Test:   isExpectedEqDec34(fbtype.DecFloat34{}),
		},
	)

	fbxtest.RunValueRoundTripTests(
		context.Background(),
		t,
		defaultConnTestRunner,
		nil,
		"decfloat(34)",
		tests,
	)
}

func TestDecFloat34_ScanTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		want    fbtype.DecFloat34
		wantErr bool
	}{
		{
			name:  "string valid",
			input: "123.45",
			want:  mustDec34(t, "123.45"),
		},
		{
			name:  "bytes valid",
			input: []byte("678.90"),
			want:  mustDec34(t, "678.90"),
		},
		{
			name:  "nil",
			input: nil,
			want:  fbtype.DecFloat34{},
		},
		{
			name:    "unsupported type",
			input:   int64(123),
			wantErr: true,
		},
		{
			name:    "invalid string",
			input:   "not-a-number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d fbtype.DecFloat34
			err := d.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.True(t, isExpectedEqDec34(tt.want)(&d))
		})
	}
}

func TestDecFloat34_Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   fbtype.DecFloat34
		want    string
		wantNil bool
	}{
		{
			name:  "valid",
			input: mustDec34(t, "123.45"),
			want:  "123.45",
		},
		{
			name:  "negative zero",
			input: mustDec34(t, "-0"),
			want:  "-0",
		},
		{
			name:  "special +Infinity",
			input: mustDec34(t, "+Infinity"),
			want:  "+Infinity",
		},
		{
			name:  "special -NaN",
			input: mustDec34(t, "-NaN"),
			want:  "-NaN",
		},
		{
			name:  "special -sNaN",
			input: mustDec34(t, "-sNaN"),
			want:  "-sNaN",
		},
		{
			name:    "invalid (NULL)",
			input:   fbtype.DecFloat34{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.input.Value()
			require.NoError(t, err)

			if tt.wantNil {
				require.Nil(t, val)
			} else {
				require.Equal(t, tt.want, val)
			}
		})
	}
}

func TestDecFloat34_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   fbtype.DecFloat34
		want    string
		wantErr bool
	}{
		{
			name:  "valid number",
			input: mustDec34(t, "123.45"),
			want:  `"123.45"`,
		},
		{
			name:  "negative zero",
			input: mustDec34(t, "-0"),
			want:  `"-0"`,
		},
		{
			name:  "special +Infinity",
			input: mustDec34(t, "+Infinity"),
			want:  `"+Infinity"`,
		},
		{
			name:  "special -NaN",
			input: mustDec34(t, "-NaN"),
			want:  `"-NaN"`,
		},
		{
			name:  "special -sNaN",
			input: mustDec34(t, "-sNaN"),
			want:  `"-sNaN"`,
		},
		{
			name:  "NULL",
			input: fbtype.DecFloat34{},
			want:  `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.MarshalJSON()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))

			var d fbtype.DecFloat34
			err = d.UnmarshalJSON(got)
			require.NoError(t, err)
			require.True(t, isExpectedEqDec34(tt.input)(&d))
		})
	}
}

func TestDecFloat34_SpecialMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      fbtype.DecFloat34
		isSpecial  bool
		isInfinity bool
		isNaN      bool
		isNegZero  bool
	}{
		{
			name:  "normal number",
			input: mustDec34(t, "123.45"),
		},
		{
			name:      "negative zero",
			input:     mustDec34(t, "-0"),
			isNegZero: true,
		},
		{
			name:       "+Infinity",
			input:      mustDec34(t, "+Infinity"),
			isSpecial:  true,
			isInfinity: true,
		},
		{
			name:       "-Infinity",
			input:      mustDec34(t, "-Infinity"),
			isSpecial:  true,
			isInfinity: true,
		},
		{
			name:      "NaN",
			input:     mustDec34(t, "NaN"),
			isSpecial: true,
			isNaN:     true,
		},
		{
			name:      "sNaN",
			input:     mustDec34(t, "sNaN"),
			isSpecial: true,
			isNaN:     true,
		},
		{
			name:      "-NaN",
			input:     mustDec34(t, "-NaN"),
			isSpecial: true,
			isNaN:     true,
		},
		{
			name:      "-sNaN",
			input:     mustDec34(t, "-sNaN"),
			isSpecial: true,
			isNaN:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isSpecial, tt.input.IsSpecial())
			require.Equal(t, tt.isInfinity, tt.input.IsInfinity())
			require.Equal(t, tt.isNaN, tt.input.IsNaN())
			require.Equal(t, tt.isNegZero, tt.input.IsNegativeZero())
		})
	}
}

func TestDecFloat34_Overflow(t *testing.T) {
	t.Parallel()

	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DEC128, 0, 16, 0)

	for _, tt := range []struct {
		name           string
		input          string
		expectEncodeOK bool
	}{
		{"34 digits", "1234567890123456789012345678901234", true},
		{"35 digits", "12345678901234567890123456789012345", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var d fbtype.DecFloat34
			require.NoError(t, d.Scan(tt.input))
			require.True(t, d.Valid)

			encoded, err := m.Encode(typeMetadata, d, nil)
			if tt.expectEncodeOK {
				require.NoError(t, err)
				require.Len(t, encoded, 16)
			} else {
				require.ErrorContains(t, err, "coefficient out of range")
			}
		})
	}
}
