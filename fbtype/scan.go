package fbtype

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

// ErrScanTargetTypeChanged is returned by scan plans when Scan is called with a
// target type different from the type used to build the plan.
var ErrScanTargetTypeChanged = errors.New("scan target type changed")

// scanPlanCodecSQLScanner is used by [Map.planScan] for database/sql
// [sql.Scanner] targets when the Firebird type has a registered [Codec]. It
// decodes Firebird bytes into a driver.Value-compatible value before calling
// Scanner.Scan, so custom scanners receive the same kind of value they would
// expect from database/sql.
type scanPlanCodecSQLScanner struct {
	c            Codec
	m            *Map
	typeMetadata TypeMetadata
}

func (plan *scanPlanCodecSQLScanner) Scan(src []byte, dst any) error {
	value, err := plan.c.DecodeDatabaseSQLValue(plan.m, plan.typeMetadata, src)
	if err != nil {
		return err
	}

	scanner := dst.(sql.Scanner)
	return scanner.Scan(value)
}

// scanPlanSQLScanner is used by [Map.planScan] for database/sql [sql.Scanner]
// targets when the Firebird type has no registered [Codec]. Without a typed
// decode step, non-NULL values are passed as raw bytes and SQL NULL is passed
// as a nil interface value.
type scanPlanSQLScanner struct{}

func (plan *scanPlanSQLScanner) Scan(src []byte, dst any) error {
	scanner := dst.(sql.Scanner)

	if src == nil {
		// An interface value holding []byte(nil) does not compare equal to nil.
		return scanner.Scan(nil)
	} else {
		return scanner.Scan(src)
	}
}

// scanPlanFail is the fallback returned by [Map.planScan] when no direct plan,
// wrapper plan, sql.Scanner adapter, or *any fallback can handle the target. It
// reports the scan mismatch, with a special NULL path for targets that can
// accept NULL independently of the server-reported type.
type scanPlanFail struct {
	m            *Map
	typeMetadata TypeMetadata
}

func (plan *scanPlanFail) Scan(src []byte, dst any) error {
	// If src is NULL it might be possible to scan into dst even though the types
	// are not compatible. This can occur when selecting NULL directly. Firebird
	// assigns it the type of text, but it would be surprising to require the
	// caller to cast the NULL (e.g. `cast(null as int)`), so try to find a
	// compatible data type for dst and scan with that.
	if src == nil {
		// Try all registered types to find any plan that can scan NULL into dst.
		for typeid := range plan.m.typeidToType {
			// Use planScan instead of Scan or PlanScan to avoid polluting the
			// planned scan cache.
			plan.typeMetadata.TypeID = typeid
			plan := plan.m.planScan(plan.typeMetadata, dst, 0)
			if _, ok := plan.(*scanPlanFail); !ok {
				return plan.Scan(src, dst)
			}
		}
		for typeid := range defaultMap.typeidToType {
			if _, ok := plan.m.typeidToType[typeid]; !ok {
				plan.typeMetadata.TypeID = typeid
				plan := plan.m.planScan(plan.typeMetadata, dst, 0)
				if _, ok := plan.(*scanPlanFail); !ok {
					return plan.Scan(src, dst)
				}
			}
		}
	}

	var dataTypeName string
	if t, ok := plan.m.TypeForTypeID(plan.typeMetadata.TypeID, plan.typeMetadata.SubType); ok {
		dataTypeName = t.Name
	} else {
		dataTypeName = "unknown type"
	}

	return fmt.Errorf("cannot scan %s (type id %d) with subtype %d into %T", dataTypeName, plan.typeMetadata.TypeID, plan.typeMetadata.SubType, dst)
}

// pointerEmptyInterfaceScanPlan is used by [Map.planScan] for *any targets. It
// delegates to the selected [Codec]'s DecodeValue method so the target receives
// the codec's default Go representation.
type pointerEmptyInterfaceScanPlan struct {
	codec        Codec
	m            *Map
	typeMetadata TypeMetadata
}

func (plan *pointerEmptyInterfaceScanPlan) Scan(src []byte, dst any) error {
	value, err := plan.codec.DecodeValue(plan.m, plan.typeMetadata, src)
	if err != nil {
		return err
	}

	ptrAny := dst.(*any)
	*ptrAny = value

	return nil
}

