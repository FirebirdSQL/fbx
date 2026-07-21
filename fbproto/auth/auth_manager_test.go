package auth

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	crypt "gitlab.com/nyarla/go-crypt"
)

func TestAuthManagerBuildListAndCurrent(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, nil)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Equal(t, "Srp256,Srp,Gss,Srp224,Srp384,Srp512,Legacy_Auth", m.BuildList())

	cur := m.GetCurrentPlugin()
	require.NotNil(t, cur)
	require.Equal(t, "Srp256", cur.Name())

	initial := m.BeginHandshake()
	require.NotNil(t, initial)
	require.Equal(t, cur.Name(), initial.PluginName)
	require.Equal(t, []string{"Srp256", "Srp", "Gss", "Srp224", "Srp384", "Srp512", "Legacy_Auth"}, initial.PluginList)
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

	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, nil)
	require.NoError(t, err)
	m.current = 999
	first := m.GetCurrentPlugin()
	require.Equal(t, m.plugins[0], first)
}

func TestSetCurrentPluginNotFound(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, nil)
	require.NoError(t, err)
	p, ok := m.SetCurrentPlugin("unknown_plugin")
	require.False(t, ok)
	require.Nil(t, p)
}

func TestAuthManagerHandleServerChallenge(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, nil)
	require.NoError(t, err)
	legacyName := "Legacy_Auth"
	m.SetCurrentPlugin(legacyName)
	step, err := m.HandleServerChallenge(legacyName, nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.Equal(t, step.AuthSpecific, m.AuthSpecificData())
	require.Empty(t, m.SessionKey())
	require.Equal(t, legacyName, m.GetCurrentPlugin().Name())
	require.NoError(t, m.HandleAuthResponse(nil))

	step, err = m.HandleServerChallenge("srp", nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.NotEmpty(t, step.Reply)
	require.Empty(t, step.AuthSpecific)
	require.Equal(t, step.Reply, m.AuthSpecificData())
	require.Empty(t, m.SessionKey())
}

func TestAuthManagerStoresWireReplyForAuthSpecificData(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, []string{"Srp"})
	require.NoError(t, err)

	plugin := m.GetCurrentPlugin().(*SrpAuthPlugin)
	salt := []byte(testSaltHex)
	serverPrivate := mustBigIntFromHex(testServerPrivateHex)
	verifier := plugin.engine.Verifier("SYSDBA", "masterkey", salt)
	serverPublic := plugin.engine.PublicServerKey(verifier, serverPrivate)

	step, err := m.HandleServerChallenge("Srp", mustAuthManagerSrpServerData(t, salt, serverPublic))
	require.NoError(t, err)
	require.NotEmpty(t, step.AuthSpecific)
	require.NotEqual(t, step.AuthSpecific, step.Reply)
	require.Equal(t, step.Reply, m.AuthSpecificData())
	require.NotEmpty(t, m.SessionKey())
}

func TestHandleServerChallengeUnsupportedPlugin(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, nil)
	require.NoError(t, err)
	_, err = m.HandleServerChallenge("unknown", nil)
	require.Error(t, err)
}

func TestAuthManagerRestrictsToServerKnownPlugins(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, nil)
	require.NoError(t, err)

	require.NoError(t, m.RestrictToServerPlugins([]byte{2, 3, 'S', 'r', 'p'}))
	require.Equal(t, "Srp", m.BuildList())
	require.Equal(t, "Srp", m.GetCurrentPlugin().Name())

	initial := m.BeginHandshake()
	require.NotNil(t, initial)
	require.Equal(t, "Srp", initial.PluginName)
	require.Equal(t, []string{"Srp"}, initial.PluginList)
}

func TestAuthManagerRestrictToServerKnownPluginsNoMatch(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, []string{"Srp"})
	require.NoError(t, err)

	err = m.RestrictToServerPlugins([]byte{2, 11, 'L', 'e', 'g', 'a', 'c', 'y', '_', 'A', 'u', 't', 'h'})
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
	userConfig := UserConfig{User: "user", Password: "secret"}
	m, err := NewAuthManager(userConfig, []string{"Legacy_Auth", "Srp", "Legacy_Auth"})
	require.NoError(t, err)
	require.Equal(t, "Legacy_Auth,Srp", m.BuildList())
	cur := m.GetCurrentPlugin()
	require.NotNil(t, cur)
	require.Equal(t, "Legacy_Auth", cur.Name())
}

func TestNewAuthManagerUnknownPlugin(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "secret"}
	_, err := NewAuthManager(userConfig, []string{"unknown"})
	require.Error(t, err)
}

func mustAuthManagerSrpServerData(t *testing.T, salt []byte, serverPublic *big.Int) []byte {
	t.Helper()

	serverHex := []byte(serverPublic.Text(16))
	data := make([]byte, 0, 4+len(salt)+len(serverHex))
	data = binary.LittleEndian.AppendUint16(data, uint16(len(salt)))
	data = append(data, salt...)
	data = binary.LittleEndian.AppendUint16(data, uint16(len(serverHex)))
	data = append(data, serverHex...)
	return data
}

func TestHandleServerChallengeProcessesGssDataWhenPluginChanges(t *testing.T) {
	userConfig := UserConfig{User: "user", Password: "masterkey"}
	m, err := NewAuthManager(userConfig, []string{"Srp256", "Gss"})
	require.NoError(t, err)
	require.Equal(t, "Srp256", m.GetCurrentPlugin().Name())

	plugin := m.plugins[0].(*SrpAuthPlugin)
	salt := []byte(testSaltHex)
	serverPrivate := mustBigIntFromHex(testServerPrivateHex)
	verifier := plugin.engine.Verifier("SYSDBA", "masterkey", salt)
	serverPublic := plugin.engine.PublicServerKey(verifier, serverPrivate)
	serverData := mustAuthManagerSrpServerData(t, salt, serverPublic)

	data := []byte{9, 0}
	data = append(data, []byte("localhost")...)
	data = append(data, 6, 0)
	data = append(data, []byte("red_db")...)

	prev := getGssTokenProvider()
	mock := &mockGssProvider{token: []byte{1, 2, 3, 4}}
	SetGssTokenProvider(mock)
	t.Cleanup(func() { SetGssTokenProvider(prev) })

	step, err := m.HandleServerChallenge("Gss", serverData)
	require.NoError(t, err)
	require.Equal(t, "Gss", m.GetCurrentPlugin().Name())
	require.NotNil(t, step)
	require.NotEmpty(t, step.AuthSpecific)
	require.Equal(t, step.Reply, m.AuthSpecificData())
}
