package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

func TestManagerNBackupWithoutOptions(t *testing.T) {
	manager, conn := newTestManager(t)
	require.NoError(t, manager.NBackup(context.Background(), "db", "full.nbk"))
	require.True(t, conn.closed)
	require.Equal(t, "db", conn.expectedDatabase)
	require.Equal(t, "db", conn.database)
	require.Equal(t, "full.nbk", conn.backupFile)
	require.Equal(t, fbconn.NBackupOptions{}, conn.nbackupOptions)
}

func TestManagerNBackupMapsOptions(t *testing.T) {
	manager, conn := newTestManager(t)
	var output bytes.Buffer
	const guid = "{01234567-89AB-CDEF-0123-456789ABCDEF}"
	options := fbservice.NBackupOptions{
		Output: &output, GUID: guid, NoDatabaseTriggers: true,
		DirectIO: new(false), CleanHistory: true, KeepDays: 7,
	}
	require.NoError(t, manager.NBackup(context.Background(), "db", "file", options))
	require.Equal(t, fbconn.NBackupOptions{
		Output: &output, GUID: guid, NoDatabaseTriggers: true,
		DirectIO: options.DirectIO, CleanHistory: true, KeepDays: 7,
	}, conn.nbackupOptions)
	require.NoError(t, manager.NBackup(context.Background(), "db", "file", fbservice.NBackupOptions{Level: 2, CleanHistory: true, KeepRows: 3}))
	require.Equal(t, fbconn.NBackupOptions{Level: 2, CleanHistory: true, KeepRows: 3}, conn.nbackupOptions)
}

func TestManagerNRestoreAndNFixupMapOptions(t *testing.T) {
	manager, conn := newTestManager(t)
	var output bytes.Buffer
	require.NoError(t, manager.NRestore(context.Background(), []string{"a", "b"}, "db", fbservice.NRestoreOptions{
		Output: &output, InPlace: true, NoDatabaseTriggers: true, PreserveSequence: true,
	}))
	require.Equal(t, fbconn.NRestoreOptions{Output: &output, InPlace: true, NoDatabaseTriggers: true, PreserveSequence: true}, conn.nrestoreOptions)
	require.Equal(t, []string{"a", "b"}, conn.nrestoreFiles)
	require.NoError(t, manager.NFixup(context.Background(), "db", fbservice.NFixupOptions{Output: &output, PreserveSequence: true}))
	require.Equal(t, fbconn.NFixupOptions{Output: &output, PreserveSequence: true}, conn.nfixupOptions)
}

func TestManagerNBackupValidationBeforeConnect(t *testing.T) {
	invalid := []fbservice.NBackupOptions{
		{Level: -1}, {Level: 32768}, {GUID: "bad"},
		{GUID: "{01234567-89AB-CDEF-0123-456789ABCDEF}", Level: 1},
		{CleanHistory: true}, {KeepDays: 7}, {KeepRows: 1},
		{CleanHistory: true, KeepDays: 7, KeepRows: 1},
		{CleanHistory: true, KeepDays: -1}, {CleanHistory: true, KeepRows: -1},
	}
	if strconv.IntSize > 32 {
		overflow := int64(1 << 31)
		invalid = append(invalid, fbservice.NBackupOptions{CleanHistory: true, KeepDays: int(overflow)}, fbservice.NBackupOptions{CleanHistory: true, KeepRows: int(overflow)})
	}
	for _, option := range invalid {
		manager, conn := newTestManager(t)
		require.Error(t, manager.NBackup(context.Background(), "db", "b", option))
		require.False(t, conn.opened)
	}
	for _, pair := range [][2]string{
		{"", "b"}, {"db", ""}, {"db\x00suffix", "b"}, {"db", "b\x00suffix"},
		{strings.Repeat("d", 65536), "b"}, {"db", strings.Repeat("я", 32768)}, {"db", "stdout"},
	} {
		manager, conn := newTestManager(t)
		require.Error(t, manager.NBackup(context.Background(), pair[0], pair[1]))
		require.False(t, conn.opened)
	}
}

