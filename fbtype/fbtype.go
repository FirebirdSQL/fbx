package fbtype

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

// ErrScanTargetTypeChanged is returned by scan plans when Scan is called with a
// target type different from the type used to build the plan.
var ErrScanTargetTypeChanged = errors.New("scan target type changed")

// Codec converts between Go and Firebird values. A Codec must not be mutated
// after it is registered with a Map.
type Codec interface {

	// PlanEncode returns an EncodePlan for encoding value into Firebird format.
	// If no plan can be found then nil is returned.
	PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan

	// PlanScan returns a ScanPlan for scanning a Firebird value into a
	// destination with the same type as target. If no plan can be found then nil
	// is returned.
	PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan

	// DecodeDatabaseSQLValue returns src decoded into a value compatible with the sql.Scanner interface.
	DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error)

	// DecodeValue returns decoded src.
	DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error)
}

// Type represents a Firebird data type. It must not be mutated after it is
// registered with a Map.
type Type struct {
	Codec  Codec
	Name   string
	TypeID uint32
}

// TypeMetadata describes a Firebird value as reported by the server. Codecs use
// it to select plans and interpret raw wire bytes.
type TypeMetadata struct {
	TypeID    uint32
	SubType   int32
	Length    uint32
	Scale     int32
	ArrayData *ArrayData
}

// NewTypeMetadata creates TypeMetadata for a non-array Firebird value.
func NewTypeMetadata(typeID uint32, subType int32, length uint32, scale int32) TypeMetadata {
	var tm TypeMetadata

	tm.TypeID = typeID
	tm.SubType = subType
	tm.Length = length
	tm.Scale = scale

	return tm
}

// SetArrayMetadata attaches array element and dimension metadata.
func (tm *TypeMetadata) SetArrayMetadata(arrayData *ArrayData) {
	tm.ArrayData = arrayData
}

// Map is the mapping between Firebird server types and Go type handling logic.
// It can encode values for transmission to a Firebird server and scan received
// values.
type Map struct {
	typeidToType      map[uint32]*Type
	nameToType        map[string]*Type
	reflectTypeToName map[reflect.Type]string

	reflectTypeToType map[reflect.Type]*Type

	memoizedEncodePlans map[TypeMetadata]map[reflect.Type]EncodePlan

	// TryWrapEncodePlanFuncs is a slice of functions that wrap a value that
	// cannot be encoded by the Codec. Every time a wrapper is found,
	// PlanEncode is recursively called with the new value. This allows
	// several layers of wrappers to be built up. Default functions should
	// usually run last, so additional functions should typically be prepended.
	TryWrapEncodePlanFuncs []TryWrapEncodePlanFunc

	// TryWrapScanPlanFuncs is a slice of functions that wrap a target that
	// cannot be scanned into by the Codec. Every time a wrapper is found,
	// PlanScan is recursively called with the new target. This allows several
	// layers of wrappers to be built up. Default functions should usually run
	// last, so additional functions should typically be prepended.
	TryWrapScanPlanFuncs []TryWrapScanPlanFunc
}

// Copy returns a new Map containing the same registered types.
func (m *Map) Copy() *Map {
	newMap := NewMap()
	for _, type_ := range m.typeidToType {
		newMap.RegisterType(type_)
	}
	return newMap
}

// NewMap creates a Map with the default wrapper functions and fallback access
// to the package-level default type registrations.
func NewMap() *Map {
	defaultMapInitOnce.Do(initDefaultMap)

	return &Map{
		typeidToType:      make(map[uint32]*Type),
		nameToType:        make(map[string]*Type),
		reflectTypeToName: make(map[reflect.Type]string),

		memoizedEncodePlans: make(map[TypeMetadata]map[reflect.Type]EncodePlan),

		TryWrapEncodePlanFuncs: []TryWrapEncodePlanFunc{
			TryWrapDerefPointerEncodePlan,
			TryWrapBuiltinTypeEncodePlan,
			TryWrapFindUnderlyingTypeEncodePlan,
			TryWrapStringerEncodePlan,
			TryWrapStructEncodePlan,
			TryWrapSliceEncodePlan,
			TryWrapMultiDimSliceEncodePlan,
			TryWrapArrayEncodePlan,
		},

		TryWrapScanPlanFuncs: []TryWrapScanPlanFunc{
			TryPointerPointerScanPlan,
			TryWrapBuiltinTypeScanPlan,
			TryFindUnderlyingTypeScanPlan,
			TryWrapStructScanPlan,
			TryWrapPtrSliceScanPlan,
			TryWrapPtrMultiDimSliceScanPlan,
			TryWrapPtrArrayScanPlan,
		},
	}
}

