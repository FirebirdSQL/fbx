package fbtype

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"reflect"

	"rdb.red-soft.ru/fbx/internal/pgio"
)

type ArrayGetter interface {
	// Dimensions returns the array dimensions. If array is nil then nil is returned.
	Dimensions() []ArrayDimension

	// Index returns the element at i.
	Index(i int) any

	// IndexType returns a non-nil scan target of the type Index will return. This is used by ArrayCodec.PlanEncode.
	IndexType() any
}

type ArraySetter interface {
	// SetDimensions prepares the value such that ScanIndex can be called for each element. This will remove any existing
	// elements. dimensions may be nil to indicate a NULL array. If unable to exactly preserve dimensions SetDimensions
	// may return an error or silently flatten the array dimensions.
	SetDimensions(dimensions []ArrayDimension) error

	// ScanIndex returns a value usable as a scan target for i. SetDimensions must be called before ScanIndex.
	ScanIndex(i int) any

	// ScanIndexType returns a non-nil scan target of the type ScanIndex will return. This is used by
	// ArrayCodec.PlanScan.
	ScanIndexType() any
}

// ArrayCodec is a codec for any array type.
type ArrayCodec struct {
	ElementType *Type
	SubType     int32
	Length      uint32
	Scale       int32
}

func (c *ArrayCodec) FormatSupported(format int16) bool {
	return c.ElementType.Codec.FormatSupported(format)
}

func (c *ArrayCodec) PreferredFormat() int16 {
	if c.ElementType.Codec.FormatSupported(BinaryFormatCode) {
		return BinaryFormatCode
	}
	return TextFormatCode
}

func (c *ArrayCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	arrayValuer, ok := value.(ArrayGetter)
	if !ok {
		return nil
	}

	elementType := arrayValuer.IndexType()

	newTypeMetadata := SetTypeMetadata(c.ElementType.TypeID, c.SubType, c.Length, c.Scale, typeMetadata.FormatCode)
	elementEncodePlan := m.PlanEncode(newTypeMetadata, elementType)
	if elementEncodePlan == nil {
		if reflect.TypeOf(elementType) != nil {
			return nil
		}
	}

	switch typeMetadata.FormatCode {
	case BinaryFormatCode:
		return &encodePlanArrayCodecBinary{ac: c, m: m, typeid: typeMetadata.TypeID}
	case TextFormatCode:
		return &encodePlanArrayCodecText{ac: c, m: m, typeid: typeMetadata.TypeID}
	}

	return nil
}

type encodePlanArrayCodecText struct {
	ac     *ArrayCodec
	m      *Map
	typeid uint32
}

func (p *encodePlanArrayCodecText) Encode(value any, buf []byte) (newBuf []byte, err error) {
	array := value.(ArrayGetter)

	dimensions := array.Dimensions()
	if dimensions == nil {
		return nil, nil
	}

	elementCount := cardinality(dimensions)
	if elementCount == 0 {
		return append(buf, '{', '}'), nil
	}

	buf = encodeTextArrayDimensions(buf, dimensions)

	// dimElemCounts is the multiples of elements that each array lies on. For
	// example, a single dimension array of length 4 would have a dimElemCounts of
	// [4]. A multi-dimensional array of lengths [3,5,2] would have a
	// dimElemCounts of [30,10,2]. This is used to simplify when to render a '{'
	// or '}'.
	dimElemCounts := make([]int, len(dimensions))
	dimElemCounts[len(dimensions)-1] = int(dimensions[len(dimensions)-1].Length)
	for i := len(dimensions) - 2; i > -1; i-- {
		dimElemCounts[i] = int(dimensions[i].Length) * dimElemCounts[i+1]
	}

	var encodePlan EncodePlan
	var lastElemType reflect.Type
	inElemBuf := make([]byte, 0, 32)
	for i := 0; i < elementCount; i++ {
		if i > 0 {
			buf = append(buf, ',')
		}

		for _, dec := range dimElemCounts {
			if i%dec == 0 {
				buf = append(buf, '{')
			}
		}

		elem := array.Index(i)
		var elemBuf []byte
		if isNil, _ := isNilDriverValuer(elem); !isNil {
			elemType := reflect.TypeOf(elem)
			if lastElemType != elemType {
				lastElemType = elemType
				typeMetadata := SetTypeMetadata(p.ac.ElementType.TypeID, p.ac.SubType, p.ac.Length, p.ac.Scale, TextFormatCode)
				encodePlan = p.m.PlanEncode(typeMetadata, elem)
				if encodePlan == nil {
					return nil, fmt.Errorf("unable to encode %v", array.Index(i))
				}
			}
			elemBuf, err = encodePlan.Encode(elem, inElemBuf)
			if err != nil {
				return nil, err
			}
		}

		if elemBuf == nil {
			buf = append(buf, `NULL`...)
		} else {
			buf = append(buf, quoteArrayElementIfNeeded(string(elemBuf))...)
		}

		for _, dec := range dimElemCounts {
			if (i+1)%dec == 0 {
				buf = append(buf, '}')
			}
		}
	}

	return buf, nil
}

