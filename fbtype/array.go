package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"time"

	"rdb.red-soft.ru/fbx/fbproto"
)

var errNullArrayElement = errors.New("NULL array element is not supported")

// ArrayGetter is the interface for sending the array to the server
type ArrayGetter interface {

	// Index returns the element at i.
	Index(i int) any

	// Dimensions returns the []Dimensions of the array
	Dimensions(lowers []int) ([]ArrayDimension, error)
}

// ArrayDimension is a struct for storage array dimensions
type ArrayDimension struct {
	LowerBound int
	Length     int
}

// ArraySetter is the interface for getting the array from the server
type ArraySetter interface {
	// SetDimensions prepares the value such that ScanIndex can be called for each element. This will remove any existing
	// elements. dimensions may be nil to indicate a NULL array. If unable to exactly preserve dimensions SetDimensions
	// may return an error or silently flatten the array dimensions.
	SetDimensions(dimensions []ArrayDimension) error

	// ScanIndex returns a value usable as a scan target for i. SetDimensions must be called before ScanIndex.
	ScanIndex(i int) any
}

// ArrayData is a struct for any array type.
type ArrayData struct {
	ElementType *Type
	Dimensions  []ArrayDimension
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
func NewArrayData(typeId uint32, subType int32, length uint32, scale int32, dimensionsMap []map[string]int) (*ArrayData, error) {
	ad := new(ArrayData)
	var err error

	ad.Length = length
	ad.Scale = scale
	ad.SubType = subType
	ad.TypeId = typeId

	dims := make([]ArrayDimension, 0, len(dimensionsMap))
	for _, elem := range dimensionsMap {
		dims = append(dims, ArrayDimension{LowerBound: elem["Lower"], Length: elem["Upper"] - elem["Lower"] + 1})
	}
	ad.Dimensions = dims

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

	realIndexes, actualDims, err := getActualSliceDimensions(ps.arrayData.Dimensions, src, ps.arrayData.TypeId, ps.arrayData.Length)
	if err != nil {
		return err
	}

	err = array.SetDimensions(actualDims)
	if err != nil {
		return err
	}
	if len(src) == 0 {
		return nil
	}

	err = ps.arrayData.decodeBinary(ps.m, src, array, realIndexes)

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
		lowers = append(lowers, d.LowerBound)
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
	if argumentElementCount == 0 {
		if buf == nil {
			buf = make([]byte, 0)
		}
		return buf, nil
	}

	var encodePlan EncodePlan
	var lastElemType reflect.Type
	var elem any
	var tmpBuf []byte
	elementTypeId := pe.arrayData.ElementType.TypeID
	lenElem := pe.arrayData.Length

	for i := 0; i < argumentElementCount; i++ {
		elem = array.Index(i)
		if isNilValue(elem) {
			return nil, fmt.Errorf("cannot encode NULL array element at index %d", i)
		}
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
			if errors.Is(err, errNullArrayElement) {
				return nil, fmt.Errorf("cannot encode NULL array element at index %d", i)
			}
			return nil, err
		}
		buf = append(buf, tmpBuf...)
	}

	return buf, nil
}

// Dimensions returns the dimensions and bounds of value. A nil-like value
// represents a SQL NULL array and returns nil dimensions.
func Dimensions(value any, lowers []int) ([]map[string]int, error) {
	if isNilValue(value) {
		return nil, nil
	}

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
		output = append(output, map[string]int{"Lower": dimension.LowerBound, "Upper": dimension.LowerBound + dimension.Length - 1})
	}
	return output, nil
}

func (ad *ArrayData) encodeStaticType(encodePlan EncodePlan, elem any) ([]byte, error) {
	buf, err := encodePlan.Encode(elem, make([]byte, 0))
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, errNullArrayElement
	}

	return buf, nil
}

