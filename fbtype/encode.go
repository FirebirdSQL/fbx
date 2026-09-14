package fbtype

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// PlanEncode returns an EncodePlan for encoding value with the given Firebird
// type metadata. If no plan can be found then nil is returned.
func (m *Map) PlanEncode(typeMetadata TypeMetadata, value any) EncodePlan {
	return m.planEncodeDepth(typeMetadata, value, 0)
}

func (m *Map) planEncodeDepth(typeMetadata TypeMetadata, value any, depth int) EncodePlan {
	// Guard against infinite recursion.
	if depth > 8 {
		return nil
	}

	typeidMemo := m.memoizedEncodePlans[typeMetadata]
	if typeidMemo == nil {
		typeidMemo = make(map[reflect.Type]EncodePlan)
		m.memoizedEncodePlans[typeMetadata] = typeidMemo
	}

	targetReflectType := reflect.TypeOf(value)
	plan := typeidMemo[targetReflectType]
	if plan == nil {
		plan = m.planEncode(typeMetadata, value, depth)
		typeidMemo[targetReflectType] = plan
	}

	return plan
}

func (m *Map) planEncode(typeMetadata TypeMetadata, value any, depth int) EncodePlan {

	var dt *Type
	if dataType, ok := m.TypeForTypeID(typeMetadata.TypeID, typeMetadata.SubType); ok {
		dt = dataType
	} else {
		// If the metadata TypeID is unknown or unregistered, try to find a
		// suitable Firebird type from the Go value.
		if dataType, ok := m.TypeForValue(value); ok {
			dt = dataType
			typeMetadata.TypeID = dt.TypeID // Preserve assumed TypeID in case we are recursively called below.
		}
	}

	if dt != nil {
		if plan := dt.Codec.PlanEncode(m, typeMetadata, value); plan != nil {
			return plan
		}
	}

	for _, f := range m.TryWrapEncodePlanFuncs {
		if wrapperPlan, nextValue, ok := f(value); ok {
			if nextPlan := m.planEncodeDepth(typeMetadata, nextValue, depth+1); nextPlan != nil {
				wrapperPlan.SetNext(nextPlan)
				return wrapperPlan
			}
		}
	}

	if _, ok := value.(driver.Valuer); ok {
		return &encodePlanDriverValuer{m: m, typeMetadata: typeMetadata}
	}

	return nil
}

// encodePlanDriverValuer encodes values through database/sql/driver.Valuer
// after direct codec and wrapper planning fails.
type encodePlanDriverValuer struct {
	m            *Map
	typeMetadata TypeMetadata
}

func (plan *encodePlanDriverValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	if isNil, callNilDriverValuer := isNilDriverValuer(value); isNil && !callNilDriverValuer {
		return nil, nil
	}

	dv := value.(driver.Valuer)
	if dv == nil {
		return nil, nil
	}
	v, err := dv.Value()
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	newBuf, err = plan.m.Encode(plan.typeMetadata, v, buf)
	if err == nil {
		return newBuf, nil
	}

	s, ok := v.(string)
	if !ok {
		return nil, err
	}

	var scannedValue any
	scanErr := plan.m.Scan(plan.typeMetadata, []byte(s), &scannedValue)
	if scanErr != nil {
		return nil, err
	}

	// Prevent infinite loop. We can't encode this.
	if reflect.TypeOf(value) == reflect.TypeOf(scannedValue) {
		return nil, fmt.Errorf("tried to encode %v via encoding to text and scanning but failed due to receiving same type back", value)
	}

	var err2 error
	newBuf, err2 = plan.m.Encode(plan.typeMetadata, scannedValue, buf)
	if err2 != nil {
		return nil, err
	}

	return newBuf, nil
}

// TryWrapEncodePlanFunc tries to create a wrapper plan for value. If
// successful, nextValue is used to find the next EncodePlan and SetNext must be
// called on the wrapper before use.
type TryWrapEncodePlanFunc func(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool)

// derefPointerEncodePlan adapts *T values to the T value expected by the next
// plan. A nil pointer encodes as SQL NULL.
type derefPointerEncodePlan struct {
	next EncodePlan
}

