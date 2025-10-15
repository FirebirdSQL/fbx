package auth

import (
	"bytes"

	crypt "gitlab.com/nyarla/go-crypt"
)

type LegacyAuthPlugin struct {
	Password string
}

func NewLegacyAuthPlugin(password string) *LegacyAuthPlugin {
	return &LegacyAuthPlugin{Password: password}
}

func (p *LegacyAuthPlugin) Name() string {
	return "Legacy_Auth"
}

func (p *LegacyAuthPlugin) Advertise() []string {
	return []string{p.Name()}
}

func (p *LegacyAuthPlugin) InitialClientData() []byte {
	return bytes.NewBufferString(crypt.Crypt(p.Password, "9z")[2:]).Bytes()
}

func (p *LegacyAuthPlugin) HandleServerData(user string, password string, pluginName string, data []byte) (*AuthPluginStep, error) {
	resp := p.InitialClientData()
	if len(resp) == 0 {
		return &AuthPluginStep{}, nil
	}
	return &AuthPluginStep{Reply: resp, AuthSpecific: cloneBytes(resp)}, nil
}

func (p *LegacyAuthPlugin) HandleAuthResponse(data []byte) error {
	return nil
}
