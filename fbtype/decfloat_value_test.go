package fbtype

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"testing"
)

func TestDecFloatValueCopiesCoefficient(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "DecFloat16",
			run: func(t *testing.T) {
				src := DecFloat16{Int: big.NewInt(42), Exp: -1, Valid: true}

				var scanned DecFloat16
				if err := scanned.ScanDec16(src); err != nil {
					t.Fatal(err)
				}
				src.Int.SetInt64(99)
				if got := scanned.String(); got != "4.2" {
					t.Fatalf("ScanDec16 coefficient changed through alias: %s", got)
				}

				value, err := scanned.Dec16Value()
				if err != nil {
					t.Fatal(err)
				}
				value.Int.SetInt64(77)
				if got := scanned.String(); got != "4.2" {
					t.Fatalf("Dec16Value coefficient changed through alias: %s", got)
				}
			},
		},
		{
			name: "DecFloat34",
			run: func(t *testing.T) {
				src := DecFloat34{Int: big.NewInt(42), Exp: -1, Valid: true}

				var scanned DecFloat34
				if err := scanned.ScanDec34(src); err != nil {
					t.Fatal(err)
				}
				src.Int.SetInt64(99)
				if got := scanned.String(); got != "4.2" {
					t.Fatalf("ScanDec34 coefficient changed through alias: %s", got)
				}

				value, err := scanned.Dec34Value()
				if err != nil {
					t.Fatal(err)
				}
				value.Int.SetInt64(77)
				if got := scanned.String(); got != "4.2" {
					t.Fatalf("Dec34Value coefficient changed through alias: %s", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestDecFloatScanAndJSONClearDestinationOnError(t *testing.T) {
	t.Run("DecFloat16", func(t *testing.T) {
		d := DecFloat16{Int: big.NewInt(42), Valid: true}
		if err := d.Scan("not-a-number"); err == nil {
			t.Fatal("Scan accepted invalid decimal text")
		}
		if d.Valid {
			t.Fatalf("Scan left stale value: %+v", d)
		}

		d = DecFloat16{Int: big.NewInt(42), Valid: true}
		if err := json.Unmarshal([]byte(`"not-a-number"`), &d); err == nil {
			t.Fatal("UnmarshalJSON accepted invalid decimal text")
		}
		if d.Valid {
			t.Fatalf("UnmarshalJSON left stale value: %+v", d)
		}
	})

	t.Run("DecFloat34", func(t *testing.T) {
		d := DecFloat34{Int: big.NewInt(42), Valid: true}
		if err := d.Scan(struct{}{}); err == nil {
			t.Fatal("Scan accepted an unsupported source")
		}
		if d.Valid {
			t.Fatalf("Scan left stale value: %+v", d)
		}

		d = DecFloat34{Int: big.NewInt(42), Valid: true}
		if err := json.Unmarshal([]byte(`{"value": 42}`), &d); err == nil {
			t.Fatal("UnmarshalJSON accepted an object")
		}
		if d.Valid {
			t.Fatalf("UnmarshalJSON left stale value: %+v", d)
		}
	})
}

func TestDecFloatParsingRejectsCombinedExponentOverflow(t *testing.T) {
	tests := []struct {
		name  string
		input string
		scan  func(string) error
	}{
		{
			name:  "DecFloat16 negative overflow",
			input: "1.0E-2147483648",
			scan:  func(s string) error { var d DecFloat16; return d.Scan(s) },
		},
		{
			name:  "DecFloat16 positive overflow",
			input: "10E+2147483647",
			scan:  func(s string) error { var d DecFloat16; return d.Scan(s) },
		},
		{
			name:  "DecFloat34 negative overflow",
			input: "1.0E-2147483648",
			scan:  func(s string) error { var d DecFloat34; return d.Scan(s) },
		},
		{
			name:  "DecFloat34 positive overflow",
			input: "10E+2147483647",
			scan:  func(s string) error { var d DecFloat34; return d.Scan(s) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scan(tt.input)
			if err == nil {
				t.Fatalf("Scan(%q) succeeded", tt.input)
			}
			if !strings.Contains(err.Error(), "exponent") || !strings.Contains(err.Error(), "int32") {
				t.Fatalf("Scan(%q) error = %v", tt.input, err)
			}
		})
	}
}

func TestDecFloatRejectsUnknownSpecialValue(t *testing.T) {
	const unknown = SpecialValue(255)

	tests := []struct {
		name     string
		decValue func() error
		value    func() error
		json     func() error
		encode   func() error
		toFloat  func() error
	}{
		{
			name: "DecFloat16",
			decValue: func() error {
				_, err := (DecFloat16{Valid: true, Special: unknown}).Dec16Value()
				return err
			},
			value: func() error {
				_, err := (DecFloat16{Valid: true, Special: unknown}).Value()
				return err
			},
			json: func() error {
				_, err := json.Marshal(DecFloat16{Valid: true, Special: unknown})
				return err
			},
			encode: func() error {
				_, err := encodeDecFloat16Binary(nil, DecFloat16{Valid: true, Special: unknown})
				return err
			},
			toFloat: func() error {
				_, err := decFloat16ToFloat64(DecFloat16{Valid: true, Special: unknown})
				return err
			},
		},
		{
			name: "DecFloat34",
			decValue: func() error {
				_, err := (DecFloat34{Valid: true, Special: unknown}).Dec34Value()
				return err
			},
			value: func() error {
				_, err := (DecFloat34{Valid: true, Special: unknown}).Value()
				return err
			},
			json: func() error {
				_, err := json.Marshal(DecFloat34{Valid: true, Special: unknown})
				return err
			},
			encode: func() error {
				_, err := encodeDecFloat34Binary(nil, DecFloat34{Valid: true, Special: unknown})
				return err
			},
			toFloat: func() error {
				_, err := decFloat34ToFloat64(DecFloat34{Valid: true, Special: unknown})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, check := range map[string]func() error{
				"DecValue":      tt.decValue,
				"Value":         tt.value,
				"MarshalJSON":   tt.json,
				"binary encode": tt.encode,
				"float64":       tt.toFloat,
			} {
				t.Run(name, func(t *testing.T) {
					if err := check(); err == nil || !strings.Contains(err.Error(), "unknown special value 255") {
						t.Fatalf("error = %v", err)
					}
				})
			}
		})
	}
}

func TestDecFloatInvalidStatePredicatesAreFalse(t *testing.T) {
	for _, tt := range []struct {
		name      string
		special   func() bool
		infinity  func() bool
		nan       func() bool
		negative0 func() bool
	}{
		{
			name:      "DecFloat16",
			special:   func() bool { return (DecFloat16{Special: SpecialPosInf}).IsSpecial() },
			infinity:  func() bool { return (DecFloat16{Special: SpecialPosInf}).IsInfinity() },
			nan:       func() bool { return (DecFloat16{Special: SpecialQuietNaN}).IsNaN() },
			negative0: func() bool { return (DecFloat16{NegativeZero: true}).IsNegativeZero() },
		},
		{
			name:      "DecFloat34",
			special:   func() bool { return (DecFloat34{Special: SpecialPosInf}).IsSpecial() },
			infinity:  func() bool { return (DecFloat34{Special: SpecialPosInf}).IsInfinity() },
			nan:       func() bool { return (DecFloat34{Special: SpecialQuietNaN}).IsNaN() },
			negative0: func() bool { return (DecFloat34{NegativeZero: true}).IsNegativeZero() },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.special() || tt.infinity() || tt.nan() || tt.negative0() {
				t.Fatal("an invalid SQL NULL value satisfied a value predicate")
			}
		})
	}
}

func TestDecFloatPrimitiveStringScanRejectsNull(t *testing.T) {
	var dst string
	if err := (scanPlanBinaryDecFloat16ToString{}).Scan(nil, &dst); err == nil || err.Error() != "cannot scan NULL into *string" {
		t.Fatalf("DecFloat16 string NULL error = %v", err)
	}
	if err := (scanPlanBinaryDecFloat34ToString{}).Scan(nil, &dst); err == nil || err.Error() != "cannot scan NULL into *string" {
		t.Fatalf("DecFloat34 string NULL error = %v", err)
	}
}

func TestDecFloatBinaryDecodeClearsDestinationOnError(t *testing.T) {
	d16 := DecFloat16{Int: big.NewInt(42), Valid: true}
	if err := decodeDecFloat16Binary(make([]byte, 7), &d16); err == nil {
		t.Fatal("decodeDecFloat16Binary accepted an invalid length")
	}
	if d16.Valid {
		t.Fatalf("decodeDecFloat16Binary left stale value: %+v", d16)
	}

	d34 := DecFloat34{Int: big.NewInt(42), Valid: true}
	if err := decodeDecFloat34Binary(make([]byte, 15), &d34); err == nil {
		t.Fatal("decodeDecFloat34Binary accepted an invalid length")
	}
	if d34.Valid {
		t.Fatalf("decodeDecFloat34Binary left stale value: %+v", d34)
	}
}

func TestFloat64ToDecFloat16RejectsUnrepresentableValue(t *testing.T) {
	value := math.Nextafter(1, 2)
	if _, err := Float64ToDecFloat16(value); err == nil || !strings.Contains(err.Error(), "coefficient out of range") {
		t.Fatalf("Float64ToDecFloat16(%g) error = %v", value, err)
	}

	if _, err := Float64ToDecFloat34(value); err != nil {
		t.Fatalf("Float64ToDecFloat34(%g) error = %v", value, err)
	}
}

func TestDecFloat16NegativeZeroConversions(t *testing.T) {
	d, err := Float64ToDecFloat16(math.Copysign(0, -1))
	if err != nil {
		t.Fatal(err)
	}
	if !d.IsNegativeZero() || d.String() != "-0" {
		t.Fatalf("Float64ToDecFloat16(-0) = %+v", d)
	}

	encoded, err := encodeDecFloat16Binary(nil, d)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(encoded); got != "a238000000000000" {
		t.Fatalf("encoded negative zero = %s", got)
	}

	var decoded DecFloat16
	if err := decodeDecFloat16Binary(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.IsNegativeZero() {
		t.Fatalf("decoded negative zero = %+v", decoded)
	}

	f, err := decFloat16ToFloat64(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if f != 0 || !math.Signbit(f) {
		t.Fatalf("negative zero converted to float64 = %v (signbit %t)", f, math.Signbit(f))
	}
}

func TestDecFloat34NegativeZeroConversions(t *testing.T) {
	d, err := Float64ToDecFloat34(math.Copysign(0, -1))
	if err != nil {
		t.Fatal(err)
	}
	if !d.IsNegativeZero() || d.String() != "-0" {
		t.Fatalf("Float64ToDecFloat34(-0) = %+v", d)
	}

	encoded, err := encodeDecFloat34Binary(nil, d)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(encoded); got != "a2080000000000000000000000000000" {
		t.Fatalf("encoded negative zero = %s", got)
	}

	var decoded DecFloat34
	if err := decodeDecFloat34Binary(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.IsNegativeZero() {
		t.Fatalf("decoded negative zero = %+v", decoded)
	}

	f, err := decFloat34ToFloat64(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if f != 0 || !math.Signbit(f) {
		t.Fatalf("negative zero converted to float64 = %v (signbit %t)", f, math.Signbit(f))
	}
}

func TestDecFloatNegativeZeroRejectsInvalidStates(t *testing.T) {
	d16 := DecFloat16{Int: big.NewInt(1), Valid: true, NegativeZero: true}
	if _, err := encodeDecFloat16Binary(nil, d16); err == nil {
		t.Fatal("DecFloat16 encoding accepted NegativeZero with a nonzero coefficient")
	}
	if _, err := d16.Value(); err == nil {
		t.Fatal("DecFloat16.Value accepted NegativeZero with a nonzero coefficient")
	}

	d34 := DecFloat34{Valid: true, Special: SpecialQuietNaN, NegativeZero: true}
	if _, err := encodeDecFloat34Binary(nil, d34); err == nil {
		t.Fatal("DecFloat34 encoding accepted NegativeZero with a special value")
	}
	if _, err := d34.MarshalJSON(); err == nil {
		t.Fatal("DecFloat34.MarshalJSON accepted NegativeZero with a special value")
	}
}

func TestDecFloatSignedNaNBinary(t *testing.T) {
	tests := []struct {
		name    string
		special SpecialValue
		encode  func(SpecialValue) ([]byte, error)
		decode  func([]byte) (SpecialValue, error)
		wantHex string
	}{
		{
			name: "decimal64 negative NaN", special: SpecialNegQuietNaN,
			encode: func(s SpecialValue) ([]byte, error) {
				return encodeDecFloat16Binary(nil, DecFloat16{Valid: true, Special: s})
			},
			decode: func(src []byte) (SpecialValue, error) {
				var d DecFloat16
				err := decodeDecFloat16Binary(src, &d)
				return d.Special, err
			},
			wantHex: "fc00000000000000",
		},
		{
			name: "decimal64 negative signaling NaN", special: SpecialNegSignalingNaN,
			encode: func(s SpecialValue) ([]byte, error) {
				return encodeDecFloat16Binary(nil, DecFloat16{Valid: true, Special: s})
			},
			decode: func(src []byte) (SpecialValue, error) {
				var d DecFloat16
				err := decodeDecFloat16Binary(src, &d)
				return d.Special, err
			},
			wantHex: "fe00000000000000",
		},
		{
			name: "decimal128 negative NaN", special: SpecialNegQuietNaN,
			encode: func(s SpecialValue) ([]byte, error) {
				return encodeDecFloat34Binary(nil, DecFloat34{Valid: true, Special: s})
			},
			decode: func(src []byte) (SpecialValue, error) {
				var d DecFloat34
				err := decodeDecFloat34Binary(src, &d)
				return d.Special, err
			},
			wantHex: "fc000000000000000000000000000000",
		},
		{
			name: "decimal128 negative signaling NaN", special: SpecialNegSignalingNaN,
			encode: func(s SpecialValue) ([]byte, error) {
				return encodeDecFloat34Binary(nil, DecFloat34{Valid: true, Special: s})
			},
			decode: func(src []byte) (SpecialValue, error) {
				var d DecFloat34
				err := decodeDecFloat34Binary(src, &d)
				return d.Special, err
			},
			wantHex: "fe000000000000000000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.encode(tt.special)
			if err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(encoded); got != tt.wantHex {
				t.Fatalf("encoded = %s, want %s", got, tt.wantHex)
			}

			decoded, err := tt.decode(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if decoded != tt.special {
				t.Fatalf("decoded special = %v, want %v", decoded, tt.special)
			}
		})
	}
}

func TestDecFloatSignedNaNFloat64(t *testing.T) {
	negativeNaN := math.Float64frombits(0xfff8000000000001)
	if !math.IsNaN(negativeNaN) || !math.Signbit(negativeNaN) {
		t.Fatal("test value is not a negative NaN")
	}

	d16, err := Float64ToDecFloat16(negativeNaN)
	if err != nil {
		t.Fatal(err)
	}
	if d16.Special != SpecialNegQuietNaN {
		t.Fatalf("Float64ToDecFloat16(-NaN) = %v", d16.Special)
	}
	f16, err := decFloat16ToFloat64(d16)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(f16) || !math.Signbit(f16) {
		t.Fatalf("DecFloat16 -NaN converted to float64 signbit %t", math.Signbit(f16))
	}

	d34, err := Float64ToDecFloat34(negativeNaN)
	if err != nil {
		t.Fatal(err)
	}
	if d34.Special != SpecialNegQuietNaN {
		t.Fatalf("Float64ToDecFloat34(-NaN) = %v", d34.Special)
	}
	f34, err := decFloat34ToFloat64(d34)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(f34) || !math.Signbit(f34) {
		t.Fatalf("DecFloat34 -NaN converted to float64 signbit %t", math.Signbit(f34))
	}
}

func TestDecFloatSignedNaNText(t *testing.T) {
	tests := []struct {
		input string
		want  SpecialValue
	}{
		{input: "NaN", want: SpecialQuietNaN},
		{input: "+NaN", want: SpecialQuietNaN},
		{input: "-NaN", want: SpecialNegQuietNaN},
		{input: "sNaN", want: SpecialSignalingNaN},
		{input: "+sNaN", want: SpecialSignalingNaN},
		{input: "-sNaN", want: SpecialNegSignalingNaN},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var d16 DecFloat16
			if err := parseDecFloat16String(tt.input, &d16); err != nil {
				t.Fatal(err)
			}
			if d16.Special != tt.want {
				t.Fatalf("DecFloat16 special = %v, want %v", d16.Special, tt.want)
			}

			var d34 DecFloat34
			if err := parseDecFloat34String(tt.input, &d34); err != nil {
				t.Fatal(err)
			}
			if d34.Special != tt.want {
				t.Fatalf("DecFloat34 special = %v, want %v", d34.Special, tt.want)
			}
		})
	}
}
