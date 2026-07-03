package fbservice

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
)

func TestManagerBackupWithoutOptions(t *testing.T) {
	manager, connection := newTestManager(t)

	err := manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.NoError(t, err)
	require.True(t, connection.closed)
	require.Equal(t, "/data/employee.fdb", connection.expectedDatabase)
	require.Equal(t, "/data/employee.fdb", connection.database)
	require.Equal(t, "/backup/employee.fbk", connection.backupFile)
	require.Equal(t, fbconn.BackupOptions{}, connection.options)
}

func TestManagerBackupMapsOptions(t *testing.T) {
	manager, connection := newTestManager(t)
	var output bytes.Buffer

	err := manager.Backup(
		context.Background(),
		"/data/employee.fdb",
		"/backup/employee.fbk",
		BackupOptions{
			Output:             &output,
			Verbose:            true,
			IgnoreChecksums:    true,
			IgnoreLimbo:        true,
			MetadataOnly:       true,
			NoGarbageCollect:   true,
			OldDescriptions:    true,
			NonTransportable:   true,
			ConvertExternal:    true,
			Expand:             true,
			Factor:             32,
			NoDatabaseTriggers: true,
			Zip:                true,
			Compressor:         "zstd:3:1",
		},
	)

	require.NoError(t, err)
	require.Equal(t, fbconn.BackupOptions{
		Output:                &output,
		Verbose:               true,
		IgnoreChecksums:       true,
		IgnoreLimbo:           true,
		MetadataOnly:          true,
		NoGarbageCollect:      true,
		OldDescriptions:       true,
		NonTransportable:      true,
		ConvertExternalTables: true,
		Expand:                true,
		Factor:                32,
		NoDatabaseTriggers:    true,
		Zip:                   true,
		Compressor:            "zstd:3:1",
	}, connection.options)
}

func TestManagerBackupRejectsMultipleOptions(t *testing.T) {
	manager, connection := newTestManager(t)

	err := manager.Backup(
		context.Background(),
		"/data/employee.fdb",
		"/backup/employee.fbk",
		BackupOptions{},
		BackupOptions{},
	)

	require.ErrorContains(t, err, "at most one")
	require.False(t, connection.opened)
}

func TestManagerBackupReturnsOperationAndCloseErrors(t *testing.T) {
	manager, connection := newTestManager(t)
	connection.backupErr = errors.New("backup failed")
	connection.closeErr = errors.New("close failed")

	err := manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.ErrorContains(t, err, "backup failed")
	require.ErrorContains(t, err, "close failed")
}

func TestNewManagerCopiesConfig(t *testing.T) {
	config, err := ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)
	manager := NewManager(config)
	config.User = "changed"
	config.AuthPlugins = append(config.AuthPlugins, "changed")

	require.Equal(t, "sysdba", manager.config.User)
	require.NotContains(t, manager.config.AuthPlugins, "changed")
}

func TestManagerMapsPublicConfigToConnectionConfig(t *testing.T) {
	config, err := ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)
	config.Host = "service-host"
	config.Port = 4040
	config.User = "service-user"
	config.Password = "service-password"
	config.Role = "RDB$ADMIN"
	config.Charset = "UTF8"
	config.SystemUser = "system-user"
	config.SystemHost = "system-host"
	config.ClientVersion = "fbservice-test"
	config.AuthPlugins = []string{"Srp256", "Srp"}
	config.WireCrypt = false
	config.ConnectTimeout = 3 * time.Second

	manager := NewManager(config)
	connection := new(fakeServiceConnection)
	manager.connect = func(_ context.Context, serviceConfig *fbconn.ServiceConfig) (serviceConnection, error) {
		connection.connectionConfig = serviceConfig.ConnectionConfig.Copy()
		return connection, nil
	}

	err = manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.NoError(t, err)
	require.Equal(t, "service-host", connection.connectionConfig.Host)
	require.Equal(t, uint16(4040), connection.connectionConfig.Port)
	require.Equal(t, "service-user", connection.connectionConfig.User)
	require.Equal(t, "service-password", connection.connectionConfig.Password)
	require.Equal(t, "RDB$ADMIN", connection.connectionConfig.Role)
	require.Equal(t, "UTF8", connection.connectionConfig.Encoding)
	require.Equal(t, "system-user", connection.connectionConfig.SystemUser)
	require.Equal(t, "system-host", connection.connectionConfig.SystemHost)
	require.Equal(t, "fbservice-test", connection.connectionConfig.ClientVersion)
	require.Equal(t, []string{"Srp256", "Srp"}, connection.connectionConfig.AuthPluginList)
	require.False(t, connection.connectionConfig.WireCrypt)
	require.Equal(t, 3*time.Second, connection.connectionConfig.ConnectTimeout)
}

func TestManagerRejectsConfigNotCreatedByParseConfig(t *testing.T) {
	manager := NewManager(&Config{Host: "localhost"})

	err := manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.ErrorContains(t, err, "must be created by ParseConfig")
}

func TestManagerBackupRejectsNilConfig(t *testing.T) {
	manager := NewManager(nil)

	err := manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.ErrorContains(t, err, "config must not be nil")
}

func newTestManager(t *testing.T) (*Manager, *fakeServiceConnection) {
	t.Helper()
	config, err := ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)

	connection := new(fakeServiceConnection)
	manager := NewManager(config)
	manager.connect = func(_ context.Context, config *fbconn.ServiceConfig) (serviceConnection, error) {
		connection.opened = true
		connection.expectedDatabase = config.ExpectedDatabase
		return connection, nil
	}
	return manager, connection
}

type fakeServiceConnection struct {
	opened           bool
	closed           bool
	expectedDatabase string
	database         string
	backupFile       string
	options          fbconn.BackupOptions
	connectionConfig *fbconn.Config
	backupErr        error
	closeErr         error
}

func (c *fakeServiceConnection) Backup(
	_ context.Context,
	database string,
	backupFile string,
	options fbconn.BackupOptions,
) error {
	c.database = database
	c.backupFile = backupFile
	c.options = options
	return c.backupErr
}

func (c *fakeServiceConnection) Close(context.Context) error {
	c.closed = true
	return c.closeErr
}
