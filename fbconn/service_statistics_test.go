package fbconn

import (
	"bytes"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn/ctxwatch"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestBuildStatisticsActionSPBEachOption(t *testing.T) {
	tests := map[string]struct {
		options StatisticsOptions
		want    []byte
	}{
		"defaults":                    {StatisticsOptions{}, []byte{108, 0, 0, 0, 0}},
		"Output":                      {StatisticsOptions{Output: io.Discard}, []byte{108, 0, 0, 0, 0}},
		"DataPages":                   {StatisticsOptions{DataPages: true}, []byte{108, 1, 0, 0, 0}},
		"HeaderPages":                 {StatisticsOptions{HeaderPages: true}, []byte{108, 4, 0, 0, 0}},
		"IndexPages":                  {StatisticsOptions{IndexPages: true}, []byte{108, 8, 0, 0, 0}},
		"SystemRelations":             {StatisticsOptions{SystemRelations: true}, []byte{108, 16, 0, 0, 0}},
		"RecordVersions":              {StatisticsOptions{RecordVersions: true}, []byte{108, 32, 0, 0, 0}},
		"NoCreation":                  {StatisticsOptions{NoCreation: true}, []byte{108, 128, 0, 0, 0}},
		"Encryption":                  {StatisticsOptions{Encryption: true}, []byte{108, 0, 1, 0, 0}},
		"Tables":                      {StatisticsOptions{Tables: []string{"A", "b c"}}, []byte{64, 1, 0, 'A', 64, 3, 0, 'b', ' ', 'c', 108, 0, 0, 0, 0}},
		"Schemas":                     {StatisticsOptions{Schemas: []string{"S", "s p"}}, []byte{65, 1, 0, 'S', 65, 3, 0, 's', ' ', 'p', 108, 0, 0, 0, 0}},
		"ParallelWorkers":             {StatisticsOptions{ParallelWorkers: 258}, []byte{100, 2, 1, 0, 0, 108, 0, 0, 0, 0}},
		"all analysis flags":          {StatisticsOptions{DataPages: true, IndexPages: true, SystemRelations: true, RecordVersions: true, NoCreation: true}, []byte{108, 185, 0, 0, 0}},
		"header without creation":     {StatisticsOptions{HeaderPages: true, NoCreation: true}, []byte{108, 132, 0, 0, 0}},
		"encryption without creation": {StatisticsOptions{Encryption: true, NoCreation: true}, []byte{108, 128, 1, 0, 0}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			spb, err := buildStatisticsActionSPB("db", tt.options)
			require.NoError(t, err)
			// Literal wire values check action, database tag, string length and each option.
			require.Equal(t, append([]byte{11, 106, 2, 0, 'd', 'b'}, tt.want...), spb)
		})
	}
}

func TestBuildStatisticsActionSPBValidation(t *testing.T) {
	type testCase struct {
		database string
		options  StatisticsOptions
	}
	tests := map[string]testCase{
		"empty database":          {},
		"long database":           {database: strings.Repeat("d", 65536)},
		"NUL database":            {database: "d\x00b"},
		"help switch database":    {database: "-?"},
		"version switch database": {database: "-z"},
		"negative workers":        {database: "db", options: StatisticsOptions{ParallelWorkers: -1}},
		"empty table":             {database: "db", options: StatisticsOptions{Tables: []string{""}}},
		"NUL table":               {database: "db", options: StatisticsOptions{Tables: []string{"a\x00b"}}},
		"switch table":            {database: "db", options: StatisticsOptions{Tables: []string{"-system"}}},
		"long table":              {database: "db", options: StatisticsOptions{Tables: []string{strings.Repeat("я", 32768)}}},
		"empty schema":            {database: "db", options: StatisticsOptions{Schemas: []string{""}}},
		"NUL schema":              {database: "db", options: StatisticsOptions{Schemas: []string{"a\x00b"}}},
		"switch schema":           {database: "db", options: StatisticsOptions{Schemas: []string{"-system"}}},
		"long schema":             {database: "db", options: StatisticsOptions{Schemas: []string{strings.Repeat("я", 32768)}}},
	}
	if strconv.IntSize > 32 {
		overflow := int64(maxServiceActionInt32) + 1
		tests["workers overflow"] = testCase{"db", StatisticsOptions{ParallelWorkers: int(overflow)}}
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := buildStatisticsActionSPB(tt.database, tt.options)
			require.Error(t, err)
		})
	}
}

func TestBuildStatisticsActionSPBDoesNotApplyManagerSemantics(t *testing.T) {
	spb, err := buildStatisticsActionSPB("db", StatisticsOptions{HeaderPages: true, Encryption: true})
	require.NoError(t, err)
	require.Equal(t, []byte{11, 106, 2, 0, 'd', 'b', 108, 4, 1, 0, 0}, spb)
	for name, option := range map[string]StatisticsOptions{
		"data": {DataPages: true}, "index": {IndexPages: true}, "system": {SystemRelations: true},
		"record": {RecordVersions: true}, "table": {Tables: []string{"T"}}, "schema": {Schemas: []string{"S"}},
	} {
		t.Run("header with "+name, func(t *testing.T) {
			option.HeaderPages = true
			_, err := buildStatisticsActionSPB("db", option)
			require.NoError(t, err)
		})
		t.Run("encryption with "+name, func(t *testing.T) {
			option.HeaderPages, option.Encryption = false, true
			_, err := buildStatisticsActionSPB("db", option)
			require.NoError(t, err)
		})
	}
}

