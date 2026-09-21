package fbtype

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"rdb.red-soft.ru/fbx/fbproto"
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

func TestDecFloatParsingRejectsUnrepresentableLargeExponents(t *testing.T) {
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
			input: "1E+2147483648",
			scan:  func(s string) error { var d DecFloat16; return d.Scan(s) },
		},
		{
			name:  "DecFloat34 negative overflow",
			input: "1.0E-2147483648",
			scan:  func(s string) error { var d DecFloat34; return d.Scan(s) },
		},
		{
			name:  "DecFloat34 positive overflow",
			input: "1E+2147483648",
			scan:  func(s string) error { var d DecFloat34; return d.Scan(s) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scan(tt.input)
			if err == nil {
				t.Fatalf("Scan(%q) succeeded", tt.input)
			}
			if !strings.Contains(err.Error(), "exponent out of range") {
				t.Fatalf("Scan(%q) error = %v", tt.input, err)
			}
		})
	}
}

func TestDecFloatTextNormalization(t *testing.T) {
	type decimalValue interface {
		Scan(any) error
		String() string
		Value() (driver.Value, error)
	}
	for _, spec := range []struct {
		name      string
		precision int
		min, max  int32
		metadata  TypeMetadata
		newValue  func() decimalValue
	}{
		{"DecFloat16", 16, -398, 369, NewTypeMetadata(fbproto.SQL_TYPE_DEC64, 0, 8, 0), func() decimalValue { return &DecFloat16{} }},
		{"DecFloat34", 34, -6176, 6111, NewTypeMetadata(fbproto.SQL_TYPE_DEC128, 0, 16, 0), func() decimalValue { return &DecFloat34{} }},
	} {
		t.Run(spec.name, func(t *testing.T) {
			fullPower := "1" + strings.Repeat("0", spec.precision-1)
			tests := []struct {
				name, input, coefficient string
				exponent                 int32
				negativeZero             bool
			}{
				{"upper one", fmt.Sprintf("1E%d", spec.max+1), "10", spec.max, false},
				{"upper ten", fmt.Sprintf("1E%d", spec.max+10), "10000000000", spec.max, false},
				{"upper negative", fmt.Sprintf("-12E%d", spec.max+1), "-120", spec.max, false},
				{"upper maximum power", fmt.Sprintf("1E%d", spec.max+int32(spec.precision)-1), fullPower, spec.max, false},
				{"lower one", fmt.Sprintf("1.0E%d", spec.min), "1", spec.min, false},
				{"lower ten", fmt.Sprintf("10000000000E%d", spec.min-10), "1", spec.min, false},
				{"lower negative", fmt.Sprintf("-12.00E%d", spec.min), "-12", spec.min, false},
				{"lower preserves remaining zeros", fmt.Sprintf("1000E%d", spec.min-1), "100", spec.min, false},
				{"excess coefficient zeros", fullPower + "0", fullPower, 1, false},
				{"excess coefficient zeros at lower bound", fmt.Sprintf("%s0E%d", fullPower, spec.min-1), fullPower, spec.min, false},
				{"excess coefficient zeros at upper bound", fmt.Sprintf("%s0E%d", fullPower, spec.max-1), fullPower, spec.max, false},
				{"excess zeros then lower rescale", fmt.Sprintf("%s0000E%d", fullPower, spec.min-5), fullPower[:len(fullPower)-1], spec.min, false},
				{"long exact integer", "1" + strings.Repeat("0", 100_000) + "E-100000", fullPower, 1 - int32(spec.precision), false},
				{"long exact fraction", "1." + strings.Repeat("0", 100_000), fullPower, 1 - int32(spec.precision), false},
				{"long leading fractional zeros", "0." + strings.Repeat("0", 100_000) + "1E100001", "1", 0, false},
				{"preserve scale", "1.2300", "12300", -4, false},
				{"preserve zero scale", "-0.00", "0", -2, true},
				{"upper zero", fmt.Sprintf("0E%d", spec.max+1), "0", spec.max, false},
				{"lower negative zero", fmt.Sprintf("-0E%d", spec.min-1), "0", spec.min, true},
				{"zero beyond int32", "-0.0E-2147483648", "0", spec.min, true},
				{"zero at int64 maximum", "0E9223372036854775807", "0", spec.max, false},
				{"zero at int64 minimum", "0E-9223372036854775808", "0", spec.min, false},
				{"zero combined exponent underflow", "-0.0E-9223372036854775808", "0", spec.min, true},
				{"leading zeros and sign", fmt.Sprintf(" +0001e+%d ", spec.max+1), "10", spec.max, false},
			}
			m := NewMap()
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					check := func(value decimalValue) {
						t.Helper()
						switch d := value.(type) {
						case *DecFloat16:
							assertDecFloat16Representation(t, *d, tt.coefficient, tt.exponent, tt.negativeZero)
						case *DecFloat34:
							assertDecFloat34Representation(t, *d, tt.coefficient, tt.exponent, tt.negativeZero)
						}
					}
					value := spec.newValue()
					for _, src := range []any{tt.input, []byte(tt.input)} {
						if err := value.Scan(src); err != nil {
							t.Fatalf("Scan(%T): %v", src, err)
						}
						check(value)
					}
					encodedJSON, err := json.Marshal(tt.input)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(encodedJSON, value); err != nil {
						t.Fatal(err)
					}
					check(value)
					driverValue, err := value.Value()
					if err != nil || driverValue != value.String() {
						t.Fatalf("Value() = %v, %v", driverValue, err)
					}
					encodedJSON, err = json.Marshal(value)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(encodedJSON, value); err != nil {
						t.Fatal(err)
					}
					check(value)
					for _, src := range []any{tt.input, value, driverValue} {
						prefix := []byte{0xaa, 0xbb}
						wire, err := m.Encode(spec.metadata, src, prefix)
						if err != nil {
							t.Fatalf("Encode(%T): %v", src, err)
						}
						if len(wire) != len(prefix)+int(spec.metadata.Length) || !bytes.Equal(wire[:len(prefix)], prefix) {
							t.Fatalf("unexpected wire buffer: %x", wire)
						}
						decoded := spec.newValue()
						if err := m.Scan(spec.metadata, wire[len(prefix):], decoded); err != nil {
							t.Fatal(err)
						}
						check(decoded)
					}
				})
			}
			for _, input := range []string{
				fmt.Sprintf("1E%d", spec.max+int32(spec.precision)),
				fmt.Sprintf("%sE%d", strings.Repeat("9", spec.precision), spec.max+1),
				fmt.Sprintf("1E%d", spec.min-1),
				fmt.Sprintf("110E%d", spec.min-2),
				fmt.Sprintf("%s0E%d", fullPower, spec.max),
				"1" + strings.Repeat("0", spec.precision-1) + "1",
				"1E1000000000", "1E-1000000000",
				"1E9223372036854775807", "1E-9223372036854775808",
				"1.0E-9223372036854775808", fullPower + "0E9223372036854775807",
				fullPower + "000x", fullPower + "000.0.0",
			} {
				t.Run("reject/"+input, func(t *testing.T) {
					value := spec.newValue()
					checkNull := func() {
						t.Helper()
						got, err := value.Value()
						if err != nil || got != nil {
							t.Fatalf("failed parse left stale value: %v, %v", got, err)
						}
					}
					for _, src := range []any{input, []byte(input)} {
						if err := value.Scan("42"); err != nil {
							t.Fatal(err)
						}
						if err := value.Scan(src); err == nil {
							t.Fatalf("Scan accepted %q", input)
						}
						checkNull()
					}
					if err := value.Scan("42"); err != nil {
						t.Fatal(err)
					}
					encodedJSON, err := json.Marshal(input)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(encodedJSON, value); err == nil {
						t.Fatalf("UnmarshalJSON accepted %q", input)
					}
					checkNull()
					if _, err := m.Encode(spec.metadata, input, nil); err == nil {
						t.Fatalf("Encode accepted %q", input)
					}
				})
			}
		})
	}
}

