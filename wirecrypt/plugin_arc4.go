package wirecrypt

import "fmt"

func newArc4Plugin() Plugin {
	return PluginFunc{
		id: Identifier{Type: TypeSymmetric, Plugin: "Arc4"},
		fn: func(cfg SessionConfig) (*InitResult, error) {
			if len(cfg.SessionKey) == 0 {
				return nil, fmt.Errorf("wirecrypt: %s missing session key", cfg.Identifier.Plugin)
			}
			return &InitResult{
				Identifier: cfg.Identifier,
			}, nil
		},
	}
}