func (plan *derefPointerEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *derefPointerEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	ptr := reflect.ValueOf(value)

	if ptr.IsNil() {
		return nil, nil
	}

	return plan.next.Encode(ptr.Elem().Interface(), buf)
}

// TryWrapDerefPointerEncodePlan wraps a pointer value by dereferencing it
// before calling the next plan.
func TryWrapDerefPointerEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	if valueType := reflect.TypeOf(value); valueType != nil && valueType.Kind() == reflect.Pointer {
		return &derefPointerEncodePlan{}, reflect.New(valueType.Elem()).Elem().Interface(), true
	}

	return nil, nil, false
}

// kindToTypes maps named scalar kinds to their builtin Go types for underlying
// type planning.
var kindToTypes map[reflect.Kind]reflect.Type = map[reflect.Kind]reflect.Type{
	reflect.Int:     reflect.TypeOf(int(0)),
	reflect.Int8:    reflect.TypeOf(int8(0)),
	reflect.Int16:   reflect.TypeOf(int16(0)),
	reflect.Int32:   reflect.TypeOf(int32(0)),
	reflect.Int64:   reflect.TypeOf(int64(0)),
	reflect.Uint:    reflect.TypeOf(uint(0)),
	reflect.Uint8:   reflect.TypeOf(uint8(0)),
	reflect.Uint16:  reflect.TypeOf(uint16(0)),
	reflect.Uint32:  reflect.TypeOf(uint32(0)),
	reflect.Uint64:  reflect.TypeOf(uint64(0)),
	reflect.Float32: reflect.TypeOf(float32(0)),
	reflect.Float64: reflect.TypeOf(float64(0)),
	reflect.String:  reflect.TypeOf(""),
	reflect.Bool:    reflect.TypeOf(false),
}

var byteSliceType = reflect.TypeOf([]byte{})

// underlyingTypeEncodePlan converts a named scalar, named []byte, or named array
// to its underlying Go type before calling the next plan.
type underlyingTypeEncodePlan struct {
	nextValueType reflect.Type
	next          EncodePlan
}

func (plan *underlyingTypeEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *underlyingTypeEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(reflect.ValueOf(value).Convert(plan.nextValueType).Interface(), buf)
}

// TryWrapFindUnderlyingTypeEncodePlan wraps named scalar, byte slice, and array
// values by converting them to their underlying Go type.
func TryWrapFindUnderlyingTypeEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if value == nil {
		return nil, nil, false
	}

	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	if _, ok := value.(SkipUnderlyingTypePlanner); ok {
		return nil, nil, false
	}

	refValue := reflect.ValueOf(value)

	nextValueType := kindToTypes[refValue.Kind()]
	if nextValueType != nil && refValue.Type() != nextValueType {
		return &underlyingTypeEncodePlan{nextValueType: nextValueType}, refValue.Convert(nextValueType).Interface(), true
	}

	// []byte is a special case. It is a slice but we treat it as a scalar type. In the case of a named type like
	// json.RawMessage which is defined as []byte the underlying type should be considered as []byte. But any other slice
	// does not have a special underlying type.
	if refValue.Type() != byteSliceType && refValue.Type().AssignableTo(byteSliceType) {
		return &underlyingTypeEncodePlan{nextValueType: byteSliceType}, refValue.Convert(byteSliceType).Interface(), true
	}

	// Get underlying type of any array.
	if refValue.Kind() == reflect.Array {
		underlyingArrayType := reflect.ArrayOf(refValue.Len(), refValue.Type().Elem())
		if refValue.Type() != underlyingArrayType {
			return &underlyingTypeEncodePlan{nextValueType: underlyingArrayType}, refValue.Convert(underlyingArrayType).Interface(), true
		}
	}

	return nil, nil, false
}

// TryWrapStringerEncodePlan wraps fmt.Stringer values as TextValuer. It runs
// after underlying-type planning so named scalars prefer their underlying codec.
func TryWrapStringerEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	if s, ok := value.(fmt.Stringer); ok {
		return &wrapFmtStringerEncodePlan{}, fmtStringerWrapper{s}, true
	}

	return nil, nil, false
}

