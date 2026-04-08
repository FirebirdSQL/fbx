package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

// ArrayGetter is the interface for sending the array to the server
type ArrayGetter interface {

	// Index returns the element at i.
	Index(i int) any

	// Dimensions returns the []Dimensions of the array
	Dimensions(lowers []int) ([]Dimension, error)
}

// Dimension is a struct for storage array dimensions
type Dimension struct {
	Lower int
	Upper int
}

// ArraySetter is the interface for getting the array from the server
type ArraySetter interface {
	// SetDimensions prepares the value such that ScanIndex can be called for each element. This will remove any existing
	// elements. dimensions may be nil to indicate a NULL array. If unable to exactly preserve dimensions SetDimensions
	// may return an error or silently flatten the array dimensions.
	SetDimensions(dimensions []Dimension) error

	// ScanIndex returns a value usable as a scan target for i. SetDimensions must be called before ScanIndex.
	ScanIndex(i int) any
}

// ArrayData is a struct for any array type.
type ArrayData struct {
	ElementType *Type
	Dimensions  []Dimension
	SubType     int32
	Length      uint32
	Scale       int32
	TypeId      uint32
}

type ArraySQLScanner struct {
	ArrayData   *ArrayData
	ArrayBuffer []byte
}

// NewArrayData is a constructor for ArrayData struct
func NewArrayData(typeId uint32, subType int32, length uint32, scale int32, DimensionsMap []map[string]int) (*ArrayData, error) {
	ad := new(ArrayData)
	var err error

	ad.Length = length
	ad.Scale = scale
	ad.SubType = subType
	ad.TypeId = typeId

	b := []Dimension{}
	for _, elem := range DimensionsMap {
		b = append(b, Dimension{Lower: elem["Lower"], Upper: elem["Upper"]})
	}
	ad.Dimensions = b

	ad.ElementType, err = getTypeFromMap(typeId)
	if err != nil {
		return nil, err
	}
	return ad, nil

}

type ArrayCodec struct {
	ElementType *Type
}

func (ArrayCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	if _, err := getArrayEncodeType(value); err == nil {
		return planEncodeArray{m: m, arrayData: typeMetadata.ArrayData}
	}
	return nil
}

func (ArrayCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	if _, err := getArrayDecodeType(target); err == nil {
		return planScanArray{m: m, arrayData: typeMetadata.ArrayData}
	}
	return nil
}

func (ArrayCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	buf := make([]byte, len(src))
	copy(buf, src)
	return buf, nil
}

func (ArrayCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var slice []any
	err := m.PlanScan(typeMetadata, &slice).Scan(src, &slice)
	return slice, err
}

type planScanArray struct {
	m         *Map
	arrayData *ArrayData
}

func (ps planScanArray) Scan(src []byte, dst any) error {
	array, err := getArrayDecodeType(dst)
	if err != nil {
		return err
	}

	if src == nil {
		array.SetDimensions(nil)
		return nil
	}

	err = array.SetDimensions(ps.arrayData.Dimensions)
	if err != nil {
		return err
	}
	if len(src) == 0 {
		return nil
	}

	ps.arrayData.ElementType, err = getTypeFromMap(ps.arrayData.TypeId)
	if err != nil {
		return err
	}

	err = ps.arrayData.decodeBinary(ps.m, src, array)

	return err
}

type planEncodeArray struct {
	m         *Map
	arrayData *ArrayData
}

func (pe planEncodeArray) Encode(value any, buf []byte) (newBuf []byte, err error) {
	array, err := getArrayEncodeType(value)
	if err != nil {
		return nil, err
	}
	var lowers []int
	for _, d := range pe.arrayData.Dimensions {
		lowers = append(lowers, d.Lower)
	}
	dimensions, err := array.Dimensions(lowers)
	if dimensions == nil {
		return nil, err
	}

	argumentElementCount := allElementsCount(dimensions)
	fieldElementCount := allElementsCount(pe.arrayData.Dimensions)

	if fieldElementCount < argumentElementCount {
		return nil, fmt.Errorf("array too long")
	}

	var encodePlan EncodePlan
	var lastElemType reflect.Type
	var elem any
	var tmpBuf []byte
	elementTypeId := pe.arrayData.ElementType.TypeID
	lenElem := pe.arrayData.Length

	for i := 0; i < argumentElementCount; i++ {
		elem = array.Index(i)
		elemType := reflect.TypeOf(elem)
		if lastElemType != elemType {
			lastElemType = elemType
			typeMetadata := NewTypeMetadata(elementTypeId, pe.arrayData.SubType, lenElem, pe.arrayData.Scale)
			encodePlan = pe.m.PlanEncode(typeMetadata, elem)
			if encodePlan == nil {
				return nil, fmt.Errorf("unable to encode %v", array.Index(i))
			}
		}

		switch elementTypeId {
		case fbproto.SQL_TYPE_VARYING:
			tmpBuf, err = pe.arrayData.encodeVaryingType(encodePlan, elem)
		case fbproto.SQL_TYPE_TEXT, fbproto.SQL_TYPE_BOOLEAN:
			tmpBuf, err = pe.arrayData.encodeWithPadding(encodePlan, elem)
		default:
			tmpBuf, err = pe.arrayData.encodeStaticType(encodePlan, elem)
		}

		if err != nil {
			return nil, err
		}
		buf = append(buf, tmpBuf...)
	}

	return buf, nil
}

