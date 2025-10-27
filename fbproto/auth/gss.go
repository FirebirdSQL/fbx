package auth

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

// GssAuthPlugin implements the Firebird/RedDatabase GSS (Kerberos) authentication
// handshake from the client side.
//
// Behavior:
//   - On first call with empty server data, it requests more data by sending an
//     empty op_cont_auth payload (server then responds with host/service info).
//   - When server data is present, it parses host and service, generates a GSS
//     init token for SPN "service/host" (or hostbased service name equivalent),
//     and returns it both as Reply (to send with op_cont_auth) and as
//     AuthSpecific (to include in DPB specific_auth_data during attach).
//
// This plugin does not derive a session key; wire encryption is negotiated
// separately via db-crypt plugins.
type GssAuthPlugin struct{}

func NewGssAuthPlugin() *GssAuthPlugin { return &GssAuthPlugin{} }

func (p *GssAuthPlugin) Name() string { return "Gss" }

func (p *GssAuthPlugin) Advertise() []string { return []string{p.Name()} }

// InitialClientData for GSS is empty; we first wait for server-specific data
// (host, service) to build the SPN.
func (p *GssAuthPlugin) InitialClientData() []byte { return nil }

func (p *GssAuthPlugin) HandleServerData(user string, password string, pluginName string, data []byte) (*AuthPluginStep, error) {
	// If server hasn't provided GSS parameters yet, request more data by
	// returning a non-nil step with an empty Reply (client will send op_cont_auth
	// with empty p_data).
	if len(data) == 0 {
		gssDebugf("gss: awaiting server data (host/service)\n")
		return &AuthPluginStep{Reply: nil}, nil
	}

	host, service, err := parseGssServerData(data)
	if err != nil {
		return nil, err
	}

	// Form SPN. Kerberos SPNs generally use service/host.
	spn := service + "/" + host
	gssDebugf("[AUTHDBG] gss: parsed host=%q service=%q spn=%q\n", host, service, spn)

	prov := getGssTokenProvider()
	if prov == nil {
		return nil, fmt.Errorf("gss: no token provider configured for SPN %q", spn)
	}

	token, err := prov.InitToken(service, host)
	if err != nil {
		return nil, fmt.Errorf("gss: token init failed for %q: %w", spn, err)
	}
	gssDebugf("[AUTHDBG] gss: token generated, len=%d\n", len(token))

	// Reply carries the client token to server; AuthSpecific duplicates it so
	// it can be included in DPB specific_auth_data later.
	return &AuthPluginStep{Reply: cloneBytes(token), AuthSpecific: cloneBytes(token)}, nil
}

func (p *GssAuthPlugin) HandleAuthResponse(data []byte) error { return nil }

// parseGssServerData parses the server-provided payload for GSS, which is
// defined as two little-endian length-prefixed strings: host and service.
func parseGssServerData(data []byte) (host, service string, err error) {
	if len(data) < 4 {
		return "", "", fmt.Errorf("gss: malformed payload")
	}
	// host
	hostLen := int(binary.LittleEndian.Uint16(data[0:2]))
	if hostLen < 0 || 2+hostLen > len(data) {
		return "", "", fmt.Errorf("gss: malformed host in payload")
	}
	host = string(data[2 : 2+hostLen])
	off := 2 + hostLen

	if off+2 > len(data) {
		return "", "", fmt.Errorf("gss: malformed payload")
	}
	// service
	svcLen := int(binary.LittleEndian.Uint16(data[off : off+2]))
	off += 2
	if svcLen < 0 || off+svcLen > len(data) {
		return "", "", fmt.Errorf("gss: malformed service in payload")
	}
	service = string(data[off : off+svcLen])

	host = strings.TrimSpace(host)
	service = strings.TrimSpace(service)
	if host == "" || service == "" {
		return "", "", fmt.Errorf("gss: empty host or service in payload")
	}
	return host, service, nil
}

// Token provider abstraction
//
// The GSS plugin relies on an environment-specific token provider to acquire
// a client GSS token (eg using system GSSAPI or a pure Go Kerberos library)
// for the host/service emitted by the server. By default, no provider is set.

type GssTokenProvider interface {
	// InitToken should return an initial client token for the SPN computed
	// from service and host (eg service/host).
	InitToken(service, host string) ([]byte, error)
}

// gssProvider holds the global token provider.
// It is intended to be set once during process startup and not changed at runtime.
// We wrap the provider in a fixed concrete type to allow atomic.Value stores of
// different implementations without hitting the "inconsistent type" panic.
type providerSlot struct{ p GssTokenProvider }

var gssProvider atomic.Value // stores providerSlot

// SetGssTokenProvider installs a global token provider used by the GSS plugin.
func SetGssTokenProvider(p GssTokenProvider) { gssProvider.Store(providerSlot{p: p}) }

func getGssTokenProvider() GssTokenProvider {
	v := gssProvider.Load()
	if v == nil {
		return nil
	}
	if slot, ok := v.(providerSlot); ok {
		return slot.p
	}
	return nil
}

// Debug logging controlled by FBX_AUTH_DEBUG.
func gssDebugf(format string, args ...any) {
	gssDebugEnabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("FBX_AUTH_DEBUG")))
	if gssDebugEnabled {
		fmt.Printf(format, args...)
	}
}
