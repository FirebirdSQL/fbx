package fbtype

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

// ArraySetter is the interface for sending the array to the server
type ArraySetter interface {

	// Index returns the element at i.
	Index(i int) any

	// Dimensions returns the []Dimensions of the array
	Dimensions() []Dimention
}

// Dimention is a struct for storage array dimensions
type Dimention struct {
	Lower int
	Upper int
}

// ArrayGetter is the interface for getting the array from the server
type ArrayGetter interface {
	// SetDimentions prepares the value such that ScanIndex can be called for each element. This will remove any existing
	// elements. dimentions may be nil to indicate a NULL array. If unable to exactly preserve dimentions SetDimentions
	// may return an error or silently flatten the array dimentions.
	SetDimentions(dimentions []Dimention) error

	// ScanIndex returns a value usable as a scan target for i. SetDimentions must be called before ScanIndex.
	ScanIndex(i int) any
}

// ArrayData is a atruct for any array type.
type ArrayData struct {
	ElementType *Type
	Dimensions  []Dimention
	SubType     int32
	Length      uint32
	Scale       int32
	TypeId      uint32
}

// EncodeArray gets an array of any type as input and encodes it in binary format
func EncodeArray(m *Map, typeId uint32, subType int32, scale int32, length uint32, dimentionsMap []map[string]int, value any, buf []byte) ([]byte, []map[string]int, error) {

	ac := new(ArrayData)
	var err error

	ac.Length = length
	ac.Scale = scale
	ac.SubType = subType
	ac.TypeId = typeId

	b := []Dimention{}
	for _, elem := range dimentionsMap {
		b = append(b, Dimention{Lower: elem["Lower"], Upper: elem["Upper"]})
	}
	ac.Dimensions = b

	ac.ElementType, err = getTypeFromBlr(typeId)
	if err != nil {
		return nil, nil, err
	}

	array, err := getArrayEncodeType(value)
	if err != nil {
		return nil, nil, err
	}
	dimentions := array.Dimensions()
	if dimentions == nil {
		return nil, nil, nil
	}

	argumentElementCount := allElementsCount(dimentions)
	fieldElementCount := allElementsCount(ac.Dimensions)

	if fieldElementCount < argumentElementCount {
		return nil, nil, fmt.Errorf("Array to long")
	}

	ac.Dimensions = dimentions

	var encodePlan EncodePlan
	var lastElemType reflect.Type
	var elem any
	var tmpBuf []byte
	elementTypeId := ac.ElementType.TypeID
	lenElem := ac.Length

	for i := 0; i < argumentElementCount; i++ {
		elem = array.Index(i)
		elemType := reflect.TypeOf(elem)
		if lastElemType != elemType {
			lastElemType = elemType
			encodePlan = m.PlanEncode(elementTypeId, ac.SubType, lenElem, ac.Scale, BinaryFormatCode, elem)
			if encodePlan == nil {
				return nil, nil, fmt.Errorf("unable to encode %v", array.Index(i))
			}
		}

		switch elementTypeId {
		case fbproto.SQL_TYPE_VARYING:
			tmpBuf, err = ac.encodeVaryingType(array, encodePlan, lastElemType, elem)
		case fbproto.SQL_TYPE_TEXT:
			tmpBuf, err = ac.encodeTextType(array, encodePlan, lastElemType, elem)
		default:
			tmpBuf, err = ac.encodeStaticType(array, encodePlan, lastElemType, elem)
		}

		if err != nil {
			return nil, nil, err
		}
		buf = append(buf, tmpBuf...)
	}

	outputDimentionsMap := []map[string]int{}

	for _, elem := range ac.Dimensions {
		outputDimentionsMap = append(outputDimentionsMap, map[string]int{"Lower": elem.Lower, "Upper": elem.Upper})
	}

	return buf, outputDimentionsMap, nil
}