// WrappedEncodePlanNextSetter is an EncodePlan wrapper that receives the next
// plan after recursive planning succeeds.
type WrappedEncodePlanNextSetter interface {
	SetNext(EncodePlan)
	EncodePlan
}

// TryWrapBuiltinTypeEncodePlan wraps builtin Go values with adapter types that
// implement codec valuer interfaces.
func TryWrapBuiltinTypeEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	switch value := value.(type) {
	case int8:
		return &wrapInt8EncodePlan{}, int8Wrapper(value), true
	case int16:
		return &wrapInt16EncodePlan{}, int16Wrapper(value), true
	case int32:
		return &wrapInt32EncodePlan{}, int32Wrapper(value), true
	case int64:
		return &wrapInt64EncodePlan{}, int64Wrapper(value), true
	case int:
		return &wrapIntEncodePlan{}, intWrapper(value), true
	case uint8:
		return &wrapUint8EncodePlan{}, uint8Wrapper(value), true
	case uint16:
		return &wrapUint16EncodePlan{}, uint16Wrapper(value), true
	case uint32:
		return &wrapUint32EncodePlan{}, uint32Wrapper(value), true
	case uint64:
		return &wrapUint64EncodePlan{}, uint64Wrapper(value), true
	case uint:
		return &wrapUintEncodePlan{}, uintWrapper(value), true
	case float32:
		return &wrapFloat32EncodePlan{}, float32Wrapper(value), true
	case float64:
		return &wrapFloat64EncodePlan{}, float64Wrapper(value), true
	case string:
		return &wrapStringEncodePlan{}, stringWrapper(value), true
	case time.Time:
		return &wrapTimeEncodePlan{}, timeWrapper(value), true
	case []byte:
		return &wrapByteSliceEncodePlan{}, byteSliceWrapper(value), true
	}

	return nil, nil, false
}

// The wrap*EncodePlan types below adapt the original builtin value to the
// matching wrapper type from builtin_wrappers.go before calling the next plan.
type wrapInt8EncodePlan struct {
	next EncodePlan
}

func (plan *wrapInt8EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapInt8EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(int8Wrapper(value.(int8)), buf)
}

type wrapInt16EncodePlan struct {
	next EncodePlan
}

func (plan *wrapInt16EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapInt16EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(int16Wrapper(value.(int16)), buf)
}

type wrapInt32EncodePlan struct {
	next EncodePlan
}

func (plan *wrapInt32EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapInt32EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(int32Wrapper(value.(int32)), buf)
}

type wrapInt64EncodePlan struct {
	next EncodePlan
}

func (plan *wrapInt64EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapInt64EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(int64Wrapper(value.(int64)), buf)
}

type wrapIntEncodePlan struct {
	next EncodePlan
}

func (plan *wrapIntEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapIntEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(intWrapper(value.(int)), buf)
}

type wrapUint8EncodePlan struct {
	next EncodePlan
}

func (plan *wrapUint8EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapUint8EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(uint8Wrapper(value.(uint8)), buf)
}

type wrapUint16EncodePlan struct {
	next EncodePlan
}

func (plan *wrapUint16EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapUint16EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(uint16Wrapper(value.(uint16)), buf)
}

type wrapUint32EncodePlan struct {
	next EncodePlan
}

func (plan *wrapUint32EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapUint32EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(uint32Wrapper(value.(uint32)), buf)
}

type wrapUint64EncodePlan struct {
	next EncodePlan
}

func (plan *wrapUint64EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapUint64EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(uint64Wrapper(value.(uint64)), buf)
}

type wrapUintEncodePlan struct {
	next EncodePlan
}

func (plan *wrapUintEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapUintEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(uintWrapper(value.(uint)), buf)
}

type wrapFloat32EncodePlan struct {
	next EncodePlan
}

func (plan *wrapFloat32EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapFloat32EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(float32Wrapper(value.(float32)), buf)
}

type wrapFloat64EncodePlan struct {
	next EncodePlan
}

