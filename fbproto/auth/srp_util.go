package auth

import (
	"crypto/sha1"
	"math/big"
)

const srpKeySize = 128

func srpPad(v *big.Int) []byte {
	buf := make([]byte, srpKeySize)
	var m big.Int
	n := new(big.Int).Add(new(big.Int), v)

	for i := range buf {
		buf[i] = byte(m.And(m.SetInt64(255), n).Int64())
		n = n.Div(n, m.SetInt64(256))
	}

	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}

	i := 0
	for i < len(buf) && buf[i] == 0 {
		i++
	}
	return buf[i:]
}

func srpBytesToBigInt(v []byte) *big.Int {
	m := new(big.Int).SetInt64(256)
	a := new(big.Int)
	r := new(big.Int)
	for _, b := range v {
		r = r.Mul(r, m)
		r = r.Add(r, a.SetInt64(int64(b)))
	}
	return r
}

func srpBigIntToSha1(n *big.Int) []byte {
	h := sha1.New()
	h.Write(n.Bytes())
	return h.Sum(nil)
}

func srpStringHash(s string) *big.Int {
	h := sha1.New()
	h.Write([]byte(s))
	return srpBytesToBigInt(h.Sum(nil))
}

func srpUserHash(salt []byte, user, password string) *big.Int {
	h1 := sha1.New()
	h1.Write([]byte(user + ":" + password))
	h2 := sha1.New()
	h2.Write(salt)
	h2.Write(h1.Sum(nil))
	return srpBytesToBigInt(h2.Sum(nil))
}

// srpXorBytes was unused and removed.
