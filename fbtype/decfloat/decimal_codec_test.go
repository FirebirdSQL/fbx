package decfloat

import (
	"encoding/hex"
	"math/big"
	"testing"
)

func TestNewDecimalFormatRejectsInvalidParameters(t *testing.T) {
	tests := []struct {
		name              string
		formatBitLength   int
		coefficientDigits int
	}{
		{name: "zero bit length", formatBitLength: 0, coefficientDigits: 16},
		{name: "non-byte bit length", formatBitLength: 63, coefficientDigits: 16},
		{name: "invalid coefficient digits", formatBitLength: 64, coefficientDigits: 15},
		{name: "coefficient precision above decimal128", formatBitLength: 128, coefficientDigits: 37},
		{name: "insufficient exponent bits", formatBitLength: 8, coefficientDigits: 4},
		{name: "exponent width above portable limit", formatBitLength: 80, coefficientDigits: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("NewDecimalFormat(%d, %d) did not panic", tt.formatBitLength, tt.coefficientDigits)
				}
			}()

			NewDecimalFormat(tt.formatBitLength, tt.coefficientDigits)
		})
	}
}

func TestNewDecimalFormatAcceptsPortableBoundary(t *testing.T) {
	format := NewDecimalFormat(144, 34)
	if format.exponentContinuationBits != 28 {
		t.Fatalf("exponentContinuationBits=%d want=28", format.exponentContinuationBits)
	}
	if err := format.validate(); err != nil {
		t.Fatalf("boundary format is invalid: %v", err)
	}
}

func TestDecimalFormatConstants(t *testing.T) {
	tests := []struct {
		name              string
		format            DecimalFormat
		byteLength        int
		coefficientDigits int
		exponentBits      int
		exponentLimit     int
		exponentBias      int
	}{
		{
			name: "decimal64", format: Decimal64, byteLength: 8,
			coefficientDigits: 16, exponentBits: 8, exponentLimit: 767, exponentBias: 398,
		},
		{
			name: "decimal128", format: Decimal128, byteLength: 16,
			coefficientDigits: 34, exponentBits: 12, exponentLimit: 12287, exponentBias: 6176,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.format.formatByteLength != tt.byteLength ||
				tt.format.coefficientDigits != tt.coefficientDigits ||
				tt.format.exponentContinuationBits != tt.exponentBits ||
				tt.format.eLimit != tt.exponentLimit ||
				tt.format.BiasedExponent(0) != tt.exponentBias {
				t.Fatalf("unexpected format: %+v", tt.format)
			}
		})
	}
}

func TestDecimalCodecEncodeFiniteValidatesAndClearsBuffer(t *testing.T) {
	codec := NewDecimalCodec(Decimal64)

	if err := codec.EncodeFinite(make([]byte, 7), big.NewInt(1), 0); err == nil {
		t.Fatal("EncodeFinite accepted a short buffer")
	}
	if err := codec.EncodeFinite(make([]byte, 8), nil, 0); err == nil {
		t.Fatal("EncodeFinite accepted a nil coefficient")
	}

	buf := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	if err := codec.EncodeFinite(buf, big.NewInt(1), 0); err != nil {
		t.Fatalf("EncodeFinite failed: %v", err)
	}
	if got := hex.EncodeToString(buf); got != "a238000000000001" {
		t.Fatalf("EncodeFinite result = %s, want a238000000000001", got)
	}
}

func TestDecimalCodecEncodeRejectsInvalidEnums(t *testing.T) {
	codec := NewDecimalCodec(Decimal64)

	if _, err := codec.Encode(DecodedDecimal{Signum: 0, Type: DecimalFinite}); err == nil {
		t.Fatal("Encode accepted an unknown Signum")
	}
	if _, err := codec.Encode(DecodedDecimal{Signum: SignumPositive, Type: DecimalType(255)}); err == nil {
		t.Fatal("Encode accepted an unknown DecimalType")
	}
}

