package fbx_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbservice"
)

type restoreIntegrationServer struct {
	profile string
	kind    string
	version string
	major   int
	minor   int
}

type restoreIntegrationCapabilities struct {
	ignoreBLRErrors bool
	includeData     bool
	replicaMode     bool
	encryption      bool
	directIO        bool
	parallelWorkers bool
	schemaFilters   bool
	tablespaces     bool
}

var restoreIntegrationVersionRegexp = regexp.MustCompile(`^(\d+)\.(\d+)`)

func TestServiceRestoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("service restore integration test is disabled in short mode")
	}

	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("service restore integration test requires FBX_TEST_DATABASE")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	server := restoreIntegrationServerFromEnvironment(ctx, t, connString)
	capabilities := server.capabilities()
	t.Logf(
		"restore integration profile: name=%s kind=%s version=%s capabilities=%+v",
		server.profile,
		server.kind,
		server.version,
		capabilities,
	)

	serviceConfig, err := fbservice.ParseConfig(connString)
	require.NoError(t, err)
	manager := fbservice.NewManager(serviceConfig)
	fixture := newRestoreIntegrationBackup(
		ctx,
		t,
		manager,
		connString,
		"common",
		populateCommonRestoreIntegrationDatabase,
	)

	t.Run("verbose interval and statistics", func(t *testing.T) {
		var output bytes.Buffer
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			Output:          &output,
			VerboseInterval: 100,
			Statistics:      fbservice.RestoreStatisticsAll,
		})

		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(t, 150, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
		require.EqualValues(t, 75, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_SKIP_DATA"))
		outputText := strings.ToLower(output.String())
		require.Contains(t, outputText, "100 records restored")
		require.Contains(t, outputText, "total statistics")
	})

	t.Run("FSS repair parameters accepted", func(t *testing.T) {
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			FixFSSData:     "WIN1251",
			FixFSSMetadata: "WIN1251",
		})

		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(t, 150, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
	})

	t.Run("metadata only", func(t *testing.T) {
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{MetadataOnly: true})

		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.Zero(t, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
		require.Zero(t, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_SKIP_DATA"))
	})

	t.Run("skip data", func(t *testing.T) {
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{SkipData: "RESTORE_SKIP_DATA"})

		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(t, 150, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
		require.Zero(t, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_SKIP_DATA"))
	})

	t.Run("ignore BLR errors parameter accepted", func(t *testing.T) {
		if !capabilities.ignoreBLRErrors {
			t.Skipf("ignore-BLR-errors restore is not supported by %s %s", server.kind, server.version)
		}

		database := fixture.restore(ctx, t, fbservice.RestoreOptions{IgnoreBLRErrors: true})
		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(t, 150, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
	})

	t.Run("include data", func(t *testing.T) {
		if !capabilities.includeData {
			if server.kind == "rdb" && server.major == 3 {
				t.Skip("Red Database 3 requires the legacy KEEP_DATA tag, which RestoreOptions does not emit")
			}
			t.Skipf("include-data restore is not supported by %s %s", server.kind, server.version)
		}

		database := fixture.restore(ctx, t, fbservice.RestoreOptions{IncludeData: "RESTORE_KEEP_DATA"})

		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(t, 150, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
		require.Zero(t, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_SKIP_DATA"))
	})

	t.Run("replica mode", func(t *testing.T) {
		if !capabilities.replicaMode {
			t.Skipf("replica-mode restore is not supported by %s %s", server.kind, server.version)
		}

		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			ReplicaMode: fbservice.RestoreReplicaModeReadOnly,
		})

		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		var replicaMode string
		err := conn.QueryRow(
			ctx,
			"select rdb$get_context('SYSTEM', 'REPLICA_MODE') from rdb$database",
		).Scan(&replicaMode)
		require.NoError(t, err)
		require.Equal(t, "READ-ONLY", replicaMode)
	})

	t.Run("direct I/O", func(t *testing.T) {
		if !capabilities.directIO {
			t.Skipf("direct-I/O restore is not supported by %s %s", server.kind, server.version)
		}

		database := fixture.restore(ctx, t, fbservice.RestoreOptions{DirectIO: true})
		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(t, 150, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
	})

	t.Run("parallel workers", func(t *testing.T) {
		if !capabilities.parallelWorkers {
			if server.kind == "rdb" && server.major == 3 {
				t.Skip("Red Database 3 requires the legacy parallel-workers tag, which RestoreOptions does not emit")
			}
			t.Skipf("parallel restore is not supported by %s %s", server.kind, server.version)
		}

		database := fixture.restore(ctx, t, fbservice.RestoreOptions{ParallelWorkers: 2})
		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(t, 150, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_DATA"))
	})

	t.Run("encryption parameters reach server", func(t *testing.T) {
		if !capabilities.encryption {
			t.Skipf("encrypted restore is not supported by %s %s", server.kind, server.version)
		}

		const missingKeyHolder = "fbx_missing_key_holder"
		database := fixture.nextDatabase(t)
		var output bytes.Buffer
		err := manager.Restore(ctx, fixture.backupFile, database, fbservice.RestoreOptions{
			Output:      &output,
			KeyHolder:   missingKeyHolder,
			KeyName:     "fbx_test_key",
			CryptPlugin: "fbx_missing_crypt_plugin",
		})
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), missingKeyHolder)
	})

	t.Run("schema filters", func(t *testing.T) {
		if !capabilities.schemaFilters {
			t.Skipf("schema-data filters are not supported by %s %s", server.kind, server.version)
		}
		testRestoreIntegrationSchemaFilters(ctx, t, manager, connString)
	})

	t.Run("tablespace paths", func(t *testing.T) {
		if !capabilities.tablespaces {
			t.Skipf("tablespace restore paths are not supported by %s %s", server.kind, server.version)
		}
		testRestoreIntegrationTablespaces(ctx, t, manager, connString)
	})
}

func (server restoreIntegrationServer) capabilities() restoreIntegrationCapabilities {
	var capabilities restoreIntegrationCapabilities

	switch server.kind {
	case "firebird":
		capabilities.includeData = server.atLeast(4, 0)
		capabilities.replicaMode = server.atLeast(4, 0)
		capabilities.encryption = server.atLeast(4, 0)
		capabilities.directIO = server.atLeast(5, 0)
		capabilities.parallelWorkers = server.atLeast(5, 0)
		capabilities.schemaFilters = server.atLeast(6, 0)
	case "rdb":
		capabilities.ignoreBLRErrors = server.atLeast(3, 0)
		capabilities.includeData = server.atLeast(5, 0)
		capabilities.replicaMode = server.atLeast(5, 0)
		capabilities.encryption = server.atLeast(5, 0)
		capabilities.directIO = server.atLeast(5, 0)
		capabilities.parallelWorkers = server.atLeast(5, 0)
		capabilities.schemaFilters = server.atLeast(6, 0)
		capabilities.tablespaces = server.atLeast(5, 2) && !server.is(6, 0)
	}

	return capabilities
}

func (server restoreIntegrationServer) atLeast(major int, minor int) bool {
	return server.major > major || server.major == major && server.minor >= minor
}

func (server restoreIntegrationServer) after(major int, minor int) bool {
	return server.major > major || server.major == major && server.minor > minor
}

func (server restoreIntegrationServer) is(major int, minor int) bool {
	return server.major == major && server.minor == minor
}

func restoreIntegrationServerFromEnvironment(
	ctx context.Context,
	t *testing.T,
	connString string,
) restoreIntegrationServer {
	t.Helper()

	server := restoreIntegrationServer{
		profile: strings.TrimSpace(os.Getenv("TARGET")),
		kind:    strings.ToLower(strings.TrimSpace(os.Getenv("DB_KIND"))),
		version: strings.TrimSpace(os.Getenv("DB_VERSION")),
	}
	if server.kind == "" && server.version == "" {
		rdbVersion := strings.TrimSpace(os.Getenv("RDB_VERSION"))
		firebirdVersion := strings.TrimSpace(os.Getenv("FB_VERSION"))
		switch {
		case rdbVersion != "" && firebirdVersion != "":
			t.Fatal("both RDB_VERSION and FB_VERSION are set")
		case rdbVersion != "":
			server.kind = "rdb"
			server.version = rdbVersion
		case firebirdVersion != "":
			server.kind = "firebird"
			server.version = firebirdVersion
		default:
			t.Skip("service restore integration test requires DB_KIND/DB_VERSION, RDB_VERSION, or FB_VERSION")
		}
	}
	if server.kind == "" || server.version == "" {
		t.Fatal("database kind and version must be provided together")
	}
	if server.kind != "firebird" && server.kind != "rdb" {
		t.Fatalf("unsupported DB_KIND %q", server.kind)
	}

	declaredVersion := server.version
	declaredMajor, declaredMinor := parseRestoreIntegrationVersion(t, declaredVersion)

	conn, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, conn)

	var engineVersion string
	err = conn.QueryRow(
		ctx,
		"select rdb$get_context('SYSTEM', 'ENGINE_VERSION') from rdb$database",
	).Scan(&engineVersion)
	require.NoError(t, err)
	engineMajor, engineMinor := parseRestoreIntegrationVersion(t, engineVersion)
	require.Equalf(
		t,
		declaredMajor,
		engineMajor,
		"DB_VERSION %q does not match server ENGINE_VERSION %q",
		declaredVersion,
		engineVersion,
	)
	require.Equalf(
		t,
		declaredMinor,
		engineMinor,
		"DB_VERSION %q does not match server ENGINE_VERSION %q",
		declaredVersion,
		engineVersion,
	)
	server.version = engineVersion
	server.major = engineMajor
	server.minor = engineMinor
	if server.profile == "" {
		server.profile = fmt.Sprintf("%s-%s", server.kind, server.version)
	}

	return server
}

func parseRestoreIntegrationVersion(t *testing.T, version string) (int, int) {
	t.Helper()
	matches := restoreIntegrationVersionRegexp.FindStringSubmatch(version)
	if len(matches) != 3 {
		t.Fatalf("failed to parse database version %q", version)
	}
	major, err := strconv.Atoi(matches[1])
	require.NoError(t, err)
	minor, err := strconv.Atoi(matches[2])
	require.NoError(t, err)
	return major, minor
}

type restoreIntegrationBackup struct {
	manager        *fbservice.Manager
	directory      string
	backupFile     string
	nextDatabaseID int
}

func newRestoreIntegrationBackup(
	ctx context.Context,
	t *testing.T,
	manager *fbservice.Manager,
	connString string,
	name string,
	populate func(context.Context, *testing.T, *fbx.Conn, string),
) *restoreIntegrationBackup {
	t.Helper()

	serviceRoot := strings.TrimSpace(os.Getenv("FBX_TEST_SERVICE_DIR"))
	if serviceRoot == "" {
		serviceRoot = os.TempDir()
	}
	require.Truef(t, filepath.IsAbs(serviceRoot), "FBX_TEST_SERVICE_DIR must be absolute: %q", serviceRoot)
	rootInfo, err := os.Stat(serviceRoot)
	require.NoError(t, err)
	require.Truef(t, rootInfo.IsDir(), "FBX_TEST_SERVICE_DIR is not a directory: %q", serviceRoot)

	directory, err := os.MkdirTemp(serviceRoot, "fbx-service-restore-"+name+"-*")
	require.NoError(t, err)
	require.NoError(t, os.Chmod(directory, 0o777))
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("remove restore integration directory %q: %v", directory, err)
		}
	})

	sourceDatabase := filepath.Join(directory, "source.fdb")
	createConfig, err := fbx.ParseConfig(connString)
	require.NoError(t, err)
	createConfig.Database = sourceDatabase

	conn, err := fbx.CreateDatabaseConfig(ctx, createConfig)
	require.NoError(t, err)
	populate(ctx, t, conn, directory)
	closeConn(t, conn)

	backupFile := filepath.Join(directory, "source.fbk")
	err = manager.Backup(ctx, sourceDatabase, backupFile)
	require.NoError(t, err)
	backupInfo, err := os.Stat(backupFile)
	require.NoError(t, err)
	require.Positive(t, backupInfo.Size())

	err = fbx.DropDatabaseConfig(ctx, createConfig)
	require.NoError(t, err)

	return &restoreIntegrationBackup{
		manager:    manager,
		directory:  directory,
		backupFile: backupFile,
	}
}

