package fbtype

import (
	"context"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"time"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

type ArrayGetter interface {

	// Index returns the element at i.
	Index(i int) any

	// IndexType returns a non-nil scan target of the type Index will return. This is used by ArrayCodec.PlanEncode.
	IndexType() any

	Bounds() []Bound
}

type Bound struct {
	Lower int
	Upper int
}

type ArraySetter interface {
	// SetBounds prepares the value such that ScanIndex can be called for each element. This will remove any existing
	// elements. bounds may be nil to indicate a NULL array. If unable to exactly preserve bounds SetBounds
	// may return an error or silently flatten the array bounds.
	SetBounds(bounds []Bound) error

	// ScanIndex returns a value usable as a scan target for i. SetBounds must be called before ScanIndex.
	ScanIndex(i int) any

	// ScanIndexType returns a non-nil scan target of the type ScanIndex will return. This is used by
	// ArrayCodec.PlanScan.
	ScanIndexType() any
}

// ArrayCodec is a codec for any array type.
type ArrayCodec struct {
	ElementType *Type
	TableName   string
	FieldName   string
	Bounds      []Bound
	SubType     int32
	Length      uint32
	Scale       int32
	TypeId      uint32
	Metadata    map[string][]byte
	FbConn      *fbconn.FbConn
	Ctx         context.Context
}

func (c *ArrayCodec) FormatSupported(format int16) bool {
	return c.ElementType.Codec.FormatSupported(format)
}

func (c *ArrayCodec) PreferredFormat() int16 {

	return BinaryFormatCode
}

func (c *ArrayCodec) PlanEncode(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, value any) EncodePlan {

	t, err := getTypeFromBlr(c.TypeId)
	if err != nil {
		return nil
	}

	c.ElementType = t

	return &encodePlanArrayCodecBinary{arrayCodec: c, m: m}
}

type encodePlanArrayCodecBinary struct {
	arrayCodec *ArrayCodec
	m          *Map
}

func (p *encodePlanArrayCodecBinary) Encode(value any, buf []byte) ([]byte, error) {

	ac := p.arrayCodec

	array, err := getArrayEncodeType(value)
	if err != nil {
		return nil, err
	}
	bounds := array.Bounds()
	if bounds == nil {
		return nil, nil
	}

	argumentElementCount := allElementsCount(bounds)
	fieldElementCount := allElementsCount(ac.Bounds)

	if fieldElementCount < argumentElementCount {
		return nil, fmt.Errorf("Array to long")
	}

	ac.Bounds = bounds

	var encodePlan EncodePlan
	var lastElemType reflect.Type
	var elem any
	var tmpBuf []byte
	typeId := ac.ElementType.TypeID
	lenElem := ac.Length

	for i := 0; i < argumentElementCount; i++ {
		elem = array.Index(i)
		elemType := reflect.TypeOf(elem)
		if lastElemType != elemType {
			lastElemType = elemType
			encodePlan = p.m.PlanEncode(typeId, ac.SubType, lenElem, ac.Scale, BinaryFormatCode, elem)
			if encodePlan == nil {
				return nil, fmt.Errorf("unable to encode %v", array.Index(i))
			}
		}

		switch typeId {
		case fbproto.SQL_TYPE_VARYING:
			tmpBuf, err = ac.EncodeVaryingType(array, encodePlan, lastElemType, elem)
		case fbproto.SQL_TYPE_TEXT:
			tmpBuf, err = ac.EncodeTextType(array, encodePlan, lastElemType, elem)
		default:
			tmpBuf, err = ac.EncodeStaticType(array, encodePlan, lastElemType, elem)
		}

		if err != nil {
			return nil, err
		}
		buf = append(buf, tmpBuf...)
	}

	return buf, nil
}

func (c *ArrayCodec) EncodeStaticType(array ArrayGetter, encodePlan EncodePlan, lastElemType reflect.Type, elem any) ([]byte, error) {
	var buf []byte
	var err error
	if elem != nil {
		buf, err = encodePlan.Encode(elem, buf)
		if err != nil {
			return nil, err
		}
	} else {
		mz := make([]byte, c.Length)
		buf = append(buf, mz...)
	}
	return buf, nil
}

func (c *ArrayCodec) EncodeTextType(array ArrayGetter, encodePlan EncodePlan, lastElemType reflect.Type, elem any) ([]byte, error) {
	var buf []byte
	var err error
	if elem != nil {
		buf, err = encodePlan.Encode(elem, buf)
		if err != nil {
			return nil, err
		}
		l := uint32(len(buf))
		padding := make([]byte, c.Length-l)
		for i := range padding {
			padding[i] = 32
		}
		buf = append(buf, padding...)
		if c.Length%4 != 0 {
			padding = make([]byte, 4-c.Length&3)
			buf = append(buf, padding...)
		}
	} else {
		mz := make([]byte, c.Length)
		buf = append(buf, mz...)
	}
	return buf, nil
}

func (c *ArrayCodec) EncodeVaryingType(array ArrayGetter, encodePlan EncodePlan, lastElemType reflect.Type, elem any) ([]byte, error) {
	var buf []byte
	if elem != nil {
		str, err := encodePlan.Encode(elem, buf)
		if err != nil {
			return nil, err
		}
		lenElem := uint32(len(str))
		bufLen := make([]byte, 4)
		binary.BigEndian.PutUint32(bufLen, lenElem)
		buf = append(buf, bufLen...)
		buf = append(buf, str...)

		padding := make([]byte, 4-lenElem&3)
		buf = append(buf, padding...)
		lenElem = 0
	} else {
		bufLen := make([]byte, 4)
		binary.BigEndian.PutUint32(bufLen, 0)
		buf = append(buf, bufLen...)
	}
	return buf, nil
}

func (c *ArrayCodec) PlanScan(m *Map, typeId uint32, subType int32, length uint32, scale int32, format int16, target any) ScanPlan {

	return &scanPlanArrayCodec{
		arrayCodec: c,
		m:          m,
		formatCode: format,
	}
}

func allElementsCount(bounds []Bound) int {
	if len(bounds) == 0 {
		return 0
	}

	elementCount := int(bounds[0].Upper - bounds[0].Lower + 1)
	for _, d := range bounds[1:] {
		elementCount *= int(d.Upper - d.Lower + 1)
	}

	return elementCount
}

func (c *ArrayCodec) decodeBinary(m *Map, arrayOID uint32, src []byte, length int, array ArraySetter) error {

	elementCount := allElementsCount(c.Bounds)
	if elementCount == 0 {
		return nil
	}

	elementScanPlan := c.ElementType.Codec.PlanScan(m, c.ElementType.TypeID, c.SubType, c.Length, c.Scale, BinaryFormatCode, array.ScanIndex(0))
	if elementScanPlan == nil {
		elementScanPlan = m.PlanScan(c.ElementType.TypeID, c.SubType, c.Length, c.Scale, BinaryFormatCode, array.ScanIndex(0))
	}
	if arrayOID == fbproto.SQL_TYPE_SHORT {
		length = 4
	}
	rp := 0
	var elemLen int
	if arrayOID == fbproto.SQL_TYPE_TEXT && length%4 != 0 {
		elemLen = length + (4 - length&3)
	} else {
		elemLen = length
	}
	for i := 0; i < elementCount && rp < len(src); i++ {
		elem := array.ScanIndex(i)
		if arrayOID == fbproto.SQL_TYPE_VARYING {
			elemLen = int(int32(binary.BigEndian.Uint32(src[rp:])))
			rp += 4
		}
		var elemSrc []byte
		if elemLen >= 0 {
			elemSrc = src[rp : rp+elemLen]
			rp += elemLen
		}
		err := elementScanPlan.Scan(elemSrc, elem)
		if err != nil {
			return fmt.Errorf("failed to scan array element %d: %w", i, err)
		}
	}

	return nil
}

type scanPlanArrayCodec struct {
	arrayCodec *ArrayCodec
	m          *Map
	formatCode int16
}

func (spac *scanPlanArrayCodec) Scan(src []byte, dst any) error {
	ac := spac.arrayCodec
	fbConn := ac.FbConn

	array, err := getArrayDecodeType(dst)
	if err != nil {
		return err
	}

	if len(src) == 0 {
		array.SetBounds(nil)
		return nil
	}

	arrayId := int64(binary.BigEndian.Uint64(src))

	err = ac.getArrayInfo()
	if err != nil {
		return err
	}

	sdl := ac.generateSDL()

	length := ac.getSliceLength()

	arr, err := fbConn.GetArray(ac.Ctx, arrayId, ac.TypeId, length, int(ac.Length), sdl)
	if err != nil {
		return err
	}

	err = array.SetBounds(ac.Bounds)
	if err != nil {
		return err
	}
	if len(arr) == 0 {
		return nil
	}

	dt, err := getSqlTypeFromBlr(ac.TypeId)
	if err != nil {
		return err
	}

	ac.ElementType, err = getTypeFromBlr(ac.TypeId)
	if err != nil {
		return err
	}

	err = ac.decodeBinary(spac.m, dt, arr, int(ac.Length), array)

	return err
}

func (ac *ArrayCodec) getSliceLength() int {
	var elements int = 1
	for _, elem := range ac.Bounds {
		elements *= elem.Upper - elem.Lower + 1
	}

	length := elements * int(ac.Length)

	switch ac.TypeId {
	case fbproto.Isc_blr_varaying, fbproto.Isc_blr_varaying2:
		length += elements * 2
	}
	return length
}

func (ac *ArrayCodec) generateSDL() []byte {
	sdl := []byte{}
	metadata := ac.Metadata
	bounds := ac.Bounds

	data_type := trimBytes(metadata["DATATYPE"], 1)[0]
	scale := trimBytes(metadata["SCALE"], 1)
	length := trimBytes(metadata["LENGTH"], 2)

	if ac.TypeId == fbproto.Isc_blr_varaying || ac.TypeId == fbproto.Isc_blr_varaying2 ||
		ac.TypeId == fbproto.Isc_blr_text || ac.TypeId == fbproto.Isc_blr_text2 {
		intLength := binary.LittleEndian.Uint16(length) * uint16(ac.FbConn.Encoding.MaxBytesPerChar)
		binary.LittleEndian.PutUint16(length, intLength)
	}

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_version1,
		fbproto.Isc_sdl_struct,
		1,
		data_type,
	}...)

	switch data_type {
	case fbproto.Isc_blr_short, fbproto.Isc_blr_long, fbproto.Isc_blr_int64, fbproto.Isc_blr_quad:
		sdl = append(sdl, scale...)
	case fbproto.Isc_blr_text, fbproto.Isc_blr_cstring, fbproto.Isc_blr_varaying:
		sdl = append(sdl, length...)
	}

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_relation,
		byte(len(metadata["TABLE"])),
	}...)
	sdl = append(sdl, metadata["TABLE"]...)

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_field,
		byte(len(metadata["FIELD"])),
	}...)
	sdl = append(sdl, metadata["FIELD"]...)

	for i, b := range bounds {
		if b.Lower == 1 {
			sdl = append(sdl, []byte{
				fbproto.Isc_sdl_do1,
				byte(i),
			}...)
		} else {
			sdl = append(sdl, []byte{
				fbproto.Isc_sdl_do2,
				byte(i),
			}...)
			sdl = append(sdl, getLiteral(b.Lower)...)
		}
		sdl = append(sdl, getLiteral(b.Upper)...)
	}

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_element,
		1,
		fbproto.Isc_sdl_scalar,
		0,
		byte(len(bounds)),
	}...)

	for i := 0; i < len(bounds); i++ {
		sdl = append(sdl, []byte{
			fbproto.Isc_sdl_variable,
			byte(i),
		}...)
	}

	sdl = append(sdl, fbproto.Isc_sdl_eoc)

	return sdl
}

