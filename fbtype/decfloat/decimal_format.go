package decfloat

import (
	"fmt"
	"math/big"
)

type DecimalFormat struct {
	formatBitLength  int
	formatByteLength int

	coefficientDigits int

	exponentContinuationBits    int
	coefficientContinuationBits int

	eLimit       int
	exponentBias int

	maxCoefficient *big.Int
	minCoefficient *big.Int
}

const (
	signBits        = 1
	combinationBits = 5
)

var (
	Decimal64  = NewDecimalFormat(64, 16)
	Decimal128 = NewDecimalFormat(128, 34)
)

func NewDecimalFormat(formatBitLength int, coefficientDigits int) DecimalFormat {
	f := DecimalFormat{
		formatBitLength:   formatBitLength,
		formatByteLength:  formatBitLength / 8,
		coefficientDigits: coefficientDigits,
	}
	f.coefficientContinuationBits = calculateCoefficientContinuationBits(coefficientDigits)
	f.exponentContinuationBits = calculateExponentContinuationBits(formatBitLength, f.coefficientContinuationBits)
	f.eLimit = calculateExponentLimit(f.exponentContinuationBits)

	eMin := -f.eLimit / 2
	f.exponentBias = -eMin + coefficientDigits - 1

	digits := make([]byte, coefficientDigits)
	for i := range digits {
		digits[i] = '9'
	}

	f.maxCoefficient, _ = new(big.Int).SetString(string(digits), 10)
	f.minCoefficient = new(big.Int).Neg(new(big.Int).Set(f.maxCoefficient))

	return f
}

func (f DecimalFormat) ValidateByteLength(decBytes []byte) error {
	if len(decBytes) != f.formatByteLength {
		return fmt.Errorf("decfloat: decBytes must be %d bytes, got %d", f.formatByteLength, len(decBytes))
	}
	return nil
}

func (f DecimalFormat) BiasedExponent(unbiased int) int {
	return unbiased + f.exponentBias
}

func (f DecimalFormat) UnbiasedExponent(biased int) int {
	return biased - f.exponentBias
}

func (f DecimalFormat) ValidateFiniteCoefficientAndExponent(coefficient *big.Int, exponent int) error {
	abs := new(big.Int).Abs(coefficient)
	if abs.Cmp(f.maxCoefficient) > 0 {
		return fmt.Errorf("decfloat: coefficient out of range for this format (digits=%d): %s", f.coefficientDigits, coefficient.String())
	}

	biased := f.BiasedExponent(exponent)
	if biased < 0 || biased > f.eLimit {
		return fmt.Errorf("decfloat: exponent out of range: exp=%d biased=%d allowed[0..%d]", exponent, biased, f.eLimit)
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
