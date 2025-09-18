package auth

import (
	"encoding/hex"
	"io"
	"math/big"
)

func GetClientSeed() (public, private *big.Int) {
	pub, priv, err := GetClientSeedForVariant(srpNameClassic)
	if err != nil {
		panic(err)
	}
	return pub, priv
}

func GetClientSeedForVariant(name string) (public, private *big.Int, err error) {
	engine, err := newSRPEngine(name)
	if err != nil {
		return nil, nil, err
	}
	return engine.GenerateClientValues(nil)
}

func getSalt() []byte {
	salt, err := generateSalt()
	if err != nil {
		panic(err)
	}
	return salt
}

func generateSalt() ([]byte, error) {
	return newSRPSalt(nil)
}

func newSRPSalt(random io.Reader) ([]byte, error) {
	engine, err := newSRPEngine(srpNameClassic)
	if err != nil {
		return nil, err
	}
	return engine.GenerateSalt(random)
}

func getVerifier(user, password string, salt []byte) *big.Int {
	verifier, err := getVerifierForVariant(srpNameClassic, user, password, salt)
	if err != nil {
		panic(err)
	}
	return verifier
}

func getVerifierForVariant(name string, user, password string, salt []byte) (*big.Int, error) {
	engine, err := newSRPEngine(name)
	if err != nil {
		return nil, err
	}
	return engine.Verifier(user, password, salt), nil
}

func getServerSeed(verifier *big.Int) (public, private *big.Int) {
	pub, priv, err := getServerSeedForVariant(srpNameClassic, verifier)
	if err != nil {
		panic(err)
	}
	return pub, priv
}

func getServerSeedForVariant(name string, verifier *big.Int) (public, private *big.Int, err error) {
	engine, err := newSRPEngine(name)
	if err != nil {
		return nil, nil, err
	}
	return engine.GenerateServerValues(verifier, nil)
}

func getClientSession(user, password string, salt []byte, A, B, a *big.Int) []byte {
	key, err := getClientSessionForVariant(srpNameClassic, user, password, salt, A, B, a)
	if err != nil {
		panic(err)
	}
	return key
}

func getClientSessionForVariant(name string, user, password string, salt []byte, A, B, a *big.Int) ([]byte, error) {
	engine, err := newSRPEngine(name)
	if err != nil {
		return nil, err
	}
	return engine.ClientSessionKey(user, password, salt, A, B, a)
}

func getServerSession(user, password string, salt []byte, A, B, b *big.Int) []byte {
	key, err := getServerSessionForVariant(srpNameClassic, user, password, salt, A, B, b)
	if err != nil {
		panic(err)
	}
	return key
}

func getServerSessionForVariant(name string, user, password string, salt []byte, A, B, b *big.Int) ([]byte, error) {
	engine, err := newSRPEngine(name)
	if err != nil {
		return nil, err
	}
	return engine.ServerSessionKey(user, password, salt, A, B, b)
}

func getClientProof(user, password string, salt []byte, A, B, a *big.Int, pluginName string) (M, K []byte) {
	M, K, err := getClientProofForVariant(pluginName, user, password, salt, A, B, a)
	if err != nil {
		panic(err)
	}
	return M, K
}

func GetClientProof(user, password string, salt []byte, A, B, a *big.Int, pluginName string) (M, K []byte) {
	return getClientProof(user, password, salt, A, B, a, pluginName)
}

func BigIntToBytes(v *big.Int) []byte {
	if v == nil {
		return nil
	}
	return padToPrimeLength(v.Bytes())
}

func getClientProofForVariant(name, user, password string, salt []byte, A, B, a *big.Int) (M, K []byte, err error) {
	engine, err := newSRPEngine(name)
	if err != nil {
		return nil, nil, err
	}

	K, err = engine.ClientSessionKey(user, password, salt, A, B, a)
	if err != nil {
		return nil, nil, err
	}

	M = engine.ClientProof(user, salt, A, B, K)
	return M, K, nil
}

type SrpAuthPlugin struct {
	Name         string
	engine       *srpEngine
	ClientPublic *big.Int
	ClientSecret *big.Int
}

func NewSrpAuthPlugin() *SrpAuthPlugin {
	plugin, err := newSrpAuthPlugin(srpNameClassic)
	if err != nil {
		panic(err)
	}
	return plugin
}

func NewSrp256AuthPlugin() *SrpAuthPlugin {
	plugin, err := newSrpAuthPlugin(srpNameSHA256)
	if err != nil {
		panic(err)
	}
	return plugin
}

func newSrpAuthPlugin(name string) (*SrpAuthPlugin, error) {
	engine, err := newSRPEngine(name)
	if err != nil {
		return nil, err
	}

	public, private, err := engine.GenerateClientValues(nil)
	if err != nil {
		return nil, err
	}

	return &SrpAuthPlugin{
		Name:         name,
		engine:       engine,
		ClientPublic: public,
		ClientSecret: private,
	}, nil
}

func (p *SrpAuthPlugin) GetName() string {
	return p.Name
}

func (p *SrpAuthPlugin) GetClientData() []byte {
	return []byte(hex.EncodeToString(p.ClientPublic.Bytes()))
}

func (p *SrpAuthPlugin) SetServerData(data []byte) {
	_ = data
}