func trimBytes(in []byte, l int) []byte {
	in = in[len(in)-l:]
	out := []byte{}

	for i := len(in) - 1; i >= 0; i-- {
		out = append(out, in[i])
	}
	return out
}

func getLiteral(value int) []byte {
	out := []byte{}
	valByte := make([]byte, 4)
	binary.BigEndian.PutUint32(valByte, uint32(value))

	if -128 <= value && value <= 127 {
		out = append(out, fbproto.Isc_sdl_tiny_integer)
		out = append(out, trimBytes(valByte, 1)...)
		return out
	}

	if -32768 <= value && value <= 32767 {
		out = append(out, fbproto.Isc_sdl_short_integer)
		out = append(out, trimBytes(valByte, 2)...)
		return out
	}

	out = append(out, fbproto.Isc_sdl_long_integer)
	out = append(out, trimBytes(valByte, 4)...)

	return out
}

func getArrayDecodeType(dst any) (ArraySetter, error) {

	switch target := dst.(type) {
	case *[]int16:
		return (*FlatArray[int16])(target), nil
	case *[]int32:
		return (*FlatArray[int32])(target), nil
	case *[]int64:
		return (*FlatArray[int64])(target), nil
	case *[]float32:
		return (*FlatArray[float32])(target), nil
	case *[]float64:
		return (*FlatArray[float64])(target), nil
	case *[]string:
		return (*FlatArray[string])(target), nil
	case *[]time.Time:
		return (*FlatArray[time.Time])(target), nil
	}

	targetType := reflect.TypeOf(dst)
	if targetType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("unsupported argument type")
	}

	targetElemType := targetType.Elem()

	targetValue := reflect.ValueOf(dst)

	if targetElemType.Kind() == reflect.Slice {
		innerElement := targetElemType.Elem().Kind()
		switch innerElement {
		case reflect.Slice:
			if !isRagged(targetValue.Elem()) {
				return &anyMultiDimSliceArray{slice: targetValue.Elem()}, nil
			}
		default:
			return &anySliceArrayReflect{slice: targetValue.Elem()}, nil
		}
	}

	if targetElemType.Kind() == reflect.Array {
		return &anyArrayArrayReflect{array: targetValue.Elem()}, nil
	}

	a, ok := dst.(ArraySetter)
	if !ok {
		return nil, fmt.Errorf("unsupported argument type")
	}

	return a, nil

}

