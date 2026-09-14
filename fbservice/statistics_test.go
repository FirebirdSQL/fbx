package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestStatisticsManagerEachOption(t *testing.T) {
	var output bytes.Buffer
	tests := map[string]struct {
		options fbservice.StatisticsOptions
		want    fbconn.StatisticsOptions
	}{
		"Output":          {fbservice.StatisticsOptions{Output: &output}, fbconn.StatisticsOptions{Output: &output}},
		"DataPages":       {fbservice.StatisticsOptions{DataPages: true}, fbconn.StatisticsOptions{DataPages: true}},
		"HeaderPages":     {fbservice.StatisticsOptions{HeaderPages: true}, fbconn.StatisticsOptions{HeaderPages: true}},
		"IndexPages":      {fbservice.StatisticsOptions{IndexPages: true}, fbconn.StatisticsOptions{IndexPages: true}},
		"SystemRelations": {fbservice.StatisticsOptions{SystemRelations: true}, fbconn.StatisticsOptions{SystemRelations: true}},
		"RecordVersions":  {fbservice.StatisticsOptions{RecordVersions: true}, fbconn.StatisticsOptions{RecordVersions: true}},
		"Tables":          {fbservice.StatisticsOptions{Tables: []string{"ONE", "Mixed table"}}, fbconn.StatisticsOptions{Tables: []string{"ONE", "Mixed table"}}},
		"Schemas":         {fbservice.StatisticsOptions{Schemas: []string{"PUBLIC", "Mixed schema"}}, fbconn.StatisticsOptions{Schemas: []string{"PUBLIC", "Mixed schema"}}},
		"NoCreation":      {fbservice.StatisticsOptions{NoCreation: true}, fbconn.StatisticsOptions{NoCreation: true}},
		"Encryption":      {fbservice.StatisticsOptions{Encryption: true}, fbconn.StatisticsOptions{Encryption: true}},
		"ParallelWorkers": {fbservice.StatisticsOptions{ParallelWorkers: 1}, fbconn.StatisticsOptions{ParallelWorkers: 1}},
	}
	// Every public option must have an independent test when the API is extended.
	for _, field := range reflect.VisibleFields(reflect.TypeFor[fbservice.StatisticsOptions]()) {
		require.Contains(t, tests, field.Name)
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestStatisticsManager(t)
			require.NoError(t, manager.Statistics(context.Background(), "/data/stats.fdb", tt.options))
			require.True(t, connection.closed)
			require.Equal(t, "/data/stats.fdb", connection.database)
			require.Equal(t, connection.database, connection.expectedDatabase)
			require.Equal(t, tt.want, connection.statisticsOptions)
			if len(tt.options.Tables) > 0 {
				tt.options.Tables[0] = "changed"
				require.Equal(t, tt.want.Tables, connection.statisticsOptions.Tables)
			}
			if len(tt.options.Schemas) > 0 {
				tt.options.Schemas[0] = "changed"
				require.Equal(t, tt.want.Schemas, connection.statisticsOptions.Schemas)
			}
		})
	}
}

func TestStatisticsManagerDefaults(t *testing.T) {
	manager, connection := newTestStatisticsManager(t)
	require.NoError(t, manager.Statistics(context.Background(), "employee"))
	require.Equal(t, fbconn.StatisticsOptions{}, connection.statisticsOptions)
	require.True(t, connection.closed)
}

