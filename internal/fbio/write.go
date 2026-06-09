package fbio

func AppendUint16BE(buf []byte, n uint16) []byte {
	return append(buf, byte(n>>8), byte(n))
}

func AppendUint32BE(buf []byte, n uint32) []byte {
	return append(buf, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func AppendUint64BE(buf []byte, n uint64) []byte {
	return append(buf,
		byte(n>>56), byte(n>>48), byte(n>>40), byte(n>>32),
		byte(n>>24), byte(n>>16), byte(n>>8), byte(n),
	)
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
	return append(buf, byte(n), byte(n>>8))
}

func AppendInt32LE(buf []byte, n int32) []byte {
	return append(buf, byte(n), byte(n>>8), byte(n>>16), byte(n>>24))
}

func AppendInt64LE(buf []byte, n int64) []byte {
	return append(buf,
		byte(n), byte(n>>8), byte(n>>16), byte(n>>24),
		byte(n>>32), byte(n>>40), byte(n>>48), byte(n>>56),
	)
}