func TestManagerNRestoreAndNFixupValidationBeforeConnect(t *testing.T) {
	for _, database := range []string{"", "db\x00suffix", strings.Repeat("d", 65536)} {
		manager, conn := newTestManager(t)
		require.Error(t, manager.NRestore(context.Background(), []string{"a"}, database))
		require.Error(t, manager.NFixup(context.Background(), database))
		require.False(t, conn.opened)
	}
	for _, files := range [][]string{nil, {}, {""}, {"a", ""}, {"a\x00b"}, {"stdin"}, {strings.Repeat("f", 65536)}, {strings.Repeat("f", 32763), strings.Repeat("f", 32763)}} {
		manager, conn := newTestManager(t)
		require.Error(t, manager.NRestore(context.Background(), files, "db"))
		require.False(t, conn.opened)
	}
	manager, conn := newTestManager(t)
	require.ErrorContains(t, manager.NBackup(context.Background(), "db", "b", fbservice.NBackupOptions{}, fbservice.NBackupOptions{}), "at most one")
	require.ErrorContains(t, manager.NRestore(context.Background(), []string{"b"}, "db", fbservice.NRestoreOptions{}, fbservice.NRestoreOptions{}), "at most one")
	require.ErrorContains(t, manager.NFixup(context.Background(), "db", fbservice.NFixupOptions{}, fbservice.NFixupOptions{}), "at most one")
	require.False(t, conn.opened)
}

func nbackupManagerCalls(manager *fbservice.Manager) map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"backup":  func(ctx context.Context) error { return manager.NBackup(ctx, "db", "b") },
		"restore": func(ctx context.Context) error { return manager.NRestore(ctx, []string{"a", "b"}, "db") },
		"fixup":   func(ctx context.Context) error { return manager.NFixup(ctx, "db") },
	}
}

func TestManagerNBackupErrorsAndCleanup(t *testing.T) {
	for name := range nbackupManagerCalls(nil) {
		t.Run(name, func(t *testing.T) {
			operationErr, closeErr := errors.New("operation failed"), errors.New("close failed")
			manager, conn := newTestManager(t)
			conn.backupErr, conn.restoreErr, conn.maintenanceErr, conn.closeErr = operationErr, operationErr, operationErr, closeErr
			err := nbackupManagerCalls(manager)[name](context.Background())
			require.ErrorIs(t, err, operationErr)
			require.ErrorIs(t, err, closeErr)
			require.True(t, conn.closed)

			manager, conn = newTestManager(t)
			connectErr := errors.New("connect failed")
			fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
				return nil, connectErr
			})
			require.ErrorIs(t, nbackupManagerCalls(manager)[name](context.Background()), connectErr)
			require.False(t, conn.closed)
		})
	}
}

func TestManagerNBackupSPBLengthBoundaries(t *testing.T) {
	const guid = "{01234567-89AB-CDEF-0123-456789ABCDEF}"
	for _, tt := range []struct {
		options fbservice.NBackupOptions
		maxFile int
	}{
		{fbservice.NBackupOptions{}, 65521},
		{fbservice.NBackupOptions{NoDatabaseTriggers: true}, 65516},
		{fbservice.NBackupOptions{DirectIO: new(true)}, 65516},
		{fbservice.NBackupOptions{DirectIO: new(false)}, 65515},
		{fbservice.NBackupOptions{CleanHistory: true, KeepRows: 1}, 65515},
		{fbservice.NBackupOptions{GUID: guid}, 65485},
		{fbservice.NBackupOptions{GUID: guid, NoDatabaseTriggers: true, DirectIO: new(false), CleanHistory: true, KeepDays: 1}, 65468},
	} {
		manager, conn := newTestManager(t)
		require.ErrorContains(t, manager.NBackup(context.Background(), "db", strings.Repeat("f", tt.maxFile+1), tt.options), "action SPB")
		require.False(t, conn.opened)
		require.NoError(t, manager.NBackup(context.Background(), "db", strings.Repeat("f", tt.maxFile), tt.options))
	}
	for _, tt := range []struct {
		options fbservice.NRestoreOptions
		maxFile int
	}{
		{fbservice.NRestoreOptions{}, 65526},
		{fbservice.NRestoreOptions{InPlace: true}, 65521},
		{fbservice.NRestoreOptions{NoDatabaseTriggers: true, InPlace: true, PreserveSequence: true}, 65521},
	} {
		manager, conn := newTestManager(t)
		require.ErrorContains(t, manager.NRestore(context.Background(), []string{strings.Repeat("f", tt.maxFile+1)}, "db", tt.options), "action SPB")
		require.False(t, conn.opened)
		require.NoError(t, manager.NRestore(context.Background(), []string{strings.Repeat("f", tt.maxFile)}, "db", tt.options))
	}
	for _, preserve := range []bool{false, true} {
		manager, conn := newTestManager(t)
		maxDatabase := 65531
		if preserve {
			maxDatabase = 65526
		}
		require.ErrorContains(t, manager.NFixup(context.Background(), strings.Repeat("d", maxDatabase+1), fbservice.NFixupOptions{PreserveSequence: preserve}), "action SPB")
		require.False(t, conn.opened)
		require.NoError(t, manager.NFixup(context.Background(), strings.Repeat("d", maxDatabase), fbservice.NFixupOptions{PreserveSequence: preserve}))
	}
}