func TestStatisticsManagerValidationBeforeConnect(t *testing.T) {
	type testCase struct {
		name, database string
		options        []fbservice.StatisticsOptions
	}
	tests := []testCase{
		{"empty database", "", nil},
		{"long database", strings.Repeat("d", 65536), nil},
		{"NUL database", "db\x00other", nil},
		{"help switch database", "-?", nil},
		{"version switch database", "-z", nil},
		{"multiple options", "db", []fbservice.StatisticsOptions{{}, {}}},
		{"negative workers", "db", []fbservice.StatisticsOptions{{ParallelWorkers: -1}}},
		{"empty table", "db", []fbservice.StatisticsOptions{{Tables: []string{""}}}},
		{"NUL table", "db", []fbservice.StatisticsOptions{{Tables: []string{"A\x00B"}}}},
		{"switch table", "db", []fbservice.StatisticsOptions{{Tables: []string{"-system"}}}},
		{"long table", "db", []fbservice.StatisticsOptions{{Tables: []string{strings.Repeat("t", 65536)}}}},
		{"empty schema", "db", []fbservice.StatisticsOptions{{Schemas: []string{""}}}},
		{"NUL schema", "db", []fbservice.StatisticsOptions{{Schemas: []string{"A\x00B"}}}},
		{"switch schema", "db", []fbservice.StatisticsOptions{{Schemas: []string{"-system"}}}},
		{"long schema", "db", []fbservice.StatisticsOptions{{Schemas: []string{strings.Repeat("s", 65536)}}}},
		{"header and encryption", "db", []fbservice.StatisticsOptions{{HeaderPages: true, Encryption: true}}},
	}
	if strconv.IntSize > 32 {
		overflow := int64(math.MaxInt32) + 1
		tests = append(tests, testCase{"workers overflow", "db", []fbservice.StatisticsOptions{{ParallelWorkers: int(overflow)}}})
	}
	for _, mode := range []string{"header", "encryption"} {
		for name, option := range map[string]fbservice.StatisticsOptions{
			"data": {DataPages: true}, "index": {IndexPages: true}, "system": {SystemRelations: true},
			"record": {RecordVersions: true}, "table": {Tables: []string{"T"}}, "schema": {Schemas: []string{"S"}},
		} {
			option.HeaderPages = mode == "header"
			option.Encryption = mode == "encryption"
			tests = append(tests, testCase{mode + " and " + name, "db", []fbservice.StatisticsOptions{option}})
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, connection := newTestStatisticsManager(t)
			require.Error(t, manager.Statistics(context.Background(), tt.database, tt.options...))
			require.False(t, connection.opened)
		})
	}
}

func TestStatisticsManagerDatabasePaths(t *testing.T) {
	for _, database := range []string{"./-name.fdb", "/data/-name.fdb", "employee"} {
		t.Run(database, func(t *testing.T) {
			manager, connection := newTestStatisticsManager(t)
			require.NoError(t, manager.Statistics(context.Background(), database))
			require.Equal(t, database, connection.database)
			require.Equal(t, database, connection.expectedDatabase)
			require.True(t, connection.closed)
		})
	}
}

func TestStatisticsManagerConstructors(t *testing.T) {
	manager, err := fbservice.NewStatisticsManager("host=localhost user=sysdba database=ignored")
	require.NoError(t, err)
	var connection *fakeStatisticsConnection
	connect := func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.StatisticsConnection, error) {
		connection = new(fakeStatisticsConnection)
		connection.connectionConfig = config.ConnectionConfig.Copy()
		connection.expectedDatabase = config.ExpectedDatabase
		return connection, nil
	}
	fbservice.SetStatisticsConnectForTest(manager, connect)
	require.NoError(t, manager.Statistics(context.Background(), "db"))
	require.Equal(t, "localhost", connection.connectionConfig.Host)
	requireDatabaseConnectionSettingsCleared(t, connection.connectionConfig)
	require.Equal(t, "db", connection.expectedDatabase)

	_, err = fbservice.NewStatisticsManager("port=invalid")
	require.Error(t, err)
	config, err := fbservice.ParseConfig("host=localhost user=sysdba auth_plugin_list=Srp")
	require.NoError(t, err)
	manager = fbservice.NewStatisticsManagerConfig(config)
	fbservice.SetStatisticsConnectForTest(manager, connect)
	config.Host = "changed"
	config.AuthPlugins[0] = "changed"
	require.NoError(t, manager.Statistics(context.Background(), "db"))
	require.Equal(t, "localhost", connection.connectionConfig.Host)
	require.Equal(t, []string{"Srp"}, connection.connectionConfig.AuthPluginList)
	for _, invalid := range []*fbservice.StatisticsManager{nil, {}, fbservice.NewStatisticsManagerConfig(nil), fbservice.NewStatisticsManagerConfig(&fbservice.Config{})} {
		require.Error(t, invalid.Statistics(context.Background(), "db"))
	}
}