func getArrayEncodeType(dst any) (ArrayGetter, error) {

	switch target := dst.(type) {
	case []int16:
		return (FlatArray[int16])(target), nil
	case []int32:
		return (FlatArray[int32])(target), nil
	case []int64:
		return (FlatArray[int64])(target), nil
	case []float32:
		return (FlatArray[float32])(target), nil
	case []float64:
		return (FlatArray[float64])(target), nil
	case []string:
		return (FlatArray[string])(target), nil
	case []time.Time:
		return (FlatArray[time.Time])(target), nil
	}

	targetType := reflect.TypeOf(dst)

	targetElemType := targetType

	targetValue := reflect.ValueOf(dst)

	if targetElemType.Kind() == reflect.Slice {
		innerElement := targetElemType.Elem().Kind()
		switch innerElement {
		case reflect.Slice:
			if !isRagged(targetValue) {
				return &anyMultiDimSliceArray{slice: targetValue}, nil
			}
			return nil, fmt.Errorf("cannot convert to ArrayGetter because it is a ragged multi-dimensional")
		default:
			return &anySliceArrayReflect{slice: targetValue}, nil
		}
	}

	if targetElemType.Kind() == reflect.Array {
		return &anyArrayArrayReflect{array: targetValue}, nil
	}

	a, ok := dst.(ArrayGetter)

	if targetType.Kind() != reflect.Slice && !ok {
		return nil, fmt.Errorf("unsupported argument type")
	}

	return a, nil

}

