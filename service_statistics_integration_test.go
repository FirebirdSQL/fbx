package fbx_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestServiceStatisticsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("service statistics integration test is disabled in short mode")
	}
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("service statistics integration test requires FBX_TEST_DATABASE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, version := newStatisticsIntegrationDatabase(ctx, t, connString)
	t.Logf("statistics server version: %s", version)
	manager, err := fbservice.NewManager(connString)
	require.NoError(t, err)
	run := func(t *testing.T, options fbservice.StatisticsOptions) string {
		t.Helper()
		var output bytes.Buffer
		options.Output = &output
		require.NoError(t, manager.Statistics(ctx, database, options))
		return output.String()
	}
	baseline := run(t, fbservice.StatisticsOptions{})
	require.Contains(t, baseline, "STATS_ONE")
	require.Contains(t, baseline, "STATS_TWO")
	require.Contains(t, baseline, "STATS_ONE_IDX")
	require.Contains(t, baseline, "Creation date")
	require.NotContains(t, baseline, "Average record length")

	t.Run("Output", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{})
		require.Contains(t, output, "Database header page information")
		require.Contains(t, output, "Page size")
		require.Contains(t, output, "STATS_ONE_IDX")
	})
	t.Run("Output nil", func(t *testing.T) {
		require.ErrorContains(t, manager.Statistics(ctx, database, fbservice.StatisticsOptions{}), "statistics output must not be nil")
	})
	t.Run("Output discard", func(t *testing.T) {
		require.NoError(t, manager.Statistics(ctx, database, fbservice.StatisticsOptions{Output: io.Discard}))
	})
	t.Run("DataPages", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{DataPages: true})
		require.Contains(t, output, "Data pages:")
		require.Contains(t, output, "STATS_ONE")
		require.NotContains(t, output, "STATS_ONE_IDX")
	})
	t.Run("HeaderPages", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{HeaderPages: true})
		require.Contains(t, output, "Database header page information")
		require.Contains(t, output, "Oldest transaction")
		require.NotContains(t, output, "STATS_ONE")
	})
	t.Run("IndexPages", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{IndexPages: true})
		require.Contains(t, output, "STATS_ONE_IDX")
		require.NotContains(t, output, "Data pages:")
	})
	t.Run("SystemRelations", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{SystemRelations: true})
		require.Contains(t, output, "RDB$RELATIONS")
		require.Contains(t, output, "STATS_ONE")
		require.NotContains(t, baseline, "RDB$RELATIONS")
	})
	t.Run("RecordVersions", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{RecordVersions: true})
		require.Contains(t, output, "Average record length")
		require.Contains(t, output, "Average version length")
		require.Contains(t, output, "total records: 3")
	})
	t.Run("Tables", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{Tables: []string{"STATS_ONE", "Mixed table"}})
		require.Contains(t, output, "STATS_ONE_IDX")
		require.Contains(t, output, "Mixed table")
		require.NotContains(t, output, "STATS_TWO")
	})
	t.Run("Schemas", func(t *testing.T) {
		option := fbservice.StatisticsOptions{Output: io.Discard, Schemas: []string{"STATS_KEEP_SCHEMA", "Mixed schema"}}
		if version.Major < 6 {
			var serverErr *fbconn.FbError
			require.ErrorAs(t, manager.Statistics(ctx, database, option), &serverErr)
			return
		}
		output := run(t, option)
		require.Contains(t, output, "STATS_SCHEMA_KEEP")
		require.Contains(t, output, "STATS_SCHEMA_MIXED")
		require.NotContains(t, output, "STATS_SCHEMA_SKIP")
		require.NotContains(t, output, "STATS_ONE")
	})
	t.Run("NoCreation", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{NoCreation: true})
		require.NotContains(t, output, "Creation date")
		require.Contains(t, output, "STATS_ONE_IDX")
	})
	t.Run("Encryption", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{Encryption: true})
		require.Contains(t, output, "Data pages: total")
		require.Contains(t, output, "Index pages: total")
		require.Contains(t, output, "encrypted 0")
		require.NotContains(t, output, "STATS_ONE_IDX")
	})
	t.Run("ParallelWorkers", func(t *testing.T) {
		server, err := serviceIntegrationServerFromEnvironment()
		require.NoError(t, err)
		if server.kind == "" {
			t.Skip("parallel gstat support requires DB_KIND/DB_VERSION, RDB_VERSION or FB_VERSION")
		}
		require.NoError(t, server.setVersion(version))
		option := fbservice.StatisticsOptions{Output: io.Discard, ParallelWorkers: 1}
		if server.kind == "rdb" && server.version.Major >= 5 {
			output := run(t, option)
			require.Contains(t, output, "STATS_ONE_IDX")
			require.Contains(t, output, "Data pages:")
		} else {
			var serverErr *fbconn.FbError
			require.ErrorAs(t, manager.Statistics(ctx, database, option), &serverErr)
		}
	})
	t.Run("combined data record and table filters", func(t *testing.T) {
		output := run(t, fbservice.StatisticsOptions{DataPages: true, RecordVersions: true, Tables: []string{"STATS_ONE"}, NoCreation: true})
		require.Contains(t, output, "total records: 3")
		require.NotContains(t, output, "STATS_TWO")
		require.NotContains(t, output, "STATS_ONE_IDX")
		require.NotContains(t, output, "Creation date")
	})
	t.Run("missing table returns server error", func(t *testing.T) {
		var serverErr *fbconn.FbError
		require.ErrorAs(t, manager.Statistics(ctx, database, fbservice.StatisticsOptions{Output: io.Discard, Tables: []string{"STATS_MISSING"}}), &serverErr)
	})
}