func TestDecimalCodecEncodeValidatesFiniteSign(t *testing.T) {
	codec := NewDecimalCodec(Decimal64)

	tests := []struct {
		name        string
		signum      Signum
		coefficient int64
		want        string
		wantErr     bool
	}{
		{name: "positive", signum: SignumPositive, coefficient: 750, want: "22380000000003d0"},
		{name: "negative", signum: SignumNegative, coefficient: -750, want: "a2380000000003d0"},
		{name: "positive zero", signum: SignumPositive, coefficient: 0, want: "2238000000000000"},
		{name: "negative zero", signum: SignumNegative, coefficient: 0, want: "a238000000000000"},
		{name: "positive sign with negative coefficient", signum: SignumPositive, coefficient: -750, wantErr: true},
		{name: "negative sign with positive coefficient", signum: SignumNegative, coefficient: 750, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := codec.Encode(DecodedDecimal{
				Signum:      tt.signum,
				Type:        DecimalFinite,
				Coefficient: big.NewInt(tt.coefficient),
			})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Encode accepted signum %d with coefficient %d", tt.signum, tt.coefficient)
				}
				return
			}
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if got := hex.EncodeToString(encoded); got != tt.want {
				t.Fatalf("Encode result = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDecimalCodecZeroValueReturnsErrors(t *testing.T) {
	var codec DecimalCodec

	if _, err := codec.ParseBytes(nil); err == nil {
		t.Fatal("zero DecimalCodec ParseBytes did not return an error")
	}
	if _, err := codec.Encode(DecodedDecimal{}); err == nil {
		t.Fatal("zero DecimalCodec Encode did not return an error")
	}
	if err := codec.EncodeFinite(nil, big.NewInt(0), 0); err == nil {
		t.Fatal("zero DecimalCodec EncodeFinite did not return an error")
	}

	var nilCodec *DecimalCodec
	if _, err := nilCodec.ParseBytes(nil); err == nil {
		t.Fatal("nil DecimalCodec ParseBytes did not return an error")
	}
}

func TestDecodeExponent_PortFromJaybird(t *testing.T) {
	if got := decodeExponent([]byte{0xFF, 0b0100}, 1, 6); got != 0b01000001 {
		t.Fatalf("got=%b want=%b", got, 0b01000001)
	}
	if got := decodeExponent([]byte{0xFF, 0b1111}, 1, 6); got != 0b01000011 {
		t.Fatalf("got=%b want=%b", got, 0b01000011)
	}
	if got := decodeExponent([]byte{0xFF, 0b01000}, 1, 6); got != 0b01000010 {
		t.Fatalf("got=%b want=%b", got, 0b01000010)
	}
	if got := decodeExponent([]byte{0xFF, 0b010000}, 1, 6); got != 0b01000100 {
		t.Fatalf("got=%b want=%b", got, 0b01000100)
	}
	if got := decodeExponent([]byte{0xFF, 0b0100000}, 1, 6); got != 0b01001000 {
		t.Fatalf("got=%b want=%b", got, 0b01001000)
	}
	if got := decodeExponent([]byte{0xFF, 0b01000000}, 1, 6); got != 0b01010000 {
		t.Fatalf("got=%b want=%b", got, 0b01010000)
	}
	if got := decodeExponent([]byte{0xFF, 0b10000000}, 1, 6); got != 0b01100000 {
		t.Fatalf("got=%b want=%b", got, 0b01100000)
	}
	if got := decodeExponent([]byte{0xFF, 0b10000000}, 1, 5); got != 0b0110000 {
		t.Fatalf("got=%b want=%b", got, 0b0110000)
	}
	if got := decodeExponent([]byte{0xFF, 0b10000000}, 1, 4); got != 0b011000 {
		t.Fatalf("got=%b want=%b", got, 0b011000)
	}
	if got := decodeExponent([]byte{0xFF, 0b10000000}, 1, 3); got != 0b01100 {
		t.Fatalf("got=%b want=%b", got, 0b01100)
	}
	if got := decodeExponent([]byte{0xFF, 0b10000000}, 1, 2); got != 0b0110 {
		t.Fatalf("got=%b want=%b", got, 0b0110)
	}
	if got := decodeExponent([]byte{0xFF, 0b10000000}, 1, 1); got != 0b011 {
		t.Fatalf("got=%b want=%b", got, 0b011)
	}
	if got := decodeExponent([]byte{0xFF, 0b10000000}, 1, 0); got != 0b01 {
		t.Fatalf("got=%b want=%b", got, 0b01)
	}
	if got := decodeExponent([]byte{0xFF, 0b1111}, 1, 8); got != 0b0100001111 {
		t.Fatalf("got=%b want=%b", got, 0b0100001111)
	}
	if got := decodeExponent([]byte{0xFF, 0b1100}, 1, 8); got != 0b0100001100 {
		t.Fatalf("got=%b want=%b", got, 0b0100001100)
	}
	if got := decodeExponent([]byte{0xFF, 0b1111, 0xFF}, 1, 9); got != 0b01000011111 {
		t.Fatalf("got=%b want=%b", got, 0b01000011111)
	}
	if got := decodeExponent([]byte{0xFF, 0b1100, 0xFF}, 1, 9); got != 0b01000011001 {
		t.Fatalf("got=%b want=%b", got, 0b01000011001)
	}
	if got := decodeExponent([]byte{0xFF, 0b1100, 0xFF}, 1, 10); got != 0b010000110011 {
		t.Fatalf("got=%b want=%b", got, 0b010000110011)
	}
}

func TestDecimal64_RoundTrip_Finite(t *testing.T) {
	c := NewDecimalCodec(Decimal64)

	tests := []struct {
		coeff string
		exp   int32
	}{
		{"0", 0},
		{"1", 0},
		{"12323123", -1},
		{"9999999999999999", 0},
		{"42", -10},
	}

	for _, tt := range tests {
		coef, _ := new(big.Int).SetString(tt.coeff, 10)
		src := DecodedDecimal{
			Signum:      SignumPositive,
			Type:        DecimalFinite,
			Exponent:    tt.exp,
			Coefficient: coef,
		}
		b, err := c.Encode(src)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}
		dec, err := c.ParseBytes(b)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if dec.Type != DecimalFinite || dec.Signum != SignumPositive || dec.Exponent != tt.exp || dec.Coefficient.Cmp(coef) != 0 {
			t.Fatalf("roundtrip mismatch: in=%v out=%v bytes=%x", src, dec, b)
		}
	}
}

func TestDecimal64_Specials(t *testing.T) {
	c := NewDecimalCodec(Decimal64)

	for _, typ := range []DecimalType{DecimalInfinity, DecimalNaN, DecimalSignalingNaN} {
		b, err := c.Encode(DecodedDecimal{Signum: SignumPositive, Type: typ})
		if err != nil {
			t.Fatal(err)
		}
		dec, err := c.ParseBytes(b)
		if err != nil {
			t.Fatal(err)
		}
		if dec.Type != typ || dec.Signum != SignumPositive {
			t.Fatalf("special mismatch: want=%s got=%v bytes=%x", typ.String(), dec, b)
		}
	}
}

func TestDecimalCodec_ReferenceByteVectors(t *testing.T) {
	tests := []struct {
		name        string
		format      DecimalFormat
		encoded     string
		canonical   string
		signum      Signum
		typ         DecimalType
		coefficient string
		exponent    int32
	}{
		{
			name: "decimal64 integer", format: Decimal64,
			encoded: "2238000000000010", canonical: "2238000000000010",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "10",
		},
		{
			name: "decimal64 negative fraction", format: Decimal64,
			encoded: "a2300000000003d0", canonical: "a2300000000003d0",
			signum: SignumNegative, typ: DecimalFinite, coefficient: "-750", exponent: -2,
		},
		{
			name: "decimal64 full coefficient", format: Decimal64,
			encoded: "263934b9c1e28e56", canonical: "263934b9c1e28e56",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "1234567890123456",
		},
		{
			name: "decimal64 msd nine", format: Decimal64,
			encoded: "6e38ff3fcff3fcff", canonical: "6e38ff3fcff3fcff",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "9999999999999999",
		},
		{
			name: "decimal64 maximum exponent", format: Decimal64,
			encoded: "77fcff3fcff3fcff", canonical: "77fcff3fcff3fcff",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "9999999999999999", exponent: 369,
		},
		{
			name: "decimal64 minimum exponent", format: Decimal64,
			encoded: "0000000000000001", canonical: "0000000000000001",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "1", exponent: -398,
		},
		{
			name: "decimal64 exponent continuation", format: Decimal64,
			encoded: "23c4000000000007", canonical: "23c4000000000007",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "7", exponent: 99,
		},
		{
			name: "decimal64 negative zero", format: Decimal64,
			encoded: "a238000000000000", canonical: "a238000000000000",
			signum: SignumNegative, typ: DecimalFinite, coefficient: "0",
		},
		{
			name: "decimal64 non-canonical DPD", format: Decimal64,
			encoded: "223800000000016e", canonical: "223800000000006e",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "888",
		},
		{
			name: "decimal64 infinity payload", format: Decimal64,
			encoded: "7a7a7a7a7a7a7a7a", canonical: "7800000000000000",
			signum: SignumPositive, typ: DecimalInfinity,
		},
		{
			name: "decimal64 NaN payload", format: Decimal64,
			encoded: "7d7d7d7d7d7d7d7d", canonical: "7c00000000000000",
			signum: SignumPositive, typ: DecimalNaN,
		},
		{
			name: "decimal64 negative signaling NaN payload", format: Decimal64,
			encoded: "ffffffffffffffff", canonical: "fe00000000000000",
			signum: SignumNegative, typ: DecimalSignalingNaN,
		},
		{
			name: "decimal128 integer", format: Decimal128,
			encoded: "22080000000000000000000000000010", canonical: "22080000000000000000000000000010",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "10",
		},
		{
			name: "decimal128 negative fraction", format: Decimal128,
			encoded: "a20780000000000000000000000003d0", canonical: "a20780000000000000000000000003d0",
			signum: SignumNegative, typ: DecimalFinite, coefficient: "-750", exponent: -2,
		},
		{
			name: "decimal128 full coefficient", format: Decimal128,
			encoded: "2608134b9c1e28e56f3c127177823534", canonical: "2608134b9c1e28e56f3c127177823534",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "1234567890123456789012345678901234",
		},
		{
			name: "decimal128 maximum exponent", format: Decimal128,
			encoded: "77ffcff3fcff3fcff3fcff3fcff3fcff", canonical: "77ffcff3fcff3fcff3fcff3fcff3fcff",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "9999999999999999999999999999999999", exponent: 6111,
		},
		{
			name: "decimal128 minimum exponent", format: Decimal128,
			encoded: "00000000000000000000000000000001", canonical: "00000000000000000000000000000001",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "1", exponent: -6176,
		},
		{
			name: "decimal128 exponent continuation", format: Decimal128,
			encoded: "2220c000000000000000000000000007", canonical: "2220c000000000000000000000000007",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "7", exponent: 99,
		},
		{
			name: "decimal128 non-canonical DPD", format: Decimal128,
			encoded: "220800000000000000000000000003ff", canonical: "220800000000000000000000000000ff",
			signum: SignumPositive, typ: DecimalFinite, coefficient: "999",
		},
		{
			name: "decimal128 infinity payload", format: Decimal128,
			encoded: "7b7b7b7b7b7b7b7b7b7b7b7b7b7b7b7b", canonical: "78000000000000000000000000000000",
			signum: SignumPositive, typ: DecimalInfinity,
		},
		{
			name: "decimal128 negative NaN payload", format: Decimal128,
			encoded: "fd7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d", canonical: "fc000000000000000000000000000000",
			signum: SignumNegative, typ: DecimalNaN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewDecimalCodec(tt.format)
			encoded, err := hex.DecodeString(tt.encoded)
			if err != nil {
				t.Fatal(err)
			}

			decoded, err := codec.ParseBytes(encoded)
			if err != nil {
				t.Fatalf("ParseBytes failed: %v", err)
			}
			if decoded.Signum != tt.signum || decoded.Type != tt.typ || decoded.Exponent != tt.exponent {
				t.Fatalf("ParseBytes(%s) = %v", tt.encoded, decoded)
			}

			if tt.typ == DecimalFinite {
				wantCoefficient, ok := new(big.Int).SetString(tt.coefficient, 10)
				if !ok {
					t.Fatalf("invalid test coefficient %q", tt.coefficient)
				}
				if decoded.Coefficient == nil || decoded.Coefficient.Cmp(wantCoefficient) != 0 {
					t.Fatalf("coefficient = %v, want %s", decoded.Coefficient, wantCoefficient)
				}
			} else if decoded.Coefficient != nil {
				t.Fatalf("special coefficient = %v, want nil", decoded.Coefficient)
			}

			canonical, err := codec.Encode(decoded)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if got := hex.EncodeToString(canonical); got != tt.canonical {
				t.Fatalf("canonical encoding = %s, want %s", got, tt.canonical)
			}
		})
	}
}

func TestDecimalTypeFromEveryFirstByte(t *testing.T) {
	for firstByte := 0; firstByte <= 0xff; firstByte++ {
		combination := (firstByte >> 2) & 0x1f
		want := DecimalFinite
		switch combination {
		case 0x1e:
			want = DecimalInfinity
		case 0x1f:
			if firstByte&0x02 == 0 {
				want = DecimalNaN
			} else {
				want = DecimalSignalingNaN
			}
		}

		if got := decimalTypeFromFirstByte(firstByte); got != want {
			t.Fatalf("decimalTypeFromFirstByte(0x%02x) = %s, want %s", firstByte, got, want)
		}
	}
}
