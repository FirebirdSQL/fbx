package auth

import (
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/red-soft/fbx/fbproto/utils"
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
	return srpPad(v)
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
	variant      string
	engine       *srpEngine
	ClientPublic *big.Int
	ClientSecret *big.Int
	lastProof    []byte
	lastSession  []byte
}

func NewSrpAuthPlugin() *SrpAuthPlugin {
	plg, err := newSrpAuthPlugin(srpNameClassic)
	if err != nil {
		panic(err)
	}
	return plg
}

func NewSrp256AuthPlugin() *SrpAuthPlugin {
	plg, err := newSrpAuthPlugin(srpNameSHA256)
	if err != nil {
		panic(err)
	}
	return plg
}

func NewSrp224AuthPlugin() *SrpAuthPlugin {
	plg, err := newSrpAuthPlugin(srpNameSHA224)
	if err != nil {
		panic(err)
	}
	return plg
}

func NewSrp384AuthPlugin() *SrpAuthPlugin {
	plg, err := newSrpAuthPlugin(srpNameSHA384)
	if err != nil {
		panic(err)
	}
	return plg
}

func NewSrp512AuthPlugin() *SrpAuthPlugin {
	plg, err := newSrpAuthPlugin(srpNameSHA512)
	if err != nil {
		panic(err)
	}
	return plg
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
		variant:      name,
		engine:       engine,
		ClientPublic: public,
		ClientSecret: private,
	}, nil
}

func (p *SrpAuthPlugin) Name() string {
	return p.variant
}

func (p *SrpAuthPlugin) Advertise() []string {
	return []string{p.variant}
}

func (p *SrpAuthPlugin) InitialClientData() []byte {
	return []byte(hex.EncodeToString(srpPad(p.ClientPublic)))
}

func (p *SrpAuthPlugin) HandleServerData(user string, password string, pluginName string, data []byte) (*AuthPluginStep, error) {
	if len(data) == 0 {
		p.lastProof = nil
		p.lastSession = nil
		reply := BigIntToBytes(p.ClientPublic)
		return &AuthPluginStep{Reply: cloneBytes(reply)}, nil
	}

	salt, serverHex, err := parseSrpServerData(data)
	if err != nil {
		return nil, err
	}

	serverPublic := utils.BigIntFromHexString(serverHex)
	M, K := GetClientProof(strings.ToUpper(user), password, salt, p.ClientPublic, serverPublic, p.ClientSecret, p.variant)
	p.lastProof = cloneBytes(M)
	p.lastSession = cloneBytes(K)
	return &AuthPluginStep{
		Reply:        cloneBytes(M),
		AuthSpecific: cloneBytes(M),
		SessionKey:   cloneBytes(K),
	}, nil
}

func (p *SrpAuthPlugin) HandleAuthResponse(data []byte) error {
	proof := extractSrpServerProof(data)
	if len(proof) == 0 {
		return nil
	}
	return p.verifyServerProof(proof)
}

func (p *SrpAuthPlugin) verifyServerProof(serverProof []byte) error {
	if len(serverProof) == 0 {
		return fmt.Errorf("auth: empty SRP server proof")
	}
	if len(p.lastProof) == 0 || len(p.lastSession) == 0 {
		return fmt.Errorf("auth: server proof received before client proof")
	}
	expected := p.engine.ServerProof(p.ClientPublic, p.lastProof, p.lastSession)
	if len(expected) != len(serverProof) {
		return fmt.Errorf("auth: invalid SRP server proof")
	}
	if subtle.ConstantTimeCompare(expected, serverProof) != 1 {
		return fmt.Errorf("auth: invalid SRP server proof")
	}
	return nil
}

func parseSrpServerData(data []byte) ([]byte, string, error) {
	offset := 0
	var salt []byte

	if len(data) >= 2 {
		sectionCount := int(binary.LittleEndian.Uint16(data[offset : offset+2]))
		if sectionCount > 0 && sectionCount <= 8 {
			offset += 2
			extendedSaltFound := false
			for i := 0; i < sectionCount; i++ {
				if offset+2 > len(data) {
					return nil, "", fmt.Errorf("invalid SRP payload")
				}
				length := int(binary.LittleEndian.Uint16(data[offset : offset+2]))
				offset += 2
				if length < 0 || offset+length > len(data) {
					return nil, "", fmt.Errorf("invalid SRP payload")
				}
				chunk := data[offset : offset+length]
				offset += length
				if len(chunk) > 0 && chunk[0] < 32 {
					chunk = chunk[1:]
				}
				if len(chunk) == 0 {
					continue
				}

				salt = append([]byte(nil), chunk...)
				extendedSaltFound = true
			}

			if extendedSaltFound {
				serverHex := string(data[offset:])
				if serverHex == "" {
					return nil, "", fmt.Errorf("invalid SRP server public")
				}
				return append([]byte(nil), salt...), serverHex, nil
			}

			offset = 0
		}
	}

	if len(data) < 2 {
		return nil, "", fmt.Errorf("invalid SRP payload")
	}

	saltLen := int(binary.LittleEndian.Uint16(data[offset : offset+2]))
	offset += 2
	if saltLen < 0 || offset+saltLen > len(data) {
		return nil, "", fmt.Errorf("invalid SRP payload")
	}
	salt = append([]byte(nil), data[offset:offset+saltLen]...)
	offset += saltLen

	if offset+2 <= len(data) {
		offset += 2
	}

	if offset > len(data) {
		return nil, "", fmt.Errorf("invalid SRP payload")
	}

	serverHex := string(data[offset:])
	if serverHex == "" {
		return nil, "", fmt.Errorf("invalid SRP server public")
	}

	return salt, serverHex, nil
}

func extractSrpServerProof(buf []byte) []byte {
	const serverProofTag = byte(2)
	for i := 0; i < len(buf); {
		if i+2 > len(buf) {
			break
		}
		tag := buf[i]
		i++
		length := int(buf[i])
		i++
		if length < 0 || i+length > len(buf) {
			break
		}
		value := buf[i : i+length]
		i += length
		if tag == serverProofTag {
			return cloneBytes(value)
		}
	}
	return nil
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
