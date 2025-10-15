package wirecrypt

import (
	"fmt"
)

func newChaChaPlugin() Plugin {
	return PluginFunc{
		id: Identifier{Type: TypeSymmetric, Plugin: "ChaCha"},
		fn: func(cfg SessionConfig) (*InitResult, error) {
			if len(cfg.SessionKey) == 0 {
				return nil, fmt.Errorf("wirecrypt: %s missing session key", cfg.Identifier.Plugin)
			}
			if len(cfg.SpecificData) != 12 {
				return nil, fmt.Errorf("wirecrypt: %s invalid nonce length %d", cfg.Identifier.Plugin, len(cfg.SpecificData))
			}
			return &InitResult{
				Identifier: cfg.Identifier,
				Nonce:      cfg.SpecificData,
			}, nil
		},
	}
}

func newChaCha64Plugin() Plugin {
	return PluginFunc{
		id: Identifier{Type: TypeSymmetric, Plugin: "ChaCha64"},
		fn: func(cfg SessionConfig) (*InitResult, error) {
			if len(cfg.SessionKey) == 0 {
				return nil, fmt.Errorf("wirecrypt: %s missing session key", cfg.Identifier.Plugin)
			}
			if len(cfg.SpecificData) != 8 && len(cfg.SpecificData) != 12 {
				return nil, fmt.Errorf("wirecrypt: %s invalid nonce length %d", cfg.Identifier.Plugin, len(cfg.SpecificData))
			}
			return &InitResult{
				Identifier: cfg.Identifier,
				Nonce:      cfg.SpecificData,
			}, nil
		},
	}
}