func TestManagerNBackupCancellationUsesIndependentCleanup(t *testing.T) {
	for name := range nbackupManagerCalls(nil) {
		t.Run(name, func(t *testing.T) {
			manager, _ := newTestManager(t)
			conn := new(traceCleanupConnection)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conn.run = func(got context.Context) error {
				require.Same(t, ctx, got)
				cancel()
				return ctx.Err()
			}
			fbservice.SetConnectForTest(manager, func(got context.Context, config *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
				require.Same(t, ctx, got)
				require.Equal(t, "db", config.ExpectedDatabase)
				return conn, nil
			})
			require.ErrorIs(t, nbackupManagerCalls(manager)[name](ctx), context.Canceled)
			require.True(t, conn.closed)
			require.NoError(t, conn.closeContextErr)
			require.True(t, conn.closeDeadline)
		})
	}
}

func TestManagerNBackupCanceledBeforeConnect(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelDeadline := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancelDeadline()
	for name := range nbackupManagerCalls(nil) {
		for _, ctx := range []context.Context{canceled, expired} {
			t.Run(name+"/"+ctx.Err().Error(), func(t *testing.T) {
				manager, conn := newTestManager(t)
				require.ErrorIs(t, nbackupManagerCalls(manager)[name](ctx), ctx.Err())
				require.False(t, conn.opened)
				require.False(t, conn.closed)
			})
		}
	}
}

func TestManagerNRestoreAndNFixupWithoutOptions(t *testing.T) {
	t.Run("restore", func(t *testing.T) {
		manager, conn := newTestManager(t)
		require.NoError(t, manager.NRestore(context.Background(), []string{"full.nbk", "inc.nbk"}, "db"))
		require.True(t, conn.closed)
		require.Equal(t, "db", conn.expectedDatabase)
		require.Equal(t, "db", conn.database)
		require.Equal(t, []string{"full.nbk", "inc.nbk"}, conn.nrestoreFiles)
		require.Equal(t, fbconn.NRestoreOptions{}, conn.nrestoreOptions)
	})
	t.Run("fixup", func(t *testing.T) {
		manager, conn := newTestManager(t)
		require.NoError(t, manager.NFixup(context.Background(), "db"))
		require.True(t, conn.closed)
		require.Equal(t, "db", conn.expectedDatabase)
		require.Equal(t, "db", conn.database)
		require.Equal(t, fbconn.NFixupOptions{}, conn.nfixupOptions)
	})
}

func (c *fakeServiceConnection) NBackup(ctx context.Context, database, backupFile string, options fbconn.NBackupOptions) error {
	c.database, c.backupFile, c.nbackupOptions = database, backupFile, options
	if c.run != nil {
		return c.run(ctx)
	}
	return c.backupErr
}

func (c *concurrentServiceConnection) NBackup(_ context.Context, database, _ string, _ fbconn.NBackupOptions) error {
	return c.recordOperation(database)
}

func (c *fakeServiceConnection) NRestore(ctx context.Context, files []string, database string, options fbconn.NRestoreOptions) error {
	c.database, c.nrestoreFiles, c.nrestoreOptions = database, files, options
	if c.run != nil {
		return c.run(ctx)
	}
	return c.restoreErr
}

func (c *fakeServiceConnection) NFixup(ctx context.Context, database string, options fbconn.NFixupOptions) error {
	c.database, c.nfixupOptions = database, options
	if c.run != nil {
		return c.run(ctx)
	}
	return c.maintenanceErr
}

func (c *concurrentServiceConnection) NRestore(_ context.Context, _ []string, database string, _ fbconn.NRestoreOptions) error {
	return c.recordOperation(database)
}

func (c *concurrentServiceConnection) NFixup(_ context.Context, database string, _ fbconn.NFixupOptions) error {
	return c.recordOperation(database)
}