func getTypeFromBlr(datatype uint32) (*Type, error) {
	oid, err := getSqlTypeFromBlr(datatype)
	if err != nil {
		return nil, err
	}
	return defaultMap.typeidToType[oid], nil
}

func getSqlTypeFromBlr(datatype uint32) (uint32, error) {
	switch datatype {
	case fbproto.Isc_blr_varaying,
		fbproto.Isc_blr_varaying2:
		return fbproto.SQL_TYPE_VARYING, nil

	case fbproto.Isc_blr_text,
		fbproto.Isc_blr_text2,
		fbproto.Isc_blr_cstring,
		fbproto.Isc_blr_cstring2:
		return fbproto.SQL_TYPE_TEXT, nil

	case fbproto.Isc_blr_short:
		return fbproto.SQL_TYPE_SHORT, nil

	case fbproto.Isc_blr_long:
		return fbproto.SQL_TYPE_LONG, nil

	case fbproto.Isc_blr_quad:
		return fbproto.SQL_TYPE_QUAD, nil

	case fbproto.Isc_blr_int64:
		return fbproto.SQL_TYPE_INT64, nil

	case fbproto.Isc_blr_double:
		return fbproto.SQL_TYPE_DOUBLE, nil

	case fbproto.Isc_blr_float:
		return fbproto.SQL_TYPE_FLOAT, nil

	case fbproto.Isc_blr_d_float:
		return fbproto.SQL_TYPE_D_FLOAT, nil

	case fbproto.Isc_blr_sql_date:
		return fbproto.SQL_TYPE_DATE, nil

	case fbproto.Isc_blr_sql_time:
		return fbproto.SQL_TYPE_TIME, nil

	case fbproto.Isc_blr_sql_time_tz:
		return fbproto.SQL_TYPE_TIME_TZ, nil

	case fbproto.Isc_blr_timestamp:
		return fbproto.SQL_TYPE_TIMESTAMP, nil

	case fbproto.Isc_blr_blob:
		return fbproto.SQL_TYPE_BLOB, nil

	case fbproto.Isc_blr_bool:
		return fbproto.SQL_TYPE_BOOLEAN, nil

	case fbproto.Isc_blr_timestamp_tz:
		return fbproto.SQL_TYPE_TIMESTAMP_TZ, nil

	case fbproto.Isc_blr_dec64:
		return fbproto.SQL_TYPE_DEC64, nil

	case fbproto.Isc_blr_dec128:
		return fbproto.SQL_TYPE_DEC128, nil

	case fbproto.Isc_blr_int128:
		return fbproto.SQL_TYPE_INT128, nil

	default:
		return 0, errors.New("unknown blr type")
	}

}

