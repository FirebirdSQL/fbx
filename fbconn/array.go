package fbconn

import (
	"context"
	"encoding/binary"
	"errors"

	"rdb.red-soft.ru/fbx/fbproto"
)

// Array struct contains the information about array
type Array struct {
	TableName         string
	FieldName         string
	Dimensions        []map[string]int
	CurrentDimensions []map[string]int
	Length            uint32
	Scale             int32
	BlrTypeId         uint32
	TypeId            uint32
	SubType           int32
}

// NewArray is a constructor for Array struct
func NewArray(table, field string, dims []map[string]int, l uint32, scale int32, typeId, blrTypeId uint32, subType int32) *Array {

	a := new(Array)

	a.TableName = table
	a.FieldName = field
	a.Dimensions = dims
	a.CurrentDimensions = dims
	a.Length = l
	a.Scale = scale
	a.TypeId = typeId
	a.BlrTypeId = blrTypeId
	a.SubType = subType

	return a
}

func (ac Array) getSliceLength() int {
	var elements int = 1
	for _, elem := range ac.CurrentDimensions {
		elements *= elem["Upper"] - elem["Lower"] + 1
	}

	length := elements * int(ac.Length)

	switch ac.TypeId {
	case fbproto.SQL_TYPE_VARYING:
		length += elements * 2
	}
	return length
}

func (ac *Array) generateSDL() []byte {
	sdl := []byte{}
	dimensions := ac.CurrentDimensions

	data_type := trimBytes(binary.BigEndian.AppendUint32(make([]byte, 4), ac.BlrTypeId), 1)[0]
	scale := trimBytes(binary.BigEndian.AppendUint32(make([]byte, 4), uint32(ac.Scale)), 1)
	length := trimBytes(binary.BigEndian.AppendUint32(make([]byte, 4), ac.Length), 2)

	binary.LittleEndian.PutUint16(length, uint16(ac.Length))

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
		byte(len(ac.TableName)),
	}...)
	sdl = append(sdl, []byte(ac.TableName)...)

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_field,
		byte(len(ac.FieldName)),
	}...)
	sdl = append(sdl, []byte(ac.FieldName)...)

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

func trimBytes(in []byte, l int) []byte {
	in = in[len(in)-l:]
	out := []byte{}

	for i := len(in) - 1; i >= 0; i-- {
		out = append(out, in[i])
	}
	return out
}

func getDimensionBound(value int) []byte {
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

// PutArray sends array to the server and returns an array id
func PutArray(fbConn *FbConn, ctx context.Context, arrayData *Array, arr []byte) ([]byte, error) {

	sdl := arrayData.generateSDL()
	length := arrayData.getSliceLength()

	id, err := fbConn.putArray(ctx, arr, length, sdl)
	arrayData.CurrentDimensions = arrayData.Dimensions
	if err != nil {
		return nil, err
	}
	return id, nil
}

// GetArray sends an array id to the server and returns an array buffer
func GetArray(fbConn *FbConn, ctx context.Context, arrayData *Array, arrId []byte) ([]byte, error) {
	if len(arrId) == 0 {
		return nil, nil
	}

	arrayId := int64(binary.BigEndian.Uint64(arrId))

	sdl := arrayData.generateSDL()
	length := arrayData.getSliceLength()

	array, err := fbConn.getArray(ctx, arrayId, arrayData.BlrTypeId, length, int(arrayData.Length), sdl)
	if err != nil {
		return nil, err
	}

	return array, err
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
