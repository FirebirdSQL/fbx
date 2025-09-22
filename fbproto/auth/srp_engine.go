package auth

import (
	crand "crypto/rand"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"math/big"

	"modernc.org/mathutil"
)

type srpEngine struct {
	variant    srpVariant
	prime      *big.Int
	generator  *big.Int
	multiplier *big.Int
}

func newSRPEngine(name string) (*srpEngine, error) {
	variant, ok := srpVariants[name]
	if !ok {
		return nil, fmt.Errorf("auth: unsupported SRP variant %q", name)
	}

	prime := new(big.Int).Set(srpPrime)
	generator := new(big.Int).Set(srpGenerator)
	multiplier := new(big.Int).Set(srpMultiplier)

	return &srpEngine{
		variant:    variant,
		prime:      prime,
		generator:  generator,
		multiplier: multiplier,
	}, nil
}

func (e *srpEngine) scramble(A, B *big.Int) *big.Int {
	h := sha1.New()
	h.Write(srpPad(A))
	h.Write(srpPad(B))
	return new(big.Int).SetBytes(h.Sum(nil))
}

func (e *srpEngine) userHash(user, password string, salt []byte) *big.Int {
	return srpUserHash(salt, user, password)
}

func (e *srpEngine) sessionKey(secret *big.Int) []byte {
	return srpBigIntToSha1(secret)
}

func (e *srpEngine) pow(base, exp *big.Int) *big.Int {
	baseMod := new(big.Int).Mod(base, e.prime)
	if baseMod.Sign() < 0 {
		baseMod.Add(baseMod, e.prime)
	}
	return new(big.Int).Exp(baseMod, exp, e.prime)
}

func (e *srpEngine) generatePrivate(random io.Reader) (*big.Int, error) {
	if random == nil {
		random = crand.Reader
	}

	limit := new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil)
	for {
		candidate, err := crand.Int(random, limit)
		if err != nil {
			return nil, err
		}
		if candidate.Sign() != 0 {
			return candidate, nil
		}
	}
}

func (e *srpEngine) GenerateClientValues(random io.Reader) (public, private *big.Int, err error) {
	private, err = e.generatePrivate(random)
	if err != nil {
		return nil, nil, err
	}
	return e.PublicClientKey(private), private, nil
}

func (e *srpEngine) GenerateServerValues(verifier *big.Int, random io.Reader) (public, private *big.Int, err error) {
	private, err = e.generatePrivate(random)
	if err != nil {
		return nil, nil, err
	}
	return e.PublicServerKey(verifier, private), private, nil
}

func (e *srpEngine) GenerateSalt(random io.Reader) ([]byte, error) {
	if random == nil {
		random = crand.Reader
	}

	salt := make([]byte, srpSaltSize)
	if _, err := io.ReadFull(random, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

func (e *srpEngine) PublicClientKey(private *big.Int) *big.Int {
	return e.pow(e.generator, private)
}

func (e *srpEngine) PublicServerKey(verifier, private *big.Int) *big.Int {
	gb := e.pow(e.generator, private)
	kv := new(big.Int).Mul(e.multiplier, verifier)
	kv.Mod(kv, e.prime)
	public := new(big.Int).Add(kv, gb)
	public.Mod(public, e.prime)
	return public
}

func (e *srpEngine) ClientSessionKey(user, password string, salt []byte, A, B, a *big.Int) ([]byte, error) {
	secret, err := e.clientSessionSecret(user, password, salt, A, B, a)
	if err != nil {
		return nil, err
	}
	return e.sessionKey(secret), nil
}

func (e *srpEngine) clientSessionSecret(user, password string, salt []byte, A, B, a *big.Int) (*big.Int, error) {
	if !isValidPublicKey(A, e.prime) || !isValidPublicKey(B, e.prime) {
		return nil, errInvalidPublicKey
	}

	u := e.scramble(A, B)
	if u.Sign() == 0 {
		return nil, errZeroScramble
	}

	x := e.userHash(user, password, salt)
	gx := e.pow(e.generator, x)
	kgx := new(big.Int).Mul(e.multiplier, gx)
	kgx.Mod(kgx, e.prime)

	base := new(big.Int).Sub(B, kgx)
	base.Mod(base, e.prime)
	if base.Sign() < 0 {
		base.Add(base, e.prime)
	}

	ux := new(big.Int).Mul(u, x)
	exponent := new(big.Int).Add(a, ux)
	exponent.Mod(exponent, e.prime)

	secret := e.pow(base, exponent)
	return secret, nil
}

func (e *srpEngine) ServerSessionKey(user, password string, salt []byte, A, B, b *big.Int) ([]byte, error) {
	secret, err := e.serverSessionSecret(user, password, salt, A, B, b)
	if err != nil {
		return nil, err
	}
	return e.sessionKey(secret), nil
}

func (e *srpEngine) serverSessionSecret(user, password string, salt []byte, A, B, b *big.Int) (*big.Int, error) {
	if !isValidPublicKey(A, e.prime) {
		return nil, errInvalidPublicKey
	}

	u := e.scramble(A, B)
	if u.Sign() == 0 {
		return nil, errZeroScramble
	}

	v := e.Verifier(user, password, salt)
	vu := e.pow(v, u)
	base := new(big.Int).Mul(A, vu)
	base.Mod(base, e.prime)

	secret := e.pow(base, b)
	return secret, nil
}

func (e *srpEngine) Verifier(user, password string, salt []byte) *big.Int {
	x := e.userHash(user, password, salt)
	return e.pow(e.generator, x)
}

func (e *srpEngine) ClientProof(user string, salt []byte, A, B *big.Int, K []byte) []byte {
	n1 := srpBytesToBigInt(srpBigIntToSha1(e.prime))
	n2 := srpBytesToBigInt(srpBigIntToSha1(e.generator))
	n3 := mathutil.ModPowBigInt(n1, n2, e.prime)
	n4 := srpStringHash(user)

	var digest hash.Hash = e.variant.hashFunc()
	toBytes := func(b *big.Int) []byte { return b.Bytes() }
	digest.Write(toBytes(n3))
	digest.Write(toBytes(n4))
	digest.Write(salt)
	digest.Write(A.Bytes())
	digest.Write(B.Bytes())
	digest.Write(K)
	return digest.Sum(nil)
}

func (e *srpEngine) ServerProof(A *big.Int, M, K []byte) []byte {
	digest := e.variant.hashFunc()
	digest.Write(srpPad(A))
	digest.Write(M)
	digest.Write(K)
	return digest.Sum(nil)
}

func isValidPublicKey(value, prime *big.Int) bool {
	mod := new(big.Int).Mod(value, prime)
	return mod.Sign() != 0
}
