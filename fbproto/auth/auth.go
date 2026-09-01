package auth

import (
	"fmt"
	"strings"
)

// ConstructorOption is a functional option for configuring AuthPlugin.
// It takes an AuthPlugin and returns an error if the configuration fails.
type ConstructorOption func(AuthPlugin) error

// PluginConfig contains settings for authentication plugins.
type PluginConfig struct {
	// User is the username used for authentication.
	User string
	// Password is the password of user.
	Password string
	// ConstructorOptions maps plugin names to their configuration options.
	// The key is the plugin identifier in lower case without spaces.
	// The value is a slice of ConstructorOption functions that will be
	// applied to the plugin during initialization.
	// Options are called each time the plugin constructor is called.
	ConstructorOptions map[string][]ConstructorOption
}

// AuthPluginStep describes the client output produced by one authentication
// plugin step. Reply contains bytes for the next wire message, AuthSpecific
// contains the plugin-native authentication value, and SessionKey contains key
// material that may be used by wire encryption.
type AuthPluginStep struct {
	Reply        []byte
	AuthSpecific []byte
	SessionKey   []byte
}

// HandshakeRequest describes the authentication fields included in op_connect.
// SpecificData is split into protocol-sized chunks for CNCT_specific_data.
type HandshakeRequest struct {
	PluginName   string
	PluginList   []string
	SpecificData []SpecificDataChunk
}

// AuthPlugin implements one client-side authentication mechanism.
// An AuthPlugin instance holds state for a single negotiation attempt and is not
// safe for concurrent use.
type AuthPlugin interface {
	// Name returns the canonical plugin name.
	Name() string
	// Advertise returns the plugin names accepted by this implementation.
	Advertise() []string
	// InitialClientData returns data sent with op_connect before a server challenge.
	InitialClientData() []byte
	// HandleServerData processes a server challenge and returns the next client step.
	HandleServerData(user string, password string, pluginName string, data []byte) (*AuthPluginStep, error)
	// HandleAuthResponse validates final authentication data returned by the server.
	HandleAuthResponse(data []byte) error
}

var defaultPluginSequence = []string{
	"Srp256",
	"Srp",
	"Srp224",
	"Srp384",
	"Srp512",
	"Legacy_Auth",
}

var pluginConstructors = map[string]func(conf PluginConfig) AuthPlugin{
	"srp":    func(PluginConfig) AuthPlugin { return NewSrpAuthPlugin() },
	"srp256": func(PluginConfig) AuthPlugin { return NewSrp256AuthPlugin() },
	"srp224": func(PluginConfig) AuthPlugin { return NewSrp224AuthPlugin() },
	"srp384": func(PluginConfig) AuthPlugin { return NewSrp384AuthPlugin() },
	"srp512": func(PluginConfig) AuthPlugin { return NewSrp512AuthPlugin() },
	"legacy_auth": func(conf PluginConfig) AuthPlugin {
		return NewLegacyAuthPlugin(conf.Password)
	},
}

// DefaultPluginSequence returns a copy of the default authentication plugin order.
func DefaultPluginSequence() []string {
	return cloneStrings(defaultPluginSequence)
}

// DefaultPluginListString returns the default authentication plugin order as a comma-separated list.
func DefaultPluginListString() string {
	return strings.Join(defaultPluginSequence, ",")
}

// AuthManager coordinates plugin selection and state across the wire protocol
// authentication exchange. An AuthManager is not safe for concurrent use.
type AuthManager struct {
	pluginConfig     PluginConfig
	plugins          []AuthPlugin
	current          int
	lastAuthSpecific []byte
	lastSessionKey   []byte
	authComplete     bool
}

// SpecificDataChunk is one CNCT_specific_data fragment.
// Part is the zero-based fragment number and Data is its payload.
type SpecificDataChunk struct {
	Part byte
	Data []byte
}

// RegisterAuthPlugin allows to register custom authentication plugins.
// To register it, you need to pass the name of the plugin and its constructor.
// Constructor must be like in pluginConstructors map.
func RegisterAuthPlugin(name string, constructor func(PluginConfig) AuthPlugin) error {
	lowerName := strings.ToLower(name)
	if _, ok := pluginConstructors[lowerName]; !ok {
		defaultPluginSequence = append(defaultPluginSequence, strings.TrimSpace(name))
		pluginConstructors[lowerName] = constructor
		return nil
	}
	return fmt.Errorf("plugin already exists")
}

// NewAuthManager creates an authentication manager for user and password.
// pluginNames defines client preference order; an empty list uses
// [DefaultPluginSequence]. Duplicate names are ignored.
func NewAuthManager(conf PluginConfig, pluginNames []string) (*AuthManager, error) {
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

		plugin, err := createAuthPlugin(trimmed, conf)
		if err != nil {
			return nil, err
		}
		err = applyAuthPluginsOptions(plugin, conf.ConstructorOptions)
		if err != nil {
			return nil, err
		}

		plugins = append(plugins, plugin)
		seen[key] = struct{}{}
	}

	if len(plugins) == 0 {
		return nil, fmt.Errorf("auth: no authentication plugins available")
	}

	return &AuthManager{
		pluginConfig: conf,
		plugins:      plugins,
		current:      0,
	}, nil
}

