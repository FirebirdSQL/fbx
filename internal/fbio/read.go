package fbio

import "encoding/binary"

func ReadUint16BE(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func ReadInt32BE(b []byte) int32 {
	return int32(binary.BigEndian.Uint32(b))
}

func ReadInt64BE(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func ReadInt16LE(b []byte) int16 {
	return int16(binary.LittleEndian.Uint16(b))
}

func ReadInt32LE(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b))
}

func ReadInt64LE(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}
