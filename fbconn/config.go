package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/red-soft/fbx/fbconn/ctxwatch"
	"github.com/red-soft/fbx/fbproto"
	"github.com/red-soft/fbx/fbproto/auth"
)

type AfterConnectFunc func(ctx context.Context, fbconn *FbConn) error
type ValidateConnectFunc func(ctx context.Context, fbconn *FbConn) error
type GetSSLPasswordFunc func(ctx context.Context) string

// Config is the settings used to establish a connection to a PostgreSQL server. It must be created by [ParseConfig]. A
// manually initialized Config will cause ConnectConfig to panic.
type Config struct {
	Host                       string // host (e.g. localhost) or absolute path to unix domain socket directory (e.g. /private/tmp)
	Port                       uint16
	Database                   string
	User                       string
	Role                       string
	SystemUser                 string
	SystemHost                 string
	Password                   string
	ConnectTimeout             time.Duration
	DialFunc                   DialFunc   // e.g. net.Dialer.DialContext
	LookupFunc                 LookupFunc // e.g. net.Resolver.LookupHost
	BuildFrontend              BuildFrontendFunc
	AuthPluginList             []string
	WireCrypt                  bool
	Encoding                   string
	DPB                        []byte
	ClientVersion              string
	DefaultTransactionReadOnly bool

	// BuildContextWatcherHandler is called to create a ContextWatcherHandler for a connection. The handler is called
	// when a context passed to a FbConn method is canceled.
	BuildContextWatcherHandler func(*FbConn) ctxwatch.Handler

	RuntimeParams map[string]string // Run-time parameters to set on connection as session default values (e.g. search_path or application_name)

	KerberosSrvName string
	KerberosSpn     string

	Fallbacks []*FallbackConfig

	// ValidateConnect is called during a connection attempt after a successful authentication with the PostgreSQL server.
	// It can be used to validate that the server is acceptable. If this returns an error the connection is closed and the next
	// fallback config is tried. This allows implementing high availability behavior such as libpq does with target_session_attrs.
	ValidateConnect ValidateConnectFunc

	// AfterConnect is called after ValidateConnect. It can be used to set up the connection (e.g. Set session variables
	// or prepare statements). If this returns an error the connection attempt fails.
	AfterConnect AfterConnectFunc

	// OnNotice is a callback function called when a notice response is received.
	OnNotice NoticeHandler

	// OnNotification is a callback function called when a notification from the LISTEN/NOTIFY system is received.
	OnNotification NotificationHandler

	// OnFbError is a callback function called when a Firebird error happened on server
	OnFbError FbErrorHandler

	// OnFbWarning is a callback function called when a Firebird warning obtained by client with success response
	OnFbWarning FbWarningHandler

	createdByParseConfig bool // Used to enforce created by ParseConfig rule.
}

// ParseConfigOptions contains options that control how a config is built such as GetSSLPassword.
type ParseConfigOptions struct {
	// GetSSLPassword gets the password to decrypt a SSL client certificate. This is analogous to the libpq function
	// PQsetSSLKeyPassHook_OpenSSL.
	GetSSLPassword GetSSLPasswordFunc
}

// Copy returns a deep copy of the config that is safe to use and modify.
// The only exception is the TLSConfig field:
// according to the tls.Config docs it must not be modified after creation.
func (c *Config) Copy() *Config {
	newConf := new(Config)
	*newConf = *c
	if newConf.RuntimeParams != nil {
		newConf.RuntimeParams = make(map[string]string, len(c.RuntimeParams))
		for k, v := range c.RuntimeParams {
			newConf.RuntimeParams[k] = v
		}
	}
	if len(newConf.AuthPluginList) > 0 {
		newList := make([]string, len(c.AuthPluginList))
		copy(newList, c.AuthPluginList)
		newConf.AuthPluginList = newList
	}
	return newConf
}

