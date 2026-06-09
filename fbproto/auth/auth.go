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
	"Srp",
	"Gss",
	"Srp224",
	"Srp384",
	"Srp512",
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
	authComplete     bool
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

		plugin, ok := createAuthPlugin(trimmed, password)
		if !ok {
			return nil, fmt.Errorf("auth: unknown authentication plugin %q", name)
		}

		plugins = append(plugins, plugin)
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
	if plugin := m.GetCurrentPlugin(); plugin != nil {
		for _, advertised := range plugin.Advertise() {
			if strings.EqualFold(advertised, name) {
				return plugin, true
			}
		}
	}

	for idx, plugin := range m.plugins {
		for _, advertised := range plugin.Advertise() {
			if strings.EqualFold(advertised, name) {
				if fresh, ok := createAuthPlugin(plugin.Name(), m.password); ok {
					plugin = fresh
					m.plugins[idx] = fresh
				}
				m.current = idx
				return plugin, true
			}
		}
	}
	return nil, false
}

func (m *AuthManager) RestrictToServerPlugins(serverKeys []byte) error {
	serverPlugins := knownAuthPlugins(serverKeys)
	if len(serverPlugins) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(serverPlugins))
	for _, name := range serverPlugins {
		allowed[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}

	currentPlugin := m.GetCurrentPlugin()
	filtered := m.plugins[:0]
	for _, plugin := range m.plugins {
		matches := false
		for _, advertised := range plugin.Advertise() {
			if _, ok := allowed[strings.ToLower(strings.TrimSpace(advertised))]; ok {
				matches = true
				break
			}
		}
		if matches {
			filtered = append(filtered, plugin)
		}
	}

	if len(filtered) == 0 {
		return fmt.Errorf("auth: no matching plugins on server")
	}

	m.plugins = filtered
	m.current = 0
	if currentPlugin != nil {
		for idx, plugin := range m.plugins {
			if strings.EqualFold(plugin.Name(), currentPlugin.Name()) {
				m.current = idx
				break
			}
		}
	}
	return nil
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
	m.authComplete = false
	plugin, ok := m.SetCurrentPlugin(pluginName)
	if !ok {
		return nil, &UnsupportedPluginError{Name: pluginName}
	}
	step, err := plugin.HandleServerData(m.user, m.password, pluginName, data)
	if err != nil {
		return nil, err
	}
	if step != nil && len(step.Reply) > 0 {
		m.lastAuthSpecific = step.Reply
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
	if err := plugin.HandleAuthResponse(data); err != nil {
		return err
	}
	m.authComplete = true
	return nil
}

func (m *AuthManager) SetAuthComplete(complete bool) {
	m.authComplete = complete
	if complete {
		m.lastAuthSpecific = nil
	}
}

func (m *AuthManager) AuthSpecificData() []byte {
	if m.authComplete {
		return nil
	}
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

func createAuthPlugin(name string, password string) (AuthPlugin, bool) {
	constructor, ok := pluginConstructors[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, false
	}
	return constructor(password), true
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

func knownAuthPlugins(serverKeys []byte) []string {
	const tagKnownPlugins = 2

	var result []string
	for offset := 0; offset+2 <= len(serverKeys); {
		tag := serverKeys[offset]
		length := int(serverKeys[offset+1])
		offset += 2
		if offset+length > len(serverKeys) {
			return result
		}
		if tag == tagKnownPlugins {
			result = append(result, splitPluginList(string(serverKeys[offset:offset+length]))...)
		}
		offset += length
	}
	return result
}

func splitPluginList(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case ' ', '\t', ',', ';':
			return true
		default:
			return false
		}
	})
	result := fields[:0]
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			result = append(result, trimmed)
		}
	}
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
