package auth

import (
	"strings"
)

type AuthPlugin interface {
	GetClientData() []byte
	// SetServerData processes server auth data and returns:
	// - response: bytes to send back to the server in op_cont_auth (may be nil)
	// - authSpecific: bytes to store in DPB specific_auth_data (may be nil)
	// - sessionKey: negotiated session key if any (may be nil)
	SetServerData(user string, password string, pluginName string, data []byte) (response []byte, authSpecific []byte, sessionKey []byte, err error)
	GetName() string
}

type AuthManager struct {
	AuthPlugins []AuthPlugin
	current     int
}

func NewAuthManager(user string, password string) *AuthManager {
	plugins := make([]AuthPlugin, 0, 6)
	plugins = append(plugins, NewSrp256AuthPlugin())
	plugins = append(plugins, NewSrp224AuthPlugin())
	plugins = append(plugins, NewSrp384AuthPlugin())
	plugins = append(plugins, NewSrp512AuthPlugin())
	plugins = append(plugins, NewSrpAuthPlugin())
	plugins = append(plugins, &LegacyAuthPlugin{Password: password})
	return &AuthManager{AuthPlugins: plugins, current: 0}
}

func (m *AuthManager) GetCurrentAuthPlugin() AuthPlugin {
	if len(m.AuthPlugins) == 0 {
		return nil
	}
	if m.current < 0 || m.current >= len(m.AuthPlugins) {
		m.current = 0
	}
	return m.AuthPlugins[m.current]
}

func (m *AuthManager) BuildList() string {
	names := make([]string, len(m.AuthPlugins))
	for i, name := range m.AuthPlugins {
		names[i] = name.GetName()
	}
	return strings.Join(names, ",")
}

func (m *AuthManager) SetCurrentPlugin(name string) (AuthPlugin, bool) {
	for idx, plugin := range m.AuthPlugins {
		if strings.EqualFold(plugin.GetName(), name) {
			m.current = idx
			return plugin, true
		}
	}
	return nil, false
}
