package decfloat

import (
	"fmt"
	"math/big"
)

// DecimalFormat describes an IEEE-754 decimal interchange format.
type DecimalFormat struct {
	formatBitLength  int
	formatByteLength int

	coefficientDigits int

	exponentContinuationBits    int
	coefficientContinuationBits int

	eLimit       int
	exponentBias int

	maxCoefficient *big.Int
}

const (
	signBits                             = 1
	combinationBits                      = 5
	maxSupportedCoefficientDigits        = 34
	maxSupportedExponentContinuationBits = 29
)

var (
	// Decimal64 describes the 64-bit format used by Firebird DECFLOAT(16).
	Decimal64 = NewDecimalFormat(64, 16)
	// Decimal128 describes the 128-bit format used by Firebird DECFLOAT(34).
	Decimal128 = NewDecimalFormat(128, 34)
)

// NewDecimalFormat constructs an IEEE-754 decimal format description.
//
// coefficientDigits must have the form n*3+1 for n greater than zero and must
// not exceed 34, the precision of IEEE-754 decimal128. The resulting format
// must leave between 2 and 29 exponent-continuation bits so all exponent
// calculations fit int32 and are portable to 32-bit systems.
// NewDecimalFormat panics when these requirements are not met.
func NewDecimalFormat(formatBitLength int, coefficientDigits int) DecimalFormat {
	if formatBitLength <= 0 || formatBitLength%dpdBitsPerByte != 0 {
		panic(fmt.Sprintf("decfloat: format bit length must be a positive multiple of 8, was %d", formatBitLength))
	}
	if coefficientDigits/dpdDigitsPerGroup <= 0 || coefficientDigits%dpdDigitsPerGroup != 1 {
		panic(fmt.Sprintf("decfloat: coefficient digits must have form n*3+1 with n > 0, was %d", coefficientDigits))
	}
	if coefficientDigits > maxSupportedCoefficientDigits {
		panic(fmt.Sprintf("decfloat: coefficient digits must not exceed %d, was %d", maxSupportedCoefficientDigits, coefficientDigits))
	}

	f := DecimalFormat{
		formatBitLength:   formatBitLength,
		formatByteLength:  formatBitLength / 8,
		coefficientDigits: coefficientDigits,
	}
	f.coefficientContinuationBits = calculateCoefficientContinuationBits(coefficientDigits)
	f.exponentContinuationBits = calculateExponentContinuationBits(formatBitLength, f.coefficientContinuationBits)
	if f.exponentContinuationBits < 2 {
		panic(fmt.Sprintf("decfloat: format leaves %d exponent-continuation bits, need at least 2", f.exponentContinuationBits))
	}
	if f.exponentContinuationBits > maxSupportedExponentContinuationBits {
		panic(fmt.Sprintf("decfloat: format leaves %d exponent-continuation bits, maximum supported is %d",
			f.exponentContinuationBits, maxSupportedExponentContinuationBits))
	}
	f.eLimit = calculateExponentLimit(f.exponentContinuationBits)

	eMin := -f.eLimit / 2
	f.exponentBias = -eMin + coefficientDigits - 1

	digits := make([]byte, coefficientDigits)
	for i := range digits {
		digits[i] = '9'
	}

	f.maxCoefficient, _ = new(big.Int).SetString(string(digits), 10)

	return f
}

// ValidateByteLength checks that decBytes has the format's encoded length.
func (f DecimalFormat) ValidateByteLength(decBytes []byte) error {
	if err := f.validate(); err != nil {
		return err
	}
	if len(decBytes) != f.formatByteLength {
		return fmt.Errorf("decfloat: decBytes must be %d bytes, got %d", f.formatByteLength, len(decBytes))
	}
	return nil
}

// BiasedExponent converts an unbiased exponent to its encoded value.
func (f DecimalFormat) BiasedExponent(unbiased int) int {
	return unbiased + f.exponentBias
}

// UnbiasedExponent converts an encoded exponent to its unbiased value.
func (f DecimalFormat) UnbiasedExponent(biased int) int {
	return biased - f.exponentBias
}

// ValidateFiniteCoefficientAndExponent checks that a finite value fits f.
func (f DecimalFormat) ValidateFiniteCoefficientAndExponent(coefficient *big.Int, exponent int) error {
	if err := f.validate(); err != nil {
		return err
	}
	if coefficient == nil {
		return fmt.Errorf("decfloat: coefficient must not be nil")
	}

	abs := new(big.Int).Abs(coefficient)
	if abs.Cmp(f.maxCoefficient) > 0 {
		return fmt.Errorf("decfloat: coefficient out of range for this format (maximum %d decimal digits)", f.coefficientDigits)
	}

	minExponent := -f.exponentBias
	maxExponent := f.eLimit - f.exponentBias
	if exponent < minExponent || exponent > maxExponent {
		return fmt.Errorf("decfloat: exponent out of range: exp=%d allowed[%d..%d]", exponent, minExponent, maxExponent)
	}
	return nil
}

func (f DecimalFormat) validate() error {
	if f.formatBitLength <= 0 || f.formatBitLength%dpdBitsPerByte != 0 ||
		f.formatByteLength != f.formatBitLength/dpdBitsPerByte ||
		f.coefficientDigits/dpdDigitsPerGroup <= 0 || f.coefficientDigits%dpdDigitsPerGroup != 1 ||
		f.coefficientDigits > maxSupportedCoefficientDigits ||
		f.exponentContinuationBits < 2 || f.exponentContinuationBits > maxSupportedExponentContinuationBits ||
		f.coefficientContinuationBits != calculateCoefficientContinuationBits(f.coefficientDigits) ||
		f.exponentContinuationBits != calculateExponentContinuationBits(f.formatBitLength, f.coefficientContinuationBits) ||
		f.eLimit != calculateExponentLimit(f.exponentContinuationBits) ||
		f.exponentBias != f.eLimit/2+f.coefficientDigits-1 ||
		f.maxCoefficient == nil {
		return fmt.Errorf("decfloat: invalid DecimalFormat")
	}
	return nil
}

func calculateCoefficientContinuationBits(coefficientDigits int) int {
	return dpdBitsPerGroup * (coefficientDigits - 1) / dpdDigitsPerGroup
}

func calculateExponentContinuationBits(formatBitLength int, coefficientContinuationBits int) int {
	return formatBitLength - signBits - combinationBits - coefficientContinuationBits
}

func calculateExponentLimit(exponentContinuationBits int) int {
	return 3*(1<<exponentContinuationBits) - 1
}
