package decfloat

import (
	"fmt"
	"math/big"
)

const (
	dpdDigitsPerGroup = 3
	dpdBitsPerGroup   = 10
	dpdBitsPerByte    = 8
)

var bigOneThousand = big.NewInt(1000)

// Signum identifies the sign of an IEEE-754 decimal value, including zero and
// special values.
type Signum int

const (
	SignumPositive Signum = 1
	SignumNegative Signum = -1
)

// DenselyPackedDecimalCodec converts decimal coefficients to and from DPD.
type DenselyPackedDecimalCodec struct {
	numberOfDigits int
	digitGroups    int
}

// NewDenselyPackedDecimalCodec constructs a DPD codec for numberOfDigits.
//
// numberOfDigits must have the form n*3+1 for n greater than zero. The most
// significant digit is handled separately from the n three-digit DPD groups.
// NewDenselyPackedDecimalCodec panics when numberOfDigits is invalid.
func NewDenselyPackedDecimalCodec(numberOfDigits int) *DenselyPackedDecimalCodec {
	if numberOfDigits/dpdDigitsPerGroup <= 0 || numberOfDigits%dpdDigitsPerGroup != 1 {
		panic(fmt.Sprintf("decfloat: number of digits must have form n*3+1 with n > 0, was %d", numberOfDigits))
	}

	return &DenselyPackedDecimalCodec{
		numberOfDigits: numberOfDigits,
		digitGroups:    numberOfDigits / dpdDigitsPerGroup,
	}
}

// DecodeValue decodes a DPD coefficient whose least-significant byte is at
// lsbIndex. firstDigit contains the separately encoded most-significant digit.
// DecodeValue panics if firstDigit or the encoded buffer bounds are invalid.
func (c *DenselyPackedDecimalCodec) DecodeValue(signum Signum, firstDigit int, decBytes []byte, lsbIndex int) *big.Int {
	if firstDigit < 0 || firstDigit > 9 {
		panic(fmt.Sprintf("decfloat: firstDigit must be 0..9, was %d", firstDigit))
	}
	c.validateLsbIndex(lsbIndex, len(decBytes))

	digitChars := make([]byte, c.numberOfDigits+1)
	for i := range digitChars {
		digitChars[i] = '0'
	}

	for digitGroup := 0; digitGroup < c.digitGroups; digitGroup++ {
		digitBitsFromEnd := digitGroup * dpdBitsPerGroup
		firstByteBitOffset := digitBitsFromEnd % dpdBitsPerByte
		firstByteIndex := lsbIndex - digitBitsFromEnd/dpdBitsPerByte

		b0 := uint16(uint8(decBytes[firstByteIndex]))
		b1 := uint16(uint8(decBytes[firstByteIndex-1]))

		var dpdGroupBits uint16
		dpdGroupBits = ((b0 >> uint(firstByteBitOffset)) |
			(b1 << uint(dpdBitsPerByte-firstByteBitOffset))) & 0x03FF

		if dpdGroupBits != 0 {
			dstPos := len(digitChars) - (digitGroup+1)*dpdDigitsPerGroup
			srcPos := int(dpdGroupBits) * dpdDigitsPerGroup
			digitChars[dstPos+0] = dpdGroupBits2Digits[srcPos+0]
			digitChars[dstPos+1] = dpdGroupBits2Digits[srcPos+1]
			digitChars[dstPos+2] = dpdGroupBits2Digits[srcPos+2]
		}
	}

	if firstDigit != 0 {
		digitChars[1] = byte('0' + firstDigit)
	}

	return bytesToBigInt(signum, digitChars)
}

// EncodeValue encodes value as DPD ending at lsbIndex and returns the
// separately encoded most-significant digit.
//
// The coefficient bits in decBytes must be zero before this method is called.
// EncodeValue panics if value is nil, the buffer bounds are invalid, or value
// has more than the configured number of digits.
func (c *DenselyPackedDecimalCodec) EncodeValue(value *big.Int, decBytes []byte, lsbIndex int) int {
	c.validateLsbIndex(lsbIndex, len(decBytes))

	remaining := new(big.Int).Abs(value)
	var q, r big.Int

	for digitGroup := 0; digitGroup < c.digitGroups; digitGroup++ {
		digitBitsFromEnd := digitGroup * dpdBitsPerGroup
		firstByteBitOffset := digitBitsFromEnd % dpdBitsPerByte
		firstByteIndex := lsbIndex - digitBitsFromEnd/dpdBitsPerByte

		q.QuoRem(remaining, bigOneThousand, &r)
		remaining.Set(&q)

		rem := int(r.Int64())
		currentGroup := uint16(bin2dpd[rem])

		decBytes[firstByteIndex] |= byte(currentGroup << uint(firstByteBitOffset))
		decBytes[firstByteIndex-1] |= byte(currentGroup >> uint(dpdBitsPerByte-firstByteBitOffset))
	}

	msd := int(remaining.Int64())
	if msd < 0 || msd > 9 {
		panic(fmt.Sprintf("decfloat: mostSignificantDigit out of range, was %d", msd))
	}
	return msd
}

func (c *DenselyPackedDecimalCodec) validateLsbIndex(lsbIndex int, decBytesLen int) {
	if lsbIndex < 0 || lsbIndex >= decBytesLen {
		panic(fmt.Sprintf("decfloat: lsbIndex must be within array length %d, was %d", decBytesLen, lsbIndex))
	}

	if (lsbIndex+1)*dpdBitsPerByte < dpdBitsPerGroup*c.digitGroups {
		needBytes := (dpdBitsPerGroup*c.digitGroups + 7) / dpdBitsPerByte
		panic(fmt.Sprintf("decfloat: need at least %d bytes (lsbIndex=%d), have %d",
			needBytes, lsbIndex, lsbIndex+1))
	}
}

func bytesToBigInt(signum Signum, digitChars []byte) *big.Int {
	i := -1
	for idx := 0; idx < len(digitChars); idx++ {
		if digitChars[idx] != '0' {
			i = idx
			break
		}
	}
	if i == -1 {
		return big.NewInt(0)
	}

	if signum == SignumNegative {
		i--
		if i < 0 {
			panic("decfloat: internal error: no space for '-'")
		}
		digitChars[i] = '-'
	}

	z := new(big.Int)
	if _, ok := z.SetString(string(digitChars[i:]), 10); !ok {
		panic("decfloat: failed to parse digits into big.Int")
	}
	return z
}
