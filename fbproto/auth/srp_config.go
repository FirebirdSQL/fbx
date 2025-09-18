package auth

import (
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"math/big"
)

const (
	srpSaltSize = 32
)

var (
	srpPrimeHex  = "E67D2E994B2F900C3F41F08F5BB2627ED0D49EE1FE767A52EFCD565CD6E768812C3E1E9CE8F0A8BEA6CB13CD29DDEBF7A96D4A93B55D488DF099A15C89DCB0640738EB2CBDD9A8F7BAB561AB1B0DC1C6CDABF303264A08D1BCA932D1F1EE428B619D970F342ABA9A65793B8B2F041AE5364350C16F735F56ECBCA87BD57B29E7"
	srpPrime     = mustBigIntFromHex(srpPrimeHex)
	srpGenerator = big.NewInt(2)

	errInvalidPublicKey = errors.New("auth: invalid SRP public key")
	errZeroScramble     = errors.New("auth: zero SRP scramble value")
)

const (
	srpNameClassic = "Srp"
	srpNameSHA256  = "Srp256"
)

type srpVariant struct {
	name     string
	hashFunc func() hash.Hash
}

var srpVariants = map[string]srpVariant{
	srpNameClassic: {name: srpNameClassic, hashFunc: sha1.New},
	srpNameSHA256:  {name: srpNameSHA256, hashFunc: sha256.New},
}

func mustBigIntFromHex(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic(fmt.Sprintf("auth: invalid hex big.Int literal %q", s))
	}
	return n
}
