package auth

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	krbclient "github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

type goKrb5Provider struct{}

func NewGoKrb5Provider() GssTokenProvider { return goKrb5Provider{} }

func init() {
	SetGssTokenProvider(NewGoKrb5Provider())
}

func (goKrb5Provider) InitToken(service, host string) ([]byte, error) {
	spn := service + "/" + host
	dbg("[AUTHDBG] gss[gokrb5]: InitToken: service=%q host=%q spn=%q\n", service, host, spn)

	// Try to determine a usable FILE ccache
	ccPath := getCCachePath()
	var cc *credentials.CCache
	if ccPath != "" {
		path := strings.TrimPrefix(ccPath, "FILE:")
		if c, err := credentials.LoadCCache(path); err == nil {
			cc = c
		} else {
			return nil, fmt.Errorf("failed to load ccache %s: %v", path, err)
		}
	}

	// Load krb5.conf if present (with sanitization)
	cfg, cfgPath, err := loadKrb5Config()
	if err != nil {
		return nil, fmt.Errorf("failed to load krb5 config: %v", err)
	} else if cfg != nil {
		dbg("[AUTHDBG] gss[gokrb5]: using krb5 config: %s\n", cfgPath)
	} else if cc != nil {
		// Derive a minimal config from ccache realm and map host -> realm
		cfg = minimalConfigFromCC(cc, host)
		cfgPath = "<derived>"
		dbg("[AUTHDBG] gss[gokrb5]: using derived config from ccache realm=%s for host=%s\n", cc.DefaultPrincipal.Realm, host)
	}

	if cc != nil && cfg != nil {
		cl, err := krbclient.NewFromCCache(cc, cfg)
		if err == nil {
			dbg("[AUTHDBG] gss[gokrb5]: credentials from ccache:%s\n", strings.TrimPrefix(ccPath, "FILE:"))
			return initSpnegoToken(cl, spn)
		}
		dbg("[AUTHDBG] gss[gokrb5]: client from ccache failed: %v\n", err)
	} else if cc == nil {
		dbg("[AUTHDBG] gss[gokrb5]: ccache not available (env/guess failed)\n")
	}

	if cfg == nil {
		// Last resort config that allows DNS discovery
		cfg = config.New()
		cfg.LibDefaults.DNSLookupKDC = true
		cfg.LibDefaults.DNSLookupRealm = true
	}
	if cl, src, err := clientFromKeytab(cfg); err == nil {
		dbg("[AUTHDBG] gss[gokrb5]: credentials from %s", src)
		return initSpnegoToken(cl, spn)
	} else {
		dbg("[AUTHDBG] gss[gokrb5]: keytab not used: %v\n", err)
	}

	if err != nil {
		return nil, fmt.Errorf("client from keytab failed: %v", err)
	}

	return nil, fmt.Errorf("no credentials: set FILE ccache or keytab (KRB5CCNAME or KRB5_CLIENT_KTNAME/FBX_GSS_PRINCIPAL)")
}

func initSpnegoToken(cl *krbclient.Client, spn string) ([]byte, error) {
	// Use SPNEGO mechanism directly (not the HTTP client wrapper)
	s := spnego.SPNEGOClient(cl, spn)
	if err := s.AcquireCred(); err != nil {
		return nil, fmt.Errorf("acquire cred failed: %w", err)
	}
	ct, err := s.InitSecContext()
	if err != nil {
		return nil, err
	}
	nb, err := ct.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal token failed: %w", err)
	}
	dbg("[AUTHDBG] gss[gokrb5]: token generated via SPNEGOClient: len=%d\n", len(nb))
	return nb, nil
}

func clientFromKeytab(cfg *config.Config) (*krbclient.Client, string, error) {
	ktEnv := strings.TrimSpace(os.Getenv("KRB5_CLIENT_KTNAME"))
	if ktEnv == "" {
		ktEnv = strings.TrimSpace(os.Getenv("KRB5_KTNAME"))
	}
	if ktEnv == "" {
		return nil, "keytab", fmt.Errorf("KRB5_CLIENT_KTNAME/KRB5_KTNAME not set")
	}
	path := strings.TrimPrefix(ktEnv, "FILE:")
	if !fileExists(path) {
		return nil, "keytab", fmt.Errorf("keytab not found: %s", path)
	}
	kt, err := keytab.Load(path)
	if err != nil {
		return nil, "keytab", fmt.Errorf("load keytab failed: %w", err)
	}
	principal := strings.TrimSpace(os.Getenv("FBX_GSS_PRINCIPAL"))
	if principal == "" {
		return nil, "keytab", fmt.Errorf("FBX_GSS_PRINCIPAL not set (e.g. user@REALM)")
	}
	parts := strings.Split(principal, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, "keytab", fmt.Errorf("invalid FBX_GSS_PRINCIPAL, expected user@REALM")
	}
	user := parts[0]
	realm := parts[1]

	cl := krbclient.NewWithKeytab(user, realm, kt, cfg)
	if err := cl.Login(); err != nil {
		return nil, "keytab", fmt.Errorf("keytab login failed: %w", err)
	}
	return cl, "keytab:" + path, nil
}

