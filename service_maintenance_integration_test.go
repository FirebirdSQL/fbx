package fbx_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestServiceMaintenanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("maintenance integration requires a server")
	}
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("maintenance integration requires FBX_TEST_DATABASE")
	}
	server, err := serviceIntegrationServerFromEnvironment()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	config, err := fbx.ParseConfig(connString)
	require.NoError(t, err)
	root := os.Getenv("FBX_TEST_SERVICE_DIR")
	if root == "" {
		root = os.TempDir()
	}
	directory, err := os.MkdirTemp(root, "fbx-service-maintenance-*")
	require.NoError(t, err)
	require.NoError(t, os.Chmod(directory, 0o777))
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(directory)) })
	config.Database = filepath.Join(directory, "maintenance.fdb")
	manager, err := fbservice.NewManager(connString)
	require.NoError(t, err)
	conn, err := fbx.CreateDatabaseConfig(ctx, config)
	require.NoError(t, err)
	t.Logf("maintenance server: %s", conn.ServerVersion())
	currentMode := fbservice.DatabaseModeNormal
	readOnlyPending := false
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		require.NoError(t, conn.Close(cleanup))
		// Force out any attachment left by a failed server-side maintenance action.
		if currentMode != fbservice.DatabaseModeFull {
			require.NoError(t, manager.SetDatabaseProperties(cleanup, config.Database, fbservice.DatabaseProperties{Shutdown: &fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce}}))
		}
		require.NoError(t, manager.SetDatabaseProperties(cleanup, config.Database, fbservice.DatabaseProperties{OnlineMode: fbservice.DatabaseModeNormal}))
		if readOnlyPending {
			require.NoError(t, manager.SetDatabaseProperties(cleanup, config.Database, fbservice.DatabaseProperties{AccessMode: fbservice.DatabaseAccessModeReadWrite}))
		}
		require.NoError(t, fbx.DropDatabaseConfig(cleanup, config))
	})
	require.NoError(t, server.setVersion(conn.ServerVersion()))
	caps := server.maintenanceCapabilities()
	t.Logf("maintenance profile: kind=%q version=%s", server.kind, server.version)
	for _, sql := range []string{
		"create table MAINT_DATA (ID integer)", "insert into MAINT_DATA values (1)",
		"create table MAINT_ICU (V varchar(30) character set utf8 collate unicode_ci_ai)",
		"create index MAINT_ICU_IDX on MAINT_ICU (V)", "insert into MAINT_ICU values ('Hello')",
	} {
		_, err := conn.Exec(ctx, sql)
		require.NoError(t, err)
	}
	require.NoError(t, conn.Close(ctx))
	set := func(t *testing.T, options fbservice.DatabaseProperties) {
		t.Helper()
		require.NoError(t, manager.SetDatabaseProperties(ctx, config.Database, options))
		if options.AccessMode != fbservice.DatabaseAccessModeUnchanged {
			readOnlyPending = options.AccessMode == fbservice.DatabaseAccessModeReadOnly
		}
		if options.Shutdown != nil {
			currentMode = options.Shutdown.Mode
		}
		if options.OnlineMode != fbservice.DatabaseModeUnchanged {
			currentMode = options.OnlineMode
		}
	}
	shutdown := func(t *testing.T, options fbservice.ShutdownOptions) {
		t.Helper()
		require.NoError(t, manager.Shutdown(ctx, config.Database, options))
		currentMode = options.Mode
	}
	header := func(t *testing.T) string {
		t.Helper()
		var out bytes.Buffer
		require.NoError(t, manager.Statistics(ctx, config.Database, fbservice.StatisticsOptions{Output: &out, HeaderPages: true}))
		return out.String()
	}
	queryInt := func(t *testing.T, sql string) int {
		t.Helper()
		c, err := fbx.ConnectConfig(ctx, config)
		require.NoError(t, err)
		defer func() { require.NoError(t, c.Close(ctx)) }()
		var n int
		require.NoError(t, c.QueryRow(ctx, sql).Scan(&n))
		return n
	}
	t.Run("access mode", func(t *testing.T) {
		set(t, fbservice.DatabaseProperties{AccessMode: fbservice.DatabaseAccessModeReadOnly})
		require.Equal(t, 1, queryInt(t, "select MON$READ_ONLY from MON$DATABASE"))
		c, err := fbx.ConnectConfig(ctx, config)
		require.NoError(t, err)
		_, err = c.Exec(ctx, "insert into MAINT_DATA values (2)")
		require.Error(t, err)
		require.NoError(t, c.Close(ctx))
		set(t, fbservice.DatabaseProperties{AccessMode: fbservice.DatabaseAccessModeReadWrite})
		require.Equal(t, 0, queryInt(t, "select MON$READ_ONLY from MON$DATABASE"))
	})
	t.Run("forced writes", func(t *testing.T) {
		set(t, fbservice.DatabaseProperties{WriteMode: fbservice.DatabaseWriteModeAsync})
		require.Equal(t, 0, queryInt(t, "select MON$FORCED_WRITES from MON$DATABASE"))
		set(t, fbservice.DatabaseProperties{WriteMode: fbservice.DatabaseWriteModeSync})
		require.Equal(t, 1, queryInt(t, "select MON$FORCED_WRITES from MON$DATABASE"))
	})
	t.Run("cache", func(t *testing.T) {
		set(t, fbservice.DatabaseProperties{PageBuffers: new(321)})
		require.Equal(t, 321, queryInt(t, "select MON$PAGE_BUFFERS from MON$DATABASE"))
		set(t, fbservice.DatabaseProperties{PageBuffers: new(0)})
		require.Regexp(t, `Page buffers\s+0`, header(t))
	})
	t.Run("sweep interval", func(t *testing.T) {
		for _, interval := range []int{1234, 0, 20000} {
			set(t, fbservice.DatabaseProperties{SweepInterval: &interval})
			require.Equal(t, interval, queryInt(t, "select MON$SWEEP_INTERVAL from MON$DATABASE"))
		}
	})
	t.Run("SQL dialect", func(t *testing.T) {
		for _, dialect := range []int{1, 3} {
			set(t, fbservice.DatabaseProperties{SQLDialect: dialect})
			require.Equal(t, dialect, queryInt(t, "select MON$SQL_DIALECT from MON$DATABASE"))
		}
	})
	t.Run("reserve space", func(t *testing.T) {
		for _, reserve := range []bool{false, true} {
			set(t, fbservice.DatabaseProperties{ReserveSpace: &reserve})
			want := 0
			if reserve {
				want = 1
			}
			require.Equal(t, want, queryInt(t, "select MON$RESERVE_SPACE from MON$DATABASE"))
		}
	})
	t.Run("shutdown and online", func(t *testing.T) {
		checkMode := func(t *testing.T, mode fbservice.DatabaseMode) {
			t.Helper()
			if mode == fbservice.DatabaseModeFull {
				// Some servers attach even for gstat header output. Full shutdown
				// must be checked without requiring that attachment to succeed.
				c, err := fbx.ConnectConfig(ctx, config)
				if c != nil {
					require.NoError(t, c.Close(ctx))
				}
				var serverErr *fbconn.FbError
				require.ErrorAs(t, err, &serverErr)
				require.Contains(t, serverErr.GdsCodes, fbproto.Isc_shutdown)
				return
			}
			report := header(t)
			switch mode {
			case fbservice.DatabaseModeMulti:
				require.Contains(t, report, "multi-user maintenance")
			case fbservice.DatabaseModeSingle:
				require.Contains(t, report, "single-user maintenance")
			case fbservice.DatabaseModeNormal:
				require.NotContains(t, report, "multi-user maintenance")
				require.NotContains(t, report, "single-user maintenance")
				require.NotContains(t, report, "full shutdown")
			}
			c, err := fbx.ConnectConfig(ctx, config)
			require.NoError(t, err)
			defer func() { require.NoError(t, c.Close(ctx)) }()
			second, err := fbx.ConnectConfig(ctx, config)
			if second != nil {
				require.NoError(t, second.Close(ctx))
			}
			if mode == fbservice.DatabaseModeSingle {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
		for _, mode := range []fbservice.DatabaseMode{fbservice.DatabaseModeMulti, fbservice.DatabaseModeSingle, fbservice.DatabaseModeFull} {
			for _, method := range []fbservice.ShutdownMethod{fbservice.ShutdownForce, fbservice.ShutdownDenyAttachments, fbservice.ShutdownDenyTransactions} {
				t.Run(fmt.Sprintf("shutdown mode=%d method=%d", mode, method), func(t *testing.T) {
					t.Cleanup(func() {
						set(t, fbservice.DatabaseProperties{OnlineMode: fbservice.DatabaseModeNormal})
						checkMode(t, fbservice.DatabaseModeNormal)
					})
					shutdown(t, fbservice.ShutdownOptions{Mode: mode, Method: method, Timeout: 1})
					checkMode(t, mode)
				})
			}
		}
		for _, mode := range []fbservice.DatabaseMode{fbservice.DatabaseModeNormal, fbservice.DatabaseModeMulti, fbservice.DatabaseModeSingle} {
			t.Run(fmt.Sprintf("online mode=%d", mode), func(t *testing.T) {
				t.Cleanup(func() {
					if currentMode != fbservice.DatabaseModeNormal {
						set(t, fbservice.DatabaseProperties{OnlineMode: fbservice.DatabaseModeNormal})
					}
				})
				shutdown(t, fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce})
				set(t, fbservice.DatabaseProperties{OnlineMode: mode})
				checkMode(t, mode)
			})
		}
		require.Equal(t, 1, queryInt(t, "select count(*) from MAINT_DATA"))
	})
	t.Run("sweep advances oldest transaction", func(t *testing.T) {
		for name, options := range map[string]fbservice.SweepOptions{
			"defaults":         {},
			"timeout":          {Timeout: 2},
			"ignore checksums": {IgnoreChecksums: true},
		} {
			t.Run(name, func(t *testing.T) {
				if options.Timeout > 0 {
					requireMaintenanceFeature(t, server, caps.sweepTimeout, "sweep timeout")
				}
				c, err := fbx.ConnectConfig(ctx, config)
				require.NoError(t, err)
				tx, err := c.Begin(ctx)
				require.NoError(t, err)
				_, err = tx.Exec(ctx, "insert into MAINT_DATA values (999)")
				require.NoError(t, err)
				require.NoError(t, tx.Rollback(ctx))
				require.NoError(t, c.Close(ctx))
				oldest := func() int64 {
					match := regexp.MustCompile(`Oldest transaction\s+(\d+)`).FindStringSubmatch(header(t))
					require.Len(t, match, 2)
					n, err := strconv.ParseInt(match[1], 10, 64)
					require.NoError(t, err)
					return n
				}
				before := oldest()
				require.NoError(t, manager.Sweep(ctx, config.Database, options))
				require.Greater(t, oldest(), before)
				require.Equal(t, 0, queryInt(t, "select count(*) from MAINT_DATA where ID=999"))
			})
		}
	})
	t.Run("repair", func(t *testing.T) {
		for name, options := range map[string]fbservice.RepairOptions{
			"validate":                  {Operation: fbservice.RepairValidate},
			"validate full":             {Operation: fbservice.RepairValidate, Full: true},
			"validate read only":        {Operation: fbservice.RepairValidate, ReadOnly: true},
			"validate ignore checksums": {Operation: fbservice.RepairValidate, IgnoreChecksums: true},
			"validate all flags":        {Operation: fbservice.RepairValidate, Full: true, ReadOnly: true, IgnoreChecksums: true},
			"mend":                      {Operation: fbservice.RepairMend},
			"mend ignore checksums":     {Operation: fbservice.RepairMend, IgnoreChecksums: true},
			"kill shadows":              {Operation: fbservice.RepairKillShadows},
		} {
			t.Run(name, func(t *testing.T) {
				var out bytes.Buffer
				options.Output = &out
				var err error
				if options.Operation == fbservice.RepairValidate {
					err = manager.Validate(ctx, config.Database, fbservice.ValidateOptions{
						Full: options.Full, ReadOnly: options.ReadOnly, IgnoreChecksums: options.IgnoreChecksums, Output: &out,
					})
				} else {
					err = manager.Repair(ctx, config.Database, options)
				}
				require.NoError(t, err, "operation %d: %s", options.Operation, out.String())
				require.Equal(t, 1, queryInt(t, "select count(*) from MAINT_DATA"))
			})
		}
	})
	t.Run("mend corrupt file", func(t *testing.T) {
		source := os.Getenv("FBX_TEST_MAINTENANCE_CORRUPT_SOURCE")
		if source == "" {
			t.Skip("requires FBX_TEST_MAINTENANCE_CORRUPT_SOURCE: an offline database with mend-recoverable record damage")
		}
		damaged := filepath.Join(directory, "damaged.fdb")
		copyMaintenanceDatabase(t, source, damaged)
		var report bytes.Buffer
		err := manager.Repair(ctx, damaged, fbservice.RepairOptions{
			Operation: fbservice.RepairValidate, Full: true, ReadOnly: true, Output: &report,
		})
		if err != nil {
			var serverErr *fbconn.FbError
			require.ErrorAs(t, err, &serverErr)
		}
		require.True(t, err != nil || report.Len() > 0, "the corrupt fixture must produce validation diagnostics")
		var mendReport bytes.Buffer
		require.NoError(t, manager.Repair(ctx, damaged, fbservice.RepairOptions{
			Operation: fbservice.RepairMend, IgnoreChecksums: true, Output: &mendReport,
		}), "%s", mendReport.String())
		backup := filepath.Join(directory, "mended.fbk")
		require.NoError(t, manager.Backup(ctx, damaged, backup, fbservice.BackupOptions{IgnoreChecksums: true}))
		info, err := os.Stat(backup)
		require.NoError(t, err)
		require.Positive(t, info.Size(), "mend must allow the remaining records to be backed up")
	})
	t.Run("ICU repair", func(t *testing.T) {
		requireMaintenanceFeature(t, server, caps.icu, "ICU repair")
		for _, workers := range []int{0, 2} {
			t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
				if workers > 0 {
					requireMaintenanceFeature(t, server, caps.parallelICU, "parallel ICU repair")
				}
				require.NoError(t, manager.Repair(ctx, config.Database, fbservice.RepairOptions{Operation: fbservice.RepairFixICU, ParallelWorkers: workers}))
				require.Equal(t, 1, queryInt(t, "select count(*) from MAINT_ICU where V='hello' plan (MAINT_ICU index (MAINT_ICU_IDX))"))
			})
		}
	})
	t.Run("ODS upgrade", func(t *testing.T) {
		requireMaintenanceFeature(t, server, caps.upgradeODS, "ODS upgrade")
		before := queryInt(t, "select MON$ODS_MAJOR * 100 + MON$ODS_MINOR from MON$DATABASE")
		err := manager.Repair(ctx, config.Database, fbservice.RepairOptions{Operation: fbservice.RepairUpgradeODS})
		require.NoError(t, err)
		require.GreaterOrEqual(t, queryInt(t, "select MON$ODS_MAJOR * 100 + MON$ODS_MINOR from MON$DATABASE"), before)
	})
	t.Run("ODS upgrade old file", func(t *testing.T) {
		requireMaintenanceFeature(t, server, caps.upgradeODS, "ODS upgrade")
		source := os.Getenv("FBX_TEST_MAINTENANCE_ODS_SOURCE")
		if source == "" {
			t.Skip("requires FBX_TEST_MAINTENANCE_ODS_SOURCE: an offline database with an older minor ODS supported by this server")
		}
		targetODS := queryInt(t, "select MON$ODS_MAJOR * 100 + MON$ODS_MINOR from MON$DATABASE")
		oldConfig := config.Copy()
		oldConfig.Database = filepath.Join(directory, "old-ods.fdb")
		copyMaintenanceDatabase(t, source, oldConfig.Database)
		read := func() (int, int) {
			c, err := fbx.ConnectConfig(ctx, oldConfig)
			require.NoError(t, err)
			defer func() { require.NoError(t, c.Close(ctx)) }()
			var ods, relations int
			require.NoError(t, c.QueryRow(ctx, "select MON$ODS_MAJOR * 100 + MON$ODS_MINOR from MON$DATABASE").Scan(&ods))
			require.NoError(t, c.QueryRow(ctx, "select count(*) from RDB$RELATIONS").Scan(&relations))
			return ods, relations
		}
		beforeODS, beforeRelations := read()
		require.Equal(t, targetODS/100, beforeODS/100, "source must use the supported major ODS")
		require.Less(t, beforeODS, targetODS, "source must actually require a minor ODS upgrade")
		require.NoError(t, manager.Repair(ctx, oldConfig.Database, fbservice.RepairOptions{Operation: fbservice.RepairUpgradeODS}))
		afterODS, afterRelations := read()
		require.Equal(t, targetODS, afterODS)
		require.Equal(t, beforeRelations, afterRelations)
		require.NoError(t, fbx.DropDatabaseConfig(ctx, oldConfig))
	})
	t.Run("limbo commit and rollback", func(t *testing.T) {
		ids, err := manager.LimboTransactions(ctx, config.Database)
		require.NoError(t, err)
		require.Empty(t, ids)
		details, err := manager.LimboTransactionDetails(ctx, config.Database)
		require.NoError(t, err)
		require.Empty(t, details)
		for _, action := range []fbservice.LimboTransactionAction{fbservice.LimboCommit, fbservice.LimboRollback} {
			id := prepareMaintenanceTransaction(t, ctx, config.Config, config.Database)
			ids, err = manager.LimboTransactions(ctx, config.Database)
			require.NoError(t, err)
			require.Equal(t, []int64{id}, ids)
			details, err = manager.LimboTransactionDetails(ctx, config.Database)
			require.NoError(t, err)
			require.Equal(t, []fbservice.LimboTransaction{{ID: id}}, details)
			result, err := manager.ResolveLimboTransactionChecked(ctx, config.Database, id, action, nil)
			require.NoError(t, err)
			require.Equal(t, fbservice.LimboResolutionNotListed, result.Status)
			require.Nil(t, result.Transaction)
			ids, err = manager.LimboTransactions(ctx, config.Database)
			require.NoError(t, err)
			require.Empty(t, ids)
			details, err = manager.LimboTransactionDetails(ctx, config.Database)
			require.NoError(t, err)
			require.Empty(t, details)
			want := 0
			if action == fbservice.LimboCommit {
				want = 1
			}
			require.Equal(t, want, queryInt(t, fmt.Sprintf("select count(*) from MAINT_DATA where ID=%d", id)))
		}
		require.Equal(t, 2, queryInt(t, "select count(*) from MAINT_DATA"))
	})
	t.Run("limbo recovery without a distributed decision", func(t *testing.T) {
		id := prepareMaintenanceTransaction(t, ctx, config.Config, config.Database)
		t.Cleanup(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.NoError(t, manager.ResolveLimboTransaction(cleanup, config.Database, id, fbservice.LimboRollback, nil))
		})
		var output bytes.Buffer
		result, err := manager.ResolveLimboTransactionChecked(ctx, config.Database, id, fbservice.LimboRecover, &output)
		require.Equal(t, fbservice.LimboResolutionUnknown, result.Status)
		require.Nil(t, result.Transaction)
		var serverErr *fbconn.FbError
		require.ErrorAs(t, err, &serverErr, output.String())
		// gfix can report the transaction prompt (message 91, no named constant)
		// before the unexpected-end-of-input error: Services API cannot answer it.
		require.True(t, slices.Contains(serverErr.GdsCodes, 335741019) ||
			slices.Contains(serverErr.GdsCodes, fbproto.Isc_gfix_unexp_eoi), "unexpected recovery error: %v", err)
		ids, err := manager.LimboTransactions(ctx, config.Database)
		require.NoError(t, err)
		require.Equal(t, []int64{id}, ids, "recovery must not guess commit or rollback without a decision")
	})
	t.Run("distributed limbo recovery", func(t *testing.T) {
		if os.Getenv("FBX_TEST_MAINTENANCE_DISTRIBUTED") != "1" {
			t.Skip("requires FBX_TEST_MAINTENANCE_DISTRIBUTED=1 and server authentication permitting gfix to reattach both participants")
		}
		for _, commit := range []bool{true, false} {
			t.Run(fmt.Sprintf("commit=%t", commit), func(t *testing.T) {
				before := queryInt(t, "select count(*) from MAINT_DATA")
				id := prepareMaintenanceDistributedTransaction(t, ctx, config.Config, commit)
				ids, err := manager.LimboTransactions(ctx, config.Database)
				require.NoError(t, err)
				require.Equal(t, []int64{id}, ids)
				details, err := manager.LimboTransactionDetails(ctx, config.Database)
				require.NoError(t, err)
				require.Len(t, details, 1)
				require.Equal(t, id, details[0].ID)
				require.True(t, details[0].Distributed)
				require.Len(t, details[0].Participants, 2)
				wantAdvice := fbservice.LimboAdviceRollback
				wantState := fbservice.LimboStateRolledBack
				if commit {
					wantAdvice, wantState = fbservice.LimboAdviceCommit, fbservice.LimboStateCommitted
				}
				require.Equal(t, wantAdvice, details[0].Advice)
				states := make([]fbservice.LimboTransactionState, 0, 2)
				for _, participant := range details[0].Participants {
					require.Positive(t, participant.ID)
					require.NotEmpty(t, participant.DatabasePath)
					states = append(states, participant.State)
				}
				// A completed participant may no longer have an RDB$TRANSACTIONS
				// record. The service then omits its state instead of confirming it.
				if slices.Contains(states, fbservice.LimboStateUnknown) {
					wantState = fbservice.LimboStateUnknown
				}
				require.ElementsMatch(t, []fbservice.LimboTransactionState{fbservice.LimboStatePrepared, wantState}, states)
				var output bytes.Buffer
				result, err := manager.ResolveLimboTransactionChecked(ctx, config.Database, id, fbservice.LimboRecover, &output)
				require.NoError(t, err, output.String())
				require.Equal(t, fbservice.LimboResolutionNotListed, result.Status)
				require.Nil(t, result.Transaction)
				ids, err = manager.LimboTransactions(ctx, config.Database)
				require.NoError(t, err)
				require.Empty(t, ids, output.String())
				want := before
				if commit {
					want++
				}
				require.Equal(t, want, queryInt(t, "select count(*) from MAINT_DATA"))
			})
		}
	})
	t.Run("shadows", func(t *testing.T) {
		before := queryInt(t, "select count(*) from MAINT_DATA")
		shadow := filepath.Join(directory, "shadow.fdb")
		c, err := fbx.ConnectConfig(ctx, config)
		require.NoError(t, err)
		_, err = c.Exec(ctx, "create shadow 1 manual '"+strings.ReplaceAll(shadow, "'", "''")+"'")
		require.NoError(t, err)
		require.NoError(t, c.Close(ctx))
		// The fixture is on a shared local filesystem, as for other service tests.
		saved := filepath.Join(directory, "saved-shadow.fdb")
		require.NoError(t, os.Rename(shadow, saved))
		t.Cleanup(func() {
			if _, err := os.Stat(saved); err == nil {
				require.NoError(t, os.Rename(saved, shadow))
			}
		})
		killErr := manager.Repair(ctx, config.Database, fbservice.RepairOptions{Operation: fbservice.RepairKillShadows})
		if killErr != nil {
			var serverErr *fbconn.FbError
			require.ErrorAs(t, killErr, &serverErr)
			require.Contains(t, serverErr.GdsCodes, fbproto.Isc_io_error)
			t.Logf("kill unavailable shadow reported: %v", killErr)
		}
		require.Equal(t, 0, queryInt(t, "select count(*) from RDB$FILES where RDB$SHADOW_NUMBER=1"))
		// Change only the primary after disconnecting the shadow. The recovered
		// database must retain the shadow's data, not a cached primary attachment.
		c, err = fbx.ConnectConfig(ctx, config)
		require.NoError(t, err)
		_, err = c.Exec(ctx, "insert into MAINT_DATA values (-1)")
		require.NoError(t, err)
		require.NoError(t, c.Close(ctx))
		require.Equal(t, before+1, queryInt(t, "select count(*) from MAINT_DATA"))
		set(t, fbservice.DatabaseProperties{Shutdown: &fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce}})
		// Restore a copy at the primary path while retaining the shadow. On 3.0
		// authentication opens the database before activation; the shadow reference
		// must still resolve to a separate file during that attachment.
		primary := filepath.Join(directory, "saved-primary.fdb")
		require.NoError(t, os.Rename(config.Database, primary))
		require.NoError(t, os.Rename(saved, shadow))
		copyMaintenanceDatabase(t, shadow, config.Database)
		currentMode = fbservice.DatabaseModeNormal // The detached shadow predates shutdown.
		require.Contains(t, header(t), "active shadow")
		require.NoError(t, manager.SetDatabaseProperties(ctx, config.Database, fbservice.DatabaseProperties{ActivateShadow: true}))
		shadowConn, err := fbx.ConnectConfig(ctx, config)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, shadowConn.Close(ctx)) })
		var count int
		require.NoError(t, shadowConn.QueryRow(ctx, "select count(*) from MAINT_DATA").Scan(&count))
		require.Equal(t, before, count)
		require.NoError(t, shadowConn.Close(ctx))
		require.NotContains(t, header(t), "active shadow")
		// The outer cleanup shuts down and drops the promoted database; the
		// original primary is an offline file removed with the temporary directory.
	})
}

