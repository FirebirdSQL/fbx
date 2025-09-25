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

func TestLegacyAuthPluginAdvertise(t *testing.T) {
	p := NewLegacyAuthPlugin("secret")
	require.Equal(t, []string{"Legacy_Auth"}, p.Advertise())
}

func TestLegacyAuthPluginInitialClientDataClone(t *testing.T) {
	p := NewLegacyAuthPlugin("masterkey")
	first := p.InitialClientData()
	require.NotEmpty(t, first)
	second := p.InitialClientData()
	require.Equal(t, first, second)
	secondSnapshot := append([]byte(nil), second...)
	first[0] ^= 0xFF
	require.Equal(t, secondSnapshot, second)
	third := p.InitialClientData()
	require.Equal(t, secondSnapshot, third)
}

func TestLegacyAuthPluginHandleServerDataIsolatedBuffers(t *testing.T) {
	pass := "distinct"
	p := NewLegacyAuthPlugin(pass)
	step, err := p.HandleServerData("user", pass, p.Name(), nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.NotEmpty(t, step.Reply)
	require.NotEmpty(t, step.AuthSpecific)
	require.Nil(t, step.SessionKey)
	authCopy := append([]byte(nil), step.AuthSpecific...)
	step.Reply[0] ^= 0xFF
	require.Equal(t, authCopy, step.AuthSpecific)
	step, err = p.HandleServerData("user", pass, p.Name(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, step.Reply)
	require.NotEmpty(t, step.AuthSpecific)
	replyCopy := append([]byte(nil), step.Reply...)
	step.AuthSpecific[0] ^= 0xFF
	require.Equal(t, replyCopy, step.Reply)
}

func TestLegacyAuthPluginHandleAuthResponse(t *testing.T) {
	p := NewLegacyAuthPlugin("secret")
	require.NoError(t, p.HandleAuthResponse([]byte{1, 2, 3}))
}
