package auth

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"modernc.org/mathutil"
)

const (
	testClientPrivateHex = "60975527035CF2AD1989806F0407210BC81EDC04E2762A56AFD529DDDA2D4393"
	testServerPrivateHex = "E487CB59D31AC550471E81F0344C5948FCCA6D4182380E0AAC7F2B06FDE03664"
	testSaltHex          = "BEB25379D1A8581EB5A727673A2441EEBEB25379D1A8581EB5A727673A2441EE"
)

func TestSRPKeyAgreementVariants(t *testing.T) {
	user := "SYSDBA"
	password := "masterkey"
	salt := mustDecodeHex(t, testSaltHex)
	clientPrivate := mustBigIntFromHex(testClientPrivateHex)
	serverPrivate := mustBigIntFromHex(testServerPrivateHex)

	for _, variant := range []string{srpNameClassic, srpNameSHA224, srpNameSHA256, srpNameSHA384, srpNameSHA512} {
		variant := variant
		t.Run(variant, func(t *testing.T) {
			t.Parallel()

			engine, err := newSRPEngine(variant)
			require.NoError(t, err)

			clientPublic := engine.PublicClientKey(clientPrivate)
			verifier := engine.Verifier(user, password, salt)
			serverPublic := engine.PublicServerKey(verifier, serverPrivate)

			clientKey, err := engine.ClientSessionKey(user, password, salt, clientPublic, serverPublic, clientPrivate)
			require.NoError(t, err)

			srvKey, err := engine.ServerSessionKey(user, password, salt, clientPublic, serverPublic, serverPrivate)
			require.NoError(t, err)
			require.Equal(t, srvKey, clientKey)

			clientProof := engine.ClientProof(user, salt, clientPublic, serverPublic, clientKey)
			require.Equal(t,
				manualClientProof(engine, user, salt, clientPublic, serverPublic, clientKey),
				clientProof,
			)

			serverProof := engine.ServerProof(clientPublic, clientProof, clientKey)
			require.Equal(t,
				manualServerProof(engine, clientPublic, clientProof, clientKey),
				serverProof,
			)

			derivedProof, derivedKey, err := getClientProofForVariant(variant, user, password, salt, clientPublic, serverPublic, clientPrivate)
			require.NoError(t, err)
			require.Equal(t, clientProof, derivedProof)
			require.Equal(t, clientKey, derivedKey)

			srvDerived, err := getServerSessionForVariant(variant, user, password, salt, clientPublic, serverPublic, serverPrivate)
			require.NoError(t, err)
			require.Equal(t, srvKey, srvDerived)
		})
	}
}

func TestGetClientProofForVariantErrors(t *testing.T) {
	clientPublic := mustBigIntFromHex(testClientPrivateHex)
	serverPublic := mustBigIntFromHex(testServerPrivateHex)
	salt := mustDecodeHex(t, testSaltHex)

	_, _, err := getClientProofForVariant("unsupported", "user", "password", salt, clientPublic, serverPublic, clientPublic)
	require.Error(t, err)
}

func TestGenerateSaltLength(t *testing.T) {
	salt, err := generateSalt()
	require.NoError(t, err)
	require.Len(t, salt, srpSaltSize)
}

// manualClientProof повторяет алгоритм ClientProof() из srp_engine.go
func manualClientProof(engine *srpEngine, user string, salt []byte, A, B *big.Int, K []byte) []byte {
	n1 := srpBytesToBigInt(srpBigIntToSha1(engine.prime))
	n2 := srpBytesToBigInt(srpBigIntToSha1(engine.generator))
	n3 := mathutil.ModPowBigInt(n1, n2, engine.prime)
	n4 := srpStringHash(user)

	digest := engine.variant.hashFunc()
	digest.Write(n3.Bytes())
	digest.Write(n4.Bytes())
	digest.Write(salt)
	digest.Write(A.Bytes())
	digest.Write(B.Bytes())
	digest.Write(K)
	return digest.Sum(nil)
}

// manualServerProof повторяет алгоритм ServerProof() из srp_engine.go
func manualServerProof(engine *srpEngine, A *big.Int, M, K []byte) []byte {
	digest := engine.variant.hashFunc()
	digest.Write(srpPad(A))
	digest.Write(M)
	digest.Write(K)
	return digest.Sum(nil)
}

func mustDecodeHex(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("invalid hex %q: %v", s, err)
	}
	return b
}