func (fixture *restoreIntegrationBackup) nextDatabase(t *testing.T) string {
	t.Helper()
	fixture.nextDatabaseID++
	database := filepath.Join(
		fixture.directory,
		fmt.Sprintf("restored-%02d.fdb", fixture.nextDatabaseID),
	)
	return database
}

func (fixture *restoreIntegrationBackup) restore(
	ctx context.Context,
	t *testing.T,
	options fbservice.RestoreOptions,
) string {
	t.Helper()
	database := fixture.nextDatabase(t)
	err := fixture.manager.Restore(ctx, fixture.backupFile, database, options)
	require.NoError(t, err)
	return database
}

func populateCommonRestoreIntegrationDatabase(
	ctx context.Context,
	t *testing.T,
	conn *fbx.Conn,
	_ string,
) {
	t.Helper()

	_, err := conn.Exec(ctx, "create table RESTORE_KEEP_DATA (ID integer not null)")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "create table RESTORE_SKIP_DATA (ID integer not null)")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `
execute block as
	declare variable N integer;
begin
	N = 1;
	while (N <= 150) do
	begin
		insert into RESTORE_KEEP_DATA (ID) values (:N);
		if (N <= 75) then
			insert into RESTORE_SKIP_DATA (ID) values (:N);
		N = N + 1;
	end
end`)
	require.NoError(t, err)
}

