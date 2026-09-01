package auth

// SRP engine (client-side).
//
// Notation (typical SRP-6 style):
// - N: large safe prime (1024-bit, see srp_config.go: srpPrime)
// - g: generator modulo N (2 in our case)
// - k: multiplier (fixed Firebird constant; see srpMultiplier)
// - s: salt (srpSaltSize bytes, random, provided by server)
// - a: client private ephemeral (128-bit random per Firebird)
// - b: server private ephemeral (server side)
// - A = g^a mod N: client public ephemeral (big.Int)
// - B = (k*v + g^b) mod N: server public ephemeral (big.Int)
// - u = SHA1(PAD(A) || PAD(B)): scramble parameter (Firebird uses SHA-1 here)
// - x = SHA1(s || SHA1(user ":" password)): password hash (SHA-1 here)
// - v = g^x mod N: verifier (server can store it instead of clear password)
// - S: shared secret computed differently by client and server (see below)
// - K = SHA1(S): session key bytes used for subsequent crypto
// - M: client proof, depends on SRP variant hash (SHA-1/224/256/384/512)
// - M2: server proof, same variant hash
//	http://srp.stanford.edu/design.html
//
// Byte/encoding conventions used throughout the code:
// - All big.Int values are treated modulo N and encoded big-endian.
// - PAD(x) means a left-padded big-endian encoding to the fixed SRP key size
//   (srpKeySize = 128 bytes for 1024-bit N). See srpPad() in srp_util.go.
// - The session key K is the SHA-1 digest of the shared secret S, yielding
//   20 bytes. The proof hashes (M, M2) may use a different hash depending on
//   the SRP variant (classic=SHA-1, Srp256=SHA-256, etc.).
// - The wire protocol for SRP in this driver uses hex-encoding for A (client
//   public) and also expects the server public B as an ASCII hex string.
//   See SrpAuthPlugin.GetClientData and parseSrpServerData for details.
//
// Handshake (client perspective):
// 1) Generate a 128-bit random private a and compute A = g^a mod N.
//    We send A to the server (hex-encoded PAD(A)).
// 2) Receive salt s (srpSaltSize bytes) and server public B (hex string).
// 3) Compute u = SHA1(PAD(A) || PAD(B)). If u == 0 => abort.
// 4) Compute x = SHA1(s || SHA1(user ":" password)).
// 5) Client shared secret: S = (B - k * g^x) ^ (a + u*x) mod N.
// 6) Session key: K = SHA1(S).
// 7) Client proof M. Firebird variant used here:
//      M = Hvar( (H(N) ^ H(g) mod N) || H(user) || s || A || B || K )
//    where H(N), H(g), H(user) are SHA-1 digests converted to big.Int and
//    Hvar is the variant hash (SHA-1/224/256/384/512). Note: Unlike the
//    usual SRP-6a definition that uses XOR(H(N), H(g)), Firebird computes
//    ModPowBigInt(H(N), H(g), N) for the first term — this is intentional
//    and matches the server.
// 8) Server proof (checked by client when received):
//      M2 = Hvar( PAD(A) || M || K ).

