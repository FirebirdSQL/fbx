package fbconn

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto/auth"
)

const constructorTestPluginName = "test"

type testPlugin struct {
	data []byte
	name string
}

func (p *testPlugin) Name() string        { return constructorTestPluginName }
func (p *testPlugin) Advertise() []string { return []string{p.Name()} }
func (p *testPlugin) InitialClientData() []byte {
	return p.data
}
func (p *testPlugin) HandleServerData(string, string, string, []byte) (*auth.AuthPluginStep, error) {
	reply := p.data
	return &auth.AuthPluginStep{Reply: reply, AuthSpecific: reply}, nil
}
func (p *testPlugin) HandleAuthResponse([]byte) error { return nil }

func withData(data []byte) auth.ConstructorOption {
	return func(pl auth.AuthPlugin) error {
		plugin, ok := pl.(*testPlugin)
		if !ok {
			return fmt.Errorf("unexpected plugin type %T", pl)
		}
		plugin.data = data
		return nil
	}
}

func withName(name string) auth.ConstructorOption {
	return func(pl auth.AuthPlugin) error {
		plugin, ok := pl.(*testPlugin)
		if !ok {
			return fmt.Errorf("unexpected plugin type %T", pl)
		}
		plugin.name = name
		return nil
	}
}

func registerPlugin() {
	_ = auth.RegisterAuthPlugin(constructorTestPluginName, func(auth.PluginConfig) auth.AuthPlugin {
		return &testPlugin{}
	})
}

func TestSetPluginConstructorOptionsNormalizesPluginName(t *testing.T) {
	config := &Config{}
	require.Len(t, config.ConstructorOptions["plugin"], 0)
	err := config.SetPluginConstructorOptions("  Plugin  ", func(auth.AuthPlugin) error { return nil })
	require.NoError(t, err)

	require.Contains(t, config.ConstructorOptions, "plugin")
	require.Len(t, config.ConstructorOptions["plugin"], 1)
}

func TestConfigCopyConstructorOptionsMapNotLinked(t *testing.T) {
	config, err := ParseConfig("")
	require.NoError(t, err)

	want := []byte{1, 2, 3}
	err = config.SetPluginConstructorOptions(constructorTestPluginName, withData(want))
	require.NoError(t, err)

	copyConfig := config.Copy()
	require.Len(t, copyConfig.ConstructorOptions, 1)

	config.ConstructorOptions["other"] = nil
	require.Len(t, config.ConstructorOptions, 2)

	require.Len(t, copyConfig.ConstructorOptions, 1)
}

func TestConfigSetOptionsSlicesNotLinked(t *testing.T) {
	registerPlugin()

	conf, _ := ParseConfig("")

	wantData1 := []byte{1, 2, 3}
	wantData2 := []byte{1, 2}

	optionsSlice := []auth.ConstructorOption{withData(wantData1)}

	err := conf.SetPluginConstructorOptions(constructorTestPluginName, optionsSlice...)
	require.NoError(t, err)

	innerSlice := conf.ConstructorOptions[constructorTestPluginName]

	require.Len(t, innerSlice, 1)
	require.Len(t, optionsSlice, 1)
	require.Equal(t, reflect.ValueOf(optionsSlice[0]).Pointer(), reflect.ValueOf(innerSlice[0]).Pointer())

	optionsSlice[0] = withData(wantData2)

	require.Len(t, innerSlice, 1)
	require.Len(t, optionsSlice, 1)
	require.NotEqual(t, reflect.ValueOf(optionsSlice[0]).Pointer(), reflect.ValueOf(innerSlice[0]).Pointer())
}

func TestConnectConfigPassesConstructorOptionsToPlugin(t *testing.T) {
	registerPlugin()

	wantData := []byte{1, 2, 3}
	wantName := "abc"

	var constructed *testPlugin
	config, err := ParseConfig("")
	require.NoError(t, err)
	err = config.SetPluginConstructorOptions(constructorTestPluginName,
		func(pl auth.AuthPlugin) error {
			plugin, ok := pl.(*testPlugin)
			if !ok {
				return fmt.Errorf("unexpected plugin type %T", pl)
			}
			constructed = plugin
			return withData(wantData)(pl)
		},
		withName(wantName),
	)
	require.NoError(t, err)
	config.AuthPluginList = []string{constructorTestPluginName}

	_, _ = ConnectConfig(context.Background(), config)

	require.NotNil(t, constructed)
	require.Equal(t, wantData, constructed.data)
	require.Equal(t, wantName, constructed.name)
}

func TestConstructorOptionsPassedWhenSetCurrentPluginReused(t *testing.T) {
	registerPlugin()

	wantData := []byte{1, 2, 3}
	wantName := "abc"
	conf := auth.PluginConfig{
		ConstructorOptions: map[string][]auth.ConstructorOption{
			constructorTestPluginName: {withData(wantData), withName(wantName)},
		},
	}
	m, err := auth.NewAuthManager(conf, []string{constructorTestPluginName, "Srp"})
	require.NoError(t, err)

	_, err = m.SetCurrentPlugin(constructorTestPluginName)
	require.NoError(t, err)
	step, err := m.HandleServerChallenge(constructorTestPluginName, nil)
	require.NoError(t, err)
	require.Equal(t, wantData, step.AuthSpecific)
	require.Equal(t, constructorTestPluginName, m.GetCurrentPlugin().Name())

	_, err = m.SetCurrentPlugin("Srp")
	require.NoError(t, err)
	step, err = m.HandleServerChallenge("Srp", nil)
	require.NoError(t, err)
	require.Empty(t, step.AuthSpecific)

	_, err = m.SetCurrentPlugin(constructorTestPluginName)
	require.NoError(t, err)
	step, err = m.HandleServerChallenge(constructorTestPluginName, nil)
	require.NoError(t, err)
	require.Equal(t, wantData, step.AuthSpecific)
	require.Equal(t, constructorTestPluginName, m.GetCurrentPlugin().Name())
}

func TestNilOptionReturnsError(t *testing.T) {
	registerPlugin()

	config, err := ParseConfig("")
	require.NoError(t, err)

	err = config.SetPluginConstructorOptions(constructorTestPluginName, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin option must not be nil")
}