// Dimensions returns the list of dimensions and their bounds for array
func Dimensions(value any, lowers []int) ([]map[string]int, error) {
	array, err := getArrayEncodeType(value)
	if err != nil {
		return nil, err
	}
	dimensions, err := array.Dimensions(lowers)
	if dimensions == nil {
		return nil, err
	}
	output := []map[string]int{}
	for _, dimension := range dimensions {
		output = append(output, map[string]int{"Lower": dimension.Lower, "Upper": dimension.Upper})
	}
	return output, nil
}

func (ad *ArrayData) encodeStaticType(encodePlan EncodePlan, elem any) ([]byte, error) {
	var buf []byte
	var err error
	if elem != nil {
		buf, err = encodePlan.Encode(elem, buf)
		if err != nil {
			return nil, err
		}
	} else {
		buf = append(buf, make([]byte, ad.Length)...)
	}
	return buf, nil
}

func (ad *ArrayData) encodeWithPadding(encodePlan EncodePlan, elem any) ([]byte, error) {
	var buf []byte
	var err error
	if elem != nil {
		buf, err = encodePlan.Encode(elem, buf)
		if err != nil {
			return nil, err
		}
		l := uint32(len(buf))
		padding := make([]byte, ad.Length-l)
		for i := range padding {
			padding[i] = 32
		}
		buf = append(buf, padding...)
		if ad.Length%4 != 0 {
			padding = make([]byte, 4-ad.Length%4)
			buf = append(buf, padding...)
		}
	} else {
		buf = append(buf, make([]byte, ad.Length)...)
	}
	return buf, nil
}

func (ArrayData) encodeVaryingType(encodePlan EncodePlan, elem any) ([]byte, error) {
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

		if lenElem%4 != 0 {
			padding := make([]byte, 4-lenElem%4)
			buf = append(buf, padding...)
		}
		lenElem = 0
	} else {
		bufLen := make([]byte, 4)
		binary.BigEndian.PutUint32(bufLen, 0)
		buf = append(buf, bufLen...)
	}
	return buf, nil
}

func allElementsCount(dimensions []Dimension) int {
	if len(dimensions) == 0 {
		return 0
	}

	elementCount := int(dimensions[0].Upper - dimensions[0].Lower + 1)
	for _, d := range dimensions[1:] {
		elementCount *= int(d.Upper - d.Lower + 1)
	}

	return elementCount
}

func (ad *ArrayData) decodeBinary(m *Map, src []byte, array ArraySetter) error {

	elementCount := allElementsCount(ad.Dimensions)
	if elementCount == 0 {
		return nil
	}
	length := int(ad.Length)

	typeMetadata := NewTypeMetadata(ad.ElementType.TypeID, ad.SubType, ad.Length, ad.Scale)
	elementScanPlan := ad.ElementType.Codec.PlanScan(m, typeMetadata, array.ScanIndex(0))
	if elementScanPlan == nil {
		elementScanPlan = m.PlanScan(typeMetadata, array.ScanIndex(0))
	}
	if ad.TypeId == fbproto.SQL_TYPE_SHORT {
		length = 4
	}
	position := 0
	var elemLen int
	if ad.TypeId == fbproto.SQL_TYPE_TEXT && length%4 != 0 {
		elemLen = length + (4 - length&3)
	} else {
		elemLen = length
	}
	for i := 0; i < elementCount && position < len(src); i++ {
		elem := array.ScanIndex(i)
		if ad.TypeId == fbproto.SQL_TYPE_VARYING {
			elemLen = int(int32(binary.BigEndian.Uint32(src[position:])))
			position += 4
		}
		var elemSrc []byte
		if elemLen >= 0 {
			elemSrc = src[position : position+elemLen]
			position += elemLen
		}
		err := elementScanPlan.Scan(elemSrc, elem)
		if ad.TypeId == fbproto.SQL_TYPE_BOOLEAN {
			position += 3 // skip padding
		}
		if err != nil {
			return fmt.Errorf("failed to scan array element %d: %w", i, err)
		}
	}

	return nil
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
		return nil, fmt.Errorf("unsupported argument type %T for array decode", dst)
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
		return nil, fmt.Errorf("unsupported argument type %T for array decode", dst)
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

	targetElemType := reflect.TypeOf(dst)

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

	if targetElemType.Kind() != reflect.Slice && !ok {
		return nil, fmt.Errorf("unsupported argument type %T for array encode", dst)
	}

	return a, nil

}

func getTypeFromMap(datatype uint32) (*Type, error) {
	return defaultMap.typeidToType[datatype], nil
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
	Dims     []Dimension
	Valid    bool
}

func (a Array[T]) Dimensions(lowers []int) ([]Dimension, error) {
	return a.Dims, nil
}

func (a Array[T]) Index(i int) any {
	return a.Elements[i]
}

func (a *Array[T]) SetDimensions(dimensions []Dimension) error {
	if dimensions == nil {
		*a = Array[T]{}
		return nil
	}

	elementCount := allElementsCount(dimensions)
	*a = Array[T]{
		Elements: make([]T, elementCount),
		Dims:     dimensions,
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

func (a FlatArray[T]) Dimensions(lowers []int) ([]Dimension, error) {
	if a == nil {
		return nil, nil
	}

	return []Dimension{{Lower: lowers[0], Upper: lowers[0] + (len(a) - 1)}}, nil
}

func (a *FlatArray[T]) SetDimensions(dimensions []Dimension) error {
	if dimensions == nil {
		*a = nil
		return nil
	}

	elementCount := allElementsCount(dimensions)
	*a = make(FlatArray[T], elementCount)
	return nil
}

func (a FlatArray[T]) ScanIndex(i int) any {
	return &a[i]
}
