package wirecrypt

import (
	"bytes"
	"errors"
	"strings"

	"github.com/firebirdsql/fbx/fbproto"
)

const (
	tagKeyType        = 0
	tagKeyPlugins     = 1
	tagKnownPlugins   = 2
	tagPluginSpecific = 3
)

// KnownServerKey represents a single key entry advertised by the server.
type KnownServerKey struct {
	Type         string
	Plugins      []string
	SpecificData map[string][]byte
}

// PluginSpecificData pairs an encryption identifier with plugin-specific data.
type PluginSpecificData struct {
	Identifier Identifier
	Data       []byte
}

// ParseKnownServerKeys parses the clumplet block sent by the server with
// available wire encryption plugins.
func ParseKnownServerKeys(buf []byte) ([]KnownServerKey, error) {
	if len(buf) == 0 {
		return nil, nil
	}

	var (
		keys    []KnownServerKey
		current *KnownServerKey
	)

	r := fbproto.NewXPBReader(buf)
	for !r.End() {
		// Read clumplet header: 1 byte tag, 1 byte length
		have, tag := r.Next()
		if !have {
			return nil, errors.New("wirecrypt: malformed clumplet header")
		}
		have, lb := r.Next()
		if !have {
			return nil, errors.New("wirecrypt: malformed clumplet header")
		}
		length := int(lb)
		if length < 0 {
			return nil, errors.New("wirecrypt: malformed clumplet length")
		}
		// Read value bytes
		if length == 0 {
			switch tag {
			case tagKeyType:
				// Start a new entry with empty type (unlikely, but keep behavior deterministic)
				if current != nil {
					keys = append(keys, *current)
				}
				current = &KnownServerKey{Type: "", SpecificData: make(map[string][]byte)}
			case tagKeyPlugins:
				if current != nil {
					current.Plugins = nil
				}
			case tagKnownPlugins:
				// ignore
			case tagPluginSpecific:
				// ignore empty specific
			default:
				// ignore unknown tag
			}
			continue
		}
		value, ok := r.GetBytes(length)
		if !ok {
			return nil, errors.New("wirecrypt: malformed clumplet length")
		}

		switch tag {
		case tagKeyType:
			if current != nil {
				keys = append(keys, *current)
			}
			current = &KnownServerKey{
				Type:         string(value),
				SpecificData: make(map[string][]byte),
			}
		case tagKeyPlugins:
			if current == nil {
				continue
			}
			current.Plugins = splitPlugins(string(value))
		case tagKnownPlugins:
			// Ignored for now, client already knows built-ins.
		case tagPluginSpecific:
			if current == nil {
				continue
			}
			name, data := splitPluginSpecific(value)
			if name == "" {
				continue
			}
			current.SpecificData[name] = data
		default:
			// Unknown tag, ignore to stay forward compatible.
			continue
		}
	}

	if current != nil {
		keys = append(keys, *current)
	}

	return keys, nil
}

func splitPlugins(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case ' ', '\t', ',', ';':
			return true
		default:
			return false
		}
	})
	result := fields[:0]
	for _, f := range fields {
		if trimmed := strings.TrimSpace(f); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitPluginSpecific(buf []byte) (string, []byte) {
	if len(buf) == 0 {
		return "", nil
	}
	parts := bytes.SplitN(buf, []byte{0}, 2)
	if len(parts) == 0 || len(parts[0]) == 0 {
		return "", nil
	}
	if len(parts) == 1 {
		return string(parts[0]), nil
	}
	return string(parts[0]), parts[1]
}

// CollectPluginSpecificData flattens known server keys to plugin-specific data
// entries in the order advertised by the server.
func CollectPluginSpecificData(keys []KnownServerKey) []PluginSpecificData {
	if len(keys) == 0 {
		return nil
	}
	result := make([]PluginSpecificData, 0, len(keys)*2)
	for _, key := range keys {
		for _, plugin := range key.Plugins {
			id := Identifier{Type: key.Type, Plugin: plugin}
			data := key.SpecificData[plugin]
			result = append(result, PluginSpecificData{Identifier: id, Data: data})
		}
	}
	return result
}
