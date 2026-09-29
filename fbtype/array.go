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

var (
	errArrayMetadataMissing = errors.New("array element metadata is missing")
	errNullArrayElement     = errors.New("NULL array element is not supported")
)

const (
	maxArrayDimensions  = 16
	minArrayBound       = -1 << 31
	maxArrayBound       = 1<<31 - 1
	maxArraySliceLength = 1<<31 - 1
)

// ArrayGetter provides array dimensions and elements for encoding.
type ArrayGetter interface {
	// Index returns the flattened array element at i.
	Index(i int) any

	// Dimensions returns dimensions for encoding. lowers contains the lower
	// bounds declared by the database column. A nil result means SQL NULL.
	Dimensions(lowers []int) ([]ArrayDimension, error)
}

// ArrayDimension describes one dimension of a Firebird array.
type ArrayDimension struct {
	LowerBound int // LowerBound is the first valid index.
	Length     int // Length is the number of elements in the dimension.
}

// ArraySetter prepares and exposes destinations for decoded array elements.
type ArraySetter interface {
	// SetDimensions resets the destination before scanning an array. nil means
	// SQL NULL. Unsupported dimensions may be flattened or rejected.
	SetDimensions(dimensions []ArrayDimension) error

	// ScanIndex returns the scan target for flattened element i.
	// SetDimensions must be called first.
	ScanIndex(i int) any
}

// ArrayData contains the element metadata and dimensions of a Firebird array.
type ArrayData struct {
	ElementType *Type
	Dimensions  []ArrayDimension
	SubType     int32
	Length      uint32
	Scale       int32
	TypeId      uint32
}

// ArraySQLScanner carries an encoded Firebird array and its metadata through
// database/sql for later scanning by [Map.SQLScanner].
type ArraySQLScanner struct {
	ArrayData   *ArrayData
	ArrayBuffer []byte
}

// NewArrayData constructs array metadata from element metadata and inclusive
// Lower and Upper dimension bounds.
func NewArrayData(typeID uint32, subType int32, length uint32, scale int32, dimensionsMap []map[string]int) (*ArrayData, error) {
	ad := new(ArrayData)
	var err error

	ad.Length = length
	ad.Scale = scale
	ad.SubType = subType
	ad.TypeId = typeID

	dims := make([]ArrayDimension, 0, len(dimensionsMap))
	for i, elem := range dimensionsMap {
		lower, lowerPresent := elem["Lower"]
		upper, upperPresent := elem["Upper"]
		if !lowerPresent || !upperPresent {
			return nil, fmt.Errorf("array field dimension %d must contain Lower and Upper bounds", i)
		}
		if lower < minArrayBound || lower > maxArrayBound {
			return nil, fmt.Errorf("array field dimension %d lower bound %d is outside int32 range", i, lower)
		}
		if upper < minArrayBound || upper > maxArrayBound {
			return nil, fmt.Errorf("array field dimension %d upper bound %d is outside int32 range", i, upper)
		}
		if lower >= upper {
			return nil, fmt.Errorf("array field dimension %d has invalid bounds %d:%d", i, lower, upper)
		}

		length64 := int64(upper) - int64(lower) + 1
		if length64 > int64(maxIntValue()) {
			return nil, fmt.Errorf("array field dimension %d length %d overflows int", i, length64)
		}
		dims = append(dims, ArrayDimension{LowerBound: lower, Length: int(length64)})
	}
	ad.Dimensions = dims

	ad.ElementType, err = getTypeFromMap(typeID)
	if err != nil {
		return nil, err
	}
	return ad, nil

}

// ArrayCodec encodes and decodes Firebird ARRAY values.
type ArrayCodec struct {
	ElementType *Type
}

// PlanEncode returns an encode plan for supported array values.
func (ArrayCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	if _, err := getArrayEncodeType(value); err == nil {
		return planEncodeArray{m: m, arrayData: typeMetadata.ArrayData}
	}
	return nil
}

// PlanScan returns a scan plan for supported array targets.
func (ArrayCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	if _, err := getArrayDecodeType(target); err == nil {
		return planScanArray{m: m, arrayData: typeMetadata.ArrayData}
	}
	return nil
}

// DecodeDatabaseSQLValue returns an owned copy of the encoded array body.
func (ArrayCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	buf := make([]byte, len(src))
	copy(buf, src)
	return buf, nil
}