func (plan *wrapFloat64EncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapFloat64EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(float64Wrapper(value.(float64)), buf)
}

type wrapStringEncodePlan struct {
	next EncodePlan
}

func (plan *wrapStringEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapStringEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(stringWrapper(value.(string)), buf)
}

type wrapTimeEncodePlan struct {
	next EncodePlan
}

func (plan *wrapTimeEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapTimeEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(timeWrapper(value.(time.Time)), buf)
}

type wrapByteSliceEncodePlan struct {
	next EncodePlan
}

func (plan *wrapByteSliceEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapByteSliceEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(byteSliceWrapper(value.([]byte)), buf)
}

type wrapFmtStringerEncodePlan struct {
	next EncodePlan
}

func (plan *wrapFmtStringerEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapFmtStringerEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(fmtStringerWrapper{value.(fmt.Stringer)}, buf)
}

// TryWrapSliceEncodePlan wraps a single-dimensional slice as an ArrayGetter.
// Common slice element types use FlatArray[T]; other one-dimensional slices use
// reflection.
func TryWrapSliceEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	// Avoid using reflect path for common types.
	switch value := value.(type) {
	case []int16:
		return &wrapSliceEncodePlan[int16]{}, (FlatArray[int16])(value), true
	case []int32:
		return &wrapSliceEncodePlan[int32]{}, (FlatArray[int32])(value), true
	case []int64:
		return &wrapSliceEncodePlan[int64]{}, (FlatArray[int64])(value), true
	case []float32:
		return &wrapSliceEncodePlan[float32]{}, (FlatArray[float32])(value), true
	case []float64:
		return &wrapSliceEncodePlan[float64]{}, (FlatArray[float64])(value), true
	case []string:
		return &wrapSliceEncodePlan[string]{}, (FlatArray[string])(value), true
	case []time.Time:
		return &wrapSliceEncodePlan[time.Time]{}, (FlatArray[time.Time])(value), true
	}

	if valueType := reflect.TypeOf(value); valueType != nil && valueType.Kind() == reflect.Slice {
		if valueType.Elem().Kind() == reflect.Slice {
			return nil, nil, false
		}
		w := anySliceArrayReflect{
			slice: reflect.ValueOf(value),
		}
		return &wrapSliceEncodeReflectPlan{}, w, true
	}

	return nil, nil, false
}

// wrapSliceEncodePlan adapts common []T values to FlatArray[T] before
// delegating to an ArrayGetter-compatible encode plan.
type wrapSliceEncodePlan[T any] struct {
	next EncodePlan
}

func (plan *wrapSliceEncodePlan[T]) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapSliceEncodePlan[T]) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode((FlatArray[T])(value.([]T)), buf)
}

// wrapSliceEncodeReflectPlan adapts uncommon one-dimensional slices to
// anySliceArrayReflect before delegating to an ArrayGetter-compatible encode
// plan.
type wrapSliceEncodeReflectPlan struct {
	next EncodePlan
}

func (plan *wrapSliceEncodeReflectPlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapSliceEncodeReflectPlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	w := anySliceArrayReflect{
		slice: reflect.ValueOf(value),
	}

	return plan.next.Encode(w, buf)
}

// TryWrapMultiDimSliceEncodePlan wraps rectangular multi-dimensional slices as
// ArrayGetter values. Ragged slices are skipped because Firebird arrays require
// fixed dimensions.
func TryWrapMultiDimSliceEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	sliceValue := reflect.ValueOf(value)
	if sliceValue.Kind() == reflect.Slice {
		valueElemType := sliceValue.Type().Elem()

		if valueElemType.Kind() == reflect.Slice {
			if !isRagged(sliceValue) {
				w := anyMultiDimSliceArray{
					slice: reflect.ValueOf(value),
				}
				return &wrapMultiDimSliceEncodePlan{}, &w, true
			}
		}
	}

	return nil, nil, false
}

// wrapMultiDimSliceEncodePlan adapts a rectangular multi-dimensional slice to
// anyMultiDimSliceArray before delegating to an ArrayGetter-compatible encode
// plan.
type wrapMultiDimSliceEncodePlan struct {
	next EncodePlan
}