func TestDecFloatParsingRejectsInt64ExponentOverflow(t *testing.T) {
	for _, tt := range []struct {
		name string
		scan func(string) error
	}{
		{name: "DecFloat16", scan: func(s string) error { var d DecFloat16; return d.Scan(s) }},
		{name: "DecFloat34", scan: func(s string) error { var d DecFloat34; return d.Scan(s) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scan("1.0E-9223372036854775808")
			if err == nil || !strings.Contains(err.Error(), "int64 range") {
				t.Fatalf("Scan error = %v, want int64 range error", err)
			}
		})
	}
}

func TestDecFloatParsingBoundsCoefficientWork(t *testing.T) {
	oversized := strings.Repeat("9", 100_000)
	leadingZeros := strings.Repeat("0", 100_000) + "1"
	oversizedExponent := "1E+" + strings.Repeat("9", 100_000)

	t.Run("DecFloat16", func(t *testing.T) {
		var d DecFloat16
		err := d.Scan(oversized)
		if err == nil || !strings.Contains(err.Error(), "more than 16 decimal digits") {
			t.Fatalf("Scan oversized coefficient error = %v", err)
		}
		if len(err.Error()) > 512 {
			t.Fatalf("oversized coefficient error contains unbounded input: %d bytes", len(err.Error()))
		}
		err = d.Scan(oversizedExponent)
		if err == nil || !strings.Contains(err.Error(), "int64 range") {
			t.Fatalf("Scan oversized exponent error = %v", err)
		}
		if len(err.Error()) > 512 {
			t.Fatalf("oversized exponent error contains unbounded input: %d bytes", len(err.Error()))
		}

		if err := d.Scan(leadingZeros); err != nil {
			t.Fatal(err)
		}
		assertDecFloat16Representation(t, d, "1", 0, false)
	})

	t.Run("DecFloat34", func(t *testing.T) {
		var d DecFloat34
		err := d.Scan(oversized)
		if err == nil || !strings.Contains(err.Error(), "more than 34 decimal digits") {
			t.Fatalf("Scan oversized coefficient error = %v", err)
		}
		if len(err.Error()) > 512 {
			t.Fatalf("oversized coefficient error contains unbounded input: %d bytes", len(err.Error()))
		}
		err = d.Scan(oversizedExponent)
		if err == nil || !strings.Contains(err.Error(), "int64 range") {
			t.Fatalf("Scan oversized exponent error = %v", err)
		}
		if len(err.Error()) > 512 {
			t.Fatalf("oversized exponent error contains unbounded input: %d bytes", len(err.Error()))
		}

		if err := d.Scan(leadingZeros); err != nil {
			t.Fatal(err)
		}
		assertDecFloat34Representation(t, d, "1", 0, false)
	})
}

