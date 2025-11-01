package wirecrypt

// Identifier identifies a wire encryption plugin with its type and name.
type Identifier struct {
	Type   string
	Plugin string
}

const (
	// TypeSymmetric matches Firebird symmetric wire encryption plugins.
	TypeSymmetric = "Symmetric"
)

// SessionConfig contains the negotiated session key and plugin-specific data
// provided by the server for a specific encryption plugin.
type SessionConfig struct {
	Identifier   Identifier
	SessionKey   []byte
	SpecificData []byte
}

// InitResult describes the successful initialization of a wire encryption plugin.
type InitResult struct {
	Identifier Identifier
	Nonce      []byte
}

// Plugin abstracts initialization for a wire encryption algorithm.
type Plugin interface {
	Identifier() Identifier
	Initialize(SessionConfig) (*InitResult, error)
}

// PluginFunc adapts a function to the Plugin interface.
type PluginFunc struct {
	id Identifier
	fn func(SessionConfig) (*InitResult, error)
}

func (p PluginFunc) Identifier() Identifier {
	return p.id
}

func (p PluginFunc) Initialize(cfg SessionConfig) (*InitResult, error) {
	return p.fn(cfg)
}