func connectRestoreIntegrationDatabase(
	ctx context.Context,
	t *testing.T,
	connString string,
	database string,
) *fbx.Conn {
	t.Helper()
	config, err := fbx.ParseConfig(connString)
	require.NoError(t, err)
	config.Database = database
	config.CreateDatabase = false
	conn, err := fbx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	return conn
}

func restoreIntegrationTableCount(
	ctx context.Context,
	t *testing.T,
	conn *fbx.Conn,
	relation string,
) int64 {
	t.Helper()
	var count int64
	err := conn.QueryRow(ctx, "select count(*) from "+relation).Scan(&count)
	require.NoError(t, err)
	return count
}

func testRestoreIntegrationSchemaFilters(
	ctx context.Context,
	t *testing.T,
	manager *fbservice.Manager,
	connString string,
) {
	t.Helper()

	fixture := newRestoreIntegrationBackup(
		ctx,
		t,
		manager,
		connString,
		"schemas",
		func(ctx context.Context, t *testing.T, conn *fbx.Conn, _ string) {
			_, err := conn.Exec(ctx, "create schema RESTORE_KEEP_SCHEMA")
			require.NoError(t, err)
			_, err = conn.Exec(ctx, "create schema RESTORE_SKIP_SCHEMA")
			require.NoError(t, err)
			_, err = conn.Exec(ctx, "create table RESTORE_KEEP_SCHEMA.DATA_ROWS (ID integer)")
			require.NoError(t, err)
			_, err = conn.Exec(ctx, "create table RESTORE_SKIP_SCHEMA.DATA_ROWS (ID integer)")
			require.NoError(t, err)
			_, err = conn.Exec(ctx, "insert into RESTORE_KEEP_SCHEMA.DATA_ROWS values (1)")
			require.NoError(t, err)
			_, err = conn.Exec(ctx, "insert into RESTORE_SKIP_SCHEMA.DATA_ROWS values (2)")
			require.NoError(t, err)
		},
	)

	t.Run("skip schema data", func(t *testing.T) {
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			SkipSchemaData: "RESTORE_SKIP_SCHEMA",
		})
		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(
			t,
			1,
			restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_SCHEMA.DATA_ROWS"),
		)
		require.Zero(
			t,
			restoreIntegrationTableCount(ctx, t, conn, "RESTORE_SKIP_SCHEMA.DATA_ROWS"),
		)
	})

	t.Run("include schema data", func(t *testing.T) {
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			IncludeSchemaData: "RESTORE_KEEP_SCHEMA",
		})
		conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
		defer closeConn(t, conn)
		require.EqualValues(
			t,
			1,
			restoreIntegrationTableCount(ctx, t, conn, "RESTORE_KEEP_SCHEMA.DATA_ROWS"),
		)
		require.Zero(
			t,
			restoreIntegrationTableCount(ctx, t, conn, "RESTORE_SKIP_SCHEMA.DATA_ROWS"),
		)
	})
}

