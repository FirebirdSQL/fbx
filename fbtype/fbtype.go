package fbtype

import (
	"database/sql/driver"
	"reflect"

	"github.com/firebirdsql/fbx/fbproto"
)

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

	// DecodeDatabaseSQLValue decodes src into a driver.Value-compatible value.
	DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error)

	// DecodeValue decodes src into the codec's default Go representation.
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

func defaultTryWrapEncodePlanFuncs() []TryWrapEncodePlanFunc {
	return []TryWrapEncodePlanFunc{
		TryWrapDerefPointerEncodePlan,
		TryWrapBuiltinTypeEncodePlan,
		TryWrapFindUnderlyingTypeEncodePlan,
		TryWrapStringerEncodePlan,
		TryWrapSliceEncodePlan,
		TryWrapMultiDimSliceEncodePlan,
		TryWrapArrayEncodePlan,
	}
}

func defaultTryWrapScanPlanFuncs() []TryWrapScanPlanFunc {
	return []TryWrapScanPlanFunc{
		TryPointerPointerScanPlan,
		TryWrapBuiltinTypeScanPlan,
		TryFindUnderlyingTypeScanPlan,
		TryWrapPtrSliceScanPlan,
		TryWrapPtrMultiDimSliceScanPlan,
		TryWrapPtrArrayScanPlan,
	}
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

		TryWrapEncodePlanFuncs: defaultTryWrapEncodePlanFuncs(),
		TryWrapScanPlanFuncs:   defaultTryWrapScanPlanFuncs(),
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
func (m *Map) TypeForTypeID(typeID uint32, subType int32) (*Type, bool) {
	id := MakeTypeSubtype(typeID, subType)
	if dt, ok := m.typeidToType[id]; ok {
		return dt, true
	}

	// Firebird may encode collation into high bits of subtype for TEXT/VARYING.
	// For binary text (OCTETS), the low byte is sufficient to identify the codec.
	if typeID == fbproto.SQL_TYPE_TEXT || typeID == fbproto.SQL_TYPE_VARYING {
		idNoCollation := MakeTypeSubtype(typeID, subType&0xFF)
		if dt, ok := m.typeidToType[idNoCollation]; ok {
			return dt, true
		}
	}

	if dt, ok := m.typeidToType[typeID]; ok {
		return dt, true
	}

	if dt, ok := defaultMap.typeidToType[id]; ok {
		return dt, true
	}

	if typeID == fbproto.SQL_TYPE_TEXT || typeID == fbproto.SQL_TYPE_VARYING {
		idNoCollation := MakeTypeSubtype(typeID, subType&0xFF)
		if dt, ok := defaultMap.typeidToType[idNoCollation]; ok {
			return dt, true
		}
	}

	dt, ok := defaultMap.typeidToType[typeID]
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