// Only the temporary copy is attached or changed; the source must be offline.
func copyMaintenanceDatabase(t *testing.T, source, destination string) {
	t.Helper()
	in, err := os.Open(source)
	require.NoError(t, err)
	defer func() { require.NoError(t, in.Close()) }()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	require.NoError(t, err)
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	require.NoError(t, copyErr)
	require.NoError(t, closeErr)
	require.NoError(t, os.Chmod(destination, 0o666))
}

// Preparing transactions is not part of the maintenance API. Build the test
// fixture with the existing low-level protocol client using op_prepare (32).
type maintenancePrepareTransaction struct {
	handle      int
	description []byte
}

func (p maintenancePrepareTransaction) Write(w *fbproto.XDRWriter) error {
	opcode := 32
	if p.description != nil {
		opcode = 51 // op_prepare2 includes the distributed transaction description.
	}
	if err := w.WriteInt32(opcode); err != nil {
		return err
	}
	if err := w.WriteInt32(p.handle); err != nil {
		return err
	}
	if p.description != nil {
		return w.WriteBuffer(p.description)
	}
	return nil
}

func prepareMaintenanceTransaction(t *testing.T, ctx context.Context, config fbconn.Config, database string) int64 {
	t.Helper()
	config.Database = database
	c, err := fbconn.ConnectConfig(ctx, &config)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, c.Close(cleanup))
	})
	tx, err := c.StartTransaction(ctx)
	require.NoError(t, err)
	result := c.ExecTx(ctx, tx, "select cast(current_transaction as varchar(30)) from RDB$DATABASE").Read()
	require.NoError(t, result.Err)
	id, err := strconv.ParseInt(strings.TrimSpace(string(result.Rows[0][0])), 10, 64)
	require.NoError(t, err)
	result = c.ExecTx(ctx, tx, fmt.Sprintf("insert into MAINT_DATA values (%d)", id)).Read()
	require.NoError(t, result.Err)
	require.NoError(t, c.Client().WriteMessage(maintenancePrepareTransaction{handle: tx.Handle}))
	require.NoError(t, c.Client().Flush())
	message, err := c.ReceiveMessage(ctx)
	require.NoError(t, err)
	require.IsType(t, &fbproto.OpResponse{}, message)
	require.NoError(t, c.Close(ctx))
	return id
}