func testRestoreIntegrationTablespaces(
	ctx context.Context,
	t *testing.T,
	manager *fbservice.Manager,
	connString string,
) {
	t.Helper()

	var originalTablespacePath string
	fixture := newRestoreIntegrationBackup(
		ctx,
		t,
		manager,
		connString,
		"tablespaces",
		func(ctx context.Context, t *testing.T, conn *fbx.Conn, directory string) {
			originalTablespacePath = filepath.Join(directory, "original-tablespace.dat")
			_, err := conn.Exec(
				ctx,
				fmt.Sprintf(
					"create tablespace RESTORE_TS file '%s'",
					restoreIntegrationSQLString(originalTablespacePath),
				),
			)
			require.NoError(t, err)
			_, err = conn.Exec(ctx, "create table RESTORE_TS_DATA (ID integer) tablespace RESTORE_TS")
			require.NoError(t, err)
			_, err = conn.Exec(ctx, "insert into RESTORE_TS_DATA values (42)")
			require.NoError(t, err)
		},
	)
	require.NoError(t, os.RemoveAll(originalTablespacePath))

	t.Run("explicit path", func(t *testing.T) {
		tablespacePath := filepath.Join(fixture.directory, "explicit-tablespace.dat")
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			TablespacePaths: []fbservice.RestoreTablespacePath{{
				Name: "RESTORE_TS",
				Path: tablespacePath,
			}},
		})
		requireRestoreIntegrationTablespace(ctx, t, connString, database, tablespacePath)
	})

	t.Run("mapping file", func(t *testing.T) {
		tablespacePath := filepath.Join(fixture.directory, "mapped-tablespace.dat")
		mappingFile := filepath.Join(fixture.directory, "tablespaces.map")
		err := os.WriteFile(
			mappingFile,
			[]byte("RESTORE_TS "+tablespacePath+"\n"),
			0o644,
		)
		require.NoError(t, err)

		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			TablespaceMappingFile: mappingFile,
		})
		requireRestoreIntegrationTablespace(ctx, t, connString, database, tablespacePath)
	})

	t.Run("original path", func(t *testing.T) {
		database := fixture.restore(ctx, t, fbservice.RestoreOptions{
			UseOriginalTablespacePaths: true,
		})
		requireRestoreIntegrationTablespace(
			ctx,
			t,
			connString,
			database,
			originalTablespacePath,
		)
	})
}

func requireRestoreIntegrationTablespace(
	ctx context.Context,
	t *testing.T,
	connString string,
	database string,
	wantTablespacePath string,
) {
	t.Helper()

	conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
	defer closeConn(t, conn)
	require.EqualValues(t, 1, restoreIntegrationTableCount(ctx, t, conn, "RESTORE_TS_DATA"))

	var gotTablespacePath string
	err := conn.QueryRow(
		ctx,
		"select rdb$file_name from rdb$tablespaces where rdb$tablespace_name = 'RESTORE_TS'",
	).Scan(&gotTablespacePath)
	require.NoError(t, err)
	require.Equal(t, filepath.Clean(wantTablespacePath), filepath.Clean(strings.TrimSpace(gotTablespacePath)))

	_, err = os.Stat(wantTablespacePath)
	require.NoError(t, err)
}

func restoreIntegrationSQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
