package auth

import (
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

func NewSrp224AuthPlugin() *SrpAuthPlugin {
	plugin, err := newSrpAuthPlugin(srpNameSHA224)
	if err != nil {
		panic(err)
	}
	return plugin
}

func NewSrp384AuthPlugin() *SrpAuthPlugin {
	plugin, err := newSrpAuthPlugin(srpNameSHA384)
	if err != nil {
		panic(err)
	}
	return plugin
}

func NewSrp512AuthPlugin() *SrpAuthPlugin {
	plugin, err := newSrpAuthPlugin(srpNameSHA512)
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
	// For SRP, the initial specific auth data is the client's public value (A)
	// encoded as hex string bytes as required by wire protocol.
	return []byte(hex.EncodeToString(srpPad(p.ClientPublic)))
}

// SetServerData handles SRP server challenges. It accepts either empty data
// (server requests client public again) or a payload containing salt and
// server public key. It returns response to send back, authSpecific DPB data
// (hex-encoded proof), and the negotiated session key.
func (p *SrpAuthPlugin) SetServerData(user string, password string, pluginName string, data []byte) (response []byte, authSpecific []byte, sessionKey []byte, err error) {
	if len(data) == 0 {
		// Server is asking for client public value again
		return BigIntToBytes(p.ClientPublic), nil, nil, nil
	}

	salt, serverHex, err := parseSrpServerData(data)
	if err != nil {
		return nil, nil, nil, err
	}

	serverPublic := utils.BigIntFromHexString(serverHex)
	// Compute client proof and session key
	M, K := GetClientProof(strings.ToUpper(user), password, salt, p.ClientPublic, serverPublic, p.ClientSecret, pluginName)
	// Response to send to server is raw proof M
	// Store raw proof in authSpecific; it will be hex-encoded once when
	// building DPB specific_auth_data.
	proofCopy := make([]byte, len(M))
	copy(proofCopy, M)
	return proofCopy, proofCopy, K, nil
}

// parseSrpServerData mirrors the logic of pgconn.parseSrpPayload but is kept
// here to avoid package dependency in the plugin layer.
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
