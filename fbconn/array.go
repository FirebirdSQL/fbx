package fbconn

import (
	"context"
	"encoding/binary"

	"rdb.red-soft.ru/fbx/fbproto"
)

type Array struct {
	TableName  string
	FieldName  string
	Dimensions []map[string]int
	Length     uint32
	Scale      int32
	TypeId     uint32
	SubType    int32
	Metadata   map[string][]byte
}

func (ac *Array) getSliceLength() int {
	var elements int = 1
	for _, elem := range ac.Dimensions {
		elements *= elem["Upper"] - elem["Lower"] + 1
	}

	length := elements * int(ac.Length)

	switch ac.TypeId {
	case fbproto.Isc_blr_varaying, fbproto.Isc_blr_varaying2:
		length += elements * 2
	}
	return length
}

func (ac *Array) generateSDL() []byte {
	sdl := []byte{}
	metadata := ac.Metadata
	dimentions := ac.Dimensions

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

	for i, b := range dimentions {
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
			sdl = append(sdl, getLiteral(b["Lower"])...)
		}
		sdl = append(sdl, getLiteral(b["Upper"])...)
	}

	sdl = append(sdl, []byte{
		fbproto.Isc_sdl_element,
		1,
		fbproto.Isc_sdl_scalar,
		0,
		byte(len(dimentions)),
	}...)

	for i := 0; i < len(dimentions); i++ {
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

func PutArray(fbConn *FbConn, ctx context.Context, arrayData *Array, arr []byte) ([]byte, error) {

	sdl := arrayData.generateSDL()
	length := arrayData.getSliceLength()

	id, err := fbConn.putArray(ctx, arr, arrayData.TypeId, length, sdl)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func GetArray(fbConn *FbConn, ctx context.Context, arrayData Array, arrId []byte) ([]byte, error) {
	if len(arrId) == 0 {
		return nil, nil
	}

	arrayId := int64(binary.BigEndian.Uint64(arrId))

	sdl := arrayData.generateSDL()
	length := arrayData.getSliceLength()

	array, err := fbConn.getArray(ctx, arrayId, arrayData.TypeId, length, int(arrayData.Length), sdl)
	if err != nil {
		return nil, err
	}

	return array, err
}
