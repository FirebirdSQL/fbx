package fbservice

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
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
	require.Equal(t, fbconn.BackupOptions{}, connection.backupOptions)
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
			ParallelWorkers:    4,
			Replace:            true,
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
		ParallelWorkers:       4,
		Replace:               true,
		NoDatabaseTriggers:    true,
		Zip:                   true,
		Compressor:            "zstd:3:1",
	}, connection.backupOptions)
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

func TestManagerBackupValidatesRequestBeforeOpeningConnection(t *testing.T) {
	type testCase struct {
		database   string
		backupFile string
		options    BackupOptions
		errText    string
	}

	tests := map[string]testCase{
		"empty database": {
			database:   "",
			backupFile: "/backup/employee.fbk",
			errText:    "backup database must not be empty",
		},
		"empty backup file": {
			database: "/data/employee.fdb",
			errText:  "backup file must not be empty",
		},
		"negative factor": {
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options:    BackupOptions{Factor: -1},
			errText:    "backup factor must be between 0",
		},
		"negative parallel workers": {
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options:    BackupOptions{ParallelWorkers: -1},
			errText:    "backup parallel workers must be between 0",
		},
		"database exceeds SPB limit": {
			database:   strings.Repeat("d", maxServiceActionStringLength+1),
			backupFile: "/backup/employee.fbk",
			errText:    "backup database is 65536 bytes; maximum is 65535",
		},
		"backup file exceeds SPB limit": {
			database:   "/data/employee.fdb",
			backupFile: strings.Repeat("b", maxServiceActionStringLength+1),
			errText:    "backup file is 65536 bytes; maximum is 65535",
		},
		"compressor exceeds SPB limit": {
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options: BackupOptions{
				Compressor: strings.Repeat("c", maxServiceActionStringLength+1),
			},
			errText: "backup compressor is 65536 bytes; maximum is 65535",
		},
	}
	if strconv.IntSize > 32 {
		aboveMaxInt32 := int64(maxServiceActionInt32)
		aboveMaxInt32++
		tests["factor exceeds int32"] = testCase{
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options: BackupOptions{
				Factor: int(aboveMaxInt32),
			},
			errText: "backup factor must be between 0",
		}
		tests["parallel workers exceeds int32"] = testCase{
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options: BackupOptions{
				ParallelWorkers: int(aboveMaxInt32),
			},
			errText: "backup parallel workers must be between 0",
		}
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)

			err := manager.Backup(context.Background(), tt.database, tt.backupFile, tt.options)

			require.ErrorContains(t, err, tt.errText)
			require.False(t, connection.opened)
		})
	}
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

func TestManagerClearsDatabaseOnlyConnectionSettings(t *testing.T) {
	config, err := ParseConfig(
		"host=localhost user=sysdba password=masterkey database=employee " +
			"create_database=true target_session_attrs=read-write " +
			"default_transaction_read_only=true application_name=fbservice-test",
	)
	require.NoError(t, err)
	requireDatabaseConnectionSettingsCleared(t, config.connectionConfig)

	config.connectionConfig.Database = "employee"
	config.connectionConfig.ValidateConnect = func(context.Context, *fbconn.FbConn) error { return nil }
	config.connectionConfig.AfterConnect = func(context.Context, *fbconn.FbConn) error { return nil }
	config.connectionConfig.CreateDatabase = true
	config.connectionConfig.DefaultTransactionReadOnly = true
	config.connectionConfig.Autocommit = true
	config.connectionConfig.BlobBufferSize = 1
	config.connectionConfig.UseStreamBlobs = true
	config.connectionConfig.MaxInlineBlobSize = 1
	config.connectionConfig.MaxBlobCacheSize = 1
	config.connectionConfig.FetchBatchSize = 1
	config.connectionConfig.DPB = []byte{1}
	config.connectionConfig.RuntimeParams = map[string]string{"application_name": "fbservice-test"}

	manager := NewManager(config)
	connection := new(fakeServiceConnection)
	manager.connect = func(_ context.Context, serviceConfig *fbconn.ServiceConfig) (serviceConnection, error) {
		connection.connectionConfig = serviceConfig.ConnectionConfig.Copy()
		connection.expectedDatabase = serviceConfig.ExpectedDatabase
		return connection, nil
	}

	err = manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.NoError(t, err)
	require.Equal(t, "/data/employee.fdb", connection.expectedDatabase)
	requireDatabaseConnectionSettingsCleared(t, connection.connectionConfig)
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
	opened            bool
	closed            bool
	expectedDatabase  string
	database          string
	backupFile        string
	backupOptions     fbconn.BackupOptions
	restoreBackupFile string
	restoreDatabase   string
	restoreOptions    fbconn.RestoreOptions
	connectionConfig  *fbconn.Config
	backupErr         error
	restoreErr        error
	closeErr          error
}

func (c *fakeServiceConnection) Backup(
	_ context.Context,
	database string,
	backupFile string,
	options fbconn.BackupOptions,
) error {
	c.database = database
	c.backupFile = backupFile
	c.backupOptions = options
	return c.backupErr
}

func (c *fakeServiceConnection) Restore(
	_ context.Context,
	backupFile string,
	database string,
	options fbconn.RestoreOptions,
) error {
	c.restoreBackupFile = backupFile
	c.restoreDatabase = database
	c.restoreOptions = options
	return c.restoreErr
}

func (c *fakeServiceConnection) Close(context.Context) error {
	c.closed = true
	return c.closeErr
}

func requireDatabaseConnectionSettingsCleared(t *testing.T, config *fbconn.Config) {
	t.Helper()
	require.Empty(t, config.Database)
	require.False(t, config.CreateDatabase)
	require.False(t, config.DefaultTransactionReadOnly)
	require.False(t, config.Autocommit)
	require.Zero(t, config.BlobBufferSize)
	require.False(t, config.UseStreamBlobs)
	require.Zero(t, config.MaxInlineBlobSize)
	require.Zero(t, config.MaxBlobCacheSize)
	require.Zero(t, config.FetchBatchSize)
	require.Nil(t, config.DPB)
	require.Empty(t, config.RuntimeParams)
	require.Nil(t, config.ValidateConnect)
	require.Nil(t, config.AfterConnect)
}