// RegisterTypes registers multiple data types in the sequence they are provided.
func (m *Map) RegisterTypes(types []*Type) {
	for _, t := range types {
		m.RegisterType(t)
	}
}

// RegisterType registers a data type with the Map. t must not be mutated after
// it is registered.
func (m *Map) RegisterType(t *Type) {
	m.typeidToType[t.TypeID] = t
	m.nameToType[t.Name] = t

	// Invalidate caches affected by type registration.
	m.reflectTypeToType = nil
	for k := range m.memoizedEncodePlans {
		delete(m.memoizedEncodePlans, k)
	}
}

// RegisterDefaultFbType registers a mapping of a Go type to a Firebird type
// name. Typically the data type to be encoded or decoded is determined by the
// Firebird TypeID. But if the TypeID of a value to be encoded or decoded is
// unknown, this additional mapping will be used by TypeForValue to determine a
// suitable data type.
func (m *Map) RegisterDefaultFbType(value any, name string) {
	m.reflectTypeToName[reflect.TypeOf(value)] = name

	// Invalidate caches affected by type registration.
	m.reflectTypeToType = nil
	for k := range m.memoizedEncodePlans {
		delete(m.memoizedEncodePlans, k)
	}
}

// TypeForTypeID returns the Type registered for the given TypeID. The returned
// Type must not be mutated.
func (m *Map) TypeForTypeID(typeid uint32, subType int32) (*Type, bool) {
	id := MakeTypeSubtype(typeid, subType)
	if dt, ok := m.typeidToType[id]; ok {
		return dt, true
	}

	// Firebird may encode collation into high bits of subtype for TEXT/VARYING.
	// For binary text (OCTETS), the low byte is sufficient to identify the codec.
	if typeid == fbproto.SQL_TYPE_TEXT || typeid == fbproto.SQL_TYPE_VARYING {
		idNoCollation := MakeTypeSubtype(typeid, subType&0xFF)
		if dt, ok := m.typeidToType[idNoCollation]; ok {
			return dt, true
		}
	}

	if dt, ok := m.typeidToType[typeid]; ok {
		return dt, true
	}

	if dt, ok := defaultMap.typeidToType[id]; ok {
		return dt, true
	}

	if typeid == fbproto.SQL_TYPE_TEXT || typeid == fbproto.SQL_TYPE_VARYING {
		idNoCollation := MakeTypeSubtype(typeid, subType&0xFF)
		if dt, ok := defaultMap.typeidToType[idNoCollation]; ok {
			return dt, true
		}
	}

	dt, ok := defaultMap.typeidToType[typeid]
	return dt, ok
}

// TypeForName returns the Type registered for the given name. The returned Type
// must not be mutated.
func (m *Map) TypeForName(name string) (*Type, bool) {
	if dt, ok := m.nameToType[name]; ok {
		return dt, true
	}
	dt, ok := defaultMap.nameToType[name]
	return dt, ok
}

func (m *Map) buildReflectTypeToType() {
	m.reflectTypeToType = make(map[reflect.Type]*Type)

	for reflectType, name := range m.reflectTypeToName {
		if dt, ok := m.TypeForName(name); ok {
			m.reflectTypeToType[reflectType] = dt
		}
	}
}

