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

	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/auth"
)

type AfterConnectFunc func(ctx context.Context, fbconn *FbConn) error
type ValidateConnectFunc func(ctx context.Context, fbconn *FbConn) error
type GetSSLPasswordFunc func(ctx context.Context) string

const DefaultBlobBufferSize = 16384
const DefaultFetchBatchSize = 400

// Config contains settings used to establish a connection to a Firebird/RedDatabase server. It must be created by [ParseConfig] (or ParseConfigWithOptions). A
// manually initialized Config will cause ConnectConfig to panic.
type Config struct {
	Host                       string
	Port                       uint16
	Database                   string
	CreateDatabase             bool
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
	Autocommit                 bool
	BlobBufferSize             int
	UseStreamBlobs             bool
	MaxInlineBlobSize          int
	MaxBlobCacheSize           int
	FetchBatchSize             int

	// BuildContextWatcherHandler is called to create a ContextWatcherHandler for a connection. The handler is called
	// when a context passed to a FbConn method is canceled.
	BuildContextWatcherHandler func(*FbConn) ctxwatch.Handler

	RuntimeParams map[string]string // Run-time parameters to set on connection as session default values (e.g. search_path or application_name)

	KerberosSrvName string
	KerberosSpn     string

	Fallbacks []*FallbackConfig

	// ValidateConnect is called during a connection attempt after a successful authentication, before the connection is returned.
	// If this returns an error the connection is closed and the next fallback config is tried.
	ValidateConnect ValidateConnectFunc

	// AfterConnect is called after ValidateConnect. It can be used to set up the connection (e.g. Set session variables
	// or prepare statements). If this returns an error the connection attempt fails.
	AfterConnect AfterConnectFunc

	// OnFbError is a callback function called when a Firebird error happened on server
	OnFbError FbErrorHandler

	// OnFbWarning is a callback function called when a Firebird warning obtained by client with success response
	OnFbWarning FbWarningHandler

	createdByParseConfig bool // Used to enforce created by ParseConfig rule.
}

// ParseConfigOptions contains options that control how a config is built.
type ParseConfigOptions struct {
	// GetSSLPassword returns the password to decrypt a client private key / certificate,
	// when encrypted transport is used (if supported).
	GetSSLPassword GetSSLPasswordFunc
}

// Copy returns a copy of the config that is safe to use and modify.
func (c *Config) Copy() *Config {
	newConf := new(Config)
	*newConf = *c
	if newConf.DPB != nil {
		newConf.DPB = append([]byte(nil), c.DPB...)
	}
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
	if len(newConf.Fallbacks) > 0 {
		fbs := make([]*FallbackConfig, len(c.Fallbacks))
		for i, fb := range c.Fallbacks {
			if fb == nil {
				continue
			}
			cp := *fb
			fbs[i] = &cp
		}
		newConf.Fallbacks = fbs
	}
	return newConf
}

