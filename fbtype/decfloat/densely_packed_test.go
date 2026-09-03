package decfloat

import (
	"math/big"
	"testing"
)

func TestDenselyPackedDecimalCodec_DecDigitCount(t *testing.T) {
	c := NewDenselyPackedDecimalCodec(16)
	if c.digitGroups != 5 {
		t.Fatalf("digitGroups=%d want=5", c.digitGroups)
	}
	if c.numberOfDigits != 16 {
		t.Fatalf("numberOfDigits=%d want=16", c.numberOfDigits)
	}

	d := NewDenselyPackedDecimalCodec(34)
	if d.digitGroups != 11 {
		t.Fatalf("digitGroups=%d want=11", d.digitGroups)
	}
	if d.numberOfDigits != 34 {
		t.Fatalf("numberOfDigits=%d want=34", d.numberOfDigits)
	}
}

func TestDenselyPackedDecimalCodec_Dec128DigitCount(t *testing.T) {
	c := NewDenselyPackedDecimalCodec(34)
	if c.digitGroups != 11 {
		t.Fatalf("digitGroups=%d want=11", c.digitGroups)
	}
}

func TestDenselyPackedDecimalCodec_RoundTrip(t *testing.T) {
	c := NewDenselyPackedDecimalCodec(16)

	val1, ok := new(big.Int).SetString("12323123", 10)
	if !ok {
		t.Fatal("cant set string into val1")
	}

	val2, ok := new(big.Int).SetString("9999999999999999", 10)
	if !ok {
		t.Fatal("cant set string into val2")
	}

	tests := []*big.Int{
		big.NewInt(0),
		big.NewInt(7),
		big.NewInt(42),
		big.NewInt(999),
		big.NewInt(1000),
		big.NewInt(123456789),
		val1,
		val2,
	}

	for _, v := range tests {
		decBytes := make([]byte, 8)
		lsb := len(decBytes) - 1

		msd := c.EncodeValue(v, decBytes, lsb)
		got := c.DecodeValue(SignumPositive, msd, decBytes, lsb)

		if got.Cmp(new(big.Int).Abs(v)) != 0 {
			t.Fatalf("roundtrip mismatch: v=%s msd=%d got=%s bytes=%x", v.String(), msd, got.String(), decBytes)
		}
	}
}

func TestDenselyPackedDecimalCodec_RoundTrip_Positive(t *testing.T) {
	c := NewDenselyPackedDecimalCodec(16)

	val1, _ := new(big.Int).SetString("1234567890123456", 10)
	val2, _ := new(big.Int).SetString("9999999999999999", 10)
	val3, _ := new(big.Int).SetString("1000000000000000", 10)
	val4, _ := new(big.Int).SetString("100000000000001", 10)

	tests := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(7),
		big.NewInt(9),
		big.NewInt(10),
		big.NewInt(99),
		big.NewInt(100),
		big.NewInt(999),
		big.NewInt(1000),
		big.NewInt(1001),
		big.NewInt(123456789),
		val1,
		val2,
		val3,
		val4,
	}

	for _, v := range tests {
		t.Run(v.String(), func(t *testing.T) {
			decBytes := make([]byte, 8)
			lsb := 7

			msd := c.EncodeValue(v, decBytes, lsb)

			if msd < 0 || msd > 9 {
				t.Fatalf("MSD out of range: %d", msd)
			}

			got := c.DecodeValue(SignumPositive, msd, decBytes, lsb)

			if got.Cmp(v) != 0 {
				t.Fatalf("roundtrip mismatch:\noriginal: %s\nMSD:      %d\nbytes:    %x\ndecoded:  %s",
					v.String(), msd, decBytes, got.String())
			}
		})
	}
}

func TestDenselyPackedDecimalCodec_RoundTrip_Dec128(t *testing.T) {
	c := NewDenselyPackedDecimalCodec(34)

	val1, _ := new(big.Int).SetString("1234567890123456789012345678901234", 10)
	val2, _ := new(big.Int).SetString("9999999999999999999999999999999999", 10)

	tests := []struct {
		name string
		val  *big.Int
	}{
		{"zero", big.NewInt(0)},
		{"small", big.NewInt(42)},
		{"max_34_digits", val1},
		{"all_nines", val2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decBytes := make([]byte, 16)
			lsb := 15

			msd := c.EncodeValue(tt.val, decBytes, lsb)
			got := c.DecodeValue(SignumPositive, msd, decBytes, lsb)

			if got.Cmp(tt.val) != 0 {
				t.Fatalf("roundtrip mismatch: v=%s got=%s", tt.val.String(), got.String())
			}
		})
	}
}

func TestDenselyPackedDecimalCodec_RoundTrip_Negative(t *testing.T) {
	c := NewDenselyPackedDecimalCodec(16)

	negVal1, _ := new(big.Int).SetString("-1234567890123456", 10)
	negVal2, _ := new(big.Int).SetString("-9999999999999999", 10)

	tests := []*big.Int{
		big.NewInt(-1),
		big.NewInt(-7),
		big.NewInt(-42),
		big.NewInt(-999),
		big.NewInt(-1000),
		big.NewInt(-123456789),
		negVal1,
		negVal2,
	}

	for _, v := range tests {
		t.Run(v.String(), func(t *testing.T) {
			decBytes := make([]byte, 8)
			lsb := 7

			msd := c.EncodeValue(v, decBytes, lsb)
			got := c.DecodeValue(SignumNegative, msd, decBytes, lsb)

			if got.Cmp(v) != 0 {
				t.Fatalf("roundtrip mismatch: v=%s got=%s", v.String(), got.String())
			}
		})
	}
}

func TestDenselyPackedDecimalCodec_RejectsOversizedCoefficientBeforeIntConversion(t *testing.T) {
	c := NewDenselyPackedDecimalCodec(16)
	groups := new(big.Int).Exp(big.NewInt(1000), big.NewInt(5), nil)
	beyondInt64 := new(big.Int).Lsh(big.NewInt(1), 64)
	beyondInt64.Mul(beyondInt64, groups)

	tests := []*big.Int{
		new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil),
		beyondInt64,
	}
	for _, value := range tests {
		t.Run(value.String(), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("EncodeValue(%s) did not panic", value)
				}
			}()

			c.EncodeValue(value, make([]byte, 8), 7)
		})
	}
}

func TestNewDenselyPackedDecimalCodecRejectsUnsupportedPrecision(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewDenselyPackedDecimalCodec(37) did not panic")
		}
	}()

	NewDenselyPackedDecimalCodec(37)
}