func GetArrayId(fbConn *fbconn.FbConn, ctx context.Context, m *Map, table, field string, arg any) ([]byte, error) {

	ac := new(ArrayCodec)

	ac.FieldName = field
	ac.TableName = table
	ac.FbConn = fbConn
	ac.Ctx = ctx

	err := ac.getArrayInfo()
	if err != nil {
		return nil, err
	}

	ep := ac.PlanEncode(m, ac.TypeId, 0, ac.Length, 0, BinaryFormatCode, arg)
	buf := []byte{}
	arr, err := ep.Encode(arg, buf)
	if err != nil {
		return nil, err
	}
	if arr == nil {
		return make([]byte, 8), nil
	}

	sdl := ac.generateSDL()
	length := ac.getSliceLength()

	id, err := ac.FbConn.PutArray(ac.Ctx, arr, ac.TypeId, length, sdl)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func (ac *ArrayCodec) getArrayInfo() error {
	metadata, err := ac.FbConn.GetArrayMetadata(ac.Ctx, ac.TableName, ac.FieldName)
	if err != nil {
		return err
	}

	ac.Metadata = metadata
	ac.Scale = int32(binary.BigEndian.Uint32(metadata["SCALE"]))
	ac.TypeId = binary.BigEndian.Uint32(metadata["DATATYPE"])
	ac.SubType = int32(binary.BigEndian.Uint32(metadata["SUBTYPE"]))
	ac.Length = binary.BigEndian.Uint32(metadata["LENGTH"])
	if ac.TypeId == fbproto.Isc_blr_varaying || ac.TypeId == fbproto.Isc_blr_varaying2 ||
		ac.TypeId == fbproto.Isc_blr_text || ac.TypeId == fbproto.Isc_blr_text2 {
		ac.Length *= ac.FbConn.Encoding.MaxBytesPerChar
	}

	bounds := []Bound{}
	boundsMap, err := ac.FbConn.GetArrayBounds(ac.Ctx, metadata["SOURCE"])
	if err != nil {
		return err
	}
	for _, elem := range boundsMap {
		bounds = append(bounds, Bound{Lower: elem["Lower"], Upper: elem["Upper"]})
	}

	ac.Bounds = bounds

	return nil
}

func (sp *scanPlanArrayCodec) SetInfo(conn *fbconn.FbConn, ctx context.Context, tableName, fieldName string) {
	sp.arrayCodec.FbConn = conn
	sp.arrayCodec.Ctx = ctx
	sp.arrayCodec.TableName = tableName
	sp.arrayCodec.FieldName = fieldName
}

func (c *ArrayCodec) DecodeDatabaseSQLValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	switch format {
	case TextFormatCode:
		return string(src), nil
	case BinaryFormatCode:
		buf := make([]byte, len(src))
		copy(buf, src)
		return buf, nil
	default:
		return nil, fmt.Errorf("unknown format code %d", format)
	}
}

