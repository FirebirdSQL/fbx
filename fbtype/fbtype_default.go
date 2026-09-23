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

// MakeTypeSubtype packs a Firebird TypeID and subtype into a single key used
// for type registration lookups.
func MakeTypeSubtype(typeID uint32, subType int32) uint32 {
	return uint32(int32(typeID) | (subType << 16))
}

// GetType extracts the Firebird TypeID from a key created by MakeTypeSubtype.
func GetType(typeID uint32) uint32 {
	return uint32(int32(typeID) & 0xFFFF)
}

// GetSubType extracts the subtype from a key created by MakeTypeSubtype.
func GetSubType(typeID uint32) int32 {
	return int32(typeID) >> 16
}

func initDefaultMap() {
	defaultMap = &Map{
		typeidToType:      make(map[uint32]*Type),
		nameToType:        make(map[string]*Type),
		reflectTypeToName: make(map[reflect.Type]string),

		memoizedEncodePlans: make(map[TypeMetadata]map[reflect.Type]EncodePlan),

		TryWrapEncodePlanFuncs: defaultTryWrapEncodePlanFuncs(),
		TryWrapScanPlanFuncs:   defaultTryWrapScanPlanFuncs(),
	}

	// Base types
	defaultMap.RegisterType(&Type{Name: "date", TypeID: fbproto.SQL_TYPE_DATE, Codec: DateCodec{}})
	defaultMap.RegisterType(&Type{Name: "smallint", TypeID: fbproto.SQL_TYPE_SHORT, Codec: Int4Codec{}})
	defaultMap.RegisterType(&Type{Name: "int", TypeID: fbproto.SQL_TYPE_LONG, Codec: Int4Codec{}})
	defaultMap.RegisterType(&Type{Name: "bigint", TypeID: fbproto.SQL_TYPE_INT64, Codec: Int8Codec{}})
	defaultMap.RegisterType(&Type{Name: "int128", TypeID: fbproto.SQL_TYPE_INT128, Codec: Int128Codec{}})
	defaultMap.RegisterType(&Type{Name: "char", TypeID: fbproto.SQL_TYPE_TEXT, Codec: TextCodec{}})
	defaultMap.RegisterType(&Type{Name: "time", TypeID: fbproto.SQL_TYPE_TIME, Codec: TimeCodec{}})
	defaultMap.RegisterType(&Type{Name: "timetz", TypeID: fbproto.SQL_TYPE_TIME_TZ, Codec: &TimetzCodec{}})
	defaultMap.RegisterType(&Type{Name: "varchar", TypeID: fbproto.SQL_TYPE_VARYING, Codec: TextCodec{}})
	defaultMap.RegisterType(&Type{Name: "float", TypeID: fbproto.SQL_TYPE_FLOAT, Codec: Float4Codec{}})
	defaultMap.RegisterType(&Type{Name: "double precision", TypeID: fbproto.SQL_TYPE_DOUBLE, Codec: Float8Codec{}})
	defaultMap.RegisterType(&Type{Name: "boolean", TypeID: fbproto.SQL_TYPE_BOOLEAN, Codec: BoolCodec{}})
	defaultMap.RegisterType(&Type{Name: "timestamp", TypeID: fbproto.SQL_TYPE_TIMESTAMP, Codec: &TimestampCodec{}})
	defaultMap.RegisterType(&Type{Name: "timestamptz", TypeID: fbproto.SQL_TYPE_TIMESTAMP_TZ, Codec: &TimestamptzCodec{}})
	defaultMap.RegisterType(&Type{Name: "numeric", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_SHORT, 1), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "numeric", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_LONG, 1), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "numeric", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_INT64, 1), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "numeric", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_INT128, 1), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "decimal", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_SHORT, 2), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "decimal", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_LONG, 2), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "decimal", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_INT64, 2), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "decimal", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_INT128, 2), Codec: &NumericCodec{}})
	defaultMap.RegisterType(&Type{Name: "octets", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_TEXT, 1), Codec: &OctetsCodec{}})
	defaultMap.RegisterType(&Type{Name: "octets", TypeID: MakeTypeSubtype(fbproto.SQL_TYPE_VARYING, 1), Codec: &OctetsCodec{}})
	defaultMap.RegisterType(&Type{Name: "decfloat16", TypeID: fbproto.SQL_TYPE_DEC64, Codec: DecFloat16Codec{}})
	defaultMap.RegisterType(&Type{Name: "decfloat34", TypeID: fbproto.SQL_TYPE_DEC128, Codec: DecFloat34Codec{}})
	defaultMap.RegisterType(&Type{Name: "null", TypeID: fbproto.SQL_TYPE_NULL, Codec: NullCodec{}})
	defaultMap.RegisterType(&Type{Name: "array", TypeID: fbproto.SQL_TYPE_ARRAY, Codec: ArrayCodec{}})

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
	registerDefaultFbTypeVariants[string](defaultMap, "varchar")
	registerDefaultFbTypeVariants[[]byte](defaultMap, "octets")

	// fbtype provided structs
	registerDefaultFbTypeVariants[Bool](defaultMap, "boolean")
	registerDefaultFbTypeVariants[Date](defaultMap, "date")
	registerDefaultFbTypeVariants[Float4](defaultMap, "float")
	registerDefaultFbTypeVariants[Float8](defaultMap, "double precision")
	registerDefaultFbTypeVariants[Int2](defaultMap, "smallint")
	registerDefaultFbTypeVariants[Int4](defaultMap, "int")
	registerDefaultFbTypeVariants[Int8](defaultMap, "bigint")
	registerDefaultFbTypeVariants[Int128](defaultMap, "int128")
	registerDefaultFbTypeVariants[Numeric](defaultMap, "numeric")
	registerDefaultFbTypeVariants[Text](defaultMap, "varchar")
	defaultMap.RegisterDefaultFbType(UUID{}, "octets")
	defaultMap.RegisterDefaultFbType(&UUID{}, "octets")
	registerDefaultFbTypeVariants[Time](defaultMap, "time")
	registerDefaultFbTypeVariants[Timetz](defaultMap, "timetz")
	registerDefaultFbTypeVariants[Timestamp](defaultMap, "timestamp")
	registerDefaultFbTypeVariants[Timestamptz](defaultMap, "timestamptz")
	registerDefaultFbTypeVariants[DecFloat16](defaultMap, "decfloat16")
	registerDefaultFbTypeVariants[DecFloat34](defaultMap, "decfloat34")

	defaultMap.buildReflectTypeToType()
}
