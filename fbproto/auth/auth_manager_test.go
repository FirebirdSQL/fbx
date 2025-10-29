package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	crypt "gitlab.com/nyarla/go-crypt"
)

func TestAuthManagerBuildListAndCurrent(t *testing.T) {
	m, err := NewAuthManager("user", "secret", nil)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Equal(t, "Srp256,Srp224,Srp384,Srp512,Srp,Gss,Legacy_Auth", m.BuildList())

	cur := m.GetCurrentPlugin()
	require.NotNil(t, cur)
	require.Equal(t, "Srp256", cur.Name())

	initial := m.BeginHandshake()
	require.NotNil(t, initial)
	require.Equal(t, cur.Name(), initial.PluginName)
	require.Equal(t, []string{"Srp256", "Srp224", "Srp384", "Srp512", "Srp", "Gss", "Legacy_Auth"}, initial.PluginList)
	require.NotEmpty(t, initial.SpecificData)
	assembled := make([]byte, 0)
	for idx, chunk := range initial.SpecificData {
		require.LessOrEqual(t, len(chunk.Data), 254)
		require.Equal(t, byte(idx), chunk.Part)
		assembled = append(assembled, chunk.Data...)
	}
	require.Equal(t, cur.InitialClientData(), assembled)

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

	m, err := NewAuthManager("user", "secret", nil)
	require.NoError(t, err)
	m.current = 999
	first := m.GetCurrentPlugin()
	require.Equal(t, m.plugins[0], first)
}

func TestSetCurrentPluginNotFound(t *testing.T) {
	m, err := NewAuthManager("user", "secret", nil)
	require.NoError(t, err)
	p, ok := m.SetCurrentPlugin("unknown_plugin")
	require.False(t, ok)
	require.Nil(t, p)
}

func TestAuthManagerHandleServerChallenge(t *testing.T) {
	m, err := NewAuthManager("user", "secret", nil)
	require.NoError(t, err)
	step, err := m.HandleServerChallenge("legacy_auth", nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.Equal(t, step.AuthSpecific, m.AuthSpecificData())
	require.Empty(t, m.SessionKey())
	require.Equal(t, "Legacy_Auth", m.GetCurrentPlugin().Name())
	require.NoError(t, m.HandleAuthResponse(nil))

	step, err = m.HandleServerChallenge("srp", nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.NotEmpty(t, step.Reply)
	require.Empty(t, step.AuthSpecific)
	require.Empty(t, m.AuthSpecificData())
	require.Empty(t, m.SessionKey())
}

func TestHandleServerChallengeUnsupportedPlugin(t *testing.T) {
	m, err := NewAuthManager("user", "secret", nil)
	require.NoError(t, err)
	_, err = m.HandleServerChallenge("unknown", nil)
	require.Error(t, err)
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

func TestNewAuthManagerCustomList(t *testing.T) {
	m, err := NewAuthManager("user", "secret", []string{"Legacy_Auth", "Srp", "Legacy_Auth"})
	require.NoError(t, err)
	require.Equal(t, "Legacy_Auth,Srp", m.BuildList())
	cur := m.GetCurrentPlugin()
	require.NotNil(t, cur)
	require.Equal(t, "Legacy_Auth", cur.Name())
}

func TestNewAuthManagerUnknownPlugin(t *testing.T) {
	_, err := NewAuthManager("user", "secret", []string{"unknown"})
	require.Error(t, err)
}