type encodePlanArrayCodecBinary struct {
	ac     *ArrayCodec
	m      *Map
	typeid uint32
}

func (p *encodePlanArrayCodecBinary) Encode(value any, buf []byte) (newBuf []byte, err error) {
	array := value.(ArrayGetter)

	dimensions := array.Dimensions()
	if dimensions == nil {
		return nil, nil
	}

	arrayHeader := arrayHeader{
		Dimensions:    dimensions,
		ElementTypeID: p.ac.ElementType.TypeID,
	}

	containsNullIndex := len(buf) + 4

	buf = arrayHeader.EncodeBinary(buf)

	elementCount := cardinality(dimensions)

	var encodePlan EncodePlan
	var lastElemType reflect.Type
	for i := 0; i < elementCount; i++ {
		sp := len(buf)
		buf = pgio.AppendInt32(buf, -1)

		elem := array.Index(i)
		var elemBuf []byte
		if isNil, _ := isNilDriverValuer(elem); !isNil {
			elemType := reflect.TypeOf(elem)
			if lastElemType != elemType {
				lastElemType = elemType
				typeMetadata := SetTypeMetadata(p.ac.ElementType.TypeID, p.ac.SubType, p.ac.Length, p.ac.Scale, BinaryFormatCode)
				encodePlan = p.m.PlanEncode(typeMetadata, elem)
				if encodePlan == nil {
					return nil, fmt.Errorf("unable to encode %v", array.Index(i))
				}
			}
			elemBuf, err = encodePlan.Encode(elem, buf)
			if err != nil {
				return nil, err
			}
		}

		if elemBuf == nil {
			pgio.SetInt32(buf[containsNullIndex:], 1)
		} else {
			buf = elemBuf
			pgio.SetInt32(buf[sp:], int32(len(buf[sp:])-4))
		}
	}

	return buf, nil
}

func (c *ArrayCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	arrayScanner, ok := target.(ArraySetter)
	if !ok {
		return nil
	}

	// target / arrayScanner might be a pointer to a nil. If it is create one so we can call ScanIndexType to plan the
	// scan of the elements.
	if isNil, _ := isNilDriverValuer(target); isNil {
		arrayScanner = reflect.New(reflect.TypeOf(target).Elem()).Interface().(ArraySetter)
	}

	elementType := arrayScanner.ScanIndexType()

	newTypeMetadata := typeMetadata
	newTypeMetadata.TypeID = c.ElementType.TypeID
	elementScanPlan := m.PlanScan(newTypeMetadata, elementType)
	if _, ok := elementScanPlan.(*scanPlanFail); ok {
		return nil
	}

	return &scanPlanArrayCodec{
		arrayCodec: c,
		m:          m,
		typeid:     typeMetadata.TypeID,
		formatCode: typeMetadata.FormatCode,
	}
}

