package wirecrypt

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Manager tracks server-provided wire encryption options and negotiates
// encryption with registered plugins.
type Manager struct {
	mu        sync.Mutex
	keys      []KnownServerKey
	plugins   map[string]Plugin
	lastError error
}

// ErrNoCommonPlugin indicates that no registered plugin matched the server's
// advertised encryption capabilities.
var ErrNoCommonPlugin = errors.New("wirecrypt: no mutually supported encryption plugin")

// NewManager creates a manager pre-populated with the provided plugins.
func NewManager(builtin ...Plugin) *Manager {
	m := &Manager{
		plugins: make(map[string]Plugin),
	}
	for _, plugin := range builtin {
		m.Register(plugin)
	}
	return m
}

// Register registers a wire encryption plugin.
func (m *Manager) Register(plugin Plugin) {
	if plugin == nil {
		return
	}
	m.mu.Lock()
	m.plugins[keyFor(plugin.Identifier())] = plugin
	m.mu.Unlock()
}

// AddServerKeys parses and stores the provided server keys.
func (m *Manager) AddServerKeys(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	keys, err := ParseKnownServerKeys(buf)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.keys = append(m.keys, keys...)
	m.mu.Unlock()
	return nil
}

// ClearServerKeys wipes stored server key information.
func (m *Manager) ClearServerKeys() {
	m.mu.Lock()
	m.keys = nil
	m.mu.Unlock()
}

// TryInitialize attempts to initialize the first supported encryption plugin.
func (m *Manager) TryInitialize(sessionKey []byte) (*InitResult, error) {
	if len(sessionKey) == 0 {
		return nil, errors.New("wirecrypt: missing session key")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []string
	for _, entry := range CollectPluginSpecificData(m.keys) {
		plugin := m.plugins[keyFor(entry.Identifier)]
		if plugin == nil {
			continue
		}
		cfg := SessionConfig{
			Identifier:   entry.Identifier,
			SessionKey:   sessionKey,
			SpecificData: entry.Data,
		}
		initResult, err := plugin.Initialize(cfg)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		m.lastError = nil
		return initResult, nil
	}

	if len(errs) > 0 {
		m.lastError = fmt.Errorf("%w: %s", ErrNoCommonPlugin, strings.Join(errs, "; "))
	} else {
		m.lastError = ErrNoCommonPlugin
	}
	return nil, m.lastError
}

// LastError returns the most recent error during plugin negotiation.
func (m *Manager) LastError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastError
}

func keyFor(id Identifier) string {
	return fmt.Sprintf("%s/%s", strings.ToLower(id.Type), strings.ToLower(id.Plugin))
}
