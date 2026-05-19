package fbio

import "encoding/binary"

func AppendUint16BE(buf []byte, n uint16) []byte {
	wp := len(buf)
	buf = append(buf, 0, 0)
	binary.BigEndian.PutUint16(buf[wp:], n)
	return buf
}

func AppendUint32BE(buf []byte, n uint32) []byte {
	wp := len(buf)
	buf = append(buf, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(buf[wp:], n)
	return buf
}

func AppendUint64BE(buf []byte, n uint64) []byte {
	wp := len(buf)
	buf = append(buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.BigEndian.PutUint64(buf[wp:], n)
	return buf
}

func AppendInt16BE(buf []byte, n int16) []byte {
	return AppendUint16BE(buf, uint16(n))
}

func AppendInt32BE(buf []byte, n int32) []byte {
	return AppendUint32BE(buf, uint32(n))
}

func AppendInt64BE(buf []byte, n int64) []byte {
	return AppendUint64BE(buf, uint64(n))
}

func AppendInt16LE(buf []byte, n int16) []byte {
	wp := len(buf)
	buf = append(buf, 0, 0)
	binary.LittleEndian.PutUint16(buf[wp:], uint16(n))
	return buf
}

func AppendInt32LE(buf []byte, n int32) []byte {
	wp := len(buf)
	buf = append(buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(buf[wp:], uint32(n))
	return buf
}

func AppendInt64LE(buf []byte, n int64) []byte {
	wp := len(buf)
	buf = append(buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(buf[wp:], uint64(n))
	return buf
}
