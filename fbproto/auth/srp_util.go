package auth

func leftPad(b []byte, size int) []byte {
	if len(b) >= size {
		out := make([]byte, len(b))
		copy(out, b)
		return out
	}

	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

func padToPrimeLength(b []byte) []byte {
	return leftPad(b, (srpPrime.BitLen()+7)/8)
}