func (c *ArrayCodec) decodeBinary(m *Map, arrayTypeID uint32, src []byte, array ArraySetter) error {
	var arrayHeader arrayHeader
	rp, err := arrayHeader.DecodeBinary(m, src)
	if err != nil {
		return err
	}

	err = array.SetDimensions(arrayHeader.Dimensions)
	if err != nil {
		return err
	}

	elementCount := cardinality(arrayHeader.Dimensions)
	if elementCount == 0 {
		return nil
	}

	typeMetadata := SetTypeMetadata(c.ElementType.TypeID, c.SubType, c.Length, c.Scale, BinaryFormatCode)
	elementScanPlan := c.ElementType.Codec.PlanScan(m, typeMetadata, array.ScanIndex(0))
	if elementScanPlan == nil {
		elementScanPlan = m.PlanScan(typeMetadata, array.ScanIndex(0))
	}

	for i := 0; i < elementCount; i++ {
		elem := array.ScanIndex(i)
		elemLen := int(int32(binary.BigEndian.Uint32(src[rp:])))
		rp += 4
		var elemSrc []byte
		if elemLen >= 0 {
			elemSrc = src[rp : rp+elemLen]
			rp += elemLen
		}
		err = elementScanPlan.Scan(elemSrc, elem)
		if err != nil {
			return fmt.Errorf("failed to scan array element %d: %w", i, err)
		}
	}

	return nil
}

func (c *ArrayCodec) decodeText(m *Map, arrayTypeID uint32, src []byte, array ArraySetter) error {
	uta, err := parseUntypedTextArray(string(src))
	if err != nil {
		return err
	}

	err = array.SetDimensions(uta.Dimensions)
	if err != nil {
		return err
	}

	if len(uta.Elements) == 0 {
		return nil
	}

	typeMetadata := SetTypeMetadata(c.ElementType.TypeID, c.SubType, c.Length, c.Scale, TextFormatCode)
	elementScanPlan := c.ElementType.Codec.PlanScan(m, typeMetadata, array.ScanIndex(0))
	if elementScanPlan == nil {
		elementScanPlan = m.PlanScan(typeMetadata, array.ScanIndex(0))
	}

	for i, s := range uta.Elements {
		elem := array.ScanIndex(i)
		var elemSrc []byte
		if s != "NULL" || uta.Quoted[i] {
			elemSrc = []byte(s)
		}

		err = elementScanPlan.Scan(elemSrc, elem)
		if err != nil {
			return err
		}
	}

	return nil
}

type scanPlanArrayCodec struct {
	arrayCodec      *ArrayCodec
	m               *Map
	typeid          uint32
	subType         int32
	length          uint32
	scale           int32
	formatCode      int16
	elementScanPlan ScanPlan
}

func (spac *scanPlanArrayCodec) Scan(src []byte, dst any) error {
	c := spac.arrayCodec
	m := spac.m
	typeid := spac.typeid
	formatCode := spac.formatCode

	array := dst.(ArraySetter)

	if src == nil {
		return array.SetDimensions(nil)
	}

	switch formatCode {
	case BinaryFormatCode:
		return c.decodeBinary(m, typeid, src, array)
	case TextFormatCode:
		return c.decodeText(m, typeid, src, array)
	default:
		return fmt.Errorf("unknown format code %d", formatCode)
	}
}

func (c *ArrayCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	switch typeMetadata.FormatCode {
	case TextFormatCode:
		return string(src), nil
	case BinaryFormatCode:
		buf := make([]byte, len(src))
		copy(buf, src)
		return buf, nil
	default:
		return nil, fmt.Errorf("unknown format code %d", typeMetadata.FormatCode)
	}
}

func (c *ArrayCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var slice []any
	err := m.PlanScan(typeMetadata, &slice).Scan(src, &slice)
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
