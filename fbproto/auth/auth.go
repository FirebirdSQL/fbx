package auth

import "strings"

type AuthPluginStep struct {
	Reply        []byte
	AuthSpecific []byte
	SessionKey   []byte
}

type HandshakeRequest struct {
	PluginName   string
	PluginList   []string
	SpecificData []SpecificDataChunk
}

type AuthPlugin interface {
	Name() string
	Advertise() []string
	InitialClientData() []byte
	HandleServerData(user string, password string, pluginName string, data []byte) (*AuthPluginStep, error)
	HandleAuthResponse(data []byte) error
}

type AuthManager struct {
	user             string
	password         string
	plugins          []AuthPlugin
	current          int
	lastAuthSpecific []byte
	lastSessionKey   []byte
}

type SpecificDataChunk struct {
	Part byte
	Data []byte
}

func NewAuthManager(user string, password string) *AuthManager {
	plugins := make([]AuthPlugin, 0, 6)
	plugins = append(plugins, NewSrp256AuthPlugin())
	plugins = append(plugins, NewSrp224AuthPlugin())
	plugins = append(plugins, NewSrp384AuthPlugin())
	plugins = append(plugins, NewSrp512AuthPlugin())
	plugins = append(plugins, NewSrpAuthPlugin())
	plugins = append(plugins, NewLegacyAuthPlugin(password))
	return &AuthManager{
		user:     user,
		password: password,
		plugins:  plugins,
		current:  0,
	}
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
	return strings.Join(m.pluginNames(), ",")
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

func (m *AuthManager) BeginHandshake() *HandshakeRequest {
	plugin := m.GetCurrentPlugin()
	if plugin == nil {
		return nil
	}
	return &HandshakeRequest{
		PluginName:   plugin.Name(),
		PluginList:   m.pluginNames(),
		SpecificData: splitSpecificData(plugin.InitialClientData()),
	}
}

func (m *AuthManager) HandleServerChallenge(pluginName string, data []byte) (*AuthPluginStep, error) {
	plugin, ok := m.SetCurrentPlugin(pluginName)
	if !ok {
		return nil, &UnsupportedPluginError{Name: pluginName}
	}
	step, err := plugin.HandleServerData(m.user, m.password, pluginName, data)
	if err != nil {
		return nil, err
	}
	if step != nil && len(step.AuthSpecific) > 0 {
		m.lastAuthSpecific = cloneBytes(step.AuthSpecific)
	} else {
		m.lastAuthSpecific = nil
	}
	if step != nil && len(step.SessionKey) > 0 {
		m.lastSessionKey = cloneBytes(step.SessionKey)
	} else {
		m.lastSessionKey = nil
	}
	return step, nil
}

func (m *AuthManager) HandleAuthResponse(data []byte) error {
	plugin := m.GetCurrentPlugin()
	if plugin == nil {
		return &UnsupportedPluginError{Name: ""}
	}
	return plugin.HandleAuthResponse(data)
}

func (m *AuthManager) AuthSpecificData() []byte {
	return cloneBytes(m.lastAuthSpecific)
}

func (m *AuthManager) SessionKey() []byte {
	return cloneBytes(m.lastSessionKey)
}

func (m *AuthManager) pluginNames() []string {
	names := make([]string, len(m.plugins))
	for i, plugin := range m.plugins {
		names[i] = plugin.Name()
	}
	return names
}

type UnsupportedPluginError struct {
	Name string
}

func (e *UnsupportedPluginError) Error() string {
	if e == nil {
		return "auth: unsupported plugin"
	}
	if e.Name == "" {
		return "auth: unsupported plugin"
	}
	return "auth: unsupported plugin \"" + e.Name + "\""
}

func cloneBytes(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

const maxSpecificChunkSize = 254

func splitSpecificData(data []byte) []SpecificDataChunk {
	if len(data) == 0 {
		return nil
	}
	chunkCount := (len(data) + maxSpecificChunkSize - 1) / maxSpecificChunkSize
	chunks := make([]SpecificDataChunk, 0, chunkCount)
	part := byte(0)
	for offset := 0; offset < len(data); offset += maxSpecificChunkSize {
		end := offset + maxSpecificChunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := cloneBytes(data[offset:end])
		chunks = append(chunks, SpecificDataChunk{Part: part, Data: chunk})
		if part < 255 {
			part++
		}
	}
	return chunks
}