// FallbackConfig is additional settings to attempt a connection with when the primary Config fails to establish a
// network connection. It is used for TLS fallback such as sslmode=prefer and high availability (HA) connections.
type FallbackConfig struct {
	Host string // host (e.g. localhost) or path to unix domain socket directory (e.g. /private/tmp)
	Port uint16
}

// connectOneConfig is the configuration for a single attempt to connect to a single host.
type connectOneConfig struct {
	network          string
	address          string
	originalHostname string // original hostname before resolving
}

// isAbsolutePath checks if the provided value is an absolute path either
// beginning with a forward slash (as on Linux-based systems) or with a capital
// letter A-Z followed by a colon and a backslash, e.g., "C:\", (as on Windows).
func isAbsolutePath(path string) bool {
	isWindowsPath := func(p string) bool {
		if len(p) < 3 {
			return false
		}
		drive := p[0]
		colon := p[1]
		backslash := p[2]
		if drive >= 'A' && drive <= 'Z' && colon == ':' && backslash == '\\' {
			return true
		}
		return false
	}
	return strings.HasPrefix(path, "/") || isWindowsPath(path)
}

// NetworkAddress converts a PostgreSQL host and port into network and address suitable for use with
// net.Dial.
func NetworkAddress(host string, port uint16) (network, address string) {
	if isAbsolutePath(host) {
		network = "tcp"
		//TODO: fbx allow override default port
		address = net.JoinHostPort("localhost", "3050")
	} else {
		network = "tcp"
		address = net.JoinHostPort(host, strconv.Itoa(int(port)))
	}
	return network, address
}

// ParseConfig builds a *Config from connString with similar behavior to the PostgreSQL standard C library libpq. It
// uses the same defaults as libpq (e.g. port=5432) and understands most PG* environment variables. ParseConfig closely
// matches the parsing behavior of libpq. connString may either be in URL format or keyword = value format. See
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING for details. connString also may be empty
// to only read from the environment. If a password is not supplied it will attempt to read the .pgpass file.
//
//	# Example Keyword/Value
//	user=jack password=secret host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca
//
//	# Example URL
//	postgres://jack:secret@pg.example.com:5432/mydb?sslmode=verify-ca
//
// The returned *Config may be modified. However, it is strongly recommended that any configuration that can be done
// through the connection string be done there. In particular the fields Host, Port, TLSConfig, and Fallbacks can be
// interdependent (e.g. TLSConfig needs knowledge of the host to validate the server certificate). These fields should
// not be modified individually. They should all be modified or all left unchanged.
//
// ParseConfig supports specifying multiple hosts in similar manner to libpq. Host and port may include comma separated
// values that will be tried in order. This can be used as part of a high availability system. See
// https://www.postgresql.org/docs/11/libpq-connect.html#LIBPQ-MULTIPLE-HOSTS for more information.
//
//	# Example URL
//	postgres://jack:secret@foo.example.com:5432,bar.example.com:5432/mydb
//
// ParseConfig currently recognizes the following environment variable and their parameter key word equivalents passed
// via database URL or keyword/value:
//
//	FBHOST
//	FBPORT
//	FBDATABASE
//	FBUSER
//	FBPASSWORD
//	FBPASSFILE
//	FBSERVICE
//	FBSERVICEFILE
//	FBSSLMODE
//	FBSSLCERT
//	FBSSLKEY
//	FBSSLROOTCERT
//	FBSSLPASSWORD
//	FBOPTIONS
//	FBAPPNAME
//	FBCONNECT_TIMEOUT
//	FBTARGETSESSIONATTRS
//	FBTZ
//
// See http://www.postgresql.org/docs/11/static/libpq-envars.html for details on the meaning of environment variables.
//
// See https://www.postgresql.org/docs/11/libpq-connect.html#LIBPQ-PARAMKEYWORDS for parameter key word names. They are
// usually but not always the environment variable name downcased and without the "PG" prefix.
//
// Important Security Notes:
//
// ParseConfig tries to match libpq behavior with regard to PGSSLMODE. This includes defaulting to "prefer" behavior if
// not set.
//
// See http://www.postgresql.org/docs/11/static/libpq-ssl.html#LIBPQ-SSL-PROTECTION for details on what level of
// security each sslmode provides.
//
// The sslmode "prefer" (the default), sslmode "allow", and multiple hosts are implemented via the Fallbacks field of
// the Config struct. If TLSConfig is manually changed it will not affect the fallbacks. For example, in the case of
// sslmode "prefer" this means it will first try the main Config settings which use TLS, then it will try the fallback
// which does not use TLS. This can lead to an unexpected unencrypted connection if the main TLS config is manually
// changed later but the unencrypted fallback is present. Ensure there are no stale fallbacks when manually setting
// TLSConfig.
//
// Other known differences with libpq:
//
// When multiple hosts are specified, libpq allows them to have different passwords set via the .pgpass file. fbconn
// does not.
//
// In addition, ParseConfig accepts the following options:
//
//   - servicefile.
//     libpq only reads servicefile from the PGSERVICEFILE environment variable. ParseConfig accepts servicefile as a
//     part of the connection string.
func ParseConfig(connString string) (*Config, error) {
	var parseConfigOptions ParseConfigOptions
	return ParseConfigWithOptions(connString, parseConfigOptions)
}

