package auth

import "strings"

type AuthPluginStep struct {
	Reply        []byte
	AuthSpecific []byte
	SessionKey   []byte
}

type AuthPlugin interface {
	Name() string
	Advertise() []string
	InitialClientData() []byte
	HandleServerData(user string, password string, pluginName string, data []byte) (*AuthPluginStep, error)
	HandleAuthResponse(data []byte) error
}

type AuthManager struct {
	plugins []AuthPlugin
	current int
}

func NewAuthManager(user string, password string) *AuthManager {
	plugins := make([]AuthPlugin, 0, 6)
	plugins = append(plugins, NewSrp256AuthPlugin())
	plugins = append(plugins, NewSrp224AuthPlugin())
	plugins = append(plugins, NewSrp384AuthPlugin())
	plugins = append(plugins, NewSrp512AuthPlugin())
	plugins = append(plugins, NewSrpAuthPlugin())
	plugins = append(plugins, NewLegacyAuthPlugin(password))
	return &AuthManager{plugins: plugins, current: 0}
}

func (m *AuthManager) GetCurrentPlugin() AuthPlugin {
	if len(m.plugins) == 0 {
		return nil
	}
	if m.current < 0 || m.current >= len(m.plugins) {
		m.current = 0
	}
	return m.plugins[m.current]
}

func (m *AuthManager) BuildList() string {
	names := make([]string, len(m.plugins))
	for i, plugin := range m.plugins {
		names[i] = plugin.Name()
	}
	return strings.Join(names, ",")
}

func (m *AuthManager) SetCurrentPlugin(name string) (AuthPlugin, bool) {
	for idx, plugin := range m.plugins {
		for _, advertised := range plugin.Advertise() {
			if strings.EqualFold(advertised, name) {
				m.current = idx
				return plugin, true
			}
		}
	}
	return nil, false
}
