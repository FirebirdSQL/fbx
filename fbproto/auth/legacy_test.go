package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	crypt "gitlab.com/nyarla/go-crypt"
)

func TestLegacyAuthPluginName(t *testing.T) {
	p := NewLegacyAuthPlugin("")
	require.Equal(t, "Legacy_Auth", p.Name())
}

func TestLegacyAuthPluginClientData(t *testing.T) {
	pass := "masterkey"
	p := NewLegacyAuthPlugin(pass)
	data := p.InitialClientData()
	expected := bytes.NewBufferString(crypt.Crypt(pass, "9z")[2:]).Bytes()
	require.Equal(t, expected, data)
}

func TestLegacyAuthPluginHandleServerData(t *testing.T) {
	pass := "p@ssw0rd"
	p := NewLegacyAuthPlugin(pass)
	step, err := p.HandleServerData("user", pass, "Legacy_Auth", []byte{1, 2, 3})
	require.NoError(t, err)
	expected := bytes.NewBufferString(crypt.Crypt(pass, "9z")[2:]).Bytes()
	require.Equal(t, expected, step.Reply)
	require.Equal(t, expected, step.AuthSpecific)
	require.Nil(t, step.SessionKey)
}
