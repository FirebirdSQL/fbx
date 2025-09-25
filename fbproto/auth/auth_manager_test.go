package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	crypt "gitlab.com/nyarla/go-crypt"
)

func TestAuthManagerBuildListAndCurrent(t *testing.T) {
	m := NewAuthManager("user", "secret")
	require.NotNil(t, m)
	require.Equal(t, "Srp256,Srp224,Srp384,Srp512,Srp,Legacy_Auth", m.BuildList())

	cur := m.GetCurrentPlugin()
	require.NotNil(t, cur)
	require.Equal(t, "Srp256", cur.Name())

	plg, ok := m.SetCurrentPlugin("srp")
	require.True(t, ok)
	require.Equal(t, "Srp", plg.Name())
	require.Equal(t, plg, m.GetCurrentPlugin())

	legacy, ok := m.SetCurrentPlugin("legacy_auth")
	require.True(t, ok)
	require.Equal(t, "Legacy_Auth", legacy.Name())
}

func TestAuthManagerEmptyAndBounds(t *testing.T) {
	var empty AuthManager
	require.Nil(t, empty.GetCurrentPlugin())

	m := NewAuthManager("user", "secret")
	m.current = 999
	first := m.GetCurrentPlugin()
	require.Equal(t, m.plugins[0], first)
}

func TestSetCurrentPluginNotFound(t *testing.T) {
	m := NewAuthManager("user", "secret")
	p, ok := m.SetCurrentPlugin("unknown_plugin")
	require.False(t, ok)
	require.Nil(t, p)
}

func TestLegacyAuthPluginInitialClientData(t *testing.T) {
	pass := "masterkey"
	p := NewLegacyAuthPlugin(pass)
	data := p.InitialClientData()
	expected := bytes.NewBufferString(crypt.Crypt(pass, "9z")[2:]).Bytes()
	require.Equal(t, expected, data)

	p2 := NewLegacyAuthPlugin("other")
	require.NotEqual(t, data, p2.InitialClientData())
}