// scanPlanString is used by [Map.planScan] as the fast path for scanning
// Firebird TEXT and VARYING values into string-like targets. It decodes the
// value with the charset from [TypeMetadata] before assigning it to *string,
// **string, or [TextScanner].
type scanPlanString struct {
	typeId   uint32
	length   uint32
	encoding *fbproto.Encoding
}

func (s scanPlanString) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	var tmp string
	if err := decodeFBString(src, s.typeId == fbproto.SQL_TYPE_VARYING, s.length, s.encoding, &tmp); err != nil {
		return err
	}

	switch d := dst.(type) {
	case *string:
		*d = tmp
		return nil

	case **string:
		if *d == nil {
			*d = new(string)
		}
		**d = tmp
		return nil

	case TextScanner:
		return d.ScanText(Text{String: tmp, Valid: true})

	default:
		return fmt.Errorf("scanPlanString: cannot scan string into %T", dst)
	}
}

// scanPlanAnyToUndecodedBytes is used by [Map.planScan] for *UndecodedBytes
// targets before type lookup. It returns the raw Firebird bytes for any server
// type without invoking a codec; non-NULL values are copied because src is only
// valid during Scan, and SQL NULL becomes a nil slice.
type scanPlanAnyToUndecodedBytes struct{}

func (scanPlanAnyToUndecodedBytes) Scan(src []byte, dst any) error {
	dstBuf := dst.(*UndecodedBytes)
	if src == nil {
		*dstBuf = nil
		return nil
	}

	*dstBuf = make([]byte, len(src))
	copy(*dstBuf, src)
	return nil
}

// TryWrapScanPlanFunc tries to create a wrapper plan for target. If successful
// it returns a plan that converts the original target before calling the next
// plan, plus nextTarget as the converted target used to find that next plan.
type TryWrapScanPlanFunc func(target any) (plan WrappedScanPlanNextSetter, nextTarget any, ok bool)

// WrappedScanPlanNextSetter is a ScanPlan wrapper that receives the next plan
// after recursive planning succeeds.
type WrappedScanPlanNextSetter interface {
	SetNext(ScanPlan)
	ScanPlan
}

// pointerPointerScanPlan adapts **T scan targets to the *T target expected by
// the next plan. SQL NULL clears the outer pointer; non-NULL values allocate a
// new inner value before delegating.
type pointerPointerScanPlan struct {
	next ScanPlan
}

func (plan *pointerPointerScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *pointerPointerScanPlan) Scan(src []byte, dst any) error {
	el := reflect.ValueOf(dst).Elem()
	if src == nil {
		el.Set(reflect.Zero(el.Type()))
		return nil
	}

	el.Set(reflect.New(el.Type().Elem()))
	return plan.next.Scan(src, el.Interface())
}

// TryPointerPointerScanPlan handles a pointer to a pointer by setting the
// target to nil for SQL NULL and allocating and scanning for non-NULL.
func TryPointerPointerScanPlan(target any) (plan WrappedScanPlanNextSetter, nextTarget any, ok bool) {
	if dstValue := reflect.ValueOf(target); dstValue.Kind() == reflect.Ptr {
		elemValue := dstValue.Elem()
		if elemValue.Kind() == reflect.Ptr {
			plan = &pointerPointerScanPlan{}
			return plan, reflect.Zero(elemValue.Type()).Interface(), true
		}
	}

	return nil, nil, false
}

// SkipUnderlyingTypePlanner prevents PlanScan and PlanEncode from trying to use
// the underlying type.
type SkipUnderlyingTypePlanner interface {
	SkipUnderlyingTypePlan()
}

// elemKindToPointerTypes maps named scalar target kinds to the pointer type
// used to find a scan plan for their underlying Go type.
var elemKindToPointerTypes map[reflect.Kind]reflect.Type = map[reflect.Kind]reflect.Type{
	reflect.Int:     reflect.TypeOf(new(int)),
	reflect.Int8:    reflect.TypeOf(new(int8)),
	reflect.Int16:   reflect.TypeOf(new(int16)),
	reflect.Int32:   reflect.TypeOf(new(int32)),
	reflect.Int64:   reflect.TypeOf(new(int64)),
	reflect.Uint:    reflect.TypeOf(new(uint)),
	reflect.Uint8:   reflect.TypeOf(new(uint8)),
	reflect.Uint16:  reflect.TypeOf(new(uint16)),
	reflect.Uint32:  reflect.TypeOf(new(uint32)),
	reflect.Uint64:  reflect.TypeOf(new(uint64)),
	reflect.Float32: reflect.TypeOf(new(float32)),
	reflect.Float64: reflect.TypeOf(new(float64)),
	reflect.String:  reflect.TypeOf(new(string)),
	reflect.Bool:    reflect.TypeOf(new(bool)),
}

