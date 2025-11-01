package wirecrypt

// NewDefaultManager returns a Manager pre-configured with built-in plugins.
func NewDefaultManager() *Manager {
	return NewManager(
		newArc4Plugin(),
		newChaChaPlugin(),
		newChaCha64Plugin(),
	)
}
