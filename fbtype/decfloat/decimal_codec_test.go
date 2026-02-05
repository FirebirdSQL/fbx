package decfloat

import (
	"math/big"
	"testing"
)

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