// ParseConfigWithOptions builds a *Config from connString and options with similar behavior to the PostgreSQL standard
// C library libpq. options contains settings that cannot be specified in a connString such as providing a function to
// get the SSL password.
func ParseConfigWithOptions(connString string, options ParseConfigOptions) (*Config, error) {
	defaultSettings := defaultSettings()
	envSettings := parseEnvSettings()

	connStringSettings := make(map[string]string)
	if connString != "" {
		var err error
		// connString may be a database URL or in PostgreSQL keyword/value format
		if strings.HasPrefix(connString, "firebird://") || strings.HasPrefix(connString, "firebirdsql://") {
			connStringSettings, err = parseURLSettings(connString)
			if err != nil {
				return nil, &ParseConfigError{ConnString: connString, msg: "failed to parse as URL", err: err}
			}
		} else {
			connStringSettings, err = parseKeywordValueSettings(connString)
			if err != nil {
				return nil, &ParseConfigError{ConnString: connString, msg: "failed to parse as keyword/value", err: err}
			}
		}
	}

	settings := mergeSettings(defaultSettings, envSettings, connStringSettings)
	encoding, err := fbproto.NewEncoding(settings["encoding"])
	if err != nil {
		return nil, err
	}

	rawPluginList := settings["auth_plugin_list"]
	pluginList := parseAuthPluginList(rawPluginList)
	if len(pluginList) == 0 {
		pluginList = auth.DefaultPluginSequence()
	}

	config := &Config{
		createdByParseConfig: true,
		Database:             settings["database"],
		User:                 settings["user"],
		Password:             settings["password"],
		Role:                 settings["role"],
		RuntimeParams:        make(map[string]string),
		BuildFrontend: func(r io.Reader, w io.Writer) *fbproto.Client {
			return fbproto.NewClient(r, w, encoding)
		},
		BuildContextWatcherHandler: func(fbConn *FbConn) ctxwatch.Handler {
			return &DeadlineContextWatcherHandler{Conn: fbConn.conn}
		},
		OnFbError: func(_ *FbConn, fbErr *FbError) bool {
			// we want to automatically close any fatal errors
			// FIXME: fbx how to check if server error is fatal?
			//if strings.EqualFold(pgErr.Severity, "FATAL") {
			//	return false
			//}
			return true
		},
		AuthPluginList:             pluginList,
		WireCrypt:                  convertToBool(settings["wire_crypt"], true),
		SystemUser:                 settings["system_user"],
		SystemHost:                 settings["system_host"],
		ClientVersion:              settings["client_version"],
		Encoding:                   settings["encoding"],
		DefaultTransactionReadOnly: convertToBool(settings["default_transaction_read_only"], false),
	}

	if connectTimeoutSetting, present := settings["connect_timeout"]; present {
		connectTimeout, err := parseConnectTimeoutSetting(connectTimeoutSetting)
		if err != nil {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid connect_timeout", err: err}
		}
		config.ConnectTimeout = connectTimeout
		config.DialFunc = makeConnectTimeoutDialFunc(connectTimeout)
	} else {
		defaultDialer := makeDefaultDialer()
		config.DialFunc = defaultDialer.DialContext
	}

	config.LookupFunc = makeDefaultResolver().LookupHost

	notRuntimeParams := map[string]struct{}{
		"host":                          {},
		"port":                          {},
		"database":                      {},
		"user":                          {},
		"password":                      {},
		"connect_timeout":               {},
		"krbspn":                        {},
		"krbsrvname":                    {},
		"auth_plugin_list":              {},
		"system_user":                   {},
		"wire_crypt":                    {},
		"system_host":                   {},
		"client_version":                {},
		"encoding":                      {},
		"default_transaction_read_only": {},
	}

	// Adding kerberos configuration
	if _, present := settings["krbsrvname"]; present {
		config.KerberosSrvName = settings["krbsrvname"]
	}
	if _, present := settings["krbspn"]; present {
		config.KerberosSpn = settings["krbspn"]
	}

	for k, v := range settings {
		if _, present := notRuntimeParams[k]; present {
			continue
		}
		config.RuntimeParams[k] = v
	}

	host := settings["host"]
	portStr := settings["port"]

	port, err := parsePort(portStr)
	if err != nil {
		return nil, &ParseConfigError{ConnString: connString, msg: "invalid port", err: err}
	}

	config.Host = host
	config.Port = port

	return config, nil
}

