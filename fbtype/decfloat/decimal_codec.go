package decfloat

import (
	"fmt"
	"math"
	"math/big"
)

// DecodedDecimal is the decomposed form of an IEEE-754 decimal value.
//
// For finite values, Coefficient contains the signed coefficient and Exponent
// contains the unbiased exponent. The sign of a nonzero Coefficient must match
// Signum; either sign is valid for zero. Coefficient is nil for special values.
type DecodedDecimal struct {
	Signum      Signum
	Type        DecimalType
	Exponent    int32
	Coefficient *big.Int
}

// DecimalCodec encodes and decodes one IEEE-754 decimal format.
type DecimalCodec struct {
	Format           DecimalFormat
	CoefficientCoder *DenselyPackedDecimalCodec
}

// NewDecimalCodec constructs a codec for format. It panics if format is invalid.
func NewDecimalCodec(format DecimalFormat) *DecimalCodec {
	if err := format.validate(); err != nil {
		panic(err)
	}

	return &DecimalCodec{
		Format:           format,
		CoefficientCoder: NewDenselyPackedDecimalCodec(format.coefficientDigits),
	}
}

const (
	combination2 = 0b0_11000_00
	negativeBit  = 0b1000_0000
)

// ParseBytes decodes an IEEE-754 decimal interchange format (decimal64/decimal128).
func (c *DecimalCodec) ParseBytes(decBytes []byte) (DecodedDecimal, error) {
	if err := c.validate(); err != nil {
		return DecodedDecimal{}, err
	}
	if err := c.Format.ValidateByteLength(decBytes); err != nil {
		return DecodedDecimal{}, err
	}

	firstByte := int(decBytes[0] & 0xFF)

	signum := SignumPositive
	if (firstByte & negativeBit) != 0 {
		signum = SignumNegative
	}

	typ := decimalTypeFromFirstByte(firstByte)
	if typ != DecimalFinite {
		return DecodedDecimal{Signum: signum, Type: typ}, nil
	}

	var exponentMSB int
	var firstDigit int

	if (firstByte & combination2) != combination2 {
		exponentMSB = ((firstByte >> 3) & 0b01100) | (firstByte & 0b011)
		firstDigit = (firstByte >> 2) & 0b0111
	} else {
		exponentMSB = ((firstByte >> 1) & 0b01100) | (firstByte & 0b011)
		firstDigit = 0b01000 | ((firstByte >> 2) & 0b01)
	}

	expBitsRemaining := c.Format.exponentContinuationBits - 2
	expAndBias := decodeExponent(decBytes, exponentMSB, expBitsRemaining)

	unbiasedExponent := c.Format.UnbiasedExponent(expAndBias)
	if unbiasedExponent < math.MinInt32 || unbiasedExponent > math.MaxInt32 {
		return DecodedDecimal{}, fmt.Errorf("decfloat: decoded exponent %d is out of int32 range", unbiasedExponent)
	}
	exponent := int32(unbiasedExponent)

	lsbIndex := len(decBytes) - 1
	coeff := c.CoefficientCoder.DecodeValue(signum, firstDigit, decBytes, lsbIndex)

	return DecodedDecimal{
		Signum:      signum,
		Type:        DecimalFinite,
		Exponent:    exponent,
		Coefficient: coeff,
	}, nil
}

// EncodeFinite encodes a finite coefficient and exponent into decBytes.
//
// The sign is taken from the existing sign bit in decBytes. All other bits are
// cleared before encoding, and the absolute value of coefficient is encoded.
func (c *DecimalCodec) EncodeFinite(decBytes []byte, coefficient *big.Int, exponent int32) error {
	if err := c.validate(); err != nil {
		return err
	}
	if err := c.Format.ValidateByteLength(decBytes); err != nil {
		return err
	}
	if err := c.Format.ValidateFiniteCoefficientAndExponent(coefficient, int(exponent)); err != nil {
		return err
	}

	signBit := decBytes[0] & negativeBit
	clear(decBytes)
	decBytes[0] = signBit

	biased := c.Format.BiasedExponent(int(exponent))
	lsbIndex := len(decBytes) - 1
	msd := c.CoefficientCoder.EncodeValue(coefficient, decBytes, lsbIndex)

	expMSB := biased >> c.Format.exponentContinuationBits
	expTwoBitCont := (biased >> (c.Format.exponentContinuationBits - 2)) & 0b11

	if msd <= 7 {
		decBytes[0] |= byte((expMSB << 5) | (msd << 2) | expTwoBitCont)
	} else {
		decBytes[0] |= byte(combination2 | (expMSB << 3) | ((msd & 0b01) << 2) | expTwoBitCont)
	}

	encodeExponentContinuation(decBytes, biased, c.Format.exponentContinuationBits-2)
	return nil
}

// Encode encodes DecodedDecimal into bytes (decimal64/decimal128).
func (c *DecimalCodec) Encode(d DecodedDecimal) ([]byte, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	decBytes := make([]byte, c.Format.formatByteLength)

	switch d.Signum {
	case SignumPositive:
	case SignumNegative:
		decBytes[0] = negativeBit
	default:
		return nil, fmt.Errorf("decfloat: unknown signum %d", d.Signum)
	}

	if d.Type != DecimalFinite {
		specialBits, err := d.Type.specialBits()
		if err != nil {
			return nil, err
		}
		decBytes[0] |= specialBits
		return decBytes, nil
	}

	if d.Coefficient == nil {
		d.Coefficient = big.NewInt(0)
	}
	if coefficientSign := d.Coefficient.Sign(); coefficientSign != 0 && coefficientSign != int(d.Signum) {
		return nil, fmt.Errorf("decfloat: signum %d does not match coefficient %s", d.Signum, d.Coefficient)
	}

	if err := c.EncodeFinite(decBytes, d.Coefficient, d.Exponent); err != nil {
		return nil, err
	}
	return decBytes, nil
}

func (c *DecimalCodec) validate() error {
	if c == nil {
		return fmt.Errorf("decfloat: nil DecimalCodec")
	}
	if err := c.Format.validate(); err != nil {
		return err
	}
	if c.CoefficientCoder == nil || c.CoefficientCoder.numberOfDigits != c.Format.coefficientDigits {
		return fmt.Errorf("decfloat: coefficient coder does not match decimal format")
	}
	return nil
}

func encodeExponentContinuation(decBytes []byte, expAndBias int, expBitsRemaining int) {
	expByteIndex := 1
	for expBitsRemaining > 8 {
		decBytes[expByteIndex] = byte(expAndBias >> (expBitsRemaining - 8))
		expBitsRemaining -= 8
		expByteIndex++
	}
	if expBitsRemaining > 0 {
		decBytes[expByteIndex] |= byte(expAndBias << (8 - expBitsRemaining))
	}
}

func decodeExponent(decBytes []byte, exponentMSB int, exponentBitsRemaining int) int {
	exponent := exponentMSB
	byteIndex := 1
	for exponentBitsRemaining > 8 {
		exponent = (exponent << 8) | int(decBytes[byteIndex]&0xFF)
		exponentBitsRemaining -= 8
		byteIndex++
	}
	if exponentBitsRemaining > 0 {
		exponent = (exponent << exponentBitsRemaining) |
			(int(decBytes[byteIndex]&0xFF) >> (8 - exponentBitsRemaining))
	}

	return exponent
}

// String returns a diagnostic representation of d.
func (d DecodedDecimal) String() string {
	return fmt.Sprintf("type=%s sign=%d coeff=%v exp=%d", d.Type.String(), d.Signum, d.Coefficient, d.Exponent)
}