func (plan *wrapMultiDimSliceEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapMultiDimSliceEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	w := anyMultiDimSliceArray{
		slice: reflect.ValueOf(value),
	}

	return plan.next.Encode(&w, buf)
}

// TryWrapArrayEncodePlan wraps fixed-length Go arrays as ArrayGetter values.
func TryWrapArrayEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	if valueType := reflect.TypeOf(value); valueType != nil && valueType.Kind() == reflect.Array {
		w := anyArrayArrayReflect{
			array: reflect.ValueOf(value),
		}
		return &wrapArrayEncodeReflectPlan{}, w, true
	}

	return nil, nil, false
}

// wrapArrayEncodeReflectPlan adapts a fixed-length Go array to
// anyArrayArrayReflect before delegating to an ArrayGetter-compatible encode
// plan.
type wrapArrayEncodeReflectPlan struct {
	next EncodePlan
}

func (plan *wrapArrayEncodeReflectPlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapArrayEncodeReflectPlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	w := anyArrayArrayReflect{
		array: reflect.ValueOf(value),
	}

	return plan.next.Encode(w, buf)
}

func newEncodeError(value any, m *Map, typeid uint32, subType int32, err error) error {

	var dataTypeName string
	if t, ok := m.TypeForTypeID(typeid, subType); ok {
		dataTypeName = t.Name
	} else {
		dataTypeName = "unknown type"
	}

	return fmt.Errorf("unable to encode %#v for %s (TypeID %d): %w", value, dataTypeName, typeid, err)
}

// Encode appends the encoded bytes of value to buf. If value is SQL NULL, it
// appends nothing and returns (nil, nil). The caller is responsible for writing
// the NULL marker or encoded data length.
func (m *Map) Encode(typeMetadata TypeMetadata, value any, buf []byte) (newBuf []byte, err error) {
	if isNil, callNilDriverValuer := isNilDriverValuer(value); isNil {
		if callNilDriverValuer {
			newBuf, err = (&encodePlanDriverValuer{m: m, typeMetadata: typeMetadata}).Encode(value, buf)
			if err != nil {
				return nil, newEncodeError(value, m, typeMetadata.TypeID, typeMetadata.SubType, err)
			}

			return newBuf, nil
		} else {
			return nil, nil
		}
	}

	plan := m.PlanEncode(typeMetadata, value)
	if plan == nil {
		return nil, newEncodeError(value, m, typeMetadata.TypeID, typeMetadata.SubType, errors.New("cannot find encode plan"))
	}

	newBuf, err = plan.Encode(value, buf)
	if err != nil {
		return nil, newEncodeError(value, m, typeMetadata.TypeID, typeMetadata.SubType, err)
	}

	return newBuf, nil
}

// appendNonNullBytes preserves the distinction between an empty encoded value
// and SQL NULL when the destination buffer has not been initialized yet.
func appendNonNullBytes(buf, value []byte) []byte {
	if buf == nil && len(value) == 0 {
		return []byte{}
	}

	return append(buf, value...)
}

var valuerReflectType = reflect.TypeFor[driver.Valuer]()

func isNilValue(value any) bool {
	if value == nil {
		return true
	}

	refVal := reflect.ValueOf(value)
	switch refVal.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return refVal.IsNil()
	default:
		return false
	}
}

// isNilDriverValuer reports whether value is nil and whether a nil
// driver.Valuer should still have Value called.
func isNilDriverValuer(value any) (isNil bool, callNilDriverValuer bool) {
	if value == nil {
		return true, false
	}

	refVal := reflect.ValueOf(value)
	kind := refVal.Kind()
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		if !refVal.IsNil() {
			return false, false
		}

		if _, ok := value.(driver.Valuer); ok {
			if kind == reflect.Pointer {
				// Call Value only when driver.Valuer is implemented on *T.
				return true, !refVal.Type().Elem().Implements(valuerReflectType)
			} else {
				return true, true
			}
		}

		return true, false
	default:
		return false, false
	}
}
