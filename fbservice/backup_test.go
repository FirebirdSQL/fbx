package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbservice"
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
		fbservice.BackupOptions{
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
		fbservice.BackupOptions{},
		fbservice.BackupOptions{},
	)

	require.ErrorContains(t, err, "at most one")
	require.False(t, connection.opened)
}

func TestManagerBackupValidatesRequestBeforeOpeningConnection(t *testing.T) {
	type testCase struct {
		database   string
		backupFile string
		options    fbservice.BackupOptions
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
			options:    fbservice.BackupOptions{Factor: -1},
			errText:    "backup factor must be between 0",
		},
		"negative parallel workers": {
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options:    fbservice.BackupOptions{ParallelWorkers: -1},
			errText:    "backup parallel workers must be between 0",
		},
		"database exceeds SPB limit": {
			database:   strings.Repeat("d", math.MaxUint16+1),
			backupFile: "/backup/employee.fbk",
			errText:    "backup database is 65536 bytes; maximum is 65535",
		},
		"backup file exceeds SPB limit": {
			database:   "/data/employee.fdb",
			backupFile: strings.Repeat("b", math.MaxUint16+1),
			errText:    "backup file is 65536 bytes; maximum is 65535",
		},
		"compressor exceeds SPB limit": {
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options: fbservice.BackupOptions{
				Compressor: strings.Repeat("c", math.MaxUint16+1),
			},
			errText: "backup compressor is 65536 bytes; maximum is 65535",
		},
	}
	if strconv.IntSize > 32 {
		aboveMaxInt32 := int64(math.MaxInt32)
		aboveMaxInt32++
		tests["factor exceeds int32"] = testCase{
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options: fbservice.BackupOptions{
				Factor: int(aboveMaxInt32),
			},
			errText: "backup factor must be between 0",
		}
		tests["parallel workers exceeds int32"] = testCase{
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options: fbservice.BackupOptions{
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

func TestNewManagerConfigCopiesConfig(t *testing.T) {
	config, err := fbservice.ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)
	config.AuthPlugins = []string{"Srp256", "Srp"}
	manager, err := fbservice.NewManagerConfig(config)
	require.NoError(t, err)
	connection := new(fakeServiceConnection)
	fbservice.SetConnectForTest(manager, func(_ context.Context, serviceConfig *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		connection.connectionConfig = serviceConfig.ConnectionConfig.Copy()
		return connection, nil
	})
	config.User = "changed"
	config.AuthPlugins[0] = "changed"
	config.AuthPlugins = append(config.AuthPlugins, "changed")

	err = manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.NoError(t, err)
	require.Equal(t, "sysdba", connection.connectionConfig.User)
	require.Equal(t, []string{"Srp256", "Srp"}, connection.connectionConfig.AuthPluginList)
}

func TestManagerMapsPublicConfigToConnectionConfig(t *testing.T) {
	config, err := fbservice.ParseConfig("firebird://sysdba:masterkey@localhost/employee")
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

	manager, err := fbservice.NewManagerConfig(config)
	require.NoError(t, err)
	connection := new(fakeServiceConnection)
	fbservice.SetConnectForTest(manager, func(_ context.Context, serviceConfig *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		connection.connectionConfig = serviceConfig.ConnectionConfig.Copy()
		return connection, nil
	})

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
	config, err := fbservice.ParseConfig(
		"host=localhost user=sysdba password=masterkey database=employee " +
			"create_database=true target_session_attrs=read-write " +
			"default_transaction_read_only=true application_name=fbservice-test " +
			"autocommit=true blob_buffer_size=1 use_stream_blobs=true " +
			"max_inline_blob_size=1 max_blob_cache_size=1 fetch_batch_size=1",
	)
	require.NoError(t, err)

	manager, err := fbservice.NewManagerConfig(config)
	require.NoError(t, err)
	connection := new(fakeServiceConnection)
	fbservice.SetConnectForTest(manager, func(_ context.Context, serviceConfig *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		connection.connectionConfig = serviceConfig.ConnectionConfig.Copy()
		connection.expectedDatabase = serviceConfig.ExpectedDatabase
		return connection, nil
	})

	err = manager.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk")

	require.NoError(t, err)
	require.Equal(t, "/data/employee.fdb", connection.expectedDatabase)
	requireDatabaseConnectionSettingsCleared(t, connection.connectionConfig)
}

func TestNewManagerConfigRejectsConfigNotCreatedByParseConfig(t *testing.T) {
	manager, err := fbservice.NewManagerConfig(&fbservice.Config{Host: "localhost"})

	require.ErrorContains(t, err, "must be created by ParseConfig")
	require.Nil(t, manager)
}

func TestNewManagerConfigRejectsNilConfig(t *testing.T) {
	manager, err := fbservice.NewManagerConfig(nil)

	require.ErrorContains(t, err, "config must not be nil")
	require.Nil(t, manager)
}

func newTestManager(t *testing.T) (*fbservice.Manager, *fakeServiceConnection) {
	t.Helper()
	config, err := fbservice.ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)

	connection := new(fakeServiceConnection)
	manager, err := fbservice.NewManagerConfig(config)
	require.NoError(t, err)
	fbservice.SetConnectForTest(manager, func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		connection.opened = true
		connection.expectedDatabase = config.ExpectedDatabase
		return connection, nil
	})
	return manager, connection
}

type fakeServiceConnection struct {
	repairOptions     fbconn.RepairOptions
	properties        fbconn.DatabaseProperties
	maintenanceErr    error
	transactionID     int64
	limboAction       fbconn.LimboTransactionAction
	limboOutput       io.Writer
	opened            bool
	closed            bool
	expectedDatabase  string
	database          string
	backupFile        string
	backupOptions     fbconn.BackupOptions
	restoreBackupFile string
	restoreDatabase   string
	restoreOptions    fbconn.RestoreOptions
	statisticsOptions fbconn.StatisticsOptions
	statisticsErr     error
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

func (c *fakeServiceConnection) Statistics(_ context.Context, database string, options fbconn.StatisticsOptions) error {
	c.database = database
	c.statisticsOptions = options
	return c.statisticsErr
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