func mergeSettings(settingSets ...map[string]string) map[string]string {
	settings := make(map[string]string)

	for _, s2 := range settingSets {
		for k, v := range s2 {
			settings[k] = v
		}
	}

	return settings
}

func parseEnvSettings() map[string]string {
	settings := make(map[string]string)

	nameMap := map[string]string{
		"FBHOST":               "host",
		"FBPORT":               "port",
		"FBDATABASE":           "database",
		"FBUSER":               "user",
		"FBPASSWORD":           "password",
		"FBPASSFILE":           "passfile",
		"FBAPPNAME":            "application_name",
		"FBCONNECT_TIMEOUT":    "connect_timeout",
		"FBSSLMODE":            "sslmode",
		"FBSSLKEY":             "sslkey",
		"FBSSLCERT":            "sslcert",
		"FBSSLSNI":             "sslsni",
		"FBSSLROOTCERT":        "sslrootcert",
		"FBSSLPASSWORD":        "sslpassword",
		"FBSSLNEGOTIATION":     "sslnegotiation",
		"FBTARGETSESSIONATTRS": "target_session_attrs",
		"FBSERVICE":            "service",
		"FBSERVICEFILE":        "servicefile",
		"FBTZ":                 "timezone",
		"FBOPTIONS":            "options",
	}

	for envname, realname := range nameMap {
		value := os.Getenv(envname)
		if value != "" {
			settings[realname] = value
		}
	}

	return settings
}

