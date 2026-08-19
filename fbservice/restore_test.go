package fbservice

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
)

func TestManagerRestoreWithoutOptions(t *testing.T) {
	manager, connection := newTestManager(t)

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
	)

	require.NoError(t, err)
	require.True(t, connection.closed)
	require.Equal(t, "/data/employee.fdb", connection.expectedDatabase)
	require.Equal(t, "/backup/employee.fbk", connection.restoreBackupFile)
	require.Equal(t, "/data/employee.fdb", connection.restoreDatabase)
	require.Equal(t, fbconn.RestoreOptions{}, connection.restoreOptions)
}

func TestManagerRestoreMapsOptions(t *testing.T) {
	manager, connection := newTestManager(t)
	var output bytes.Buffer

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{
			Output:                  &output,
			Verbose:                 true,
			Replace:                 true,
			AccessMode:              RestoreAccessModeReadOnly,
			DeactivateIndexes:       true,
			SkipShadows:             true,
			SkipValidityConstraints: true,
			CommitAfterEachTable:    true,
			UseAllPageSpace:         true,
			PageBuffers:             512,
			PageSize:                16384,
			ParallelWorkers:         4,
		},
	)

	require.NoError(t, err)
	require.Equal(t, fbconn.RestoreOptions{
		Output:                  &output,
		Verbose:                 true,
		Replace:                 true,
		AccessMode:              fbconn.RestoreAccessModeReadOnly,
		DeactivateIndexes:       true,
		SkipShadows:             true,
		SkipValidityConstraints: true,
		CommitAfterEachTable:    true,
		UseAllPageSpace:         true,
		PageBuffers:             512,
		PageSize:                16384,
		ParallelWorkers:         4,
	}, connection.restoreOptions)
}

func TestManagerRestoreMapsAccessMode(t *testing.T) {
	tests := map[string]struct {
		accessMode RestoreAccessMode
		want       fbconn.RestoreAccessMode
	}{
		"from backup": {
			accessMode: RestoreAccessModeFromBackup,
			want:       fbconn.RestoreAccessModeFromBackup,
		},
		"read-only": {
			accessMode: RestoreAccessModeReadOnly,
			want:       fbconn.RestoreAccessModeReadOnly,
		},
		"read-write": {
			accessMode: RestoreAccessModeReadWrite,
			want:       fbconn.RestoreAccessModeReadWrite,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)

			err := manager.Restore(
				context.Background(),
				"/backup/employee.fbk",
				"/data/employee.fdb",
				RestoreOptions{AccessMode: tt.accessMode},
			)

			require.NoError(t, err)
			require.Equal(t, tt.want, connection.restoreOptions.AccessMode)
		})
	}
}

func TestManagerRestoreRejectsMultipleOptions(t *testing.T) {
	manager, connection := newTestManager(t)

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{},
		RestoreOptions{},
	)

	require.ErrorContains(t, err, "at most one")
	require.False(t, connection.opened)
}

func TestManagerRestoreValidatesRequestBeforeOpeningConnection(t *testing.T) {
	type testCase struct {
		backupFile string
		database   string
		options    RestoreOptions
		errText    string
	}

	tests := map[string]testCase{
		"empty backup file": {
			database: "/data/employee.fdb",
			errText:  "restore backup file must not be empty",
		},
		"empty database": {
			backupFile: "/backup/employee.fbk",
			errText:    "restore database must not be empty",
		},
		"backup file exceeds SPB limit": {
			backupFile: strings.Repeat("b", maxServiceActionStringLength+1),
			database:   "/data/employee.fdb",
			errText:    "restore backup file is 65536 bytes; maximum is 65535",
		},
		"database exceeds SPB limit": {
			backupFile: "/backup/employee.fbk",
			database:   strings.Repeat("d", maxServiceActionStringLength+1),
			errText:    "restore database is 65536 bytes; maximum is 65535",
		},
		"negative page buffers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageBuffers: -1},
			errText:    "restore page buffers must be between 0",
		},
		"invalid page size": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageSize: 12345},
			errText:    "restore page size must be one of",
		},
		"negative parallel workers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ParallelWorkers: -1},
			errText:    "restore parallel workers must be between 0",
		},
		"invalid access mode": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{AccessMode: RestoreAccessMode(255)},
			errText:    "invalid restore access mode 255",
		},
	}
	if strconv.IntSize > 32 {
		aboveMaxInt32 := int64(maxServiceActionInt32) + 1
		tests["page buffers exceed int32"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageBuffers: int(aboveMaxInt32)},
			errText:    "restore page buffers must be between 0",
		}
		tests["parallel workers exceed int32"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ParallelWorkers: int(aboveMaxInt32)},
			errText:    "restore parallel workers must be between 0",
		}
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)

			err := manager.Restore(
				context.Background(),
				tt.backupFile,
				tt.database,
				tt.options,
			)

			require.ErrorContains(t, err, tt.errText)
			require.False(t, connection.opened)
		})
	}
}

func TestManagerRestoreReturnsOperationAndCloseErrors(t *testing.T) {
	manager, connection := newTestManager(t)
	connection.restoreErr = errors.New("restore failed")
	connection.closeErr = errors.New("close failed")

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
	)

	require.ErrorContains(t, err, "restore failed")
	require.ErrorContains(t, err, "close failed")
}

func TestManagerRestoreReturnsConnectionError(t *testing.T) {
	manager, _ := newTestManager(t)
	connectionErr := errors.New("connection failed")
	manager.connect = func(context.Context, *fbconn.ServiceConfig) (serviceConnection, error) {
		return nil, connectionErr
	}

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
	)

	require.ErrorIs(t, err, connectionErr)
	require.ErrorContains(t, err, "connect to service manager")
}
