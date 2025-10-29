package auth

import (
	"fmt"
	"strings"
)

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

var defaultPluginSequence = []string{
	"Srp256",
	"Srp224",
	"Srp384",
	"Srp512",
	"Srp",
	"Gss",
	"Legacy_Auth",
}

var pluginConstructors = map[string]func(password string) AuthPlugin{
	"srp":    func(string) AuthPlugin { return NewSrpAuthPlugin() },
	"srp256": func(string) AuthPlugin { return NewSrp256AuthPlugin() },
	"srp224": func(string) AuthPlugin { return NewSrp224AuthPlugin() },
	"srp384": func(string) AuthPlugin { return NewSrp384AuthPlugin() },
	"srp512": func(string) AuthPlugin { return NewSrp512AuthPlugin() },
	"gss":    func(string) AuthPlugin { return NewGssAuthPlugin() },
	"legacy_auth": func(password string) AuthPlugin {
		return NewLegacyAuthPlugin(password)
	},
}

// DefaultPluginSequence returns a copy of the default authentication plugin order.
func DefaultPluginSequence() []string {
	return cloneStrings(defaultPluginSequence)
}

// DefaultPluginListString returns the default authentication plugin order as a comma separated list.
func DefaultPluginListString() string {
	return strings.Join(defaultPluginSequence, ",")
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

func NewAuthManager(user string, password string, pluginNames []string) (*AuthManager, error) {
	order := pluginNames
	if len(order) == 0 {
		order = DefaultPluginSequence()
	} else {
		order = cloneStrings(pluginNames)
	}

	plugins := make([]AuthPlugin, 0, len(order))
	seen := make(map[string]struct{}, len(order))
	for _, name := range order {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, dup := seen[key]; dup {
			continue
		}

		constructor, ok := pluginConstructors[key]
		if !ok {
			return nil, fmt.Errorf("auth: unknown authentication plugin %q", name)
		}

		plugins = append(plugins, constructor(password))
		seen[key] = struct{}{}
	}

	if len(plugins) == 0 {
		return nil, fmt.Errorf("auth: no authentication plugins available")
	}

	return &AuthManager{
		user:     user,
		password: password,
		plugins:  plugins,
		current:  0,
	}, nil
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
		m.lastAuthSpecific = step.AuthSpecific
	} else {
		m.lastAuthSpecific = nil
	}
	if step != nil && len(step.SessionKey) > 0 {
		m.lastSessionKey = step.SessionKey
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

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func prioritizePlugin(names []string, target string) []string {
	idx := -1
	targetLower := strings.ToLower(strings.TrimSpace(target))
	for i, name := range names {
		if strings.ToLower(strings.TrimSpace(name)) == targetLower {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return names
	}

	result := make([]string, 0, len(names))
	result = append(result, names[idx])
	result = append(result, names[:idx]...)
	result = append(result, names[idx+1:]...)
	return result
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