func parseAuthPluginList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	splitFn := func(r rune) bool {
		switch r {
		case ',', ';', ' ', '\t', '\n', '\r':
			return true
		default:
			return false
		}
	}

	parts := strings.FieldsFunc(raw, splitFn)
	if len(parts) == 0 {
		return nil
	}

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseURLSettings(connString string) (map[string]string, error) {
	settings := make(map[string]string)

	parsedURL, err := url.Parse(connString)
	if err != nil {
		if urlErr := new(url.Error); errors.As(err, &urlErr) {
			return nil, urlErr.Err
		}
		return nil, err
	}

	if parsedURL.User != nil {
		settings["user"] = parsedURL.User.Username()
		if password, present := parsedURL.User.Password(); present {
			settings["password"] = password
		}
	}

	// Handle multiple host:port's in url.Host by splitting them into host,host,host and port,port,port.
	var hosts []string
	var ports []string
	for _, host := range strings.Split(parsedURL.Host, ",") {
		if host == "" {
			continue
		}
		if isIPOnly(host) {
			hosts = append(hosts, strings.Trim(host, "[]"))
			continue
		}
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return nil, fmt.Errorf("failed to split host:port in '%s', err: %w", host, err)
		}
		if h != "" {
			hosts = append(hosts, h)
		}
		if p != "" {
			ports = append(ports, p)
		}
	}
	if len(hosts) > 0 {
		settings["host"] = strings.Join(hosts, ",")
	}
	if len(ports) > 0 {
		settings["port"] = strings.Join(ports, ",")
	}

	var database string
	if len(parsedURL.Path) > 0 {
		database = parsedURL.Path[1:]
	}
	if parsedURL.Path != "" {
		settings["database"] = database
	}

	nameMap := map[string]string{
		"dbname": "database",
	}

	for k, v := range parsedURL.Query() {
		if k2, present := nameMap[k]; present {
			k = k2
		}

		settings[k] = v[0]
	}

	return settings, nil
}

