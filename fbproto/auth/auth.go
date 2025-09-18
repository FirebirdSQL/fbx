package auth

import (
	"strings"
)

type AuthPlugin interface {
	GetClientData() []byte
	SetServerData(data []byte)
	GetName() string
}

type AuthManager struct {
	AuthPlugins   []AuthPlugin
	CurrentPlugin *AuthPlugin
}

func NewAuthManager(user string, password string) *AuthManager {
	plugins := make([]AuthPlugin, 0, 3)
	plugins = append(plugins, NewSrp256AuthPlugin())
	plugins = append(plugins, NewSrpAuthPlugin())
	plugins = append(plugins, &LegacyAuthPlugin{Password: password})
	return &AuthManager{AuthPlugins: plugins, CurrentPlugin: &plugins[0]}
}

func (m *AuthManager) GetCurrentAuthPlugin() AuthPlugin {
	if len(m.AuthPlugins) == 0 {
		return nil
	}
	return m.AuthPlugins[0]
}

func (m *AuthManager) BuildList() string {
	names := make([]string, len(m.AuthPlugins))
	for i, name := range m.AuthPlugins {
		names[i] = name.GetName()
	}
	return strings.Join(names, ",")
}