func newStatisticsIntegrationDatabase(ctx context.Context, t *testing.T, connString string) (string, fbx.ServerVersion) {
	t.Helper()
	root := os.Getenv("FBX_TEST_SERVICE_DIR")
	if root == "" {
		root = os.TempDir()
	}
	directory, err := os.MkdirTemp(root, "fbx-service-statistics-*")
	require.NoError(t, err)
	require.NoError(t, os.Chmod(directory, 0o777))
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(directory)) })
	config, err := fbx.ParseConfig(connString)
	require.NoError(t, err)
	config.Database = filepath.Join(directory, "statistics.fdb")
	conn, err := fbx.CreateDatabaseConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, conn.Close(cleanupCtx))
		require.NoError(t, fbx.DropDatabaseConfig(cleanupCtx, config))
	})
	version := conn.ServerVersion()
	for _, sql := range []string{
		"create table STATS_ONE (ID integer not null)",
		"create index STATS_ONE_IDX on STATS_ONE (ID)",
		"create table STATS_TWO (ID integer)",
		`create table "Mixed table" (ID integer)`,
		"insert into STATS_ONE values (1)", "insert into STATS_ONE values (2)", "insert into STATS_ONE values (3)",
		"insert into STATS_TWO values (42)", `insert into "Mixed table" values (7)`,
	} {
		_, err := conn.Exec(ctx, sql)
		require.NoError(t, err)
	}
	if version.Major >= 6 {
		for _, sql := range []string{
			"create schema STATS_KEEP_SCHEMA", "create schema STATS_SKIP_SCHEMA", `create schema "Mixed schema"`,
			"create table STATS_KEEP_SCHEMA.STATS_SCHEMA_KEEP (ID integer)",
			"create table STATS_SKIP_SCHEMA.STATS_SCHEMA_SKIP (ID integer)",
			`create table "Mixed schema".STATS_SCHEMA_MIXED (ID integer)`,
			"insert into STATS_KEEP_SCHEMA.STATS_SCHEMA_KEEP values (1)",
			"insert into STATS_SKIP_SCHEMA.STATS_SCHEMA_SKIP values (2)",
			`insert into "Mixed schema".STATS_SCHEMA_MIXED values (3)`,
		} {
			_, err := conn.Exec(ctx, sql)
			require.NoError(t, err)
		}
	}
	require.NoError(t, conn.Close(ctx))
	return config.Database, version
}