// FallbackConfig contains additional host/port pairs to try when the primary address fails
type FallbackConfig struct {
	Host string // host (e.g. localhost) or absolute path (see NetworkAddress)
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

// NetworkAddress converts host and port into network and address suitable for use with net.Dial.
// If host is an absolute path, it currently falls back to a local TCP connection to localhost:3050.
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

// ParseConfig builds a *Config from connString.
//
// connString may be:
// - URL format (firebird:// or firebirdsql://)
// - keyword/value format (key=value pairs separated by spaces)
//
// connString may also be empty to only read from the environment and defaults.
//
// Defaults come from defaultSettings (e.g. host=localhost, port=3050).
//
// Examples:
//
//	# Keyword/Value
//	user=sysdba password=masterkey host=localhost port=3050 database=employee.fdb
//	user=sysdba password=masterkey host=localhost port=3050 database=new.fdb create_database=true
//
// # URL
//
//	firebird://sysdba:masterkey@localhost:3050/employee.fdb
//	firebird://sysdba:masterkey@localhost:3050/new.fdb?create_database=true
//
// ParseConfig recognizes the following environment variables (and their equivalents in the connection string):
//
//	FBHOST, FBPORT, FBDATABASE, FBUSER, FBPASSWORD
//	FBAPPNAME, FBTZ, FBCONNECT_TIMEOUT, FBFETCH_BATCH_SIZE
//	FBTARGETSESSIONATTRS, FBOPTIONS
//
// Any unrecognized options are preserved in Config.RuntimeParams.
//
// Multiple hosts/ports may be specified using comma-separated values; they will be tried in order.
// This is exposed via Config.Fallbacks and can be used for HA / failover.
func ParseConfig(connString string) (*Config, error) {
	var parseConfigOptions ParseConfigOptions
	return ParseConfigWithOptions(connString, parseConfigOptions)
}

// ParseConfigWithOptions builds a *Config from connString and options
func ParseConfigWithOptions(connString string, options ParseConfigOptions) (*Config, error) {
	defaultSettings := defaultSettings()
	envSettings := parseEnvSettings()

	connStringSettings := make(map[string]string)
	if connString != "" {
		var err error
		// connString may be a database URL or in keyword/value format
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
		CreateDatabase:       convertToBool(settings["create_database"], false),
		User:                 settings["user"],
		Password:             settings["password"],
		Role:                 settings["role"],
		RuntimeParams:        make(map[string]string),
		BuildFrontend: func(r io.Reader, w io.Writer) *fbproto.Client {
			return fbproto.NewClient(r, w, encoding)
		},
		BuildContextWatcherHandler: func(fbConn *FbConn) ctxwatch.Handler {
			return &CancelRequestContextWatcherHandler{Conn: fbConn}
		},
		OnFbError: func(_ *FbConn, fbErr *FbError) bool {
			// we want to automatically close any fatal errors
			// FIXME: fbx how to check if server error is fatal?
			//if strings.EqualFold(fbErr.Severity, "FATAL") {
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
		Autocommit:                 convertToBool(settings["autocommit"], true),
		UseStreamBlobs:             convertToBool(settings["use_stream_blobs"], true),
		FetchBatchSize:             DefaultFetchBatchSize,
	}
	if s, ok := settings["max_inline_blob_size"]; ok {
		value, err := parsePositiveIntSetting(s)
		if err != nil {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid max_inline_blob_size", err: err}
		}
		config.MaxInlineBlobSize = value
	}
	config.BlobBufferSize = DefaultBlobBufferSize
	if s, ok := settings["blob_buffer_size"]; ok {
		value, err := parsePositiveIntSetting(s)
		if err != nil {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid blob_buffer_size", err: err}
		}
		if value <= 0 || value > blobSegmentSizeMax {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid blob_buffer_size", err: fmt.Errorf("outside range 1..%d", blobSegmentSizeMax)}
		}
		config.BlobBufferSize = value
	}
	if s, ok := settings["max_blob_cache_size"]; ok {
		value, err := parsePositiveIntSetting(s)
		if err != nil {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid max_blob_cache_size", err: err}
		}
		config.MaxBlobCacheSize = value
	}
	if s, ok := settings["fetch_batch_size"]; ok {
		value, err := parsePositiveIntSetting(s)
		if err != nil {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid fetch_batch_size", err: err}
		}
		if value <= 0 {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid fetch_batch_size", err: errors.New("outside range 1..2147483647")}
		}
		config.FetchBatchSize = value
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
		"create_database":               {},
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
		"autocommit":                    {},
		"blob_buffer_size":              {},
		"use_stream_blobs":              {},
		"max_inline_blob_size":          {},
		"max_blob_cache_size":           {},
		"fetch_batch_size":              {},
		"target_session_attrs":          {},
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
		if k == "timezone" {
			v = strings.ReplaceAll(v, " ", "+")
		}
		config.RuntimeParams[k] = v
	}

	hosts := strings.Split(settings["host"], ",")
	ports := strings.Split(settings["port"], ",")
	fallbacks := []*FallbackConfig{}

	for i, host := range hosts {
		var portStr string
		if i < len(ports) {
			portStr = ports[i]
		} else {
			portStr = ports[0]
		}

		port, err := parsePort(portStr)
		if err != nil {
			return nil, &ParseConfigError{ConnString: connString, msg: "invalid port", err: err}
		}

		fallbacks = append(fallbacks, &FallbackConfig{
			Host: host,
			Port: port,
		})

	}

	config.Host = fallbacks[0].Host
	config.Port = fallbacks[0].Port
	config.Fallbacks = fallbacks[1:]

	switch tsa := settings["target_session_attrs"]; tsa {
	case "primary":
		config.ValidateConnect = ValidateConnectTargetSessionAttrsPrimary
	case "replica":
		config.ValidateConnect = ValidateConnectTargetSessionAttrsReplica
	case "prefer-replica":
		config.ValidateConnect = ValidateConnectTargetSessionAttrsPreferReplica
	case "read-only":
		config.ValidateConnect = ValidateConnectTargetSessionAttrsReadOnly
	case "read-write":
		config.ValidateConnect = ValidateConnectTargetSessionAttrsReadWrite
	case "any":
		// do nothing
	default:
		return nil, &ParseConfigError{ConnString: connString, msg: fmt.Sprintf("unknown target_session_attrs value: %v", tsa)}
	}

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
		"FBAPPNAME":            "application_name",
		"FBCONNECT_TIMEOUT":    "connect_timeout",
		"FBTARGETSESSIONATTRS": "target_session_attrs",
		"FBFETCH_BATCH_SIZE":   "fetch_batch_size",
		"FBTZ":                 "timezone",
		"FBOPTIONS":            "options",
		"FBSEARCHPATH":         "search_path",
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
		"dbname":                       "database",
		"createdb":                     "create_database",
		"create_db":                    "create_database",
		"blob_buffer_size":             "blob_buffer_size",
		"use_stream_blobs":             "use_stream_blobs",
		"useStreamBlobs":               "use_stream_blobs",
		"max_inline_blob_size":         "max_inline_blob_size",
		"max_blob_cache_size":          "max_blob_cache_size",
		"isc_dpb_max_inline_blob_size": "max_inline_blob_size",
		"isc_dpb_max_blob_cache_size":  "max_blob_cache_size",
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
		"dbname":                       "database",
		"createdb":                     "create_database",
		"create_db":                    "create_database",
		"blob_buffer_size":             "blob_buffer_size",
		"use_stream_blobs":             "use_stream_blobs",
		"useStreamBlobs":               "use_stream_blobs",
		"max_inline_blob_size":         "max_inline_blob_size",
		"max_blob_cache_size":          "max_blob_cache_size",
		"isc_dpb_max_inline_blob_size": "max_inline_blob_size",
		"isc_dpb_max_blob_cache_size":  "max_blob_cache_size",
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

func parsePositiveIntSetting(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	if value < 0 {
		return 0, errors.New("negative value")
	}
	return int(value), nil
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

func checkSessionAttrIsPrimary(ctx context.Context, fbConn *FbConn) (bool, error) {
	res := fbConn.Exec(ctx, "select rdb$get_context('SYSTEM', 'REPLICATING'), rdb$get_context('SYSTEM', 'REPLICA_MODE') from rdb$database").Read()
	if res.Err != nil {
		return false, res.Err
	}

	if len(res.Rows[0][1]) == 0 && string(res.Rows[0][0]) == "TRUE" {
		return true, nil
	} else if len(res.Rows[0][1]) != 0 {
		return false, nil
	}

	return false, errors.New("Replication is not configured")
}

// ValidateConnectTargetSessionAttrsReplica is a ValidateConnectFunc that implements
// target_session_attrs=replica.
func ValidateConnectTargetSessionAttrsReplica(ctx context.Context, fbConn *FbConn) error {
	isPrimary, err := checkSessionAttrIsPrimary(ctx, fbConn)

	if err != nil {
		return err
	}

	if isPrimary {
		return errors.New("Not replica")
	}

	return nil
}

// ValidateConnectTargetSessionAttrsPrimary is a ValidateConnectFunc that implements
// target_session_attrs=primary.
func ValidateConnectTargetSessionAttrsPrimary(ctx context.Context, fbConn *FbConn) error {
	isPrimary, err := checkSessionAttrIsPrimary(ctx, fbConn)

	if err != nil {
		return err
	}

	if !isPrimary {
		return errors.New("Not primary")
	}

	return nil
}

// ValidateConnectTargetSessionAttrsPreferReplica is a ValidateConnectFunc that implements
// target_session_attrs=prefer-replica.
func ValidateConnectTargetSessionAttrsPreferReplica(ctx context.Context, fbConn *FbConn) error {
	isPrimary, err := checkSessionAttrIsPrimary(ctx, fbConn)

	if err != nil {
		return err
	}

	if isPrimary {
		return &NotPreferredError{err: errors.New("Not replica")}
	}

	return nil
}

func convertToBool(s string, defaultValue bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		v = defaultValue
	}
	return v
}