func (c *ArrayData) encodeStaticType(array ArraySetter, encodePlan EncodePlan, lastElemType reflect.Type, elem any) ([]byte, error) {
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

func (c *ArrayData) encodeTextType(array ArraySetter, encodePlan EncodePlan, lastElemType reflect.Type, elem any) ([]byte, error) {
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

func (c *ArrayData) encodeVaryingType(array ArraySetter, encodePlan EncodePlan, lastElemType reflect.Type, elem any) ([]byte, error) {
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

func allElementsCount(dimentions []Dimention) int {
	if len(dimentions) == 0 {
		return 0
	}

	elementCount := int(dimentions[0].Upper - dimentions[0].Lower + 1)
	for _, d := range dimentions[1:] {
		elementCount *= int(d.Upper - d.Lower + 1)
	}

	return elementCount
}

func (c *ArrayData) decodeBinary(m *Map, arrayOID uint32, src []byte, length int, array ArrayGetter) error {

	elementCount := allElementsCount(c.Dimensions)
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

// ScanArray gets an array in binary format, decodes, and writes it to dst
func ScanArray(m *Map, typeId uint32, subType int32, scale int32, length uint32, dimentionsMap []map[string]int, src []byte, dst any) error {

	ac := new(ArrayData)
	var err error

	ac.Length = length
	ac.Scale = scale
	ac.SubType = subType
	ac.TypeId = typeId

	b := []Dimention{}
	for _, elem := range dimentionsMap {
		b = append(b, Dimention{Lower: elem["Lower"], Upper: elem["Upper"]})
	}
	ac.Dimensions = b

	array, err := getArrayDecodeType(dst)
	if err != nil {
		return err
	}

	if src == nil {
		array.SetDimentions(nil)
		return nil
	}

	err = array.SetDimentions(ac.Dimensions)
	if err != nil {
		return err
	}
	if len(src) == 0 {
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

	err = ac.decodeBinary(m, dt, src, int(ac.Length), array)

	return err
}

func getArrayDecodeType(dst any) (ArrayGetter, error) {

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

	a, ok := dst.(ArrayGetter)
	if !ok {
		return nil, fmt.Errorf("unsupported argument type")
	}

	return a, nil

}

func getArrayEncodeType(dst any) (ArraySetter, error) {

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
			return nil, fmt.Errorf("cannot convert to ArraySetter because it is a ragged multi-dimensional")
		default:
			return &anySliceArrayReflect{slice: targetValue}, nil
		}
	}

	if targetElemType.Kind() == reflect.Array {
		return &anyArrayArrayReflect{array: targetValue}, nil
	}

	a, ok := dst.(ArraySetter)

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
	Dims     []Dimention
	Valid    bool
}

func (a Array[T]) Dimensions() []Dimention {
	return a.Dims
}

func (a Array[T]) Index(i int) any {
	return a.Elements[i]
}

func (a *Array[T]) SetDimentions(dimentions []Dimention) error {
	if dimentions == nil {
		*a = Array[T]{}
		return nil
	}

	elementCount := allElementsCount(dimentions)
	*a = Array[T]{
		Elements: make([]T, elementCount),
		Dims:     dimentions,
		Valid:    true,
	}

	return nil
}

func (a Array[T]) ScanIndex(i int) any {
	return &a.Elements[i]
}

type FlatArray[T any] []T

func (a FlatArray[T]) Index(i int) any {
	return a[i]
}

func (a FlatArray[T]) Dimensions() []Dimention {
	if a == nil {
		return nil
	}

	return []Dimention{{Lower: 1, Upper: len(a)}}
}

func (a *FlatArray[T]) SetDimentions(dimentions []Dimention) error {
	if dimentions == nil {
		*a = nil
		return nil
	}

	elementCount := allElementsCount(dimentions)
	*a = make(FlatArray[T], elementCount)
	return nil
}

func (a FlatArray[T]) ScanIndex(i int) any {
	return &a[i]
}
