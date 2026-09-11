package fbservice_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestManagerSupportsConcurrentOperations(t *testing.T) {
	config, err := fbservice.ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)

	const operationCount = 100
	var opened atomic.Int64
	var closed atomic.Int64

	manager := fbservice.NewManager(config)
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
			if i%2 == 0 {
				errs <- manager.Backup(context.Background(), database, backupFile)
				return
			}
			errs <- manager.Restore(context.Background(), backupFile, database)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int64(operationCount), opened.Load())
	require.Equal(t, int64(operationCount), closed.Load())
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
