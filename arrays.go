package fbx

import (
	"context"
	"encoding/binary"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
)

type array struct {
	TableName string
	FieldName string
	Bounds    []fbtype.Bound
	Length    uint32
	Scale     int32
	TypeId    uint32
	Metadata  map[string][]byte
}

func (ac *array) getSliceLength() int {
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

func (ac *array) generateSDL() []byte {
	sdl := []byte{}
	metadata := ac.Metadata
	bounds := ac.Bounds

	data_type := trimBytes(metadata["DATATYPE"], 1)[0]
	scale := trimBytes(metadata["SCALE"], 1)
	length := trimBytes(metadata["LENGTH"], 2)

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

func putArray(fbConn *fbconn.FbConn, ctx context.Context, arrayData *fbconn.Array, arr fbtype.Arr) ([]byte, error) {

	ac := new(array)
	ac.Length = arrayData.Length
	ac.Scale = arrayData.Scale
	ac.FieldName = arrayData.FieldName
	ac.TableName = arrayData.TableName
	ac.TypeId = arrayData.TypeId
	ac.Metadata = arrayData.Metadata

	ac.Bounds = arr.Bounds

	sdl := ac.generateSDL()
	length := ac.getSliceLength()

	id, err := fbConn.PutArray(ctx, arr.Buf, ac.TypeId, length, sdl)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func getArray(fbConn *fbconn.FbConn, ctx context.Context, arr fbconn.Array, arrId []byte) ([]byte, error) {
	if len(arrId) == 0 {
		return nil, nil
	}

	arrayId := int64(binary.BigEndian.Uint64(arrId))

	ac := new(array)
	ac.Length = arr.Length
	ac.TypeId = arr.TypeId
	ac.FieldName = arr.FieldName
	ac.TableName = arr.TableName
	ac.Metadata = arr.Metadata

	bounds := []fbtype.Bound{}
	for _, elem := range arr.Bounds {
		bounds = append(bounds, fbtype.Bound{Lower: elem["Lower"], Upper: elem["Upper"]})
	}
	ac.Bounds = bounds

	sdl := ac.generateSDL()

	length := ac.getSliceLength()

	array, err := fbConn.GetArray(ctx, arrayId, ac.TypeId, length, int(ac.Length), sdl)
	if err != nil {
		return nil, err
	}

	return array, err
}
