package fbtype_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func mustDec16(t *testing.T, s string) fbtype.DecFloat16 {
	t.Helper()
	var d fbtype.DecFloat16
	require.NoError(t, d.Scan(s))
	return d
}

func isExpectedEqDec16(want fbtype.DecFloat16) func(any) bool {
	return func(got any) bool {
		var d fbtype.DecFloat16
		switch v := got.(type) {
		case *fbtype.DecFloat16:
			if v == nil {
				return !want.Valid
			}
			d = *v
		case fbtype.DecFloat16:
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

func isExpectedEqString(want string) func(any) bool {
	return func(got any) bool {
		if p, ok := got.(*string); ok {
			return p != nil && *p == want
		}
		s, ok := got.(string)
		return ok && s == want
	}
}

func TestDecFloat16Codec_RoundTrip_DB(t *testing.T) {
	t.Parallel()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("decfloat(16) is not supported by server engine version <= 3")
		}
	})

	type tv struct {
		in  string
		out string
	}

	values := []tv{
		{"0", "0"},
		{"1", "1"},
		{"-1", "-1"},
		{"-0", "0"},

		{"0.1", "0.1"},
		{"0.00001", "0.00001"},
		{"0.0000023", "0.0000023"},
		{"-42.125", "-42.125"},
		{"3.141592653589793", "3.141592653589793"},
		{"2343.2323", "2343.2323"},
		{"123.231233", "123.231233"},
		{"3333333333.3", "3333333333.3"},
		{"-0.0000421254333", "-0.0000421254333"},

		{"1.0", "1.0"},
		{"10.00", "10.00"},
		{"100.000", "100.000"},

		{"1E+20", "1E+20"},
		{"1E-20", "1E-20"},
		{"1.234567890123456E+20", "1.234567890123456E+20"},
		{"9.999999999999999E-20", "9.999999999999999E-20"},
		{"1.2E+6", "1200000"},
		{"0.123E+3", "123"},
		{"1230E-2", "12.3"},
		{"0.001E+6", "1000"},
		{"123.0E+2", "12300"},

		{"9999999999999999", "9999999999999999"},
		{"-9999999999999999", "-9999999999999999"},
		{"0.9999999999999999", "0.9999999999999999"},
		{"1.999999999999999", "1.999999999999999"},
		{"9.999999999999999", "9.999999999999999"},
		{"1.000000000000000E+17", "1.000000000000000E+17"},
		{"9.999999999999999E+16", "9.999999999999999E+16"},

		{"1E-398", "1E-398"},
		{"1E-397", "1E-397"},
		{"1E-395", "1E-395"},
		{"1.23E-390", "1.23E-390"},
		{"1E+368", "1E+368"},
		{"9E+368", "9E+368"},
		{"1E+369", "1E+369"},
		{"9.999999999999999E+369", "9.999999999999999E+369"},
		{"9.999999999999999E+384", "9.999999999999999E+384"},

		{"0.0000000000000001", "1E-16"},

		{"999", "999"},
		{"1000", "1000"},
		{"1001", "1001"},
		{"9990009990009990", "9990009990009990"},

		{"+Infinity", "+Infinity"},
		{"-Infinity", "-Infinity"},
		{"NaN", "NaN"},
		{"sNaN", "sNaN"},
	}

	tests := make([]fbxtest.ValueRoundTripTest, 0, len(values)*2+2)

	for _, v := range values {
		d := mustDec16(t, v.in)

		tests = append(tests, fbxtest.ValueRoundTripTest{
			Param:  d,
			Result: new(fbtype.DecFloat16),
			Test:   isExpectedEqDec16(d),
		})
		tests = append(tests, fbxtest.ValueRoundTripTest{
			Param:  d,
			Result: new(string),
			Test:   isExpectedEqString(v.out),
		})
	}

	tests = append(tests,
		fbxtest.ValueRoundTripTest{
			Param:  fbtype.DecFloat16{},
			Result: new(fbtype.DecFloat16),
			Test:   isExpectedEqDec16(fbtype.DecFloat16{}),
		},
		fbxtest.ValueRoundTripTest{
			Param:  nil,
			Result: new(fbtype.DecFloat16),
			Test:   isExpectedEqDec16(fbtype.DecFloat16{}),
		},
	)

	fbxtest.RunValueRoundTripTests(
		context.Background(),
		t,
		defaultConnTestRunner,
		nil,
		"decfloat(16)",
		tests,
	)
}

func TestDecFloat16_ScanTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		want    fbtype.DecFloat16
		wantErr bool
	}{
		{
			name:  "string valid",
			input: "123.45",
			want:  mustDec16(t, "123.45"),
		},
		{
			name:  "bytes valid",
			input: []byte("678.90"),
			want:  mustDec16(t, "678.90"),
		},
		{
			name:  "nil",
			input: nil,
			want:  fbtype.DecFloat16{},
		},
		{
			name:    "unsupported type int64",
			input:   int64(123),
			wantErr: true,
		},
		{
			name:    "unsupported type float64",
			input:   float64(123.45),
			wantErr: true,
		},
		{
			name:    "invalid string",
			input:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "invalid string multiple dots",
			input:   "1.2.3",
			wantErr: true,
		},
		{
			name:    "invalid string exponent only",
			input:   "E10",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d fbtype.DecFloat16
			err := d.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.True(t, isExpectedEqDec16(tt.want)(&d))
		})
	}
}