func TestDecFloatTextRoundTripPreservesRepresentation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		coefficient  string
		exponent     int32
		negativeZero bool
		canonical    string
	}{
		{name: "positive zero", input: "0", coefficient: "0", canonical: "0"},
		{name: "negative zero", input: "-0", coefficient: "0", negativeZero: true, canonical: "-0"},
		{name: "zero with scale", input: "0.00", coefficient: "0", exponent: -2, canonical: "0.00"},
		{name: "negative zero with scale", input: "-0.00", coefficient: "0", exponent: -2, negativeZero: true, canonical: "-0.00"},
		{name: "zero with positive exponent", input: "0E+5", coefficient: "0", exponent: 5, canonical: "0E+5"},
		{name: "negative zero with minimum decimal64 exponent", input: "-0E-398", coefficient: "0", exponent: -398, negativeZero: true, canonical: "-0E-398"},
		{name: "integer trailing zero", input: "120", coefficient: "120", canonical: "120"},
		{name: "positive exponent", input: "1.2E+6", coefficient: "12", exponent: 5, canonical: "1.2E+6"},
		{name: "trailing coefficient zero", input: "1230E-2", coefficient: "1230", exponent: -2, canonical: "12.30"},
		{name: "positive exponent from fraction", input: "0.001E+6", coefficient: "1", exponent: 3, canonical: "1E+3"},
		{name: "positive exponent with scale", input: "123.0E+2", coefficient: "1230", exponent: 1, canonical: "1.230E+4"},
		{name: "plain notation boundary", input: "0.000001", coefficient: "1", exponent: -6, canonical: "0.000001"},
		{name: "scientific notation boundary", input: "0.0000001", coefficient: "1", exponent: -7, canonical: "1E-7"},
	}

	t.Run("DecFloat16", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var value DecFloat16
				if err := value.Scan(tt.input); err != nil {
					t.Fatal(err)
				}
				assertDecFloat16Representation(t, value, tt.coefficient, tt.exponent, tt.negativeZero)
				if got := value.String(); got != tt.canonical {
					t.Fatalf("String() = %q, want %q", got, tt.canonical)
				}

				driverValue, err := value.Value()
				if err != nil {
					t.Fatal(err)
				}
				if driverValue != tt.canonical {
					t.Fatalf("Value() = %q, want %q", driverValue, tt.canonical)
				}

				jsonValue, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				var decoded DecFloat16
				if err := json.Unmarshal(jsonValue, &decoded); err != nil {
					t.Fatal(err)
				}
				assertDecFloat16Representation(t, decoded, tt.coefficient, tt.exponent, tt.negativeZero)
			})
		}
	})

	t.Run("DecFloat34", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var value DecFloat34
				if err := value.Scan(tt.input); err != nil {
					t.Fatal(err)
				}
				assertDecFloat34Representation(t, value, tt.coefficient, tt.exponent, tt.negativeZero)
				if got := value.String(); got != tt.canonical {
					t.Fatalf("String() = %q, want %q", got, tt.canonical)
				}

				driverValue, err := value.Value()
				if err != nil {
					t.Fatal(err)
				}
				if driverValue != tt.canonical {
					t.Fatalf("Value() = %q, want %q", driverValue, tt.canonical)
				}

				jsonValue, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				var decoded DecFloat34
				if err := json.Unmarshal(jsonValue, &decoded); err != nil {
					t.Fatal(err)
				}
				assertDecFloat34Representation(t, decoded, tt.coefficient, tt.exponent, tt.negativeZero)
			})
		}
	})
}

