package fbx_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

func TestServiceNBackupIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("nbackup integration test is disabled in short mode")
	}
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("nbackup integration test requires FBX_TEST_DATABASE and a shared server-side directory")
	}
	server, err := serviceIntegrationServerFromEnvironment()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	conn, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	version := conn.ServerVersion()
	closeConn(t, conn)
	require.NoError(t, server.setVersion(version))
	capabilities := server.nbackupCapabilities()
	t.Logf("nbackup integration profile: name=%s kind=%s version=%s capabilities=%+v", server.profile, server.kind, server.version, capabilities)
	manager, err := fbservice.NewManager(connString)
	require.NoError(t, err)

	t.Run("full and incremental levels", func(t *testing.T) {
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		files := []string{filepath.Join(dir, "full.nbk"), filepath.Join(dir, "level1.nbk"), filepath.Join(dir, "level2.nbk")}
		for level, file := range files {
			if level > 0 {
				nbackupIntegrationExec(ctx, t, connString, database, "insert into NBACKUP_DATA values (?)", level+1)
			}
			require.NoError(t, manager.NBackup(ctx, database, file, fbservice.NBackupOptions{Level: level, DirectIO: new(false)}))
			info, err := os.Stat(file)
			require.NoError(t, err)
			require.Positive(t, info.Size())
		}
		nbackupIntegrationExec(ctx, t, connString, database, "insert into NBACKUP_DATA values (4)")
		for count := 1; count <= len(files); count++ {
			restored := filepath.Join(dir, fmt.Sprintf("restored-%d.fdb", count))
			require.NoError(t, manager.NRestore(ctx, files[:count], restored))
			require.Equal(t, []int{1, 2, 3}[:count], nbackupIntegrationIDs(ctx, t, connString, restored))
			// Default restore must leave an existing database intact.
			require.Error(t, manager.NRestore(ctx, files[:count], restored))
			require.Equal(t, []int{1, 2, 3}[:count], nbackupIntegrationIDs(ctx, t, connString, restored))
		}
		require.Error(t, manager.NRestore(ctx, []string{files[0], files[2], files[1]}, filepath.Join(dir, "bad-order.fdb")))
		require.Equal(t, []int{1, 2, 3, 4}, nbackupIntegrationIDs(ctx, t, connString, database))
	})

	t.Run("GUID selects an earlier base", func(t *testing.T) {
		if !capabilities.guid {
			t.Skipf("backup by GUID requires a supporting server profile: %s %s", server.kind, server.version)
		}
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		full := filepath.Join(dir, "base.nbk")
		require.NoError(t, manager.NBackup(ctx, database, full))
		guid := nbackupIntegrationString(ctx, t, connString, database,
			"select first 1 RDB$GUID from RDB$BACKUP_HISTORY order by RDB$BACKUP_ID desc")
		nbackupIntegrationExec(ctx, t, connString, database, "insert into NBACKUP_DATA values (2)")
		require.NoError(t, manager.NBackup(ctx, database, filepath.Join(dir, "newer-base.nbk")))
		nbackupIntegrationExec(ctx, t, connString, database, "insert into NBACKUP_DATA values (3)")
		incremental := filepath.Join(dir, "by-guid.nbk")
		require.NoError(t, manager.NBackup(ctx, database, incremental, fbservice.NBackupOptions{GUID: guid}))
		restored := filepath.Join(dir, "restored.fdb")
		require.NoError(t, manager.NRestore(ctx, []string{full, incremental}, restored))
		require.Equal(t, []int{1, 2, 3}, nbackupIntegrationIDs(ctx, t, connString, restored))
	})

	t.Run("in-place incremental restore", func(t *testing.T) {
		if !capabilities.inPlace {
			t.Skipf("in-place restore requires a supporting server profile: %s %s", server.kind, server.version)
		}
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		full, incremental := filepath.Join(dir, "full.nbk"), filepath.Join(dir, "incremental.nbk")
		require.NoError(t, manager.NBackup(ctx, database, full))
		nbackupIntegrationExec(ctx, t, connString, database, "insert into NBACKUP_DATA values (2)")
		require.NoError(t, manager.NBackup(ctx, database, incremental, fbservice.NBackupOptions{Level: 1}))
		restored := filepath.Join(dir, "restored.fdb")
		require.NoError(t, manager.NRestore(ctx, []string{full}, restored))
		require.Equal(t, []int{1}, nbackupIntegrationIDs(ctx, t, connString, restored))
		require.NoError(t, manager.NRestore(ctx, []string{incremental}, restored, fbservice.NRestoreOptions{InPlace: true, NoDatabaseTriggers: true}))
		require.Equal(t, []int{1, 2}, nbackupIntegrationIDs(ctx, t, connString, restored))
	})

	t.Run("fixup and restore preserve sequence", func(t *testing.T) {
		if !capabilities.fixup || !capabilities.preserveSequence {
			t.Skipf("fixup and preserve sequence require a supporting server profile: %s %s", server.kind, server.version)
		}
		for _, preserve := range []bool{false, true} {
			t.Run(fmt.Sprintf("preserve=%t", preserve), func(t *testing.T) {
				database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
				const dbGUIDSQL = "select rdb$get_context('SYSTEM', 'DB_GUID') from rdb$database"
				guid := nbackupIntegrationString(ctx, t, connString, database, dbGUIDSQL)
				copyFile := filepath.Join(dir, "physical-copy.fdb")
				// A level-zero nbackup file is a physical database copy in backup mode.
				require.NoError(t, manager.NBackup(ctx, database, copyFile))
				restored := filepath.Join(dir, "restored.fdb")
				require.NoError(t, manager.NRestore(ctx, []string{copyFile}, restored, fbservice.NRestoreOptions{PreserveSequence: preserve}))
				require.NoError(t, manager.NFixup(ctx, copyFile, fbservice.NFixupOptions{PreserveSequence: preserve}))
				for _, target := range []string{copyFile, restored} {
					gotGUID := nbackupIntegrationString(ctx, t, connString, target, dbGUIDSQL)
					if preserve {
						require.Equal(t, guid, gotGUID)
					} else {
						require.NotEqual(t, guid, gotGUID)
					}
					require.Equal(t, []int{1}, nbackupIntegrationIDs(ctx, t, connString, target))
					nbackupIntegrationExec(ctx, t, connString, target, "insert into NBACKUP_DATA values (2)")
					require.Equal(t, []int{1, 2}, nbackupIntegrationIDs(ctx, t, connString, target))
				}
				require.Equal(t, []int{1}, nbackupIntegrationIDs(ctx, t, connString, database))
			})
		}
	})

	t.Run("history retention", func(t *testing.T) {
		if !capabilities.cleanHistory {
			t.Skipf("history cleanup requires a supporting server profile: %s %s", server.kind, server.version)
		}
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		for i := range 2 {
			require.NoError(t, manager.NBackup(ctx, database, filepath.Join(dir, fmt.Sprintf("full-%d.nbk", i))))
		}
		require.NoError(t, manager.NBackup(ctx, database, filepath.Join(dir, "days.nbk"), fbservice.NBackupOptions{CleanHistory: true, KeepDays: 1}))
		require.Equal(t, 3, nbackupIntegrationHistoryCount(ctx, t, connString, database))
		require.NoError(t, manager.NBackup(ctx, database, filepath.Join(dir, "rows.nbk"), fbservice.NBackupOptions{CleanHistory: true, KeepRows: 1}))
		require.Equal(t, 1, nbackupIntegrationHistoryCount(ctx, t, connString, database))
		file := nbackupIntegrationString(ctx, t, connString, database, "select RDB$FILE_NAME from RDB$BACKUP_HISTORY")
		require.Equal(t, filepath.Join(dir, "rows.nbk"), file)
	})

	t.Run("database triggers can be suppressed", func(t *testing.T) {
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		nbackupIntegrationExec(ctx, t, connString, database, "create exception NBACKUP_CONNECT 'nbackup connect trigger fired'")
		nbackupIntegrationExec(ctx, t, connString, database, "create trigger NBACKUP_CONNECT active on connect as begin exception NBACKUP_CONNECT; end")
		err := manager.NBackup(ctx, database, filepath.Join(dir, "with-triggers.nbk"))
		require.ErrorContains(t, err, "nbackup connect trigger fired")
		require.NoError(t, manager.NBackup(ctx, database, filepath.Join(dir, "without-triggers.nbk"), fbservice.NBackupOptions{NoDatabaseTriggers: true}))
	})

	t.Run("server errors remain available", func(t *testing.T) {
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		err := manager.NBackup(ctx, database, filepath.Join(dir, "missing-base.nbk"), fbservice.NBackupOptions{Level: 1})
		var serverErr *fbconn.FbError
		require.ErrorAs(t, err, &serverErr)
		require.False(t, fbconn.SafeToRetry(err))
		require.Equal(t, []int{1}, nbackupIntegrationIDs(ctx, t, connString, database))
	})

	t.Run("server paths with spaces and quotes", func(t *testing.T) {
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		dir = filepath.Join(dir, "copies 'backup'")
		require.NoError(t, os.Mkdir(dir, 0o777))
		require.NoError(t, os.Chmod(dir, 0o777))
		file, restored := filepath.Join(dir, "full copy.nbk"), filepath.Join(dir, "restored database.fdb")
		require.NoError(t, manager.NBackup(ctx, database, file))
		require.NoError(t, manager.NRestore(ctx, []string{file}, restored))
		require.Equal(t, []int{1}, nbackupIntegrationIDs(ctx, t, connString, restored))
	})

	t.Run("unsupported options return server errors", func(t *testing.T) {
		if server.kind == "" {
			t.Skip("unsupported-option checks require a declared server profile")
		}
		database, dir := newNBackupIntegrationDatabase(ctx, t, connString)
		full := filepath.Join(dir, "full.nbk")
		require.NoError(t, manager.NBackup(ctx, database, full))
		guid := nbackupIntegrationString(ctx, t, connString, database,
			"select first 1 RDB$GUID from RDB$BACKUP_HISTORY order by RDB$BACKUP_ID desc")
		for _, tt := range []struct {
			name      string
			supported bool
			run       func() error
		}{
			{"GUID", capabilities.guid, func() error {
				return manager.NBackup(ctx, database, filepath.Join(dir, "guid.nbk"), fbservice.NBackupOptions{GUID: guid})
			}},
			{"in-place", capabilities.inPlace, func() error {
				return manager.NRestore(ctx, []string{full}, filepath.Join(dir, "inplace.fdb"), fbservice.NRestoreOptions{InPlace: true})
			}},
			{"fixup", capabilities.fixup, func() error { return manager.NFixup(ctx, full) }},
			{"sequence", capabilities.preserveSequence, func() error {
				return manager.NRestore(ctx, []string{full}, filepath.Join(dir, "sequence.fdb"), fbservice.NRestoreOptions{PreserveSequence: true})
			}},
			{"history", capabilities.cleanHistory, func() error {
				return manager.NBackup(ctx, database, filepath.Join(dir, "history.nbk"), fbservice.NBackupOptions{CleanHistory: true, KeepRows: 1})
			}},
		} {
			if tt.supported {
				continue
			}
			t.Run(tt.name, func(t *testing.T) {
				var serverErr *fbconn.FbError
				err := tt.run()
				require.ErrorAs(t, err, &serverErr)
				require.False(t, fbconn.SafeToRetry(err))
			})
		}
	})
}

