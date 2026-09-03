package decfloat

import "fmt"

// DecimalType identifies a finite value or an IEEE-754 decimal special value.
type DecimalType uint8

const (
	DecimalFinite DecimalType = iota
	DecimalInfinity
	DecimalNaN
	DecimalSignalingNaN
)

// String returns the name of t.
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

func (t DecimalType) specialBits() (byte, error) {
	switch t {
	case DecimalInfinity:
		return inf0, nil
	case DecimalNaN:
		return nanQuiet, nil
	case DecimalSignalingNaN:
		return nanSignal, nil
	case DecimalFinite:
		return 0, fmt.Errorf("decfloat: finite has no special bits")
	default:
		return 0, fmt.Errorf("decfloat: unknown DecimalType %d", t)
	}
}
