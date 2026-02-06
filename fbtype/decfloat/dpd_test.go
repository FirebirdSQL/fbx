package decfloat

import "testing"

func TestEncodeDecode3Digits_RoundTrip_All(t *testing.T) {
	var out [3]byte

	for d := 0; d <= 999; d++ {
		bits := encode3Digits(d)

		if bits > 0x3FF {
			t.Fatalf("encode3Digits(%d) returned out-of-range bits: 0x%x", d, bits)
		}

		decode3Digits(bits, out[:])

		got := int(out[0]-'0')*100 + int(out[1]-'0')*10 + int(out[2]-'0')
		if got != d {
			t.Fatalf("round-trip mismatch for %d: bits=0x%x decoded=%q got=%d",
				d, bits, string(out[:]), got)
		}
	}
}

func TestEncode3Digits_OutOfRangePanics(t *testing.T) {
	cases := []int{-1, -10, 1000, 9999}

	for _, d := range cases {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for encode3Digits(%d), got none", d)
				}
			}()
			_ = encode3Digits(d)
		}()
	}
}

func TestDecode3Digits_OutTooShortPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for decode3Digits with short out, got none")
		}
	}()

	decode3Digits(0, make([]byte, 2))
}

func TestDecode3Digits_KnownValues(t *testing.T) {
	type tc struct {
		d int
	}
	tests := []tc{
		{0},
		{1},
		{9},
		{10},
		{42},
		{123},
		{999},
	}

	var out [3]byte
	for _, tt := range tests {
		bits := encode3Digits(tt.d)
		decode3Digits(bits, out[:])

		want := tt.d
		got := int(out[0]-'0')*100 + int(out[1]-'0')*10 + int(out[2]-'0')
		if got != want {
			t.Fatalf("mismatch for %d: bits=0x%x decoded=%q got=%d",
				tt.d, bits, string(out[:]), got)
		}
	}
}
