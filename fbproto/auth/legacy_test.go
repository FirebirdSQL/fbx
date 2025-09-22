package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	crypt "gitlab.com/nyarla/go-crypt"
)

func TestLegacyAuthGetName(t *testing.T) {
	p := &LegacyAuthPlugin{}
	require.Equal(t, "Legacy_Auth", p.GetName())
}

func TestLegacyAuthClientData(t *testing.T) {
	pass := "masterkey"
	p := &LegacyAuthPlugin{Password: pass}
	data := p.GetClientData()
	expected := bytes.NewBufferString(crypt.Crypt(pass, "9z")[2:]).Bytes()
	require.Equal(t, expected, data)
}

func TestLegacyAuthSetServerDataNoop(t *testing.T) {
	pass := "p@ssw0rd"
	p := &LegacyAuthPlugin{Password: pass}
	before := p.GetClientData()

	// must not panic and must not change behaviour/state
	_, _, _, err := p.SetServerData("user", pass, "Legacy_Auth", []byte{1, 2, 3})
	require.NoError(t, err)

	after := p.GetClientData()
	require.Equal(t, pass, p.Password)
	require.Equal(t, before, after)
}
