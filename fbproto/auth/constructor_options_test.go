package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type testPlugin struct {
	name string
	data []byte
}

func (p *testPlugin) Name() string        { return p.name }
func (p *testPlugin) Advertise() []string { return []string{p.name} }
func (p *testPlugin) InitialClientData() []byte {
	return cloneBytes(p.data)
}
func (p *testPlugin) HandleServerData(user, password, pluginName string, serverData []byte) (*AuthPluginStep, error) {
	reply := cloneBytes(p.data)
	return &AuthPluginStep{Reply: reply, AuthSpecific: reply}, nil
}
func (p *testPlugin) HandleAuthResponse([]byte) error { return nil }

func withData(data []byte) ConstructorOption {
	return func(pl AuthPlugin) error {
		plugin, ok := pl.(*testPlugin)
		if !ok {
			return fmt.Errorf("unexpected plugin type %T", pl)
		}
		plugin.data = cloneBytes(data)
		return nil
	}
}

func registerPlugin(name string) {
	_ = RegisterAuthPlugin(name, func(PluginConfig) AuthPlugin {
		return &testPlugin{name: name}
	})
}

func TestConstructorOptionsAppliedOnNewAuthManager(t *testing.T) {
	const name = "new_manager"
	registerPlugin(name)

	want := []byte{1, 2, 3}
	conf := PluginConfig{
		User:     "user",
		Password: "password",
		ConstructorOptions: map[string][]ConstructorOption{
			name: {withData(want)},
		},
	}
	m, err := NewAuthManager(conf, []string{name})
	require.NoError(t, err)

	plugin := m.GetCurrentPlugin().(*testPlugin)
	require.Equal(t, want, plugin.data)
	require.Equal(t, want, plugin.InitialClientData())
}

func TestConstructorOptionsCaseInsensitiveKey(t *testing.T) {
	const name = "case"
	registerPlugin(name)

	want := []byte{1, 2}
	conf := PluginConfig{
		ConstructorOptions: map[string][]ConstructorOption{
			"CASE": {withData(want)},
		},
	}
	m, err := NewAuthManager(conf, []string{name})
	require.NoError(t, err)

	plugin := m.GetCurrentPlugin().(*testPlugin)
	require.Equal(t, want, plugin.data)
}

func TestConstructorOptionsNilOptionReturnsError(t *testing.T) {
	const name = "nil_plugin"
	registerPlugin(name)

	conf := PluginConfig{
		ConstructorOptions: map[string][]ConstructorOption{
			name: {nil},
		},
	}
	_, err := NewAuthManager(conf, []string{name})
	require.Error(t, err)
	require.Contains(t, err.Error(), "option function of plugin nil_plugin is nil")
}

func TestConstructorOptionsError(t *testing.T) {
	const name = "error"
	registerPlugin(name)

	conf := PluginConfig{
		ConstructorOptions: map[string][]ConstructorOption{
			name: {
				func(AuthPlugin) error {
					return fmt.Errorf("option failed")
				},
			},
		},
	}
	_, err := NewAuthManager(conf, []string{name})
	require.Error(t, err)
	require.Contains(t, err.Error(), "option failed")
}

func TestSetCurrentPluginReturnsUnsupportedPluginError(t *testing.T) {
	m, err := NewAuthManager(PluginConfig{User: "user", Password: "password"}, nil)
	require.NoError(t, err)

	plugin, err := m.SetCurrentPlugin("unknown_plugin")
	require.Error(t, err)
	require.Nil(t, plugin)
	var unsupportedErr *UnsupportedPluginError
	require.ErrorAs(t, err, &unsupportedErr)
	require.Equal(t, "unknown_plugin", unsupportedErr.Name)
}