func (c *ArrayCodec) DecodeValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var slice []any
	err := m.PlanScan(typeid, subType, length, scale, format, &slice).Scan(src, &slice)
	return slice, err
}

func isRagged(slice reflect.Value) bool {
	if slice.Type().Elem().Kind() != reflect.Slice {
		return false
	}

	sliceLen := slice.Len()
	innerLen := 0
	for i := 0; i < sliceLen; i++ {
		if i == 0 {
			innerLen = slice.Index(i).Len()
		} else {
			if slice.Index(i).Len() != innerLen {
				return true
			}
		}
		if isRagged(slice.Index(i)) {
			return true
		}
	}

	return false
}

type Array[T any] struct {
	Elements []T
	Dims     []Bound
	Valid    bool
}

func (a Array[T]) Dimensions() []Bound {
	return a.Dims
}

func (a Array[T]) Index(i int) any {
	return a.Elements[i]
}

func (a Array[T]) IndexType() any {
	var el T
	return el
}

func (a Array[T]) Bounds() []Bound {
	return a.Dims
}

func (a *Array[T]) SetBounds(bounds []Bound) error {
	if bounds == nil {
		*a = Array[T]{}
		return nil
	}

	elementCount := allElementsCount(bounds)
	*a = Array[T]{
		Elements: make([]T, elementCount),
		Dims:     bounds,
		Valid:    true,
	}

	return nil
}

func (a Array[T]) ScanIndex(i int) any {
	return &a.Elements[i]
}

func (a Array[T]) ScanIndexType() any {
	return new(T)
}

type FlatArray[T any] []T

func (a FlatArray[T]) Index(i int) any {
	return a[i]
}

func (a FlatArray[T]) IndexType() any {
	var el T
	return el
}

func (a FlatArray[T]) Bounds() []Bound {
	if a == nil {
		return nil
	}

	return []Bound{{Lower: 1, Upper: len(a)}}
}

func (a *FlatArray[T]) SetBounds(bounds []Bound) error {
	if bounds == nil {
		*a = nil
		return nil
	}

	elementCount := allElementsCount(bounds)
	*a = make(FlatArray[T], elementCount)
	return nil
}

func (a FlatArray[T]) ScanIndex(i int) any {
	return &a[i]
}

func (a FlatArray[T]) ScanIndexType() any {
	return new(T)
}
