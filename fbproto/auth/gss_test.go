package auth

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// mock token provider for tests
type mockGssProvider struct {
	token               []byte
	gotService, gotHost string
}

func (m *mockGssProvider) InitToken(service, host string) ([]byte, error) {
	m.gotService, m.gotHost = service, host
	return m.token, nil
}

func TestParseGssServerData(t *testing.T) {
	// host="localhost" (len=9), service="red_db" (len=6)
	data := []byte{9, 0}
	data = append(data, []byte("localhost")...)
	data = append(data, 6, 0)
	data = append(data, []byte("red_db")...)

	host, service, err := parseGssServerData(data)
	require.NoError(t, err)
	require.Equal(t, "localhost", host)
	require.Equal(t, "red_db", service)
}

func TestGssAuthPluginPhases(t *testing.T) {
	p := NewGssAuthPlugin()

	// Phase 1: no server data yet -> request more data
	step, err := p.HandleServerData("", "", "Gss", nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.Nil(t, step.Reply)

	// Phase 2: with server data, provider returns token
	data := []byte{9, 0}
	data = append(data, []byte("localhost")...)
	data = append(data, 6, 0)
	data = append(data, []byte("red_db")...)

	prev := getGssTokenProvider()
	mock := &mockGssProvider{token: []byte{1, 2, 3, 4}}
	SetGssTokenProvider(mock)
	t.Cleanup(func() { SetGssTokenProvider(prev) })

	step, err = p.HandleServerData("", "", "Gss", data)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.Equal(t, []byte{1, 2, 3, 4}, step.Reply)
	require.Equal(t, []byte{1, 2, 3, 4}, step.AuthSpecific)
	require.Equal(t, "red_db", mock.gotService)
	require.Equal(t, "localhost", mock.gotHost)
}

func TestAuthManagerPrefersGssWhenUserEmpty(t *testing.T) {
	m := NewAuthManager("", "")
	hs := m.BeginHandshake()
	require.NotNil(t, hs)
	require.Equal(t, "Gss", hs.PluginName)
}
