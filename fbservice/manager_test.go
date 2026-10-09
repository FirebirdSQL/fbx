package fbservice_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestNewManager(t *testing.T) {
	for name, connString := range map[string]string{
		"URL":           "firebird://service-user:service-password@service-host:4040/ignored.fdb?create_database=true",
		"keyword/value": "host=service-host port=4040 user=service-user password=service-password database=ignored.fdb create_database=true",
	} {
		t.Run(name, func(t *testing.T) {
			manager, err := fbservice.NewManager(connString)
			require.NoError(t, err)
			require.NotNil(t, manager)

			operationCtx, cancel := context.WithCancel(context.Background())
			defer cancel()
			connection := new(fakeServiceConnection)
			fbservice.SetConnectForTest(manager, func(ctx context.Context, config *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
				require.Equal(t, operationCtx, ctx)
				require.NoError(t, ctx.Err())
				connection.connectionConfig = config.ConnectionConfig.Copy()
				connection.expectedDatabase = config.ExpectedDatabase
				return connection, nil
			})

			err = manager.Backup(operationCtx, "/data/employee.fdb", "/backup/employee.fbk")

			require.NoError(t, err)
			require.Equal(t, "service-host", connection.connectionConfig.Host)
			require.Equal(t, uint16(4040), connection.connectionConfig.Port)
			require.Equal(t, "service-user", connection.connectionConfig.User)
			require.Equal(t, "service-password", connection.connectionConfig.Password)
			requireDatabaseConnectionSettingsCleared(t, connection.connectionConfig)
			require.Equal(t, "/data/employee.fdb", connection.expectedDatabase)
			require.Equal(t, "/data/employee.fdb", connection.database)
			require.Equal(t, "/backup/employee.fbk", connection.backupFile)
			require.True(t, connection.closed)
		})
	}
}

func TestNewManagerRejectsInvalidConnectionString(t *testing.T) {
	for _, connString := range []string{"firebird://localhost:invalid/employee", "host='unterminated", "port=65536"} {
		t.Run(connString, func(t *testing.T) {
			manager, err := fbservice.NewManager(connString)

			var parseErr *fbconn.ParseConfigError
			require.ErrorAs(t, err, &parseErr)
			require.Nil(t, manager)
		})
	}
}

func TestManagerSupportsConcurrentOperations(t *testing.T) {
	config, err := fbservice.ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)

	const operationCount = 100
	var opened atomic.Int64
	var closed atomic.Int64

	manager, err := fbservice.NewManagerConfig(config)
	require.NoError(t, err)
	fbservice.SetConnectForTest(manager, func(
		_ context.Context,
		config *fbconn.ServiceConfig,
	) (fbservice.ServiceConnection, error) {
		opened.Add(1)
		return &concurrentServiceConnection{
			expectedDatabase: config.ExpectedDatabase,
			closed:           &closed,
		}, nil
	})

	var wg sync.WaitGroup
	errs := make(chan error, operationCount)
	for i := range operationCount {
		wg.Add(1)
		go func() {
			defer wg.Done()

			database := fmt.Sprintf("/data/employee-%d.fdb", i)
			backupFile := fmt.Sprintf("/backup/employee-%d.fbk", i)
			switch i % 12 {
			case 0:
				errs <- manager.Backup(context.Background(), database, backupFile)
			case 1:
				errs <- manager.Restore(context.Background(), backupFile, database)
			case 3:
				errs <- manager.Repair(context.Background(), database, fbservice.RepairOptions{Operation: fbservice.RepairSweep})
			case 4:
				errs <- manager.SetDatabaseProperties(context.Background(), database, fbservice.DatabaseProperties{SQLDialect: 3})
			case 5:
				_, err := manager.LimboTransactions(context.Background(), database)
				errs <- err
			case 6:
				errs <- manager.ResolveLimboTransaction(context.Background(), database, 42, fbservice.LimboRollback, nil)
			case 7:
				_, err := manager.LimboTransactionDetails(context.Background(), database)
				errs <- err
			case 8:
				errs <- manager.Validate(context.Background(), database, fbservice.ValidateOptions{Output: io.Discard})
			case 9:
				errs <- manager.Sweep(context.Background(), database)
			case 10:
				errs <- manager.Shutdown(context.Background(), database, fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce})
			case 11:
				_, err := manager.ResolveLimboTransactionChecked(context.Background(), database, 42, fbservice.LimboRecover, io.Discard)
				errs <- err
			case 2:
				errs <- manager.Statistics(context.Background(), database, fbservice.StatisticsOptions{Output: io.Discard})
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	// A checked resolution opens one connection for the action and another for
	// the list query; each connection still runs just one service action.
	wantConnections := int64(operationCount + operationCount/12)
	require.Equal(t, wantConnections, opened.Load())
	require.Equal(t, wantConnections, closed.Load())
}

type concurrentServiceConnection struct {
	expectedDatabase string
	operationCalled  bool
	closed           *atomic.Int64
}

func (c *concurrentServiceConnection) Backup(
	_ context.Context,
	database string,
	_ string,
	_ fbconn.BackupOptions,
) error {
	return c.recordOperation(database)
}

func (c *concurrentServiceConnection) Restore(
	_ context.Context,
	_ string,
	database string,
	_ fbconn.RestoreOptions,
) error {
	return c.recordOperation(database)
}

func (c *concurrentServiceConnection) Close(context.Context) error {
	if !c.operationCalled {
		return errors.New("service connection closed before an operation")
	}
	c.closed.Add(1)
	return nil
}

func (c *concurrentServiceConnection) Statistics(_ context.Context, database string, _ fbconn.StatisticsOptions) error {
	return c.recordOperation(database)
}

func (c *concurrentServiceConnection) recordOperation(database string) error {
	if c.operationCalled {
		return errors.New("service connection reused")
	}
	c.operationCalled = true
	if database != c.expectedDatabase {
		return fmt.Errorf(
			"operation database %q does not match expected database %q",
			database,
			c.expectedDatabase,
		)
	}
	return nil
}