// GetCurrentPlugin returns the plugin currently selected by the client/server
// negotiation. If the saved index is out of range, it falls back to the first
// available plugin to keep the manager in a usable state.
func (m *AuthManager) GetCurrentPlugin() AuthPlugin {
	if len(m.plugins) == 0 {
		return nil
	}
	if m.current < 0 || m.current >= len(m.plugins) {
		m.current = 0
	}
	return m.plugins[m.current]
}

// BuildList returns the comma-separated plugin preference list currently
// advertised to the server. The list may be narrowed after the server sends
// TAG_KNOWN_PLUGINS in p_acpt_keys.
func (m *AuthManager) BuildList() string {
	return strings.Join(m.pluginNames(), ",")
}

// SetCurrentPlugin switches the active plugin to the server-selected name.
// When switching away from another plugin, it recreates the plugin instance so
// per-plugin handshake state from the previous attempt cannot leak into the new
// authentication flow.
func (m *AuthManager) SetCurrentPlugin(name string) (AuthPlugin, error) {
	if plugin := m.GetCurrentPlugin(); plugin != nil {
		for _, advertised := range plugin.Advertise() {
			if strings.EqualFold(advertised, name) {
				return plugin, nil
			}
		}
	}

	for idx, plugin := range m.plugins {
		for _, advertised := range plugin.Advertise() {
			if strings.EqualFold(advertised, name) {
				fresh, err := createAuthPlugin(plugin.Name(), m.pluginConfig)
				if err == nil {
					plugin = fresh
					m.plugins[idx] = fresh
					err = applyAuthPluginsOptions(plugin, m.pluginConfig.ConstructorOptions)
					if err != nil {
						return nil, err
					}
				}
				m.current = idx
				return plugin, nil
			}
		}
	}
	return nil, &UnsupportedPluginError{Name: name}
}

// RestrictToServerPlugins narrows the client plugin list to the intersection
// advertised by the server in TAG_KNOWN_PLUGINS. The client order is preserved,
// because the list still represents client preference after removing plugins
// the server says it cannot use.
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

// BeginHandshake builds the initial auth payload for op_connect using the
// current first-choice plugin. Only that plugin's initial data is sent; the rest
// of the list is advertised by name for server-side selection.
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

// HandleServerChallenge applies server-provided data to the selected plugin and
// returns the next client response. It also stores the wire-format reply so an
// attach-through-DPB flow can put the exact bytes into isc_dpb_specific_auth_data.
func (m *AuthManager) HandleServerChallenge(pluginName string, data []byte) (*AuthPluginStep, error) {
	var step *AuthPluginStep
	var err error
	if m.GetCurrentPlugin().Name() != pluginName {
		data = make([]byte, 0)
	}
	m.authComplete = false
	plugin, err := m.SetCurrentPlugin(pluginName)
	if err != nil {
		return nil, err
	}
	step, err = plugin.HandleServerData(m.pluginConfig.User, m.pluginConfig.Password, pluginName, data)
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

// HandleAuthResponse verifies final server auth data, such as an SRP server
// proof, and marks authentication complete when the active plugin accepts it.
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

// SetAuthComplete records completion reported by protocol messages that do not
// go through HandleAuthResponse. Once complete, stale DPB auth data is cleared.
func (m *AuthManager) SetAuthComplete(complete bool) {
	m.authComplete = complete
	if complete {
		m.lastAuthSpecific = nil
	}
}

// AuthSpecificData returns the last wire-format client auth response that still
// needs to be carried in a DPB. It returns nil after authentication completes.
func (m *AuthManager) AuthSpecificData() []byte {
	if m.authComplete {
		return nil
	}
	return cloneBytes(m.lastAuthSpecific)
}

// SessionKey returns the negotiated auth session key, when the active plugin
// produced one. Wire encryption negotiation uses this key later.
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

// The method creates an authentication plugin if such a name exists in pluginConstructors.
func createAuthPlugin(name string, conf PluginConfig) (AuthPlugin, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	constructor, ok := pluginConstructors[key]
	if !ok {
		return nil, &UnsupportedPluginError{Name: name}
	}
	return constructor(conf), nil
}

// applyAuthPluginsOptions applies functional options to plugin
func applyAuthPluginsOptions(plugin AuthPlugin, opts map[string][]ConstructorOption) error {
	for k, v := range opts {
		if strings.EqualFold(strings.ToLower(strings.TrimSpace(k)), strings.ToLower(plugin.Name())) {
			for _, opt := range v {
				if opt == nil {
					return fmt.Errorf("option function of plugin %s is nil", plugin.Name())
				}
				err := opt(plugin)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// UnsupportedPluginError reports a server-selected authentication plugin that
// is not available in the current client plugin list.
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
