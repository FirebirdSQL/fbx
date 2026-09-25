package fbtype

import (
	"database/sql/driver"
	"fmt"
	"reflect"
)

// NullCodec encodes Firebird SQL_TYPE_NULL parameters. SQL_TYPE_NULL carries
// only the NULL or NOT NULL state and has no value payload.
type NullCodec struct{}

// PlanEncode returns a plan that determines the SQL NULL state of value.
func (NullCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	return encodePlanNullCodec{}
}

type encodePlanNullCodec struct{}

func (encodePlanNullCodec) Encode(value any, buf []byte) (newBuf []byte, err error) {
	isNull, err := nullCodecIsNull(value, 0)
	if err != nil {
		return nil, err
	}
	if isNull {
		return nil, nil
	}

	return appendNonNullBytes(buf, nil), nil
}

const maxNullCodecDepth = 8

func nullCodecIsNull(value any, depth int) (bool, error) {
	if depth > maxNullCodecDepth {
		return false, fmt.Errorf("cannot determine SQL_TYPE_NULL state: maximum indirection depth %d exceeded", maxNullCodecDepth)
	}
	if value == nil {
		return true, nil
	}

	if isNil, callNilDriverValuer := isNilDriverValuer(value); isNil && !callNilDriverValuer {
		return true, nil
	}
	if valuer, ok := value.(driver.Valuer); ok {
		converted, err := valuer.Value()
		if err != nil {
			return false, err
		}
		return nullCodecIsNull(converted, depth+1)
	}

	if isNull, handled, err := nullCodecNativeValuerIsNull(value); handled {
		return isNull, err
	}

	refValue := reflect.ValueOf(value)
	switch refValue.Kind() {
	case reflect.Pointer, reflect.Interface:
		return nullCodecIsNull(refValue.Elem().Interface(), depth+1)
	default:
		return false, nil
	}
}

func nullCodecNativeValuerIsNull(value any) (isNull bool, handled bool, err error) {
	var valid bool
	switch value := value.(type) {
	case TextValuer:
		var v Text
		v, err = value.TextValue()
		valid = v.Valid
	case BoolValuer:
		var v Bool
		v, err = value.BoolValue()
		valid = v.Valid
	case NumericValuer:
		var v Numeric
		v, err = value.NumericValue()
		valid = v.Valid
	case Int128Valuer:
		var v Int128
		v, err = value.Int128Value()
		valid = v.Valid
	case Dec16Valuer:
		var v DecFloat16
		v, err = value.Dec16Value()
		valid = v.Valid
	case Dec34Valuer:
		var v DecFloat34
		v, err = value.Dec34Value()
		valid = v.Valid
	case DateValuer:
		var v Date
		v, err = value.DateValue()
		valid = v.Valid
	case TimeValuer:
		var v Time
		v, err = value.TimeValue()
		valid = v.Valid
	case TimetzValuer:
		var v Timetz
		v, err = value.TimetzValue()
		valid = v.Valid
	case TimestampValuer:
		var v Timestamp
		v, err = value.TimestampValue()
		valid = v.Valid
	case TimestamptzValuer:
		var v Timestamptz
		v, err = value.TimestamptzValue()
		valid = v.Valid
	case UUIDValuer:
		var v UUID
		v, err = value.UUIDValue()
		valid = v.Valid
	case BytesValuer:
		var v []byte
		v, err = value.BytesValue()
		valid = v != nil
	case Float64Valuer:
		var v Float8
		v, err = value.Float64Value()
		valid = v.Valid
	case Int64Valuer:
		var v Int8
		v, err = value.Int64Value()
		valid = v.Valid
	default:
		return false, false, nil
	}
	if err != nil {
		return false, true, err
	}
	return !valid, true, nil
}

// PlanScan returns no plan because Firebird uses SQL_TYPE_NULL for input
// parameters rather than result values.
func (NullCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	return nil
}

// DecodeDatabaseSQLValue decodes the NULL state as a database/sql value.
func (c NullCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}
	return "", nil
}

// DecodeValue decodes the NULL state as a Go value.
func (c NullCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}
	return []byte{}, nil
}
