package fbconn

import (
	"errors"
	"fmt"
	"maps"
	"strings"
)

func parseURLSettings(connString string) (settings map[string]string, isEmptyPortList bool, err error) {
	settings = make(map[string]string)

	if strings.IndexByte(connString, 0) >= 0 {
		return nil, isEmptyPortList, errors.New("forbidden NUL byte in connection string")
	}

	p, ok := strings.CutPrefix(connString, "firebirdsql://")
	if !ok {
		p, ok = strings.CutPrefix(connString, "firebird://")
	}
	if !ok {
		return nil, isEmptyPortList, errors.New("invalid URI propagated to internal parser routine")
	}

	if i := strings.IndexAny(p, "@/"); i >= 0 && p[i] == '@' {
		user, password, hasPassword := strings.Cut(p[:i], ":")
		p = p[i+1:]
		val, err := uriDecode(user, "user")
		if err != nil {
			return nil, isEmptyPortList, err
		}
		settings["user"] = val
		if hasPassword {
			val, err := uriDecode(password, "password")
			if err != nil {
				return nil, isEmptyPortList, err
			}
			settings["password"] = val
		}

	}

	var hostBuf, portBuf strings.Builder

	for {
		if peekByte(p) == '[' {
			// IPv6 address. The bracket contents are not validated.
			end := strings.IndexByte(p, ']')
			if end < 0 {
				return nil, isEmptyPortList, errors.New(`end of string reached when looking for matching "]" in IPv6 host address in URI`)
			}
			if end == 1 {
				return nil, isEmptyPortList, errors.New("IPv6 host address may not be empty in URI")
			}
			hostBuf.WriteString(p[1:end])
			p = p[end+1:]
			if c := peekByte(p); c != 0 && c != ':' && c != '/' && c != '?' && c != ',' {
				return nil, isEmptyPortList, fmt.Errorf(`unexpected character "%c" at position %d in URI (expected ":", "/", ",", "?")`, c, len(connString)-len(p)+1)
			}
		} else {
			// DNS-named or IPv4 netloc: everything up to a ':', '/', '?', or ','.
			i := strings.IndexAny(p, ":/?,")
			if i < 0 {
				i = len(p)
			}
			hostBuf.WriteString(p[:i])
			p = p[i:]
		}

		if peekByte(p) == ':' {
			p = p[1:]
			i := strings.IndexAny(p, "/?,")
			if i < 0 {
				i = len(p)
			}
			portBuf.WriteString(p[:i])
			p = p[i:]
		}

		if peekByte(p) != ',' {
			break
		}
		p = p[1:]
		hostBuf.WriteByte(',')
		portBuf.WriteByte(',')
	}

	// The joined lists are percent-decoded as a unit, after joining. This
	// means %2C decodes to a comma that later splits like a real separator --
	// a host name containing a literal comma is inexpressible.
	if hostBuf.Len() > 0 {
		hostList, err := uriDecode(hostBuf.String(), "host")
		if err != nil {
			return nil, isEmptyPortList, err
		}
		settings["host"] = hostList
	}
	// The port list is stored even when every element is empty
	// (firebird://h1,h2 yields port=","). This matters: a connection string
	// port shadows FBPORT during settings merge, so with FBPORT=1,2 a
	// three-host URI without ports must use the default port for every host,
	// not fail the port/host count check. Such an implied all-empty list
	// carries no user-supplied port text, which is what
	// isEmptyPortList reports -- judged on the raw bytes, before
	// percent-decoding trims spaces, so a port the user actually typed never
	// counts as implied even when it decodes to empty.
	if portBuf.Len() > 0 {
		isEmptyPortList = strings.Trim(portBuf.String(), ",") == ""
		portList, err := uriDecode(portBuf.String(), "port")
		if err != nil {
			return nil, isEmptyPortList, err
		}
		settings["port"] = portList
	}

	if peekByte(p) == '/' {
		p = p[1:]
		dbname := p
		p = ""
		if i := strings.IndexByte(dbname, '?'); i >= 0 {
			p = dbname[i:]
			dbname = dbname[:i]
		}
		val, err := uriDecode(dbname, "dbname")
		if err != nil {
			return nil, isEmptyPortList, err
		}
		settings["database"] = val
	}

	if peekByte(p) == '?' {
		// Query parameters land in their own map first: an explicit ?port=
		// overrides the netloc-derived port list and is user-supplied port
		// text, so it cancels the implied-list exemption even when its value
		// is empty.
		query := make(map[string]string)
		if err := parseURLQueryParams(p[1:], query); err != nil {
			return nil, isEmptyPortList, err
		}
		if _, ok := query["port"]; ok {
			isEmptyPortList = false
		}
		maps.Copy(settings, query)
	}

	return settings, isEmptyPortList, nil
}