func TestDecFloat16_Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   fbtype.DecFloat16
		want    string
		wantNil bool
	}{
		{
			name:  "valid number",
			input: mustDec16(t, "123.45"),
			want:  "123.45",
		},
		{
			name:  "negative",
			input: mustDec16(t, "-678.90"),
			want:  "-678.90",
		},
		{
			name:  "zero",
			input: mustDec16(t, "0"),
			want:  "0",
		},
		{
			name:  "special +Infinity",
			input: mustDec16(t, "+Infinity"),
			want:  "+Infinity",
		},
		{
			name:  "special -Infinity",
			input: mustDec16(t, "-Infinity"),
			want:  "-Infinity",
		},
		{
			name:  "special NaN",
			input: mustDec16(t, "NaN"),
			want:  "NaN",
		},
		{
			name:    "invalid (NULL)",
			input:   fbtype.DecFloat16{},
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

func TestDecFloat16_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   fbtype.DecFloat16
		want    string
		wantErr bool
	}{
		{
			name:  "valid number",
			input: mustDec16(t, "123.45"),
			want:  `"123.45"`,
		},
		{
			name:  "zero",
			input: mustDec16(t, "0"),
			want:  `"0"`,
		},
		{
			name:  "negative",
			input: mustDec16(t, "-678.90"),
			want:  `"-678.90"`,
		},
		{
			name:  "special +Infinity",
			input: mustDec16(t, "+Infinity"),
			want:  `"+Infinity"`,
		},
		{
			name:  "special -Infinity",
			input: mustDec16(t, "-Infinity"),
			want:  `"-Infinity"`,
		},
		{
			name:  "special NaN",
			input: mustDec16(t, "NaN"),
			want:  `"NaN"`,
		},
		{
			name:  "NULL",
			input: fbtype.DecFloat16{},
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

			var d fbtype.DecFloat16
			err = d.UnmarshalJSON(got)
			require.NoError(t, err)
			require.True(t, isExpectedEqDec16(tt.input)(&d))
		})
	}
}

func TestDecFloat16_UnmarshalJSON_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"invalid json", `{invalid}`},
		{"invalid number", `"not-a-number"`},
		{"array", `[1,2,3]`},
		{"object", `{"value": 123}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d fbtype.DecFloat16
			err := d.UnmarshalJSON([]byte(tt.input))
			require.Error(t, err)
		})
	}
}

func TestDecFloat16_SpecialMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      fbtype.DecFloat16
		isSpecial  bool
		isInfinity bool
		isNaN      bool
	}{
		{
			name:  "normal number",
			input: mustDec16(t, "123.45"),
		},
		{
			name:  "zero",
			input: mustDec16(t, "0"),
		},
		{
			name:       "+Infinity",
			input:      mustDec16(t, "+Infinity"),
			isSpecial:  true,
			isInfinity: true,
		},
		{
			name:       "-Infinity",
			input:      mustDec16(t, "-Infinity"),
			isSpecial:  true,
			isInfinity: true,
		},
		{
			name:      "NaN",
			input:     mustDec16(t, "NaN"),
			isSpecial: true,
			isNaN:     true,
		},
		{
			name:      "sNaN",
			input:     mustDec16(t, "sNaN"),
			isSpecial: true,
			isNaN:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isSpecial, tt.input.IsSpecial())
			require.Equal(t, tt.isInfinity, tt.input.IsInfinity())
			require.Equal(t, tt.isNaN, tt.input.IsNaN())
		})
	}
}

func TestDecFloat16_Overflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectEncodeOK bool
	}{
		{
			name:           "16 digits OK",
			input:          "1234567890123456",
			expectEncodeOK: true,
		},
		{
			name:           "17 digits - scan OK, encode fail",
			input:          "12345678901234567",
			expectEncodeOK: false,
		},
		{
			name:           "20 digits - too many",
			input:          "12345678901234567890",
			expectEncodeOK: false,
		},
	}
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DEC64, 0, 8, 0)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d fbtype.DecFloat16
			require.NoError(t, d.Scan(tt.input))
			require.True(t, d.Valid)

			encoded, err := m.Encode(typeMetadata, d, nil)
			if tt.expectEncodeOK {
				require.NoError(t, err)
				require.Len(t, encoded, 8)
			} else {
				require.ErrorContains(t, err, "coefficient out of range")
			}
		})
	}
}

func TestDecFloat16_ExponentBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		wantEncodeErr bool
	}{
		{"min_exponent", "1E-398", false},
		{"max_exponent", "9.999999999999999E+369", false},
		{"near_min", "1E-397", false},
		{"near_max", "1E+368", false},

		{"underflow", "1E-399", true},
		{"overflow", "1E+370", true},
	}
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DEC64, 0, 8, 0)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d fbtype.DecFloat16
			require.NoError(t, d.Scan(tt.input))
			require.True(t, d.Valid)

			encoded, err := m.Encode(typeMetadata, d, nil)
			if tt.wantEncodeErr {
				require.ErrorContains(t, err, "exponent out of range")
			} else {
				require.NoError(t, err)
				require.Len(t, encoded, 8)
			}
		})
	}
}