func TestFormatDecFloatUsesInt64AdjustedExponent(t *testing.T) {
	if got := formatDecFloat(big.NewInt(10), math.MaxInt32, false); got != "1.0E+2147483648" {
		t.Fatalf("formatDecFloat adjusted exponent = %q", got)
	}
}

func TestDecFloatPublicAPIsRejectUnrepresentableFiniteValues(t *testing.T) {
	coefficient16, _ := new(big.Int).SetString("12345678901234567", 10)
	coefficient34, _ := new(big.Int).SetString("12345678901234567890123456789012345", 10)

	t.Run("DecFloat16", func(t *testing.T) {
		for _, invalid := range []DecFloat16{
			{Int: coefficient16, Valid: true},
			{Int: big.NewInt(1), Exp: 370, Valid: true},
		} {
			var scanned DecFloat16
			if err := scanned.ScanDec16(invalid); err == nil {
				t.Fatalf("ScanDec16 accepted %+v", invalid)
			}
			if scanned.Valid {
				t.Fatalf("ScanDec16 left a valid value: %+v", scanned)
			}

			checks := []func() error{
				func() error { _, err := invalid.Dec16Value(); return err },
				func() error { _, err := invalid.Value(); return err },
				func() error { _, err := json.Marshal(invalid); return err },
				func() error { _, err := decFloat16ToFloat64(invalid); return err },
				func() error { _, err := encodeDecFloat16Binary(nil, invalid); return err },
			}
			for i, check := range checks {
				if err := check(); err == nil {
					t.Fatalf("DecFloat16 check %d accepted %+v", i, invalid)
				}
			}
		}
	})

	t.Run("DecFloat34", func(t *testing.T) {
		for _, invalid := range []DecFloat34{
			{Int: coefficient34, Valid: true},
			{Int: big.NewInt(1), Exp: 6112, Valid: true},
		} {
			var scanned DecFloat34
			if err := scanned.ScanDec34(invalid); err == nil {
				t.Fatalf("ScanDec34 accepted %+v", invalid)
			}
			if scanned.Valid {
				t.Fatalf("ScanDec34 left a valid value: %+v", scanned)
			}

			checks := []func() error{
				func() error { _, err := invalid.Dec34Value(); return err },
				func() error { _, err := invalid.Value(); return err },
				func() error { _, err := json.Marshal(invalid); return err },
				func() error { _, err := decFloat34ToFloat64(invalid); return err },
				func() error { _, err := encodeDecFloat34Binary(nil, invalid); return err },
			}
			for i, check := range checks {
				if err := check(); err == nil {
					t.Fatalf("DecFloat34 check %d accepted %+v", i, invalid)
				}
			}
		}
	})
}