func newNBackupIntegrationDatabase(ctx context.Context, t *testing.T, connString string) (string, string) {
	t.Helper()
	root := os.Getenv("FBX_TEST_SERVICE_DIR")
	if root == "" {
		root = os.TempDir()
	}
	require.True(t, filepath.IsAbs(root), "FBX_TEST_SERVICE_DIR must be absolute")
	dir, err := os.MkdirTemp(root, "fbx-nbackup-*")
	require.NoError(t, err)
	require.NoError(t, os.Chmod(dir, 0o777))
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	config, err := fbx.ParseConfig(connString)
	require.NoError(t, err)
	config.Database = filepath.Join(dir, "source.fdb")
	conn, err := fbx.CreateDatabaseConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)
	_, err = conn.Exec(ctx, "create table NBACKUP_DATA (ID integer not null primary key)")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "insert into NBACKUP_DATA values (1)")
	require.NoError(t, err)
	return config.Database, dir
}

func nbackupIntegrationExec(ctx context.Context, t *testing.T, connString, database, sql string, args ...any) {
	t.Helper()
	conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
	defer closeConn(t, conn)
	_, err := conn.Exec(ctx, sql, args...)
	require.NoError(t, err)
}

func nbackupIntegrationIDs(ctx context.Context, t *testing.T, connString, database string) []int {
	t.Helper()
	conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
	defer closeConn(t, conn)
	rows, err := conn.Query(ctx, "select ID from NBACKUP_DATA order by ID")
	require.NoError(t, err)
	ids, err := fbx.CollectRows(rows, fbx.RowTo[int])
	require.NoError(t, err)
	return ids
}

func nbackupIntegrationString(ctx context.Context, t *testing.T, connString, database, sql string) string {
	t.Helper()
	conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
	defer closeConn(t, conn)
	var value string
	require.NoError(t, conn.QueryRow(ctx, sql).Scan(&value))
	return strings.TrimSpace(value)
}

func nbackupIntegrationHistoryCount(ctx context.Context, t *testing.T, connString, database string) int {
	t.Helper()
	conn := connectRestoreIntegrationDatabase(ctx, t, connString, database)
	defer closeConn(t, conn)
	var count int
	require.NoError(t, conn.QueryRow(ctx, "select count(*) from RDB$BACKUP_HISTORY").Scan(&count))
	return count
}