// TypeForValue finds a data type suitable for v. Use RegisterType to register
// types that can encode and decode themselves. Use RegisterDefaultFbType to
// register Go types that can be handled by a registered data type. The returned
// Type must not be mutated.
func (m *Map) TypeForValue(v any) (*Type, bool) {
	if m.reflectTypeToType == nil {
		m.buildReflectTypeToType()
	}

	if dt, ok := m.reflectTypeToType[reflect.TypeOf(v)]; ok {
		return dt, true
	}

	dt, ok := defaultMap.reflectTypeToType[reflect.TypeOf(v)]
	return dt, ok
}

// EncodePlan is a precompiled plan to encode a particular type into a
// particular TypeID.
type EncodePlan interface {
	// Encode appends the encoded bytes of value to buf. If value is SQL NULL,
	// Encode appends nothing and returns (nil, nil). The caller is responsible
	// for writing the correct NULL marker or data length.
	Encode(value any, buf []byte) (newBuf []byte, err error)
}

// ScanPlan is a precompiled plan to scan into a type of destination.
type ScanPlan interface {
	// Scan scans src into target. src is only valid during the call to Scan. The
	// ScanPlan must not retain a reference to src.
	Scan(src []byte, target any) error
}

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

// TryWrapStructScanPlan wraps a pointer to a struct as an index-addressable
// field scanner for codecs that scan composite-like values field by field.
func TryWrapStructScanPlan(target any) (plan WrappedScanPlanNextSetter, nextValue any, ok bool) {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return nil, nil, false
	}

	var targetElemValue reflect.Value
	if targetValue.IsNil() {
		targetElemValue = reflect.Zero(targetValue.Type().Elem())
	} else {
		targetElemValue = targetValue.Elem()
	}
	targetElemType := targetElemValue.Type()

	if targetElemType.Kind() == reflect.Struct {
		exportedFields := getExportedFieldValues(targetElemValue)
		if len(exportedFields) == 0 {
			return nil, nil, false
		}

		w := ptrStructWrapper{
			s:              target,
			exportedFields: exportedFields,
		}
		return &wrapAnyPtrStructScanPlan{}, &w, true
	}

	return nil, nil, false
}

type wrapAnyPtrStructScanPlan struct {
	next ScanPlan
}

func (plan *wrapAnyPtrStructScanPlan) SetNext(next ScanPlan) { plan.next = next }

func (plan *wrapAnyPtrStructScanPlan) Scan(src []byte, target any) error {
	w := ptrStructWrapper{
		s:              target,
		exportedFields: getExportedFieldValues(reflect.ValueOf(target).Elem()),
	}

	return plan.next.Scan(src, &w)
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
			codec = ByteaCodec{}
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
	_, err := getArrayDecodeType(w.v)
	if err != nil {
		return err
	}
	if src == nil {
		return nil
	}
	array, ok := src.(ArraySQLScanner)
	if !ok {
		return fmt.Errorf("cannot convert to sql.Scanner: cannot find registered type for %T", w.v)
	}
	sp := planScanArray{m: w.m, arrayData: array.ArrayData}

	return sp.Scan(array.ArrayBuffer, w.v)
}

// PlanEncode returns an EncodePlan for encoding value into Firebird format for typeid. If no plan can be
// found then nil is returned.
func (m *Map) PlanEncode(typeMetadata TypeMetadata, value any) EncodePlan {
	return m.planEncodeDepth(typeMetadata, value, 0)
}

