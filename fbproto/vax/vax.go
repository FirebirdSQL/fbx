package vax

func Integer2(buf []byte, off int) int {
	return int(buf[off]&0xFF) | (int(buf[off+1]&0xFF) << 8)
}

func Integer(buf []byte, off int, len int) int {
	if len > 4 {
		return 0
	}

	value := 0
	shift := 0
	offset := off

	for ; len > 0; len-- {
		value += int(buf[offset]&0xFF) << shift
		shift += 8
		offset++
	}
	return value
}
