package fbconn

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
)

type dimensionsMapKey struct {
	source string
	schema string
}

type metadataMapKey struct {
	schema string
	table  string
	field  string
}

// Array contains the protocol metadata needed to read or write a Firebird
// array slice.
type Array struct {
	TableName         string
	FieldName         string
	Schema            string
	Dimensions        []map[string]int
	CurrentDimensions []map[string]int
	Length            uint32
	Scale             int32
	BlrTypeId         uint32
	TypeId            uint32
	SubType           int32
}

func (arr Array) getSliceLength(dimensions []map[string]int) (int, error) {
	const (
		maxArrayDimensions = 16
		minArrayBound      = -1 << 31
		maxArrayBound      = 1<<31 - 1
		maxSliceLength     = 1<<31 - 1
	)

	if len(dimensions) == 0 {
		return 0, fmt.Errorf("array slice must have at least one dimension")
	}
	if len(dimensions) > maxArrayDimensions {
		return 0, fmt.Errorf("array slice has %d dimensions; maximum is %d", len(dimensions), maxArrayDimensions)
	}

	elements := uint64(1)
	for i, dimension := range dimensions {
		lower, lowerPresent := dimension["Lower"]
		upper, upperPresent := dimension["Upper"]
		if !lowerPresent || !upperPresent {
			return 0, fmt.Errorf("array slice dimension %d must contain Lower and Upper bounds", i)
		}
		if lower < minArrayBound || lower > maxArrayBound {
			return 0, fmt.Errorf("array slice dimension %d lower bound %d is outside int32 range", i, lower)
		}
		if upper < minArrayBound || upper > maxArrayBound {
			return 0, fmt.Errorf("array slice dimension %d upper bound %d is outside int32 range", i, upper)
		}

		length := int64(upper) - int64(lower) + 1
		if length < 0 {
			return 0, fmt.Errorf("array slice dimension %d has invalid bounds %d:%d", i, lower, upper)
		}
		if length == 0 {
			elements = 0
			continue
		}
		if elements > uint64(maxSliceLength)/uint64(length) {
			return 0, fmt.Errorf("array slice element count exceeds protocol limit at dimension %d", i)
		}
		elements *= uint64(length)
	}

	elementLength := uint64(arr.Length)
	if arr.TypeId == fbproto.SQL_TYPE_VARYING {
		elementLength += 2
	}
	if elements > 0 && elementLength == 0 {
		return 0, fmt.Errorf("array element length must be greater than zero")
	}
	if elementLength > 0 && elements > uint64(maxSliceLength)/elementLength {
		return 0, fmt.Errorf("array slice byte length exceeds protocol limit")
	}

	return int(elements * elementLength), nil
}

func (arr *Array) generateSDL(dimensions []map[string]int) []byte {
	sdl := []byte{}

	dataType := byte(arr.BlrTypeId)
	if arr.SubType&0xFF == fbproto.CS_BINARY {
		switch arr.TypeId {
		case fbproto.SQL_TYPE_TEXT:
			dataType = fbproto.Isc_blr_text2
		case fbproto.SQL_TYPE_VARYING:
			dataType = fbproto.Isc_blr_varying2
		}
	}
	length := binary.LittleEndian.AppendUint16(make([]byte, 0), uint16(arr.Length))

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_version1,
		fbproto.Isc_sdl_struct,
		1,
		dataType,
	}...)

	switch dataType {
	case fbproto.Isc_blr_short, fbproto.Isc_blr_long, fbproto.Isc_blr_int64, fbproto.Isc_blr_quad:
		sdl = append(sdl, byte(arr.Scale))
	case fbproto.Isc_blr_text, fbproto.Isc_blr_cstring, fbproto.Isc_blr_varying:
		sdl = append(sdl, length...)
	case fbproto.Isc_blr_text2, fbproto.Isc_blr_cstring2, fbproto.Isc_blr_varying2:
		sdl = append(sdl, byte(arr.SubType), byte(arr.SubType>>8))
		sdl = append(sdl, length...)
	}

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_relation,
		byte(len(arr.TableName)),
	}...)
	sdl = append(sdl, []byte(arr.TableName)...)

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_field,
		byte(len(arr.FieldName)),
	}...)
	sdl = append(sdl, []byte(arr.FieldName)...)

	if arr.Schema != "" {
		sdl = append(sdl, []byte{
			fbproto.Isc_sdl_schema,
			byte(len(arr.Schema)),
		}...)
		sdl = append(sdl, []byte(arr.Schema)...)
	}

	for i, b := range dimensions {
		if b["Lower"] == 1 {
			sdl = append(sdl, []byte{
				fbproto.Isc_sdl_do1,
				byte(i),
			}...)
		} else {
			sdl = append(sdl, []byte{
				fbproto.Isc_sdl_do2,
				byte(i),
			}...)
			sdl = append(sdl, getDimensionBound(b["Lower"])...)
		}
		sdl = append(sdl, getDimensionBound(b["Upper"])...)
	}

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_element,
		1,
		fbproto.Isc_sdl_scalar,
		0,
		byte(len(dimensions)),
	}...)

	for i := 0; i < len(dimensions); i++ {
		sdl = append(sdl, []byte{
			fbproto.Isc_sdl_variable,
			byte(i),
		}...)
	}

	sdl = append(sdl, fbproto.Isc_sdl_eoc)

	return sdl
}

func getDimensionBound(value int) []byte {
	out := []byte{}
	valByte := make([]byte, 4)
	binary.BigEndian.PutUint32(valByte, uint32(value))

	if -128 <= value && value <= 127 {
		out = append(out, fbproto.Isc_sdl_tiny_integer)
		out = append(out, byte(value))
		return out
	}

	if -32768 <= value && value <= 32767 {
		out = append(out, fbproto.Isc_sdl_short_integer)
		out = append(out, binary.LittleEndian.AppendUint16(make([]byte, 0), uint16(value))...)
		return out
	}

	out = append(out, fbproto.Isc_sdl_long_integer)
	out = append(out, binary.LittleEndian.AppendUint32(make([]byte, 0), uint32(value))...)

	return out
}

func putArray(fbConn *FbConn, ctx context.Context, tx *FBTransaction, arrayData *Array, arr []byte) ([]byte, error) {
	defer func() {
		arrayData.CurrentDimensions = arrayData.Dimensions
	}()

	length, err := arrayData.getSliceLength(arrayData.CurrentDimensions)
	if err != nil {
		return nil, err
	}
	sdl := arrayData.generateSDL(arrayData.CurrentDimensions)

	id, err := fbConn.putArray(ctx, tx, arr, length, sdl)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func getArray(fbConn *FbConn, ctx context.Context, tx *FBTransaction, arrayData *Array, arrId []byte) ([]byte, error) {
	if len(arrId) == 0 {
		return nil, nil
	}

	arrayId := int64(binary.BigEndian.Uint64(arrId))

	length, err := arrayData.getSliceLength(arrayData.Dimensions)
	if err != nil {
		return nil, err
	}
	sdl := arrayData.generateSDL(arrayData.Dimensions)

	array, err := fbConn.getArray(ctx, tx, arrayId, arrayData.BlrTypeId, length, int(arrayData.Length), sdl)
	if err != nil {
		return nil, err
	}

	return array, err
}

func getSqlTypeFromBlr(datatype uint32) (uint32, error) {
	switch datatype {
	case fbproto.Isc_blr_varying,
		fbproto.Isc_blr_varying2:
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
