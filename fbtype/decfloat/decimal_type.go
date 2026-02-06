package decfloat

import "fmt"

type DecimalType uint8

const (
	DecimalFinite DecimalType = iota
	DecimalInfinity
	DecimalNaN
	DecimalSignalingNaN
)

func (t DecimalType) String() string {
	switch t {
	case DecimalFinite:
		return "finite"
	case DecimalInfinity:
		return "infinity"
	case DecimalNaN:
		return "nan"
	case DecimalSignalingNaN:
		return "snan"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

const (
	decimalTypeMask = 0b0_11111_10
	inf0            = 0b0_11110_00
	inf2            = 0b0_11110_10
	nanQuiet        = 0b0_11111_00
	nanSignal       = 0b0_11111_10
)

func decimalTypeFromFirstByte(firstByte int) DecimalType {
	typ := firstByte & decimalTypeMask
	switch typ {
	case inf0, inf2:
		return DecimalInfinity
	case nanQuiet:
		return DecimalNaN
	case nanSignal:
		return DecimalSignalingNaN
	default:
		return DecimalFinite
	}
}

func (t DecimalType) specialBits() byte {
	switch t {
	case DecimalInfinity:
		return inf0
	case DecimalNaN:
		return nanQuiet
	case DecimalSignalingNaN:
		return nanSignal
	case DecimalFinite:
		panic("decfloat: finite has no special bits")
	default:
		panic("decfloat: unknown DecimalType")
	}
}