func loadKrb5Config() (*config.Config, string, error) {
	cfgPath := strings.TrimSpace(os.Getenv("KRB5_CONFIG"))
	if cfgPath == "" {
		if runtime.GOOS == "windows" {
			cfgPath = `C:\\Windows\\krb5.ini`
		} else {
			cfgPath = "/etc/krb5.conf"
		}
	}
	cfg, err := config.Load(cfgPath)
	if err == nil {
		return cfg, cfgPath, nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "dns_canonicalize_hostname") && strings.Contains(strings.ToLower(err.Error()), "invalid boolean") {
		b, rerr := os.ReadFile(cfgPath)
		if rerr != nil {
			return nil, cfgPath, rerr
		}
		sanitized := sanitizeKrb5Conf(string(b))
		cfg2, lerr := config.NewFromString(sanitized)
		if lerr != nil {
			return nil, cfgPath, lerr
		}
		return cfg2, cfgPath + " (sanitized)", nil
	}
	return nil, cfgPath, err
}

// sanitizeKrb5Conf replaces unsupported values like 'fallback' for
// dns_canonicalize_hostname with 'false'.
func sanitizeKrb5Conf(s string) string {
	re := regexp.MustCompile(`(?mi)^\s*dns_canonicalize_hostname\s*=\s*fallback\s*$`)
	return re.ReplaceAllString(s, "dns_canonicalize_hostname = false")
}

func minimalConfigFromCC(cc *credentials.CCache, host string) *config.Config {
	cfg := config.New()
	cfg.LibDefaults.DefaultRealm = cc.DefaultPrincipal.Realm
	cfg.LibDefaults.DNSLookupKDC = true
	cfg.LibDefaults.DNSLookupRealm = true
	if host != "" {
		if cfg.DomainRealm == nil {
			cfg.DomainRealm = make(config.DomainRealm)
		}
		cfg.DomainRealm[strings.ToLower(host)] = cc.DefaultPrincipal.Realm
	}
	return cfg
}

func getCCachePath() string {
	cc := strings.TrimSpace(os.Getenv("KRB5CCNAME"))
	if strings.HasPrefix(cc, "FILE:") {
		return cc
	}
	if cc != "" {
		return "" // Non-FILE caches unsupported by gokrb5
	}
	// Try to read default_ccache_name from krb5.conf if it's FILE:
	if c := defaultCCacheFromConf(); strings.HasPrefix(c, "FILE:") {
		p := strings.TrimPrefix(c, "FILE:")
		if fileExists(p) {
			dbg("[AUTHDBG] gss[gokrb5]: default_ccache_name -> %s\n", c)
			return c
		}
	}
	if runtime.GOOS != "windows" {
		if u, err := user.Current(); err == nil && u != nil {
			p := "/tmp/krb5cc_" + u.Uid
			if fileExists(p) {
				dbg("[AUTHDBG] gss[gokrb5]: guessed ccache %s\n", p)
				return "FILE:" + p
			}
			// Broader fallback: only consider caches for current uid
			pattern := "/tmp/krb5cc_" + u.Uid + "*"
			if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
				for _, m := range matches {
					if fileExists(m) {
						dbg("[AUTHDBG] gss[gokrb5]: found ccache candidate for uid %s: %s\n", u.Uid, m)
						return "FILE:" + m
					}
				}
			}
		}
	}
	return ""
}

// defaultCCacheFromConf parses KRB5_CONFIG (or platform default) for
// default_ccache_name and returns it as-is for FILE: caches. Supports
// substituting %{uid} and %{euid}.
func defaultCCacheFromConf() string {
	cfgPath := strings.TrimSpace(os.Getenv("KRB5_CONFIG"))
	if cfgPath == "" {
		if runtime.GOOS == "windows" {
			cfgPath = `C:\\Windows\\krb5.ini`
		} else {
			cfgPath = "/etc/krb5.conf"
		}
	}
	b, err := os.ReadFile(cfgPath)
	if err != nil || len(b) == 0 {
		return ""
	}
	// Quick & lenient parse: look for default_ccache_name = ...
	re := regexp.MustCompile(`(?mi)^\s*default_ccache_name\s*=\s*([^#;\r\n]+)`)
	m := re.FindSubmatch(b)
	if len(m) < 2 {
		return ""
	}
	val := strings.TrimSpace(string(bytes.TrimSpace(m[1])))
	if val == "" || !strings.HasPrefix(strings.ToUpper(val), "FILE:") {
		return ""
	}
	// Expand %{uid}/%{euid}
	if u, err := user.Current(); err == nil && u != nil {
		val = strings.ReplaceAll(val, "%{uid}", u.Uid)
		val = strings.ReplaceAll(val, "%{euid}", u.Uid)
	}
	return val
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	if !filepath.IsAbs(p) {
		// Accept relative as well, resolve to absolute for debug.
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Debug helpers
func dbg(format string, args ...any) {
	gssDebugEnabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("FBX_AUTH_DEBUG")))
	if gssDebugEnabled {
		fmt.Printf(format, args...)
	}
}