// underlyingTypeScanPlan adapts pointers to named scalar, byte-slice, or array
// types to a pointer to their underlying Go type before delegating.
type underlyingTypeScanPlan struct {
	nextDstType reflect.Type
	next        ScanPlan
}

func (plan *underlyingTypeScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *underlyingTypeScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, reflect.ValueOf(dst).Convert(plan.nextDstType).Interface())
}

// TryFindUnderlyingTypeScanPlan tries to convert a target pointer to a pointer
// to its underlying Go type. For example, *MyString where MyString is defined
// as string can be planned as *string.
func TryFindUnderlyingTypeScanPlan(dst any) (plan WrappedScanPlanNextSetter, nextDst any, ok bool) {
	if _, ok := dst.(SkipUnderlyingTypePlanner); ok {
		return nil, nil, false
	}

	dstValue := reflect.ValueOf(dst)

	if dstValue.Kind() == reflect.Ptr {
		var elemValue reflect.Value
		if dstValue.IsNil() {
			elemValue = reflect.New(dstValue.Type().Elem()).Elem()
		} else {
			elemValue = dstValue.Elem()
		}
		nextDstType := elemKindToPointerTypes[elemValue.Kind()]
		if nextDstType == nil {
			if elemValue.Kind() == reflect.Slice {
				if elemValue.Type().Elem().Kind() == reflect.Uint8 {
					var v *[]byte
					nextDstType = reflect.TypeOf(v)
				}
			}

			// Get underlying type of any array.
			if elemValue.Kind() == reflect.Array {
				nextDstType = reflect.PointerTo(reflect.ArrayOf(elemValue.Len(), elemValue.Type().Elem()))
			}
		}

		if nextDstType != nil && dstValue.Type() != nextDstType && dstValue.CanConvert(nextDstType) {
			return &underlyingTypeScanPlan{nextDstType: nextDstType}, dstValue.Convert(nextDstType).Interface(), true
		}
	}

	return nil, nil, false
}

// TryWrapBuiltinTypeScanPlan tries to wrap a pointer to a Go builtin type with
// a wrapper type that implements fbtype scanner interfaces. For example, *int32
// can be planned as *int32Wrapper, which implements [Int64Scanner].
func TryWrapBuiltinTypeScanPlan(target any) (plan WrappedScanPlanNextSetter, nextDst any, ok bool) {
	switch target := target.(type) {
	case *int8:
		return &wrapInt8ScanPlan{}, (*int8Wrapper)(target), true
	case *int16:
		return &wrapInt16ScanPlan{}, (*int16Wrapper)(target), true
	case *int32:
		return &wrapInt32ScanPlan{}, (*int32Wrapper)(target), true
	case *int64:
		return &wrapInt64ScanPlan{}, (*int64Wrapper)(target), true
	case *int:
		return &wrapIntScanPlan{}, (*intWrapper)(target), true
	case *uint8:
		return &wrapUint8ScanPlan{}, (*uint8Wrapper)(target), true
	case *uint16:
		return &wrapUint16ScanPlan{}, (*uint16Wrapper)(target), true
	case *uint32:
		return &wrapUint32ScanPlan{}, (*uint32Wrapper)(target), true
	case *uint64:
		return &wrapUint64ScanPlan{}, (*uint64Wrapper)(target), true
	case *uint:
		return &wrapUintScanPlan{}, (*uintWrapper)(target), true
	case *float32:
		return &wrapFloat32ScanPlan{}, (*float32Wrapper)(target), true
	case *float64:
		return &wrapFloat64ScanPlan{}, (*float64Wrapper)(target), true
	case *string:
		return &wrapStringScanPlan{}, (*stringWrapper)(target), true
	case *time.Time:
		return &wrapTimeScanPlan{}, (*timeWrapper)(target), true
	case *[]byte:
		return &wrapByteSliceScanPlan{}, (*byteSliceWrapper)(target), true
	}

	return nil, nil, false
}