import (
	crand "crypto/rand"
	"crypto/sha1"
	"fmt"
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

// scramble returns u = SHA1(PAD(A) || PAD(B)).
// Firebird uses SHA-1 here regardless of the variant; the variant hash
// applies to the final proof digests (M/M2) only.
func (e *srpEngine) scramble(A, B *big.Int) *big.Int {
	h := sha1.New()
	h.Write(srpPad(A))
	h.Write(srpPad(B))
	return new(big.Int).SetBytes(h.Sum(nil))
}

// userHash returns x = SHA1(s || SHA1(user ":" password)).
// Note: SHA-1 is used here per Firebird SRP compatibility.
func (e *srpEngine) userHash(user, password string, salt []byte) *big.Int {
	return srpUserHash(salt, user, password)
}

// sessionKey derives K = SHA1(S). K is a byte slice of length 20.
func (e *srpEngine) sessionKey(secret *big.Int) []byte {
	return srpBigIntToSha1(secret)
}

// pow computes (base mod N) ^ exp mod N with proper normalization.
func (e *srpEngine) pow(base, exp *big.Int) *big.Int {
	baseMod := new(big.Int).Mod(base, e.prime)
	if baseMod.Sign() < 0 {
		baseMod.Add(baseMod, e.prime)
	}
	return new(big.Int).Exp(baseMod, exp, e.prime)
}

// generatePrivate produces a non-zero 128-bit random private exponent
// as required by the Firebird SRP implementation.
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

// GenerateClientValues returns (A, a), where a is a fresh private exponent and
// A = g^a mod N. A is later hex-encoded PAD(A) for the wire.
func (e *srpEngine) GenerateClientValues(random io.Reader) (public, private *big.Int, err error) {
	private, err = e.generatePrivate(random)
	if err != nil {
		return nil, nil, err
	}
	return e.PublicClientKey(private), private, nil
}

// GenerateServerValues returns (B, b) for testing/compatibility utilities,
// where B = (k*v + g^b) mod N for a given verifier v.
func (e *srpEngine) GenerateServerValues(verifier *big.Int, random io.Reader) (public, private *big.Int, err error) {
	private, err = e.generatePrivate(random)
	if err != nil {
		return nil, nil, err
	}
	return e.PublicServerKey(verifier, private), private, nil
}

// GenerateSalt returns a fresh random salt s (srpSaltSize bytes).
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

// PublicClientKey computes A = g^a mod N.
func (e *srpEngine) PublicClientKey(private *big.Int) *big.Int {
	return e.pow(e.generator, private)
}

// PublicServerKey computes B = (k*v + g^b) mod N.
func (e *srpEngine) PublicServerKey(verifier, private *big.Int) *big.Int {
	gb := e.pow(e.generator, private)
	kv := new(big.Int).Mul(e.multiplier, verifier)
	kv.Mod(kv, e.prime)
	public := new(big.Int).Add(kv, gb)
	public.Mod(public, e.prime)
	return public
}

// ClientSessionKey derives K for the client given inputs from the handshake.
// Inputs:
// - user: username (Firebird expects it uppercase on the wire; see plugin)
// - password: user password
// - salt: s received from server (srpSaltSize bytes)
// - A: client public (g^a mod N)
// - B: server public
// - a: client private
// Output:
// - K: 20-byte session key = SHA1(S).
func (e *srpEngine) ClientSessionKey(user, password string, salt []byte, A, B, a *big.Int) ([]byte, error) {
	secret, err := e.clientSessionSecret(user, password, salt, A, B, a)
	if err != nil {
		return nil, err
	}
	return e.sessionKey(secret), nil
}

// clientSessionSecret computes the shared secret
//
//	S = (B - k * g^x) ^ (a + u*x) mod N
//
// with checks for valid public keys and non-zero scramble u.
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

// ServerSessionKey is the server-side analogue for tests/utilities.
// It derives K from inputs as the server would do.
func (e *srpEngine) ServerSessionKey(user, password string, salt []byte, A, B, b *big.Int) ([]byte, error) {
	secret, err := e.serverSessionSecret(user, password, salt, A, B, b)
	if err != nil {
		return nil, err
	}
	return e.sessionKey(secret), nil
}

// serverSessionSecret computes the shared secret
//
//	S = (A * v^u) ^ b mod N
//
// with checks for valid public key A and non-zero scramble u.
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

// Verifier computes v = g^x mod N for storage/derivation on the server.
func (e *srpEngine) Verifier(user, password string, salt []byte) *big.Int {
	x := e.userHash(user, password, salt)
	return e.pow(e.generator, x)
}

// ClientProof constructs the Firebird SRP client proof M.
// It uses SHA-1 for H(N), H(g), H(user) (converted to big.Int), but the final
// digest uses the variant hash function.
//
// M = Hvar( (H(N) ^ H(g) mod N) || H(user) || s || A || B || K )
//
// All big.Int are written as raw big-endian bytes (no PAD) except A is already
// assumed to be computed with the same N; the server-side matches this format.
func (e *srpEngine) ClientProof(user string, salt []byte, A, B *big.Int, K []byte) []byte {
	n1 := srpBytesToBigInt(srpBigIntToSha1(e.prime))
	n2 := srpBytesToBigInt(srpBigIntToSha1(e.generator))
	n3 := mathutil.ModPowBigInt(n1, n2, e.prime)
	n4 := srpStringHash(user)

	digest := e.variant.hashFunc()
	toBytes := func(b *big.Int) []byte { return b.Bytes() }
	digest.Write(toBytes(n3))
	digest.Write(toBytes(n4))
	digest.Write(salt)
	digest.Write(A.Bytes())
	digest.Write(B.Bytes())
	digest.Write(K)
	return digest.Sum(nil)
}

// ServerProof constructs the server proof M2 = Hvar( PAD(A) || M || K ).
// The client verifies M2 to confirm the server also derived the same key.
func (e *srpEngine) ServerProof(A *big.Int, M, K []byte) []byte {
	digest := e.variant.hashFunc()
	digest.Write(srpPad(A))
	digest.Write(M)
	digest.Write(K)
	return digest.Sum(nil)
}

// isValidPublicKey checks that value % N != 0, as required by SRP.
func isValidPublicKey(value, prime *big.Int) bool {
	mod := new(big.Int).Mod(value, prime)
	return mod.Sign() != 0
}
