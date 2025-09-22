package auth

import (
	"bytes"
	"gitlab.com/nyarla/go-crypt"
)

type LegacyAuthPlugin struct {
	Password string
}

func (p *LegacyAuthPlugin) GetName() string {
	return "Legacy_Auth"
}

func (p *LegacyAuthPlugin) GetClientData() []byte {
	return bytes.NewBufferString(crypt.Crypt(p.Password, "9z")[2:]).Bytes()
}

func (p *LegacyAuthPlugin) SetServerData(user string, password string, pluginName string, data []byte) (response []byte, authSpecific []byte, sessionKey []byte, err error) {
	// Legacy auth uses scrambled password as client response; no session key.
	resp := p.GetClientData()
	if len(resp) > 0 {
		copyResp := make([]byte, len(resp))
		copy(copyResp, resp)
		return copyResp, copyResp, nil, nil
	}
	return nil, nil, nil, nil
}