func TestBuildStatisticsActionSPBDatabasePaths(t *testing.T) {
	for _, database := range []string{"./-name.fdb", "/data/-name.fdb", "employee"} {
		t.Run(database, func(t *testing.T) {
			spb, err := buildStatisticsActionSPB(database, StatisticsOptions{})
			require.NoError(t, err)
			requireServiceActionString(t, spb, 1, 106, database)
		})
	}
}

func TestBuildStatisticsActionSPBStringBytesAndMaximumWorkers(t *testing.T) {
	spb, err := buildStatisticsActionSPB("база", StatisticsOptions{
		Tables: []string{"Mixed \" table", "Таблица"}, Schemas: []string{"Схема"}, ParallelWorkers: maxServiceActionInt32,
	})
	require.NoError(t, err)
	offset := requireServiceActionString(t, spb, 1, 106, "база")
	offset = requireServiceActionString(t, spb, offset, 65, "Схема")
	offset = requireServiceActionString(t, spb, offset, 64, "Mixed \" table")
	offset = requireServiceActionString(t, spb, offset, 64, "Таблица")
	require.Equal(t, []byte{100, 255, 255, 255, 127, 108, 0, 0, 0, 0}, spb[offset:])
	_, err = buildStatisticsActionSPB(strings.Repeat("d", 65535), StatisticsOptions{})
	require.NoError(t, err)
}

func TestServiceConnStatisticsOutput(t *testing.T) {
	for name, discard := range map[string]bool{"writer": false, "nil writer": true} {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			resultCh := make(chan serviceActionProbeResult, 1)
			const report = "Database header page information:\n\tPage size 8192\n"
			go func() {
				defer func() { _ = server.Close() }()
				resultCh <- runServiceActionProbeServerWithOutput(server, report)
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			var output bytes.Buffer
			option := StatisticsOptions{HeaderPages: true, Output: &output}
			if discard {
				option.Output = nil
			}
			require.NoError(t, conn.Statistics(ctx, "db", option))
			if discard {
				require.Empty(t, output.String())
			} else {
				require.Equal(t, report, output.String())
			}
			require.False(t, conn.actionActive)
			require.NoError(t, conn.Close(ctx))
			select {
			case result := <-resultCh:
				require.NoError(t, result.err)
				require.Equal(t, []byte{11, 106, 2, 0, 'd', 'b', 108, 4, 0, 0, 0}, result.actionSPB)
				require.Equal(t, []int{1024, 2048, 2048, 2048}, result.bufferLengths)
				require.Equal(t, []byte{fbproto.Isc_info_svc_to_eof}, result.receiveItems)
				require.Equal(t, 42, result.detachedHandle)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}

func TestServiceConnStatisticsOutputError(t *testing.T) {
	client, server := net.Pipe()
	done := make(chan error, 1)
	go func() {
		defer func() { _ = server.Close() }()
		done <- runBackupOutputProbeServer(server)
	}()
	conn, ctx := newStatisticsProbeConnection(t, client)
	require.ErrorContains(t, conn.Statistics(ctx, "db", StatisticsOptions{Output: errorWriter{}}), "database statistics: write service output: output failed")
	require.NoError(t, conn.Close(ctx))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestServiceConnStatisticsCancellation(t *testing.T) {
	client, server := net.Pipe()
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		defer func() { _ = server.Close() }()
		done <- runCanceledServiceActionProbeServer(server, started, 11)
	}()
	conn, deadline := newStatisticsProbeConnection(t, client)
	ctx, cancel := context.WithCancel(deadline)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- conn.Statistics(ctx, "db", StatisticsOptions{}) }()
	select {
	case <-started:
		cancel()
	case <-deadline.Done():
		t.Fatal(deadline.Err())
	}
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-deadline.Done():
		t.Fatal(deadline.Err())
	}
	require.True(t, conn.conn.IsClosed())
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-deadline.Done():
		t.Fatal(deadline.Err())
	}
}

func newStatisticsProbeConnection(t *testing.T, client net.Conn) (*ServiceConn, context.Context) {
	t.Helper()
	t.Cleanup(func() { _ = client.Close() })
	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil }
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) { return client, nil }
	config.BuildContextWatcherHandler = func(c *FbConn) ctxwatch.Handler { return &DeadlineContextWatcherHandler{Conn: c.conn} }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	conn, err := ConnectServiceConfig(ctx, testServiceConfig(config))
	require.NoError(t, err)
	return conn, ctx
}
