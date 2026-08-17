package fbservice

import (
	"errors"
	"time"

	"rdb.red-soft.ru/fbx/fbconn"
)

// Config contains settings used to connect to the Firebird service manager.
//
// Config must be created by [ParseConfig]. After parsing, its fields can be
// modified before passing the config to [NewManager].
type Config struct {
	// Host is the server hostname or Unix socket path.
	Host string

	// Port is the server port.
	Port uint16

	// User is the service manager user name.
	User string

	// Password is the service manager password.
	Password string

	// Role is the SQL role used during authentication.
	Role string

	// Charset is the connection character set.
	Charset string

	// SystemUser identifies the operating system user reported to the server.
	SystemUser string

	// SystemHost identifies the client host reported to the server.
	SystemHost string

	// ClientVersion identifies the client version reported to the server.
	ClientVersion string

	// AuthPlugins lists authentication plugins in preference order.
	AuthPlugins []string

	// WireCrypt enables wire encryption.
	WireCrypt bool

	// ConnectTimeout limits the time spent establishing a service connection. A
	// zero value applies no additional connection deadline.
	ConnectTimeout time.Duration

	connectionConfig *fbconn.Config
}

// ParseConfig parses connString into service manager connection settings.
//
// A database specified by connString is not used as the target of a service
// operation. Each operation receives its target database as a method argument.
// Database attach settings, such as "create_database" and
// "target_session_attrs", are ignored because service manager connections do
// not attach to a database.
func ParseConfig(connString string) (*Config, error) {
	connectionConfig, err := fbconn.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	return configFromConnectionConfig(connectionConfig), nil
}

// Copy returns an independent copy of config.
func (c *Config) Copy() *Config {
	if c == nil {
		return nil
	}
	configCopy := *c
	configCopy.AuthPlugins = append([]string(nil), c.AuthPlugins...)
	if c.connectionConfig != nil {
		configCopy.connectionConfig = c.connectionConfig.Copy()
	}
	return &configCopy
}

func configFromConnectionConfig(connectionConfig *fbconn.Config) *Config {
	serviceConfig := connectionConfig.Copy()
	scrubDatabaseConnectionSettings(serviceConfig)

	return &Config{
		Host:             connectionConfig.Host,
		Port:             connectionConfig.Port,
		User:             connectionConfig.User,
		Password:         connectionConfig.Password,
		Role:             connectionConfig.Role,
		Charset:          connectionConfig.Encoding,
		SystemUser:       connectionConfig.SystemUser,
		SystemHost:       connectionConfig.SystemHost,
		ClientVersion:    connectionConfig.ClientVersion,
		AuthPlugins:      append([]string(nil), connectionConfig.AuthPluginList...),
		WireCrypt:        connectionConfig.WireCrypt,
		ConnectTimeout:   connectionConfig.ConnectTimeout,
		connectionConfig: serviceConfig,
	}
}

func (c *Config) toConnectionConfig() (*fbconn.Config, error) {
	if c == nil {
		return nil, errors.New("service manager config must not be nil")
	}
	if c.connectionConfig == nil {
		return nil, errors.New("service manager config must be created by ParseConfig")
	}

	connectionConfig := c.connectionConfig.Copy()
	connectionConfig.Host = c.Host
	connectionConfig.Port = c.Port
	connectionConfig.User = c.User
	connectionConfig.Password = c.Password
	connectionConfig.Role = c.Role
	connectionConfig.Encoding = c.Charset
	connectionConfig.SystemUser = c.SystemUser
	connectionConfig.SystemHost = c.SystemHost
	connectionConfig.ClientVersion = c.ClientVersion
	connectionConfig.AuthPluginList = append([]string(nil), c.AuthPlugins...)
	connectionConfig.WireCrypt = c.WireCrypt
	connectionConfig.ConnectTimeout = c.ConnectTimeout
	scrubDatabaseConnectionSettings(connectionConfig)
	return connectionConfig, nil
}

func scrubDatabaseConnectionSettings(config *fbconn.Config) {
	config.Database = ""
	config.CreateDatabase = false
	config.DefaultTransactionReadOnly = false
	config.Autocommit = false
	config.BlobBufferSize = 0
	config.UseStreamBlobs = false
	config.MaxInlineBlobSize = 0
	config.MaxBlobCacheSize = 0
	config.FetchBatchSize = 0
	config.DPB = nil
	config.RuntimeParams = nil
	config.ValidateConnect = nil
	config.AfterConnect = nil
}