// peekByte returns the first byte of s, or 0 if s is empty. A NUL byte is a
// safe end-of-input sentinel because parseURLSettings rejects literal NULs.
func peekByte(s string) byte {
	if len(s) == 0 {
		return 0
	}
	return s[0]
}

// parseURLQueryParams parses the query part of a connection URI into settings:
// '&'-separated pairs, exactly one raw '=' per pair, both halves percent-decoded,
// last occurrence wins.
func parseURLQueryParams(params string, settings map[string]string) error {
	for params != "" {
		pair := params
		if i := strings.IndexByte(params, '&'); i >= 0 {
			pair = params[:i]
			params = params[i+1:]
		} else {
			params = ""
		}

		rawKey, rawValue, found := strings.Cut(pair, "=")
		if !found {
			return fmt.Errorf(`missing key/value separator "=" in URI query parameter: "%s"`, rawKey)
		}
		if strings.IndexByte(rawValue, '=') >= 0 {
			return fmt.Errorf(`extra key/value separator "=" in URI query parameter: "%s"`, rawKey)
		}

		key, err := uriDecode(rawKey, "parameter_key")
		if err != nil {
			return err
		}
		key = canonicalConnStringKey(key)

		secretName := ""
		if key == "password" {
			secretName = key
		}
		value, err := uriDecode(rawValue, secretName)
		if err != nil {
			return err
		}

		settings[key] = value
	}

	return nil
}

// uriDecode percent-decodes %HH sequences in a URI component.
//
// If secretName is non-empty, error messages identify that component by name
// instead of quoting its raw value. This prevents the known component's raw
// contents from being copied into the decode error; it does not make the
// separate, best-effort redaction of an invalid connection string complete.
func uriDecode(raw, secretName string) (string, error) {
	// The component may be a secret; every error must be built by fail so the
	// quote-the-raw-value-or-not redaction decision lives in exactly one place.
	fail := func(secretFormat, publicFormat string) (string, error) {
		if secretName != "" {
			return "", fmt.Errorf(secretFormat, secretName)
		}
		return "", fmt.Errorf(publicFormat, raw)
	}

	var b strings.Builder
	b.Grow(len(raw))

	i := 0
	for i < len(raw) && raw[i] == ' ' {
		i++
	}
	for i < len(raw) && raw[i] != ' ' {
		if raw[i] != '%' {
			b.WriteByte(raw[i])
			i++
			continue
		}

		var hi, lo byte
		var ok1, ok2 bool
		if i+1 < len(raw) {
			hi, ok1 = hexDigit(raw[i+1])
		}
		if i+2 < len(raw) {
			lo, ok2 = hexDigit(raw[i+2])
		}
		if !ok1 || !ok2 {
			return fail("invalid percent-encoded token in %s", `invalid percent-encoded token: "%s"`)
		}

		c := hi<<4 | lo
		if c == 0 {
			return fail("forbidden value %%00 in percent-encoded value in %s", `forbidden value %%00 in percent-encoded value: "%s"`)
		}
		b.WriteByte(c)
		i += 3
	}
	for i < len(raw) && raw[i] == ' ' {
		i++
	}
	if i < len(raw) {
		return fail("unexpected spaces found in %s, use percent-encoded spaces (%%20) instead", `unexpected spaces found in "%s", use percent-encoded spaces (%%20) instead`)
	}

	return b.String(), nil
}

// uriDecodeLenient is a non-failing variant of uriDecode used for best-effort
// redaction: valid %HH sequences are decoded without validating the resulting bytes;
// malformed sequences are preserved. Redaction processing itself must not fail,
// and it must see the same key spelling the parser would (pass%77ord decodes
// to password), so encoded password keys are recognized even in connection
// strings that do not parse. This helps recognize keys but cannot make
// redaction complete when malformed input has ambiguous component boundaries.
func uriDecodeLenient(raw string) string {
	raw = strings.Trim(raw, " ")

	var b strings.Builder
	b.Grow(len(raw))

	for i := 0; i < len(raw); {
		if raw[i] == '%' && i+2 < len(raw) {
			hi, ok1 := hexDigit(raw[i+1])
			lo, ok2 := hexDigit(raw[i+2])
			if ok1 && ok2 {
				b.WriteByte(hi<<4 | lo)
				i += 3
				continue
			}
		}
		b.WriteByte(raw[i])
		i++
	}

	return b.String()
}

func hexDigit(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	}
	return 0, false
}