// The wrap*ScanPlan types below convert the original builtin target pointer to
// its matching wrapper type from builtin_wrappers.go, then delegate to the next
// plan selected for that wrapper.
type wrapInt8ScanPlan struct {
	next ScanPlan
}

func (plan *wrapInt8ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapInt8ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*int8Wrapper)(dst.(*int8)))
}

type wrapInt16ScanPlan struct {
	next ScanPlan
}

func (plan *wrapInt16ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapInt16ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*int16Wrapper)(dst.(*int16)))
}

type wrapInt32ScanPlan struct {
	next ScanPlan
}

func (plan *wrapInt32ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapInt32ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*int32Wrapper)(dst.(*int32)))
}

type wrapInt64ScanPlan struct {
	next ScanPlan
}

func (plan *wrapInt64ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapInt64ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*int64Wrapper)(dst.(*int64)))
}

type wrapIntScanPlan struct {
	next ScanPlan
}

func (plan *wrapIntScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapIntScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*intWrapper)(dst.(*int)))
}

type wrapUint8ScanPlan struct {
	next ScanPlan
}

func (plan *wrapUint8ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapUint8ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*uint8Wrapper)(dst.(*uint8)))
}

type wrapUint16ScanPlan struct {
	next ScanPlan
}

func (plan *wrapUint16ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapUint16ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*uint16Wrapper)(dst.(*uint16)))
}

type wrapUint32ScanPlan struct {
	next ScanPlan
}

func (plan *wrapUint32ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapUint32ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*uint32Wrapper)(dst.(*uint32)))
}

type wrapUint64ScanPlan struct {
	next ScanPlan
}

func (plan *wrapUint64ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapUint64ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*uint64Wrapper)(dst.(*uint64)))
}

type wrapUintScanPlan struct {
	next ScanPlan
}

func (plan *wrapUintScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapUintScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*uintWrapper)(dst.(*uint)))
}

type wrapFloat32ScanPlan struct {
	next ScanPlan
}

func (plan *wrapFloat32ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapFloat32ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*float32Wrapper)(dst.(*float32)))
}

type wrapFloat64ScanPlan struct {
	next ScanPlan
}

func (plan *wrapFloat64ScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapFloat64ScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*float64Wrapper)(dst.(*float64)))
}

type wrapStringScanPlan struct {
	next ScanPlan
}

func (plan *wrapStringScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapStringScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*stringWrapper)(dst.(*string)))
}

type wrapTimeScanPlan struct {
	next ScanPlan
}

func (plan *wrapTimeScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapTimeScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*timeWrapper)(dst.(*time.Time)))
}

type wrapByteSliceScanPlan struct {
	next ScanPlan
}

func (plan *wrapByteSliceScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapByteSliceScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*byteSliceWrapper)(dst.(*[]byte)))
}

// TryWrapPtrSliceScanPlan adapts a pointer to a single-dimensional slice to an
// ArraySetter-compatible target. Common slice element types use FlatArray[T];
// other single-dimensional slices use reflection.
func TryWrapPtrSliceScanPlan(target any) (plan WrappedScanPlanNextSetter, nextValue any, ok bool) {
	// Avoid using reflect path for common types.
	switch target := target.(type) {
	case *[]int16:
		return &wrapPtrSliceScanPlan[int16]{}, (*FlatArray[int16])(target), true
	case *[]int32:
		return &wrapPtrSliceScanPlan[int32]{}, (*FlatArray[int32])(target), true
	case *[]int64:
		return &wrapPtrSliceScanPlan[int64]{}, (*FlatArray[int64])(target), true
	case *[]float32:
		return &wrapPtrSliceScanPlan[float32]{}, (*FlatArray[float32])(target), true
	case *[]float64:
		return &wrapPtrSliceScanPlan[float64]{}, (*FlatArray[float64])(target), true
	case *[]string:
		return &wrapPtrSliceScanPlan[string]{}, (*FlatArray[string])(target), true
	case *[]time.Time:
		return &wrapPtrSliceScanPlan[time.Time]{}, (*FlatArray[time.Time])(target), true
	}

	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return nil, nil, false
	}

	targetElemType := targetType.Elem()

	if targetElemType.Kind() == reflect.Slice {
		if targetElemType.Elem().Kind() == reflect.Slice {
			return nil, nil, false
		}

		slice := reflect.New(targetElemType).Elem()
		return &wrapPtrSliceReflectScanPlan{}, &anySliceArrayReflect{slice: slice}, true
	}
	return nil, nil, false
}

