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

func (p *LegacyAuthPlugin) SetServerData(data []byte) {
}
