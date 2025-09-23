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

	// By default, the current plugin is the first one
	cur := m.GetCurrentAuthPlugin()
	require.NotNil(t, cur)
	require.Equal(t, "Srp256", cur.GetName())

	// Switch by name case-insensitively
	p, ok := m.SetCurrentPlugin("srp")
	require.True(t, ok)
	require.Equal(t, "Srp", p.GetName())
	require.Equal(t, p, m.GetCurrentAuthPlugin())

	// Switch to Legacy_Auth
	p2, ok := m.SetCurrentPlugin("legacy_auth")
	require.True(t, ok)
	require.Equal(t, "Legacy_Auth", p2.GetName())
}

func TestAuthManagerEmptyAndBounds(t *testing.T) {
	var empty AuthManager
	require.Nil(t, empty.GetCurrentAuthPlugin())

	m := NewAuthManager("user", "secret")
	// Out-of-bounds index — reset to 0
	m.current = 999
	first := m.GetCurrentAuthPlugin()
	require.Equal(t, m.AuthPlugins[0], first)
}

func TestSetCurrentPluginNotFound(t *testing.T) {
	m := NewAuthManager("user", "secret")
	p, ok := m.SetCurrentPlugin("unknown_plugin")
	require.False(t, ok)
	require.Nil(t, p)
}

func TestLegacyAuthPluginClientData(t *testing.T) {
	pass := "masterkey"
	p := &LegacyAuthPlugin{Password: pass}
	data := p.GetClientData()
	expected := bytes.NewBufferString(crypt.Crypt(pass, "9z")[2:]).Bytes()
	require.Equal(t, expected, data)

	// Different password — different output
	p2 := &LegacyAuthPlugin{Password: "other"}
	require.NotEqual(t, data, p2.GetClientData())
}