// wrapPtrSliceScanPlan adapts common *[]T targets to *FlatArray[T] before
// delegating to an ArraySetter-compatible scan plan.
type wrapPtrSliceScanPlan[T any] struct {
	next ScanPlan
}

func (plan *wrapPtrSliceScanPlan[T]) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapPtrSliceScanPlan[T]) Scan(src []byte, target any) error {
	return plan.next.Scan(src, (*FlatArray[T])(target.(*[]T)))
}

// wrapPtrSliceReflectScanPlan adapts uncommon *[]T targets to an
// anySliceArrayReflect before delegating to an ArraySetter-compatible scan
// plan.
type wrapPtrSliceReflectScanPlan struct {
	next ScanPlan
}

func (plan *wrapPtrSliceReflectScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapPtrSliceReflectScanPlan) Scan(src []byte, target any) error {
	return plan.next.Scan(src, &anySliceArrayReflect{slice: reflect.ValueOf(target).Elem()})
}

// TryWrapPtrMultiDimSliceScanPlan adapts a pointer to a multi-dimensional
// slice to an ArraySetter-compatible target. Existing slice contents may be
// ragged because scanning replaces them with the dimensions from the source
// value.
func TryWrapPtrMultiDimSliceScanPlan(target any) (plan WrappedScanPlanNextSetter, nextValue any, ok bool) {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return nil, nil, false
	}

	targetElemValue := targetValue.Elem()

	if targetElemValue.Kind() == reflect.Slice {
		elemElemKind := targetElemValue.Type().Elem().Kind()
		if elemElemKind == reflect.Slice {
			return &wrapPtrMultiDimSliceScanPlan{}, &anyMultiDimSliceArray{slice: targetValue.Elem()}, true
		}
	}

	return nil, nil, false
}

// wrapPtrMultiDimSliceScanPlan adapts *[][]T and deeper slice targets to an
// anyMultiDimSliceArray before delegating to an ArraySetter-compatible scan
// plan.
type wrapPtrMultiDimSliceScanPlan struct {
	next ScanPlan
}

func (plan *wrapPtrMultiDimSliceScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapPtrMultiDimSliceScanPlan) Scan(src []byte, target any) error {
	return plan.next.Scan(src, &anyMultiDimSliceArray{slice: reflect.ValueOf(target).Elem()})
}

// TryWrapPtrArrayScanPlan adapts a pointer to a fixed-length Go array to an
// ArraySetter-compatible target. Unlike slice targets, fixed arrays cannot be
// resized or cleared when scanning.
func TryWrapPtrArrayScanPlan(target any) (plan WrappedScanPlanNextSetter, nextValue any, ok bool) {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return nil, nil, false
	}

	targetElemValue := targetValue.Elem()

	if targetElemValue.Kind() == reflect.Array {
		return &wrapPtrArrayReflectScanPlan{}, &anyArrayArrayReflect{array: targetElemValue}, true
	}
	return nil, nil, false
}

// wrapPtrArrayReflectScanPlan adapts *[N]T targets to an anyArrayArrayReflect
// before delegating to an ArraySetter-compatible scan plan.
type wrapPtrArrayReflectScanPlan struct {
	next ScanPlan
}

func (plan *wrapPtrArrayReflectScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapPtrArrayReflectScanPlan) Scan(src []byte, target any) error {
	return plan.next.Scan(src, &anyArrayArrayReflect{array: reflect.ValueOf(target).Elem()})
}

// PlanScan prepares a plan to scan a value into target.
func (m *Map) PlanScan(typeMetadata TypeMetadata, target any) ScanPlan {
	return m.planScan(typeMetadata, target, 0)
}