func TestStatisticsManagerErrors(t *testing.T) {
	manager, connection := newTestStatisticsManager(t)
	operationErr := errors.New("statistics failed")
	closeErr := errors.New("detach failed")
	connection.statisticsErr, connection.closeErr = operationErr, closeErr
	err := manager.Statistics(context.Background(), "db")
	require.ErrorIs(t, err, operationErr)
	require.ErrorIs(t, err, closeErr)
	require.True(t, connection.closed)
	connectErr := errors.New("connect failed")
	fbservice.SetStatisticsConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.StatisticsConnection, error) {
		return nil, connectErr
	})
	require.ErrorIs(t, manager.Statistics(context.Background(), "db"), connectErr)
}

func TestStatisticsManagerClosesAfterCancellation(t *testing.T) {
	manager, _ := newTestStatisticsManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := &statisticsCancelConnection{cancel: cancel}
	fbservice.SetStatisticsConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.StatisticsConnection, error) {
		return connection, nil
	})
	require.ErrorIs(t, manager.Statistics(ctx, "db"), context.Canceled)
	require.True(t, connection.closed)
	require.NoError(t, connection.closeContextErr)
	require.True(t, connection.closeHasDeadline)
}

func TestStatisticsManagerCanceledBeforeConnect(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelDeadline := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancelDeadline()

	for _, ctx := range []context.Context{canceled, expired} {
		t.Run(ctx.Err().Error(), func(t *testing.T) {
			manager, connection := newTestStatisticsManager(t)
			err := manager.Statistics(ctx, "db")
			require.ErrorIs(t, err, ctx.Err())
			require.False(t, connection.opened)
			require.False(t, connection.closed)
		})
	}
}

type statisticsCancelConnection struct {
	fakeStatisticsConnection
	cancel           context.CancelFunc
	closeContextErr  error
	closeHasDeadline bool
}

func (c *statisticsCancelConnection) Statistics(ctx context.Context, _ string, _ fbconn.StatisticsOptions) error {
	c.cancel()
	return ctx.Err()
}

func (c *statisticsCancelConnection) Close(ctx context.Context) error {
	c.closed = true
	c.closeContextErr = ctx.Err()
	deadline, ok := ctx.Deadline()
	c.closeHasDeadline = ok && time.Until(deadline) <= 5*time.Second
	return nil
}

func TestStatisticsManagerConcurrentOperations(t *testing.T) {
	manager, _ := newTestStatisticsManager(t)
	var opened, closed atomic.Int64
	fbservice.SetStatisticsConnectForTest(manager, func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.StatisticsConnection, error) {
		opened.Add(1)
		return &fakeStatisticsConnection{expectedDatabase: config.ExpectedDatabase, closedCount: &closed}, nil
	})
	const count = 20
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := range count {
		wg.Go(func() { errs <- manager.Statistics(context.Background(), fmt.Sprintf("db%d", i)) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, count, opened.Load())
	require.EqualValues(t, count, closed.Load())
}

func newTestStatisticsManager(t *testing.T) (*fbservice.StatisticsManager, *fakeStatisticsConnection) {
	t.Helper()
	config, err := fbservice.ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)

	connection := new(fakeStatisticsConnection)
	manager := fbservice.NewStatisticsManagerConfig(config)
	fbservice.SetStatisticsConnectForTest(manager, func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.StatisticsConnection, error) {
		connection.opened = true
		connection.expectedDatabase = config.ExpectedDatabase
		return connection, nil
	})
	return manager, connection
}

// This connection intentionally implements only Statistics and Close.
type fakeStatisticsConnection struct {
	opened            bool
	closed            bool
	expectedDatabase  string
	database          string
	statisticsOptions fbconn.StatisticsOptions
	statisticsErr     error
	connectionConfig  *fbconn.Config
	closeErr          error
	closedCount       *atomic.Int64
	statisticsCalled  bool
}

func (c *fakeStatisticsConnection) Statistics(_ context.Context, database string, options fbconn.StatisticsOptions) error {
	if c.statisticsCalled {
		return errors.New("statistics connection reused")
	}
	c.statisticsCalled = true
	if database != c.expectedDatabase {
		return fmt.Errorf("statistics database %q does not match expected database %q", database, c.expectedDatabase)
	}
	c.database = database
	c.statisticsOptions = options
	return c.statisticsErr
}

func (c *fakeStatisticsConnection) Close(context.Context) error {
	c.closed = true
	if c.closedCount != nil {
		c.closedCount.Add(1)
	}
	return c.closeErr
}