func (m *Map) planEncodeDepth(typeMetadata TypeMetadata, value any, depth int) EncodePlan {
	// Guard against infinite recursion.
	if depth > 8 {
		return nil
	}

	//memoize type id + subtype
	// memoId := MakeTypeSubtype(typeid, subType)
	// typeidMemo := m.memoizedEncodePlans[memoId]
	// if typeidMemo == nil {
	// 	typeidMemo = make(map[reflect.Type][2]EncodePlan)
	// 	m.memoizedEncodePlans[memoId] = typeidMemo
	// }
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
		// If no type for the TypeID was found, then either it is unknowable (e.g. the simple protocol) or it is an
		// unregistered type. In either case try to find the type and TypeID that matches the value (e.g. a []byte would be
		// registered to Firebird bytea).
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

type encodePlanDriverValuer struct {
	m            *Map
	typeMetadata TypeMetadata
}

func (plan *encodePlanDriverValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
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

// TryWrapEncodePlanFunc is a function that tries to create a wrapper plan for value. If successful it returns a plan
// that will convert the value passed to Encode and then call the next plan. nextValue is value as it will be converted
// by plan. It must be used to find another suitable EncodePlan. When it is found SetNext must be called on plan for it
// to be usabled. ok indicates if a suitable wrapper was found.
type TryWrapEncodePlanFunc func(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool)

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

// TryWrapDerefPointerEncodePlan tries to dereference a pointer. e.g. If value was of type *string then a wrapper plan
// would be returned that dereferences the value.
func TryWrapDerefPointerEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	if valueType := reflect.TypeOf(value); valueType != nil && valueType.Kind() == reflect.Ptr {
		return &derefPointerEncodePlan{}, reflect.New(valueType.Elem()).Elem().Interface(), true
	}

	return nil, nil, false
}

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

type underlyingTypeEncodePlan struct {
	nextValueType reflect.Type
	next          EncodePlan
}

func (plan *underlyingTypeEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *underlyingTypeEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(reflect.ValueOf(value).Convert(plan.nextValueType).Interface(), buf)
}

// TryWrapFindUnderlyingTypeEncodePlan tries to convert to a Go builtin type. e.g. If value was of type MyString and
// MyString was defined as a string then a wrapper plan would be returned that converts MyString to string.
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

// TryWrapStringerEncodePlan tries to wrap a fmt.Stringer type with a wrapper that provides TextValuer. This is
// intentionally a separate function from TryWrapBuiltinTypeEncodePlan so it can be ordered after
// TryWrapFindUnderlyingTypeEncodePlan. This ensures that named types with an underlying builtin type (e.g. type MyEnum
// int32 with a String() method) prefer encoding via the underlying type's codec (e.g. as an integer) rather than via
// Stringer. Stringer is only used as a fallback when no type-specific encoding plan succeeds.
func TryWrapStringerEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	if s, ok := value.(fmt.Stringer); ok {
		return &wrapFmtStringerEncodePlan{}, fmtStringerWrapper{s}, true
	}

	return nil, nil, false
}

type WrappedEncodePlanNextSetter interface {
	SetNext(EncodePlan)
	EncodePlan
}

// TryWrapBuiltinTypeEncodePlan tries to wrap a builtin type with a wrapper that provides additional methods. e.g. If
// value was of type int32 then a wrapper plan would be returned that converts value to a type that implements
// Int64Valuer.
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
	// case time.Duration:
	// 	return &wrapDurationEncodePlan{}, durationWrapper(value), true
	// case map[string]*string:
	// 	return &wrapMapStringToPointerStringEncodePlan{}, mapStringToPointerStringWrapper(value), true
	// case map[string]string:
	// return &wrapMapStringToStringEncodePlan{}, mapStringToStringWrapper(value), true
	// case [16]byte:
	// 	return &wrapByte16EncodePlan{}, byte16Wrapper(value), true
	case []byte:
		return &wrapByteSliceEncodePlan{}, byteSliceWrapper(value), true
	}

	return nil, nil, false
}

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

// type wrapDurationEncodePlan struct {
// 	next EncodePlan
// }

// func (plan *wrapDurationEncodePlan) SetNext(next EncodePlan) { plan.next = next }

// func (plan *wrapDurationEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
// 	return plan.next.Encode(durationWrapper(value.(time.Duration)), buf)
// }

// type wrapMapStringToPointerStringEncodePlan struct {
// 	next EncodePlan
// }

// func (plan *wrapMapStringToPointerStringEncodePlan) SetNext(next EncodePlan) { plan.next = next }

// func (plan *wrapMapStringToPointerStringEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
// 	return plan.next.Encode(mapStringToPointerStringWrapper(value.(map[string]*string)), buf)
// }

// type wrapMapStringToStringEncodePlan struct {
// 	next EncodePlan
// }

// func (plan *wrapMapStringToStringEncodePlan) SetNext(next EncodePlan) { plan.next = next }

// func (plan *wrapMapStringToStringEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
// 	return plan.next.Encode(mapStringToStringWrapper(value.(map[string]string)), buf)
// }

// type wrapByte16EncodePlan struct {
// 	next EncodePlan
// }

// func (plan *wrapByte16EncodePlan) SetNext(next EncodePlan) { plan.next = next }

// func (plan *wrapByte16EncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
// 	return plan.next.Encode(byte16Wrapper(value.([16]byte)), buf)
// }

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

// TryWrapStructPlan tries to wrap a struct with a wrapper that implements CompositeIndexGetter.
func TryWrapStructEncodePlan(value any) (plan WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	if _, ok := value.(driver.Valuer); ok {
		return nil, nil, false
	}

	if valueType := reflect.TypeOf(value); valueType != nil && valueType.Kind() == reflect.Struct {
		exportedFields := getExportedFieldValues(reflect.ValueOf(value))
		if len(exportedFields) == 0 {
			return nil, nil, false
		}

		w := structWrapper{
			s:              value,
			exportedFields: exportedFields,
		}
		return &wrapAnyStructEncodePlan{}, w, true
	}

	return nil, nil, false
}

type wrapAnyStructEncodePlan struct {
	next EncodePlan
}

func (plan *wrapAnyStructEncodePlan) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapAnyStructEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	w := structWrapper{
		s:              value,
		exportedFields: getExportedFieldValues(reflect.ValueOf(value)),
	}

	return plan.next.Encode(w, buf)
}

func getExportedFieldValues(structValue reflect.Value) []reflect.Value {
	structType := structValue.Type()
	exportedFields := make([]reflect.Value, 0, structValue.NumField())
	for i := 0; i < structType.NumField(); i++ {
		sf := structType.Field(i)
		if sf.IsExported() {
			exportedFields = append(exportedFields, structValue.Field(i))
		}
	}

	return exportedFields
}

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
		w := anySliceArrayReflect{
			slice: reflect.ValueOf(value),
		}
		return &wrapSliceEncodeReflectPlan{}, w, true
	}

	return nil, nil, false
}

type wrapSliceEncodePlan[T any] struct {
	next EncodePlan
}

func (plan *wrapSliceEncodePlan[T]) SetNext(next EncodePlan) { plan.next = next }

func (plan *wrapSliceEncodePlan[T]) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode((FlatArray[T])(value.([]T)), buf)
}

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

// Encode appends the encoded bytes of value to buf. If value is the SQL value NULL then append nothing and return
// (nil, nil). The caller of Encode is responsible for writing the correct NULL value or the length of the data
// written.
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

var valuerReflectType = reflect.TypeFor[driver.Valuer]()

// isNilDriverValuer reports whether value is nil and whether a nil
// driver.Valuer should still have Value called.
func isNilDriverValuer(value any) (isNil bool, callNilDriverValuer bool) {
	if value == nil {
		return true, false
	}

	refVal := reflect.ValueOf(value)
	kind := refVal.Kind()
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		if !refVal.IsNil() {
			return false, false
		}

		if _, ok := value.(driver.Valuer); ok {
			if kind == reflect.Ptr {
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

// CharacterSetForType returns the Firebird character set id for type metadata.
func CharacterSetForType(typeID uint32, subType int32, scale int32) int32 {
	switch typeID {
	case fbproto.SQL_TYPE_TEXT, fbproto.SQL_TYPE_VARYING:
		return subType & 0xFF
	case fbproto.SQL_TYPE_BLOB:
		if subType == fbproto.Isc_blob_text {
			return scale & 0xFF
		}
		return fbproto.CS_BINARY
	}
	return fbproto.CS_dynamic
}