func (m *Map) planScan(typeMetadata TypeMetadata, target any, depth int) ScanPlan {
	if depth > 8 {
		return &scanPlanFail{m: m, typeMetadata: typeMetadata}
	}

	if target == nil {
		return &scanPlanFail{m: m, typeMetadata: typeMetadata}
	}

	if _, ok := target.(*UndecodedBytes); ok {
		return scanPlanAnyToUndecodedBytes{}
	}

	isTextType := typeMetadata.TypeID == fbproto.SQL_TYPE_TEXT || typeMetadata.TypeID == fbproto.SQL_TYPE_VARYING
	if _, ok := target.(*string); ok && isTextType {
		charsetId := CharacterSetForType(typeMetadata.TypeID, typeMetadata.SubType, typeMetadata.Scale)
		name, err := fbproto.CharacterSetIdIntoName(charsetId)
		if err == nil {
			encoding, err := fbproto.NewEncoding(name)
			if err == nil {
				return scanPlanString{typeId: typeMetadata.TypeID, encoding: encoding, length: typeMetadata.Length}
			}
		}
	}

	var dt *Type

	if dataType, ok := m.TypeForTypeID(typeMetadata.TypeID, typeMetadata.SubType); ok {
		dt = dataType
	} else if dataType, ok := m.TypeForValue(target); ok {
		dt = dataType
		typeMetadata.TypeID = dt.TypeID // Preserve assumed TypeID in case we are recursively called below.
	}

	if dt != nil {
		if plan := dt.Codec.PlanScan(m, typeMetadata, target); plan != nil {
			return plan
		}
	}

	// This needs to happen before trying m.TryWrapScanPlanFuncs. Otherwise, a
	// sql.Scanner would not get called if it was defined on a type that could be
	// unwrapped such as `type myString string`.
	if _, ok := target.(sql.Scanner); ok {
		if dt == nil {
			return &scanPlanSQLScanner{}
		} else {
			return &scanPlanCodecSQLScanner{c: dt.Codec, m: m, typeMetadata: typeMetadata}
		}
	}

	for _, f := range m.TryWrapScanPlanFuncs {
		if wrapperPlan, nextDst, ok := f(target); ok {
			if nextPlan := m.planScan(typeMetadata, nextDst, depth+1); nextPlan != nil {
				if _, failed := nextPlan.(*scanPlanFail); !failed {
					wrapperPlan.SetNext(nextPlan)
					return wrapperPlan
				}
			}
		}
	}

	if _, ok := target.(*any); ok {
		var codec Codec
		if dt != nil {
			codec = dt.Codec
		} else {
			codec = OctetsCodec{}
		}
		return &pointerEmptyInterfaceScanPlan{codec: codec, m: m, typeMetadata: typeMetadata}
	}

	return &scanPlanFail{m: m, typeMetadata: typeMetadata}
}

// Scan scans src into dst using a plan selected from typeMetadata and dst. It is
// a convenience wrapper around PlanScan for callers that do not need to reuse
// the resulting plan.
func (m *Map) Scan(typeMetadata TypeMetadata, src []byte, dst any) error {
	if dst == nil {
		return nil
	}

	plan := m.PlanScan(typeMetadata, dst)
	return plan.Scan(src, dst)
}

// codecScan scans src into dst using a single codec directly. Codec
// DecodeValue and DecodeDatabaseSQLValue implementations use it to avoid
// repeating their own PlanScan boilerplate.
func codecScan(codec Codec, m *Map, typeMetadata TypeMetadata, src []byte, dst any) error {
	scanPlan := codec.PlanScan(m, typeMetadata, dst)
	if scanPlan == nil {
		return fmt.Errorf("PlanScan did not find a plan")
	}
	return scanPlan.Scan(src, dst)
}

// SQLScanner returns a database/sql.Scanner adapter for v. It is primarily used
// with the stdlib database/sql layer to scan Firebird arrays, where the driver
// value carries both array metadata and the raw array buffer.
//
// If v already implements sql.Scanner it is returned unchanged. Otherwise v
// must be a target supported by the Firebird array scan path, such as *[]T,
// *Array[T], *FlatArray[T], or a compatible array/slice target.
func (m *Map) SQLScanner(v any) sql.Scanner {
	if s, ok := v.(sql.Scanner); ok {
		return s
	}

	return &sqlScannerWrapper{m: m, v: v}
}

type sqlScannerWrapper struct {
	m *Map
	v any
}

func (w *sqlScannerWrapper) Scan(src any) error {
	arrayTarget, err := getArrayDecodeType(w.v)
	if err != nil {
		return err
	}
	if src == nil {
		return arrayTarget.SetDimensions(nil)
	}
	array, ok := src.(ArraySQLScanner)
	if !ok {
		return fmt.Errorf("cannot convert to sql.Scanner: cannot find registered type for %T", w.v)
	}
	sp := planScanArray{m: w.m, arrayData: array.ArrayData}

	return sp.Scan(array.ArrayBuffer, w.v)
}