func (ad *ArrayData) encodeWithPadding(encodePlan EncodePlan, elem any) ([]byte, error) {
	buf, err := encodePlan.Encode(elem, make([]byte, 0))
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, errNullArrayElement
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

	return buf, nil
}

func (ArrayData) encodeVaryingType(encodePlan EncodePlan, elem any) ([]byte, error) {
	str, err := encodePlan.Encode(elem, make([]byte, 0))
	if err != nil {
		return nil, err
	}
	if str == nil {
		return nil, errNullArrayElement
	}

	lenElem := uint32(len(str))
	bufLen := make([]byte, 4)
	binary.BigEndian.PutUint32(bufLen, lenElem)
	buf := append(bufLen, str...)

	if lenElem%4 != 0 {
		padding := make([]byte, 4-lenElem%4)
		buf = append(buf, padding...)
	}

	return buf, nil
}

func allElementsCount(dimensions []ArrayDimension) int {
	if len(dimensions) == 0 {
		return 0
	}

	maxInt := int(^uint(0) >> 1)
	elementCount := 1
	for _, d := range dimensions {
		if d.Length <= 0 {
			return 0
		}
		if elementCount > maxInt/d.Length {
			return 0
		}
		elementCount *= d.Length
	}

	return elementCount
}

func (ad *ArrayData) decodeBinary(m *Map, src []byte, array ArraySetter, realIndexes []realIndexes) error {

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
	indexPosition := 0
	for i := 0; i < elementCount && position < len(src); i++ {
		if ad.TypeId == fbproto.SQL_TYPE_VARYING {
			if len(src[position:]) < 4 {
				return fmt.Errorf("array body truncated at element %d", i)
			}
			elemLen = int(int32(binary.BigEndian.Uint32(src[position:])))
			position += 4
		}
		var elemSrc []byte
		if len(realIndexes) > 1 && i > realIndexes[0].end {
			realIndexes = realIndexes[1:]
		}
		if elemLen >= 0 {
			if len(src[position:]) < elemLen {
				return fmt.Errorf("array element %d length %d exceeds remaining %d bytes", i, elemLen, len(src[position:]))
			}
			elemSrc = src[position : position+elemLen]
			position += elemLen
		}
		if len(realIndexes) > 0 && i >= realIndexes[0].start {
			elem := array.ScanIndex(indexPosition)
			err := elementScanPlan.Scan(elemSrc, elem)
			if err != nil {
				return fmt.Errorf("failed to scan array element %d: %w", indexPosition, err)
			}
			indexPosition++
		}
		if ad.TypeId == fbproto.SQL_TYPE_BOOLEAN {
			if len(src[position:]) < 3 {
				return fmt.Errorf("array boolean element %d padding exceeds remaining %d bytes", i, len(src[position:]))
			}
			position += 3 // skip padding
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

func getTypeFromMap(typeID uint32) (*Type, error) {
	if datatype, ok := defaultMap.typeidToType[typeID]; ok {
		return datatype, nil
	}
	return nil, fmt.Errorf("no matching type in map by specified typeid: %d", typeID)
}

func isRagged(slice reflect.Value) bool {
	return isSliceRagged(slice, 0, &[]int{})
}

func isSliceRagged(slice reflect.Value, lvl int, lens *[]int) bool {
	if slice.Type().Elem().Kind() != reflect.Slice {
		return false
	}

	sliceLen := slice.Len()
	innerLen := 0
	for i := 0; i < sliceLen; i++ {
		if i == 0 {
			innerLen = slice.Index(i).Len()
			if len(*lens)-1 >= lvl {
				if (*lens)[lvl] != innerLen {
					return true
				}
			} else {
				*lens = append(*lens, innerLen)
			}
		} else {
			if slice.Index(i).Len() != innerLen {
				return true
			}
		}
		if isSliceRagged(slice.Index(i), lvl+1, lens) {
			return true
		}
	}

	return false
}

type Array[T any] struct {
	Elements []T
	Dims     []ArrayDimension
	Valid    bool
}

func (a Array[T]) Dimensions(lowers []int) ([]ArrayDimension, error) {
	return a.Dims, nil
}

func (a Array[T]) Index(i int) any {
	return a.Elements[i]
}

func (a *Array[T]) SetDimensions(dimensions []ArrayDimension) error {
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

func (a FlatArray[T]) Dimensions(lowers []int) ([]ArrayDimension, error) {
	if a == nil {
		return nil, nil
	}

	return []ArrayDimension{{LowerBound: lowers[0], Length: len(a)}}, nil
}

func (a *FlatArray[T]) SetDimensions(dimensions []ArrayDimension) error {
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

func getActualSliceDimensions(dims []ArrayDimension, src []byte, typeID, length uint32) ([]realIndexes, []ArrayDimension, error) {
	actualLength, err := getActualArrayLength(src, typeID, length)
	if err != nil {
		return nil, nil, err
	}
	actualDims, err := getActualArrayDimensions(dims, actualLength)
	if err != nil {
		return nil, nil, err
	}
	ind, _, err := getActualSlice(make([]realIndexes, 0), dims, actualDims, 0)
	if err != nil {
		return nil, nil, err
	}
	return ind, actualDims, nil
}

type realIndexes struct {
	start int
	end   int
}

func elementsInDimensionsCount(dimensions []ArrayDimension, start int) int {
	total := start
	for i := 1; i < len(dimensions); i++ {
		total *= dimensions[i].Length
	}
	return total
}

func getActualArrayLength(buf []byte, typeID, length uint32) (int, error) {
	count := 0
	switch typeID {
	case fbproto.SQL_TYPE_VARYING:
		for len(buf) > 0 {
			if len(buf) < 4 {
				return 0, fmt.Errorf("length of decoded buffer is less than expected")
			}
			l := int(binary.BigEndian.Uint32(buf[:4])) + 4
			if len(buf) < l {
				return 0, fmt.Errorf("length of decoded buffer is less than expected")
			}
			buf = buf[l:]
			count++
		}
	case fbproto.SQL_TYPE_BOOLEAN, fbproto.SQL_TYPE_SHORT:
		count = len(buf) / 4
	default:
		if length < 1 {
			return 0, fmt.Errorf("length of element must not be less than 1")
		}
		count = len(buf) / int(length)
	}
	return count, nil
}

func getActualArrayDimensions(dims []ArrayDimension, fact int) ([]ArrayDimension, error) {
	expected := allElementsCount(dims)
	less := expected - fact
	factDims := make([]ArrayDimension, len(dims))
	for i := range dims {
		elementsCount := elementsInDimensionsCount(dims[i:], 1)
		if elementsCount <= 0 {
			return nil, fmt.Errorf("number of elements in dimensions must not be less than 1")
		}
		factDims[i].Length = dims[i].Length - (less / elementsCount)
		factDims[i].LowerBound = dims[i].LowerBound
		less %= elementsCount
	}
	return factDims, nil
}

func getActualSlice(sliceOut []realIndexes, dims, factDims []ArrayDimension, nowPos int) ([]realIndexes, int, error) {
	var err error
	if len(dims) > 1 {
		for i := 0; i < factDims[0].Length; i++ {
			sliceOut, nowPos, err = getActualSlice(sliceOut, dims[1:], factDims[1:], nowPos)
			if err != nil {
				return nil, 0, err
			}
		}
		nowPos += elementsInDimensionsCount(dims, dims[0].Length-factDims[0].Length)
	} else if len(dims) == 1 {
		realIndexes := realIndexes{
			start: nowPos,
			end:   nowPos + factDims[0].Length - 1,
		}
		sliceOut = append(sliceOut, realIndexes)
		nowPos += dims[0].Length
		return sliceOut, nowPos, nil
	} else {
		return nil, 0, fmt.Errorf("length of dimensions less than 1")
	}
	return sliceOut, nowPos, nil
}
