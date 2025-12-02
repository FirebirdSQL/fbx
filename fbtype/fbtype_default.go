package fbtype

import (
	"reflect"
	"sync"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

var (
	// defaultMap contains default mappings between PostgreSQL server types and Go type handling logic.
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

	// Range types
	//defaultMap.RegisterType(&Type{Name: "daterange", OID: DaterangeOID, Codec: &RangeCodec{ElementType: defaultMap.oidToType[DateOID]}})
	//defaultMap.RegisterType(&Type{Name: "int4range", OID: Int4rangeOID, Codec: &RangeCodec{ElementType: defaultMap.oidToType[Int4OID]}})
	//defaultMap.RegisterType(&Type{Name: "int8range", OID: Int8rangeOID, Codec: &RangeCodec{ElementType: defaultMap.oidToType[Int8OID]}})
	//defaultMap.RegisterType(&Type{Name: "numrange", OID: NumrangeOID, Codec: &RangeCodec{ElementType: defaultMap.oidToType[NumericOID]}})
	//defaultMap.RegisterType(&Type{Name: "tsrange", OID: TsrangeOID, Codec: &RangeCodec{ElementType: defaultMap.oidToType[TimestampOID]}})
	//defaultMap.RegisterType(&Type{Name: "tstzrange", OID: TstzrangeOID, Codec: &RangeCodec{ElementType: defaultMap.oidToType[TimestamptzOID]}})

	// Multirange types
	//defaultMap.RegisterType(&Type{Name: "datemultirange", OID: DatemultirangeOID, Codec: &MultirangeCodec{ElementType: defaultMap.oidToType[DaterangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "int4multirange", OID: Int4multirangeOID, Codec: &MultirangeCodec{ElementType: defaultMap.oidToType[Int4rangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "int8multirange", OID: Int8multirangeOID, Codec: &MultirangeCodec{ElementType: defaultMap.oidToType[Int8rangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "nummultirange", OID: NummultirangeOID, Codec: &MultirangeCodec{ElementType: defaultMap.oidToType[NumrangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "tsmultirange", OID: TsmultirangeOID, Codec: &MultirangeCodec{ElementType: defaultMap.oidToType[TsrangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "tstzmultirange", OID: TstzmultirangeOID, Codec: &MultirangeCodec{ElementType: defaultMap.oidToType[TstzrangeOID]}})

	// Array types
	//defaultMap.RegisterType(&Type{Name: "_aclitem", OID: ACLItemArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[ACLItemOID]}})
	//defaultMap.RegisterType(&Type{Name: "_bit", OID: BitArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[BitOID]}})
	//defaultMap.RegisterType(&Type{Name: "_bool", OID: BoolArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[BoolOID]}})
	//defaultMap.RegisterType(&Type{Name: "_box", OID: BoxArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[BoxOID]}})
	//defaultMap.RegisterType(&Type{Name: "_bpchar", OID: BPCharArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[BPCharOID]}})
	//defaultMap.RegisterType(&Type{Name: "_bytea", OID: ByteaArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[ByteaOID]}})
	//defaultMap.RegisterType(&Type{Name: "_char", OID: QCharArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[QCharOID]}})
	//defaultMap.RegisterType(&Type{Name: "_cid", OID: CIDArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[CIDOID]}})
	//defaultMap.RegisterType(&Type{Name: "_cidr", OID: CIDRArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[CIDROID]}})
	//defaultMap.RegisterType(&Type{Name: "_circle", OID: CircleArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[CircleOID]}})
	//defaultMap.RegisterType(&Type{Name: "_date", OID: DateArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[DateOID]}})
	//defaultMap.RegisterType(&Type{Name: "_daterange", OID: DaterangeArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[DaterangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "_float4", OID: Float4ArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[Float4OID]}})
	//defaultMap.RegisterType(&Type{Name: "_float8", OID: Float8ArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[Float8OID]}})
	//defaultMap.RegisterType(&Type{Name: "_inet", OID: InetArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[InetOID]}})
	//defaultMap.RegisterType(&Type{Name: "_int2", OID: Int2ArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[Int2OID]}})
	//defaultMap.RegisterType(&Type{Name: "_int4", OID: Int4ArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[Int4OID]}})
	//defaultMap.RegisterType(&Type{Name: "_int4range", OID: Int4rangeArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[Int4rangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "_int8", OID: Int8ArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[Int8OID]}})
	//defaultMap.RegisterType(&Type{Name: "_int8range", OID: Int8rangeArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[Int8rangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "_interval", OID: IntervalArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[IntervalOID]}})
	//defaultMap.RegisterType(&Type{Name: "_json", OID: JSONArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[JSONOID]}})
	//defaultMap.RegisterType(&Type{Name: "_jsonb", OID: JSONBArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[JSONBOID]}})
	//defaultMap.RegisterType(&Type{Name: "_jsonpath", OID: JSONPathArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[JSONPathOID]}})
	//defaultMap.RegisterType(&Type{Name: "_line", OID: LineArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[LineOID]}})
	//defaultMap.RegisterType(&Type{Name: "_lseg", OID: LsegArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[LsegOID]}})
	//defaultMap.RegisterType(&Type{Name: "_macaddr", OID: MacaddrArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[MacaddrOID]}})
	//defaultMap.RegisterType(&Type{Name: "_name", OID: NameArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[NameOID]}})
	//defaultMap.RegisterType(&Type{Name: "_numeric", OID: NumericArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[NumericOID]}})
	//defaultMap.RegisterType(&Type{Name: "_numrange", OID: NumrangeArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[NumrangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "_oid", OID: OIDArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[OIDOID]}})
	//defaultMap.RegisterType(&Type{Name: "_path", OID: PathArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[PathOID]}})
	//defaultMap.RegisterType(&Type{Name: "_point", OID: PointArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[PointOID]}})
	//defaultMap.RegisterType(&Type{Name: "_polygon", OID: PolygonArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[PolygonOID]}})
	//defaultMap.RegisterType(&Type{Name: "_record", OID: RecordArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[RecordOID]}})
	//defaultMap.RegisterType(&Type{Name: "_text", OID: TextArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[TextOID]}})
	//defaultMap.RegisterType(&Type{Name: "_tid", OID: TIDArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[TIDOID]}})
	//defaultMap.RegisterType(&Type{Name: "_time", OID: TimeArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[TimeOID]}})
	//defaultMap.RegisterType(&Type{Name: "_timestamp", OID: TimestampArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[TimestampOID]}})
	//defaultMap.RegisterType(&Type{Name: "_timestamptz", OID: TimestamptzArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[TimestamptzOID]}})
	//defaultMap.RegisterType(&Type{Name: "_tsrange", OID: TsrangeArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[TsrangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "_tstzrange", OID: TstzrangeArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[TstzrangeOID]}})
	//defaultMap.RegisterType(&Type{Name: "_uuid", OID: UUIDArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[UUIDOID]}})
	//defaultMap.RegisterType(&Type{Name: "_varbit", OID: VarbitArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[VarbitOID]}})
	//defaultMap.RegisterType(&Type{Name: "_varchar", OID: VarcharArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[VarcharOID]}})
	//defaultMap.RegisterType(&Type{Name: "_xid", OID: XIDArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[XIDOID]}})
	//defaultMap.RegisterType(&Type{Name: "_xid8", OID: XID8ArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[XID8OID]}})
	//defaultMap.RegisterType(&Type{Name: "_xml", OID: XMLArrayOID, Codec: &ArrayCodec{ElementType: defaultMap.oidToType[XMLOID]}})

	// Integer types that directly map to a PostgreSQL type
	registerDefaultPgTypeVariants[int16](defaultMap, "smallint")
	registerDefaultPgTypeVariants[int32](defaultMap, "int")
	registerDefaultPgTypeVariants[int64](defaultMap, "bigint")

	// Integer types that do not have a direct match to a PostgreSQL type
	registerDefaultPgTypeVariants[int8](defaultMap, "bigint")
	registerDefaultPgTypeVariants[int](defaultMap, "bigint")
	registerDefaultPgTypeVariants[uint8](defaultMap, "bigint")
	registerDefaultPgTypeVariants[uint16](defaultMap, "bigint")
	registerDefaultPgTypeVariants[uint32](defaultMap, "bigint")
	registerDefaultPgTypeVariants[uint64](defaultMap, "numeric")
	registerDefaultPgTypeVariants[uint](defaultMap, "numeric")

	registerDefaultPgTypeVariants[float32](defaultMap, "float")
	registerDefaultPgTypeVariants[float64](defaultMap, "double precision")

	registerDefaultPgTypeVariants[bool](defaultMap, "boolean")
	registerDefaultPgTypeVariants[time.Time](defaultMap, "timestamptz")
	registerDefaultPgTypeVariants[string](defaultMap, "text")
	registerDefaultPgTypeVariants[[]byte](defaultMap, "bytea")

	// pgtype provided structs
	registerDefaultPgTypeVariants[Bool](defaultMap, "boolean")
	registerDefaultPgTypeVariants[Date](defaultMap, "date")
	// registerDefaultPgTypeVariants[Range[Date]](defaultMap, "daterange")
	// registerDefaultPgTypeVariants[Multirange[Range[Date]]](defaultMap, "datemultirange")
	registerDefaultPgTypeVariants[Float4](defaultMap, "float")
	registerDefaultPgTypeVariants[Float8](defaultMap, "double precision")
	// registerDefaultPgTypeVariants[Range[Float8]](defaultMap, "numrange")                  // There is no PostgreSQL builtin float8range so map it to numrange.
	// registerDefaultPgTypeVariants[Multirange[Range[Float8]]](defaultMap, "nummultirange") // There is no PostgreSQL builtin float8multirange so map it to nummultirange.
	registerDefaultPgTypeVariants[Int2](defaultMap, "smallint")
	registerDefaultPgTypeVariants[Int4](defaultMap, "int")
	// registerDefaultPgTypeVariants[Range[Int4]](defaultMap, "int4range")
	// registerDefaultPgTypeVariants[Multirange[Range[Int4]]](defaultMap, "int4multirange")
	registerDefaultPgTypeVariants[Int8](defaultMap, "bigint")
	// registerDefaultPgTypeVariants[Range[Int8]](defaultMap, "int8range")
	// registerDefaultPgTypeVariants[Multirange[Range[Int8]]](defaultMap, "int8multirange")
	registerDefaultPgTypeVariants[Numeric](defaultMap, "numeric")
	// registerDefaultPgTypeVariants[Range[Numeric]](defaultMap, "numrange")
	// registerDefaultPgTypeVariants[Multirange[Range[Numeric]]](defaultMap, "nummultirange")
	registerDefaultPgTypeVariants[Text](defaultMap, "text")
	registerDefaultPgTypeVariants[Time](defaultMap, "time")
	registerDefaultPgTypeVariants[Timestamp](defaultMap, "timestamp")
	registerDefaultPgTypeVariants[Timestamptz](defaultMap, "timestamptz")
	// registerDefaultPgTypeVariants[Range[Timestamp]](defaultMap, "tsrange")
	// registerDefaultPgTypeVariants[Multirange[Range[Timestamp]]](defaultMap, "tsmultirange")
	// registerDefaultPgTypeVariants[Range[Timestamptz]](defaultMap, "tstzrange")
	// registerDefaultPgTypeVariants[Multirange[Range[Timestamptz]]](defaultMap, "tstzmultirange")

	defaultMap.buildReflectTypeToType()
}