// DecodeValue decodes an array into a flat []any value.
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
		return array.SetDimensions(nil)
	}
	if ps.arrayData == nil {
		return errArrayMetadataMissing
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
	if pe.arrayData == nil {
		return nil, errArrayMetadataMissing
	}
	var lowers []int
	for _, d := range pe.arrayData.Dimensions {
		lowers = append(lowers, d.LowerBound)
	}
	dimensions, err := array.Dimensions(lowers)
	if err != nil {
		return nil, err
	}
	if dimensions == nil {
		return nil, nil
	}

	argumentElementCount, err := checkedArrayElementCount(dimensions)
	if err != nil {
		return nil, err
	}
	if err := validateArraySliceByteLength(argumentElementCount, pe.arrayData.TypeId, pe.arrayData.Length); err != nil {
		return nil, err
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
	elementTypeId := pe.arrayData.TypeId
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
			return nil, fmt.Errorf("cannot encode array element at index %d: %w", i, err)
		}
		buf = append(buf, tmpBuf...)
	}

	return buf, nil
}

// Dimensions returns array bounds as maps with "Lower" and "Upper" keys.
// lowers supplies lower bounds for Go slices and arrays. A nil value returns nil.
func Dimensions(value any, lowers []int) ([]map[string]int, error) {
	if isNilValue(value) {
		return nil, nil
	}

	array, err := getArrayEncodeType(value)
	if err != nil {
		return nil, err
	}
	dimensions, err := array.Dimensions(lowers)
	if err != nil {
		return nil, err
	}
	if dimensions == nil {
		return nil, nil
	}
	if len(dimensions) == 0 {
		return nil, fmt.Errorf("non-NULL array must have at least one dimension")
	}
	if _, err := checkedArrayElementCount(dimensions); err != nil {
		return nil, err
	}
	output := []map[string]int{}
	for i, dimension := range dimensions {
		upper, err := checkedArrayDimensionUpper(i, dimension)
		if err != nil {
			return nil, err
		}
		output = append(output, map[string]int{"Lower": dimension.LowerBound, "Upper": upper})
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
	if uint64(len(buf)) > uint64(ad.Length) {
		return nil, fmt.Errorf("encoded array element length %d exceeds declared length %d", len(buf), ad.Length)
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

func (ad ArrayData) encodeVaryingType(encodePlan EncodePlan, elem any) ([]byte, error) {
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

func (ad *ArrayData) decodeBinary(m *Map, src []byte, array ArraySetter, realIndexes []realIndexes) error {
	elementCount, err := checkedDeclaredArrayElementCount(ad.Dimensions)
	if err != nil {
		return err
	}
	if elementCount == 0 {
		return nil
	}
	length := int(ad.Length)

	typeMetadata := NewTypeMetadata(ad.TypeId, ad.SubType, ad.Length, ad.Scale)
	elementTarget := array.ScanIndex(0)
	elementScanPlan := m.PlanScan(typeMetadata, elementTarget)
	if elementScanPlan == nil {
		return fmt.Errorf("unable to scan array element into %T", elementTarget)
	}
	if ad.TypeId == fbproto.SQL_TYPE_SHORT {
		length = 4
	}
	position := 0
	var elemLen int
	var wireElemLen int
	if ad.TypeId == fbproto.SQL_TYPE_TEXT && length%4 != 0 {
		elemLen = length
		wireElemLen = length + (4 - length&3)
	} else {
		elemLen = length
		wireElemLen = length
	}
	indexPosition := 0
	for i := 0; i < elementCount && position < len(src); i++ {
		if ad.TypeId == fbproto.SQL_TYPE_VARYING {
			if len(src[position:]) < 4 {
				return fmt.Errorf("array body truncated at element %d", i)
			}
			elemLen = int(int32(binary.BigEndian.Uint32(src[position:])))
			wireElemLen = elemLen
			position += 4
		}
		var elemSrc []byte
		if len(realIndexes) > 1 && i > realIndexes[0].end {
			realIndexes = realIndexes[1:]
		}
		if elemLen >= 0 {
			if len(src[position:]) < wireElemLen {
				return fmt.Errorf("array element %d length %d exceeds remaining %d bytes", i, wireElemLen, len(src[position:]))
			}
			elemSrc = src[position : position+elemLen]
			position += wireElemLen
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
	targetValue := reflect.ValueOf(dst)
	if !targetValue.IsValid() || targetValue.Kind() != reflect.Pointer {
		return nil, fmt.Errorf("unsupported argument type %T for array decode", dst)
	}
	if targetValue.IsNil() {
		return nil, fmt.Errorf("cannot decode array into nil target %T", dst)
	}

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

	targetType := targetValue.Type()
	targetElemType := targetType.Elem()

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
	targetValue := reflect.ValueOf(dst)
	if !targetValue.IsValid() {
		return nil, fmt.Errorf("unsupported argument type %T for array encode", dst)
	}
	if targetValue.Kind() == reflect.Pointer && targetValue.IsNil() {
		return nil, fmt.Errorf("cannot encode array from nil target %T", dst)
	}

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

	targetElemType := targetValue.Type()

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
	defaultMapInitOnce.Do(initDefaultMap)

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

// Array represents a nullable Firebird array with explicit dimensions.
// Valid is false for SQL NULL. Otherwise, Dims must be non-empty and their
// lengths must match len(Elements).
type Array[T any] struct {
	Elements []T
	Dims     []ArrayDimension
	Valid    bool
}

// Dimensions implements [ArrayGetter]. It validates that Dims describes
// exactly the number of values in Elements.
func (a Array[T]) Dimensions(_ []int) ([]ArrayDimension, error) {
	if !a.Valid {
		return nil, nil
	}
	if len(a.Dims) == 0 {
		return nil, fmt.Errorf("valid array must have at least one dimension")
	}

	elementCount, err := checkedArrayElementCount(a.Dims)
	if err != nil {
		return nil, err
	}
	if elementCount != len(a.Elements) {
		return nil, fmt.Errorf(
			"array dimensions imply element count %d, but Elements has length %d",
			elementCount,
			len(a.Elements),
		)
	}

	return a.Dims, nil
}

// Index implements [ArrayGetter].
func (a Array[T]) Index(i int) any {
	return a.Elements[i]
}

// SetDimensions implements [ArraySetter]. It replaces the current value and
// treats nil dimensions as SQL NULL.
func (a *Array[T]) SetDimensions(dimensions []ArrayDimension) error {
	if dimensions == nil {
		*a = Array[T]{}
		return nil
	}
	if len(dimensions) == 0 {
		*a = Array[T]{}
		return fmt.Errorf("non-NULL array must have at least one dimension")
	}

	elementCount, err := checkedArrayElementCount(dimensions)
	if err != nil {
		*a = Array[T]{}
		return err
	}
	*a = Array[T]{
		Elements: make([]T, elementCount),
		Dims:     dimensions,
		Valid:    true,
	}

	return nil
}

// ScanIndex implements [ArraySetter].
func (a Array[T]) ScanIndex(i int) any {
	return &a.Elements[i]
}

// FlatArray adapts a slice to [ArrayGetter] and [ArraySetter]. It represents a
// nil slice as SQL NULL and flattens multidimensional values when scanning.
type FlatArray[T any] []T

// Index implements [ArrayGetter].
func (a FlatArray[T]) Index(i int) any {
	return a[i]
}

// Dimensions implements [ArrayGetter].
func (a FlatArray[T]) Dimensions(lowers []int) ([]ArrayDimension, error) {
	if a == nil {
		return nil, nil
	}
	if len(lowers) == 0 {
		return nil, fmt.Errorf("array lower bound is missing")
	}

	return []ArrayDimension{{LowerBound: lowers[0], Length: len(a)}}, nil
}

// SetDimensions implements [ArraySetter].
func (a *FlatArray[T]) SetDimensions(dimensions []ArrayDimension) error {
	if dimensions == nil {
		*a = nil
		return nil
	}
	if len(dimensions) == 0 {
		*a = nil
		return fmt.Errorf("non-NULL array must have at least one dimension")
	}

	elementCount, err := checkedArrayElementCount(dimensions)
	if err != nil {
		*a = nil
		return err
	}
	*a = make(FlatArray[T], elementCount)
	return nil
}

// ScanIndex implements [ArraySetter].
func (a FlatArray[T]) ScanIndex(i int) any {
	return &a[i]
}

func maxIntValue() int {
	return int(^uint(0) >> 1)
}

func checkedArrayElementCount(dimensions []ArrayDimension) (int, error) {
	if len(dimensions) > maxArrayDimensions {
		return 0, fmt.Errorf("array has %d dimensions; maximum is %d", len(dimensions), maxArrayDimensions)
	}

	for i, dimension := range dimensions {
		if dimension.Length < 0 {
			return 0, fmt.Errorf("array dimension %d has negative length %d", i, dimension.Length)
		}
		if _, err := checkedArrayDimensionUpper(i, dimension); err != nil {
			return 0, err
		}
	}

	if len(dimensions) == 0 {
		return 0, nil
	}

	maxInt := maxIntValue()
	elementCount := 1
	for i, dimension := range dimensions {
		if dimension.Length == 0 {
			return 0, nil
		}
		if elementCount > maxInt/dimension.Length {
			return 0, fmt.Errorf("array element count overflows int at dimension %d", i)
		}
		elementCount *= dimension.Length
	}

	return elementCount, nil
}

func checkedDeclaredArrayElementCount(dimensions []ArrayDimension) (int, error) {
	if len(dimensions) == 0 {
		return 0, fmt.Errorf("array field must have at least one dimension")
	}

	for i, dimension := range dimensions {
		if dimension.Length < 2 {
			return 0, fmt.Errorf("array field dimension %d has length %d; minimum is 2", i, dimension.Length)
		}
	}

	elementCount, err := checkedArrayElementCount(dimensions)
	if err != nil {
		return 0, fmt.Errorf("invalid array field metadata: %w", err)
	}
	return elementCount, nil
}

func checkedArrayDimensionUpper(index int, dimension ArrayDimension) (int, error) {
	if dimension.LowerBound < minArrayBound || dimension.LowerBound > maxArrayBound {
		return 0, fmt.Errorf("array dimension %d lower bound %d is outside int32 range", index, dimension.LowerBound)
	}
	if dimension.Length < 0 {
		return 0, fmt.Errorf("array dimension %d has negative length %d", index, dimension.Length)
	}

	lower := int64(dimension.LowerBound)
	if dimension.Length == 0 {
		if lower == minArrayBound {
			return 0, fmt.Errorf("array dimension %d empty upper bound is outside int32 range", index)
		}
		return int(lower - 1), nil
	}

	maxLength := int64(maxArrayBound) - lower + 1
	if uint64(dimension.Length) > uint64(maxLength) {
		return 0, fmt.Errorf("array dimension %d upper bound is outside int32 range", index)
	}

	return int(lower + int64(dimension.Length) - 1), nil
}

func validateArraySliceByteLength(elementCount int, typeID, elementLength uint32) error {
	if elementCount < 0 {
		return fmt.Errorf("array element count must not be negative: %d", elementCount)
	}

	length := uint64(elementLength)
	if typeID == fbproto.SQL_TYPE_VARYING {
		length += 2
	}
	if elementCount > 0 && length == 0 {
		return fmt.Errorf("array element length must be greater than zero")
	}
	if length > 0 && uint64(elementCount) > uint64(maxArraySliceLength)/length {
		return fmt.Errorf("array slice byte length exceeds protocol limit")
	}

	return nil
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

func elementsInDimensionsCount(dimensions []ArrayDimension, start int) (int, error) {
	if start < 0 {
		return 0, fmt.Errorf("array element count has negative initial value %d", start)
	}

	total := start
	for i := 1; i < len(dimensions); i++ {
		if dimensions[i].Length < 0 {
			return 0, fmt.Errorf("array dimension %d has negative length %d", i, dimensions[i].Length)
		}
		if dimensions[i].Length != 0 && total > maxIntValue()/dimensions[i].Length {
			return 0, fmt.Errorf("array element count overflows int at dimension %d", i)
		}
		total *= dimensions[i].Length
	}
	return total, nil
}

func getActualArrayLength(buf []byte, typeID, length uint32) (int, error) {
	count := 0
	switch typeID {
	case fbproto.SQL_TYPE_VARYING:
		for len(buf) > 0 {
			if len(buf) < 4 {
				return 0, fmt.Errorf("length of decoded buffer is less than expected")
			}
			elementLength := uint64(binary.BigEndian.Uint32(buf[:4])) + 4
			if elementLength > uint64(len(buf)) {
				return 0, fmt.Errorf("length of decoded buffer is less than expected")
			}
			buf = buf[int(elementLength):]
			count++
		}
	case fbproto.SQL_TYPE_BOOLEAN, fbproto.SQL_TYPE_SHORT:
		if len(buf)%4 != 0 {
			return 0, fmt.Errorf("array body length %d is not divisible by element length 4", len(buf))
		}
		count = len(buf) / 4
	default:
		if length < 1 {
			return 0, fmt.Errorf("length of element must not be less than 1")
		}
		elementLength := int(length)
		if typeID == fbproto.SQL_TYPE_TEXT && elementLength%4 != 0 {
			elementLength += 4 - elementLength%4
		}
		if len(buf)%elementLength != 0 {
			return 0, fmt.Errorf("array body length %d is not divisible by element length %d", len(buf), elementLength)
		}
		count = len(buf) / elementLength
	}
	return count, nil
}

func getActualArrayDimensions(dims []ArrayDimension, fact int) ([]ArrayDimension, error) {
	expected, err := checkedDeclaredArrayElementCount(dims)
	if err != nil {
		return nil, err
	}
	if fact < 0 {
		return nil, fmt.Errorf("actual array element count must not be negative: %d", fact)
	}
	if fact > expected {
		return nil, fmt.Errorf("array body contains %d elements, but field holds at most %d", fact, expected)
	}

	less := expected - fact
	factDims := make([]ArrayDimension, len(dims))
	for i := range dims {
		elementsCount, err := elementsInDimensionsCount(dims[i:], 1)
		if err != nil {
			return nil, err
		}
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
		skipped, err := elementsInDimensionsCount(dims, dims[0].Length-factDims[0].Length)
		if err != nil {
			return nil, 0, err
		}
		nowPos += skipped
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
