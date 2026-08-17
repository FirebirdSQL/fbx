package decfloat

import "testing"

func TestDPDEncodeDecodeTablesRoundTripAllCanonicalValues(t *testing.T) {
	for value := 0; value <= 999; value++ {
		bits := bin2dpd[value]
		if bits > 0x3ff {
			t.Fatalf("bin2dpd[%d] contains out-of-range bits: 0x%x", value, bits)
		}

		i := int(bits) * dpdDigitsPerGroup
		got := int(dpdGroupBits2Digits[i]-'0')*100 +
			int(dpdGroupBits2Digits[i+1]-'0')*10 +
			int(dpdGroupBits2Digits[i+2]-'0')
		if got != value {
			t.Fatalf("round-trip mismatch for %d: bits=0x%x got=%d", value, bits, got)
		}
	}
}

func TestDPDDecodeTableAllBitPatterns(t *testing.T) {
	wantLength := (1 << dpdBitsPerGroup) * dpdDigitsPerGroup
	if len(dpdGroupBits2Digits) != wantLength {
		t.Fatalf("decode table length = %d, want %d", len(dpdGroupBits2Digits), wantLength)
	}

	for bits := 0; bits < 1<<dpdBitsPerGroup; bits++ {
		i := bits * dpdDigitsPerGroup
		for offset := 0; offset < dpdDigitsPerGroup; offset++ {
			digit := dpdGroupBits2Digits[i+offset]
			if digit < '0' || digit > '9' {
				t.Fatalf("decode table entry 0x%x contains non-digit %q", bits, digit)
			}
		}
	}
}
