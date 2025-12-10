package fbtype

import (
	"reflect"
	"sync"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

var (
	// defaultMap contains default mappings between Firebird server types and Go type handling logic.
	defaultMap         *Map
	defaultMapInitOnce = sync.Once{}
)

func MakeTypeSubtype(t uint32, sub int32) uint32 {
	return uint32(int32(t) | (sub << 16))
}

func GetType(t uint32) uint32 {
	return uint32(int32(t) & 0xFFFF)
}

func GetSubType(t uint32) int32 {
	return int32(t) >> 16
}

func initDefaultMap() {
	defaultMap = &Map{
		oidToType:         make(map[uint32]*Type),
		nameToType:        make(map[string]*Type),
		reflectTypeToName: make(map[reflect.Type]string),
		oidToFormatCode:   make(map[uint32]int16),

		memoizedEncodePlans: make(map[uint32]map[reflect.Type][2]EncodePlan),

		TryWrapEncodePlanFuncs: []TryWrapEncodePlanFunc{
			TryWrapDerefPointerEncodePlan,
			TryWrapBuiltinTypeEncodePlan,
			TryWrapFindUnderlyingTypeEncodePlan,
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

	// Base types
	defaultMap.RegisterType(&Type{Name: "date", OID: fbproto.SQL_TYPE_DATE, Codec: DateCodec{}})
	defaultMap.RegisterType(&Type{Name: "smallint", OID: fbproto.SQL_TYPE_SHORT, Codec: Int4Codec{}})
	defaultMap.RegisterType(&Type{Name: "int", OID: fbproto.SQL_TYPE_LONG, Codec: Int4Codec{}})
	defaultMap.RegisterType(&Type{Name: "bigint", OID: fbproto.SQL_TYPE_INT64, Codec: Int8Codec{}})
	defaultMap.RegisterType(&Type{Name: "text", OID: fbproto.SQL_TYPE_TEXT, Codec: TextCodec{}})
	defaultMap.RegisterType(&Type{Name: "time", OID: fbproto.SQL_TYPE_TIME, Codec: TimeCodec{}})
	defaultMap.RegisterType(&Type{Name: "varchar", OID: fbproto.SQL_TYPE_VARYING, Codec: TextCodec{}})
	defaultMap.RegisterType(&Type{Name: "float", OID: fbproto.SQL_TYPE_FLOAT, Codec: Float4Codec{}})
	defaultMap.RegisterType(&Type{Name: "double precision", OID: fbproto.SQL_TYPE_DOUBLE, Codec: Float8Codec{}})
	defaultMap.RegisterType(&Type{Name: "boolean", OID: fbproto.SQL_TYPE_BOOLEAN, Codec: BoolCodec{}})
	defaultMap.RegisterType(&Type{Name: "timestamp", OID: fbproto.SQL_TYPE_TIMESTAMP, Codec: &TimestampCodec{}})
	defaultMap.RegisterType(&Type{Name: "numeric", OID: MakeTypeSubtype(fbproto.SQL_TYPE_SHORT, 1), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "numeric", OID: MakeTypeSubtype(fbproto.SQL_TYPE_LONG, 1), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "numeric", OID: MakeTypeSubtype(fbproto.SQL_TYPE_INT64, 1), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "octets", OID: MakeTypeSubtype(fbproto.SQL_TYPE_TEXT, 1), Codec: &ByteaCodec{}})
	defaultMap.RegisterType(&Type{Name: "octets", OID: MakeTypeSubtype(fbproto.SQL_TYPE_VARYING, 1), Codec: &ByteaCodec{}})

	// Integer types that directly map to a Firebird type
	registerDefaultFbTypeVariants[int16](defaultMap, "smallint")
	registerDefaultFbTypeVariants[int32](defaultMap, "int")
	registerDefaultFbTypeVariants[int64](defaultMap, "bigint")

	// Integer types that do not have a direct match to a Firebird type
	registerDefaultFbTypeVariants[int8](defaultMap, "bigint")
	registerDefaultFbTypeVariants[int](defaultMap, "bigint")
	registerDefaultFbTypeVariants[uint8](defaultMap, "bigint")
	registerDefaultFbTypeVariants[uint16](defaultMap, "bigint")
	registerDefaultFbTypeVariants[uint32](defaultMap, "bigint")
	registerDefaultFbTypeVariants[uint64](defaultMap, "numeric")
	registerDefaultFbTypeVariants[uint](defaultMap, "numeric")

	registerDefaultFbTypeVariants[float32](defaultMap, "float")
	registerDefaultFbTypeVariants[float64](defaultMap, "double precision")

	registerDefaultFbTypeVariants[bool](defaultMap, "boolean")
	registerDefaultFbTypeVariants[time.Time](defaultMap, "timestamptz")
	registerDefaultFbTypeVariants[string](defaultMap, "text")
	registerDefaultFbTypeVariants[[]byte](defaultMap, "bytea")

	// fbtype provided structs
	registerDefaultFbTypeVariants[Bool](defaultMap, "boolean")
	registerDefaultFbTypeVariants[Date](defaultMap, "date")
	registerDefaultFbTypeVariants[Float4](defaultMap, "float")
	registerDefaultFbTypeVariants[Float8](defaultMap, "double precision")
	registerDefaultFbTypeVariants[Int2](defaultMap, "smallint")
	registerDefaultFbTypeVariants[Int4](defaultMap, "int")
	registerDefaultFbTypeVariants[Int8](defaultMap, "bigint")
	registerDefaultFbTypeVariants[Numeric](defaultMap, "numeric")
	registerDefaultFbTypeVariants[Text](defaultMap, "text")
	registerDefaultFbTypeVariants[Time](defaultMap, "time")
	registerDefaultFbTypeVariants[Timestamp](defaultMap, "timestamp")
	registerDefaultFbTypeVariants[Timestamptz](defaultMap, "timestamptz")

	defaultMap.buildReflectTypeToType()
}