func assertDecFloat16Representation(t *testing.T, value DecFloat16, coefficient string, exponent int32, negativeZero bool) {
	t.Helper()
	if !value.Valid || value.Special != SpecialNone || value.Int == nil || value.Int.String() != coefficient ||
		value.Exp != exponent || value.NegativeZero != negativeZero {
		t.Fatalf("DecFloat16 = %+v, want coefficient=%s exponent=%d negativeZero=%t", value, coefficient, exponent, negativeZero)
	}
}

func assertDecFloat34Representation(t *testing.T, value DecFloat34, coefficient string, exponent int32, negativeZero bool) {
	t.Helper()
	if !value.Valid || value.Special != SpecialNone || value.Int == nil || value.Int.String() != coefficient ||
		value.Exp != exponent || value.NegativeZero != negativeZero {
		t.Fatalf("DecFloat34 = %+v, want coefficient=%s exponent=%d negativeZero=%t", value, coefficient, exponent, negativeZero)
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

func TestDecFloatBinaryZeroExponentText(t *testing.T) {
	t.Run("DecFloat16", func(t *testing.T) {
		values := []DecFloat16{
			{Int: big.NewInt(0), Exp: -2, Valid: true},
			{Int: big.NewInt(0), Exp: 5, Valid: true},
			{Int: big.NewInt(0), Exp: -398, Valid: true, NegativeZero: true},
		}
		for _, value := range values {
			encoded, err := encodeDecFloat16Binary(nil, value)
			if err != nil {
				t.Fatal(err)
			}

			var decoded DecFloat16
			if err := decodeDecFloat16Binary(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			assertDecFloat16Representation(t, decoded, "0", value.Exp, value.NegativeZero)

			var text string
			if err := (scanPlanBinaryDecFloat16ToString{}).Scan(encoded, &text); err != nil {
				t.Fatal(err)
			}
			if text != value.String() {
				t.Fatalf("binary string = %q, want %q", text, value.String())
			}
		}
	})

	t.Run("DecFloat34", func(t *testing.T) {
		values := []DecFloat34{
			{Int: big.NewInt(0), Exp: -2, Valid: true},
			{Int: big.NewInt(0), Exp: 5, Valid: true},
			{Int: big.NewInt(0), Exp: -6176, Valid: true, NegativeZero: true},
		}
		for _, value := range values {
			encoded, err := encodeDecFloat34Binary(nil, value)
			if err != nil {
				t.Fatal(err)
			}

			var decoded DecFloat34
			if err := decodeDecFloat34Binary(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			assertDecFloat34Representation(t, decoded, "0", value.Exp, value.NegativeZero)

			var text string
			if err := (scanPlanBinaryDecFloat34ToString{}).Scan(encoded, &text); err != nil {
				t.Fatal(err)
			}
			if text != value.String() {
				t.Fatalf("binary string = %q, want %q", text, value.String())
			}
		}
	})
}

func TestFloat64ToDecFloat16RejectsUnrepresentableValue(t *testing.T) {
	value := math.Nextafter(1, 2)
	if _, err := Float64ToDecFloat16(value); err == nil || !strings.Contains(err.Error(), "coefficient out of range") {
		t.Fatalf("Float64ToDecFloat16(%g) error = %v", value, err)
	}

	if _, err := Float64ToDecFloat34(value); err != nil {
		t.Fatalf("Float64ToDecFloat34(%g) error = %v", value, err)
	}

	d16, err := Float64ToDecFloat16(1e20)
	if err != nil {
		t.Fatal(err)
	}
	assertDecFloat16Representation(t, d16, "1", 20, false)
	if got := d16.String(); got != "1E+20" {
		t.Fatalf("Float64ToDecFloat16(1e20).String() = %q", got)
	}

	d34, err := Float64ToDecFloat34(1e20)
	if err != nil {
		t.Fatal(err)
	}
	assertDecFloat34Representation(t, d34, "1", 20, false)
	if got := d34.String(); got != "1E+20" {
		t.Fatalf("Float64ToDecFloat34(1e20).String() = %q", got)
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
