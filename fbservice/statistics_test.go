package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

func TestManagerStatisticsEachOption(t *testing.T) {
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
			if tt.options.Output == nil {
				tt.options.Output = io.Discard
				tt.want.Output = io.Discard
			}
			manager, connection := newTestManager(t)
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

func TestManagerStatisticsDefaults(t *testing.T) {
	manager, connection := newTestManager(t)
	require.NoError(t, manager.Statistics(context.Background(), "employee", fbservice.StatisticsOptions{Output: io.Discard}))
	require.Equal(t, fbconn.StatisticsOptions{Output: io.Discard}, connection.statisticsOptions)
	require.True(t, connection.closed)
}

func TestManagerStatisticsRequiresOutput(t *testing.T) {
	for name, options := range map[string][]fbservice.StatisticsOptions{
		"omitted options": nil,
		"empty options":   {{}},
		"header only":     {{HeaderPages: true}},
	} {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)
			err := manager.Statistics(context.Background(), "employee", options...)
			require.ErrorContains(t, err, "statistics output must not be nil")
			require.False(t, connection.opened)
			require.False(t, connection.closed)
		})
	}
}

func TestManagerStatisticsValidationBeforeConnect(t *testing.T) {
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
			if len(tt.options) == 0 {
				tt.options = []fbservice.StatisticsOptions{{}}
			}
			for i := range tt.options {
				tt.options[i].Output = io.Discard
			}
			manager, connection := newTestManager(t)
			require.Error(t, manager.Statistics(context.Background(), tt.database, tt.options...))
			require.False(t, connection.opened)
		})
	}
}

func TestManagerStatisticsDatabasePaths(t *testing.T) {
	for _, database := range []string{"./-name.fdb", "/data/-name.fdb", "employee"} {
		t.Run(database, func(t *testing.T) {
			manager, connection := newTestManager(t)
			require.NoError(t, manager.Statistics(context.Background(), database, fbservice.StatisticsOptions{Output: io.Discard}))
			require.Equal(t, database, connection.database)
			require.Equal(t, database, connection.expectedDatabase)
			require.True(t, connection.closed)
		})
	}
}

func TestManagerStatisticsConstructors(t *testing.T) {
	manager, err := fbservice.NewManager("host=localhost user=sysdba database=ignored")
	require.NoError(t, err)
	var connection *fakeServiceConnection
	connect := func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		connection = new(fakeServiceConnection)
		connection.connectionConfig = config.ConnectionConfig.Copy()
		connection.expectedDatabase = config.ExpectedDatabase
		return connection, nil
	}
	fbservice.SetConnectForTest(manager, connect)
	require.NoError(t, manager.Statistics(context.Background(), "db", fbservice.StatisticsOptions{Output: io.Discard}))
	require.Equal(t, "localhost", connection.connectionConfig.Host)
	requireDatabaseConnectionSettingsCleared(t, connection.connectionConfig)
	require.Equal(t, "db", connection.expectedDatabase)

	_, err = fbservice.NewManager("port=invalid")
	require.Error(t, err)
	config, err := fbservice.ParseConfig("host=localhost user=sysdba auth_plugin_list=Srp")
	require.NoError(t, err)
	manager, err = fbservice.NewManagerConfig(config)
	require.NoError(t, err)
	fbservice.SetConnectForTest(manager, connect)
	config.Host = "changed"
	config.AuthPlugins[0] = "changed"
	require.NoError(t, manager.Statistics(context.Background(), "db", fbservice.StatisticsOptions{Output: io.Discard}))
	require.Equal(t, "localhost", connection.connectionConfig.Host)
	require.Equal(t, []string{"Srp"}, connection.connectionConfig.AuthPluginList)
	for _, invalid := range []*fbservice.Manager{nil, {}} {
		require.ErrorContains(t, invalid.Statistics(context.Background(), "db", fbservice.StatisticsOptions{Output: io.Discard}), "service manager config must not be nil")
	}
}

func TestManagerStatisticsErrors(t *testing.T) {
	manager, connection := newTestManager(t)
	operationErr := errors.New("statistics failed")
	closeErr := errors.New("detach failed")
	connection.statisticsErr, connection.closeErr = operationErr, closeErr
	err := manager.Statistics(context.Background(), "db", fbservice.StatisticsOptions{Output: io.Discard})
	require.ErrorIs(t, err, operationErr)
	require.ErrorIs(t, err, closeErr)
	require.True(t, connection.closed)
	connectErr := errors.New("connect failed")
	fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		return nil, connectErr
	})
	require.ErrorIs(t, manager.Statistics(context.Background(), "db", fbservice.StatisticsOptions{Output: io.Discard}), connectErr)
}

func TestManagerStatisticsClosesAfterCancellation(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := &statisticsCancelConnection{cancel: cancel}
	fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		return connection, nil
	})
	require.ErrorIs(t, manager.Statistics(ctx, "db", fbservice.StatisticsOptions{Output: io.Discard}), context.Canceled)
	require.True(t, connection.closed)
	require.NoError(t, connection.closeContextErr)
	require.True(t, connection.closeHasDeadline)
}

func TestManagerStatisticsCanceledBeforeConnect(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelDeadline := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancelDeadline()

	for _, ctx := range []context.Context{canceled, expired} {
		t.Run(ctx.Err().Error(), func(t *testing.T) {
			manager, connection := newTestManager(t)
			err := manager.Statistics(ctx, "db", fbservice.StatisticsOptions{Output: io.Discard})
			require.ErrorIs(t, err, ctx.Err())
			require.False(t, connection.opened)
			require.False(t, connection.closed)
		})
	}
}

type statisticsCancelConnection struct {
	fakeServiceConnection
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