func isIPOnly(host string) bool {
	return net.ParseIP(strings.Trim(host, "[]")) != nil || !strings.Contains(host, ":")
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

func parseKeywordValueSettings(s string) (map[string]string, error) {
	settings := make(map[string]string)

	nameMap := map[string]string{
		"dbname": "database",
	}

	for len(s) > 0 {
		var key, val string
		eqIdx := strings.IndexRune(s, '=')
		if eqIdx < 0 {
			return nil, errors.New("invalid keyword/value")
		}

		key = strings.Trim(s[:eqIdx], " \t\n\r\v\f")
		s = strings.TrimLeft(s[eqIdx+1:], " \t\n\r\v\f")
		if len(s) == 0 {
		} else if s[0] != '\'' {
			end := 0
			for ; end < len(s); end++ {
				if asciiSpace[s[end]] == 1 {
					break
				}
				if s[end] == '\\' {
					end++
					if end == len(s) {
						return nil, errors.New("invalid backslash")
					}
				}
			}
			val = strings.Replace(strings.Replace(s[:end], "\\\\", "\\", -1), "\\'", "'", -1)
			if end == len(s) {
				s = ""
			} else {
				s = s[end+1:]
			}
		} else { // quoted string
			s = s[1:]
			end := 0
			for ; end < len(s); end++ {
				if s[end] == '\'' {
					break
				}
				if s[end] == '\\' {
					end++
				}
			}
			if end == len(s) {
				return nil, errors.New("unterminated quoted string in connection info string")
			}
			val = strings.Replace(strings.Replace(s[:end], "\\\\", "\\", -1), "\\'", "'", -1)
			if end == len(s) {
				s = ""
			} else {
				s = s[end+1:]
			}
		}

		if k, ok := nameMap[key]; ok {
			key = k
		}

		if key == "" {
			return nil, errors.New("invalid keyword/value")
		}

		settings[key] = val
	}

	return settings, nil
}

func parseServiceSettings(servicefilePath, serviceName string) (map[string]string, error) {
	//servicefile, err := pgservicefile.ReadServicefile(servicefilePath)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to read service file: %v", servicefilePath)
	//}
	//
	//service, err := servicefile.GetService(serviceName)
	//if err != nil {
	//	return nil, fmt.Errorf("unable to find service: %v", serviceName)
	//}
	//
	//nameMap := map[string]string{
	//	"dbname": "database",
	//}
	//
	//settings := make(map[string]string, len(service.Settings))
	//for k, v := range service.Settings {
	//	if k2, present := nameMap[k]; present {
	//		k = k2
	//	}
	//	settings[k] = v
	//}
	//
	//return settings, nil
	panic("implement me")
}

func parsePort(s string) (uint16, error) {
	port, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > math.MaxUint16 {
		return 0, errors.New("outside range")
	}
	return uint16(port), nil
}

func makeDefaultDialer() *net.Dialer {
	// rely on GOLANG KeepAlive settings
	return &net.Dialer{}
}

func makeDefaultResolver() *net.Resolver {
	return net.DefaultResolver
}

func parseConnectTimeoutSetting(s string) (time.Duration, error) {
	timeout, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if timeout < 0 {
		return 0, errors.New("negative timeout")
	}
	return time.Duration(timeout) * time.Second, nil
}

func makeConnectTimeoutDialFunc(timeout time.Duration) DialFunc {
	d := makeDefaultDialer()
	d.Timeout = timeout
	return d.DialContext
}

// ValidateConnectTargetSessionAttrsReadWrite is a ValidateConnectFunc that implements libpq compatible
// target_session_attrs=read-write.
func ValidateConnectTargetSessionAttrsReadWrite(ctx context.Context, fbConn *FbConn) error {
	res := fbConn.Exec(ctx, "select mon$read_only from mon$transactions where mon$transaction_id = current_transaction").Read()
	if res.Err != nil {
		return res.Err
	}

	if res.Rows[0][0][3] == 1 {
		return errors.New("read only connection")
	}

	return nil
}

// ValidateConnectTargetSessionAttrsReadOnly is a ValidateConnectFunc that implements libpq compatible
// target_session_attrs=read-only.
func ValidateConnectTargetSessionAttrsReadOnly(ctx context.Context, fbConn *FbConn) error {
	res := fbConn.Exec(ctx, "select mon$read_only from mon$transactions where mon$transaction_id = current_transaction").Read()
	if res.Err != nil {
		return res.Err
	}

	if res.Rows[0][0][3] != 1 {
		return errors.New("connection is not read only")
	}

	return nil
}

//FIXME: fbx, possible to implement it by scanning MON$REPLICATION or obtaining status from context variables https://tracker.red-soft.ru/issues/261350
// ValidateConnectTargetSessionAttrsStandby is a ValidateConnectFunc that implements libpq compatible
// target_session_attrs=standby.
//func ValidateConnectTargetSessionAttrsStandby(ctx context.Context, fbConn *FbConn) error {
//	result, err := fbConn.Exec(ctx, "select pg_is_in_recovery()").ReadAll()
//	if err != nil {
//		return err
//	}
//
//	if string(result[0].Rows[0][0]) != "t" {
//		return errors.New("server is not in hot standby mode")
//	}
//
//	return nil
//}

// ValidateConnectTargetSessionAttrsPrimary is a ValidateConnectFunc that implements libpq compatible
// target_session_attrs=primary.
//func ValidateConnectTargetSessionAttrsPrimary(ctx context.Context, fbConn *FbConn) error {
//	result, err := fbConn.Exec(ctx, "select pg_is_in_recovery()").ReadAll()
//	if err != nil {
//		return err
//	}
//
//	if string(result[0].Rows[0][0]) == "t" {
//		return errors.New("server is in standby mode")
//	}
//
//	return nil
//}

// ValidateConnectTargetSessionAttrsPreferStandby is a ValidateConnectFunc that implements libpq compatible
// target_session_attrs=prefer-standby.
//func ValidateConnectTargetSessionAttrsPreferStandby(ctx context.Context, fbConn *FbConn) error {
//	result, err := fbConn.Exec(ctx, "select pg_is_in_recovery()").ReadAll()
//	if err != nil {
//		return err
//	}
//
//	if string(result[0].Rows[0][0]) != "t" {
//		return &NotPreferredError{err: errors.New("server is not in hot standby mode")}
//	}
//
//	return nil
//}

func convertToBool(s string, defaultValue bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		v = defaultValue
	}
	return v
}
