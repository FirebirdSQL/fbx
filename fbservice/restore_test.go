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
	tablespacePaths := []RestoreTablespacePath{
		{Name: "TS1", Path: "/data/ts1"},
		{Name: "TS2", Path: "/data/ts2"},
	}

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{
			Output:                     &output,
			VerboseInterval:            100,
			Statistics:                 RestoreStatisticsAll,
			Replace:                    true,
			AccessMode:                 RestoreAccessModeReadOnly,
			ReplicaMode:                RestoreReplicaModeReadOnly,
			DeactivateIndexes:          true,
			SkipShadows:                true,
			SkipValidityConstraints:    true,
			CommitAfterEachTable:       true,
			UseAllPageSpace:            true,
			DirectIO:                   true,
			IgnoreBLRErrors:            true,
			FixFSSData:                 "WIN1251",
			FixFSSMetadata:             "WIN1251",
			SkipSchemaData:             "SKIP_SCHEMA",
			SkipData:                   "SKIP_%",
			IncludeSchemaData:          "KEEP_SCHEMA",
			IncludeData:                "KEEP_%",
			KeyHolder:                  "KeyHolder",
			KeyName:                    "KeyName",
			CryptPlugin:                "DbCrypt",
			TablespaceMappingFile:      "/config/tablespaces.map",
			TablespacePaths:            tablespacePaths,
			UseOriginalTablespacePaths: true,
			PageBuffers:                512,
			PageSize:                   16384,
			ParallelWorkers:            4,
		},
	)

	require.NoError(t, err)
	tablespacePaths[0].Name = "changed after restore"
	require.Equal(t, fbconn.RestoreOptions{
		Output:                     &output,
		VerboseInterval:            100,
		Statistics:                 fbconn.RestoreStatisticsAll,
		Replace:                    true,
		AccessMode:                 fbconn.RestoreAccessModeReadOnly,
		ReplicaMode:                fbconn.RestoreReplicaModeReadOnly,
		DeactivateIndexes:          true,
		SkipShadows:                true,
		SkipValidityConstraints:    true,
		CommitAfterEachTable:       true,
		UseAllPageSpace:            true,
		DirectIO:                   true,
		IgnoreBLRErrors:            true,
		FixFSSData:                 "WIN1251",
		FixFSSMetadata:             "WIN1251",
		SkipSchemaData:             "SKIP_SCHEMA",
		SkipData:                   "SKIP_%",
		IncludeSchemaData:          "KEEP_SCHEMA",
		IncludeData:                "KEEP_%",
		KeyHolder:                  "KeyHolder",
		KeyName:                    "KeyName",
		CryptPlugin:                "DbCrypt",
		TablespaceMappingFile:      "/config/tablespaces.map",
		TablespacePaths:            []fbconn.RestoreTablespacePath{{Name: "TS1", Path: "/data/ts1"}, {Name: "TS2", Path: "/data/ts2"}},
		UseOriginalTablespacePaths: true,
		PageBuffers:                512,
		PageSize:                   16384,
		ParallelWorkers:            4,
	}, connection.restoreOptions)
}

func TestManagerRestoreMapsStandaloneOptions(t *testing.T) {
	tests := map[string]struct {
		options RestoreOptions
		check   func(*testing.T, fbconn.RestoreOptions)
	}{
		"verbose": {
			options: RestoreOptions{Verbose: true},
			check: func(t *testing.T, options fbconn.RestoreOptions) {
				require.True(t, options.Verbose)
			},
		},
		"metadata only with data filters": {
			options: RestoreOptions{
				MetadataOnly:      true,
				SkipSchemaData:    "SKIP_SCHEMA",
				SkipData:          "SKIP_%",
				IncludeSchemaData: "KEEP_SCHEMA",
				IncludeData:       "KEEP_%",
			},
			check: func(t *testing.T, options fbconn.RestoreOptions) {
				require.True(t, options.MetadataOnly)
				require.Equal(t, "SKIP_SCHEMA", options.SkipSchemaData)
				require.Equal(t, "SKIP_%", options.SkipData)
				require.Equal(t, "KEEP_SCHEMA", options.IncludeSchemaData)
				require.Equal(t, "KEEP_%", options.IncludeData)
			},
		},
		"key name with key holder": {
			options: RestoreOptions{KeyHolder: "KeyHolder", KeyName: "KeyName"},
			check: func(t *testing.T, options fbconn.RestoreOptions) {
				require.Equal(t, "KeyHolder", options.KeyHolder)
				require.Equal(t, "KeyName", options.KeyName)
				require.Empty(t, options.CryptPlugin)
			},
		},
		"crypt plugin with key holder": {
			options: RestoreOptions{KeyHolder: "KeyHolder", CryptPlugin: "DbCrypt"},
			check: func(t *testing.T, options fbconn.RestoreOptions) {
				require.Equal(t, "KeyHolder", options.KeyHolder)
				require.Empty(t, options.KeyName)
				require.Equal(t, "DbCrypt", options.CryptPlugin)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)

			err := manager.Restore(
				context.Background(),
				"/backup/employee.fbk",
				"/data/employee.fdb",
				tt.options,
			)

			require.NoError(t, err)
			tt.check(t, connection.restoreOptions)
		})
	}
}

func TestManagerRestoreMapsStatistics(t *testing.T) {
	tests := map[string]struct {
		statistics RestoreStatistics
		want       fbconn.RestoreStatistics
	}{
		"none": {
			statistics: 0,
			want:       0,
		},
		"elapsed time": {
			statistics: RestoreStatisticsElapsedTime,
			want:       fbconn.RestoreStatisticsElapsedTime,
		},
		"delta time": {
			statistics: RestoreStatisticsDeltaTime,
			want:       fbconn.RestoreStatisticsDeltaTime,
		},
		"page reads": {
			statistics: RestoreStatisticsPageReads,
			want:       fbconn.RestoreStatisticsPageReads,
		},
		"page writes": {
			statistics: RestoreStatisticsPageWrites,
			want:       fbconn.RestoreStatisticsPageWrites,
		},
		"all": {
			statistics: RestoreStatisticsAll,
			want:       fbconn.RestoreStatisticsAll,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)

			err := manager.Restore(
				context.Background(),
				"/backup/employee.fbk",
				"/data/employee.fdb",
				RestoreOptions{Statistics: tt.statistics},
			)

			require.NoError(t, err)
			require.Equal(t, tt.want, connection.restoreOptions.Statistics)
		})
	}
}

func TestManagerRestoreAcceptsMaximumValues(t *testing.T) {
	manager, connection := newTestManager(t)
	exactLimit := strings.Repeat("я", 32767) + "x"
	require.Len(t, []byte(exactLimit), maxServiceActionStringLength)

	err := manager.Restore(
		context.Background(),
		exactLimit,
		exactLimit,
		RestoreOptions{
			VerboseInterval:       maxServiceActionInt32,
			Statistics:            RestoreStatisticsAll,
			FixFSSData:            exactLimit,
			FixFSSMetadata:        exactLimit,
			SkipSchemaData:        exactLimit,
			SkipData:              exactLimit,
			IncludeSchemaData:     exactLimit,
			IncludeData:           exactLimit,
			KeyHolder:             exactLimit,
			KeyName:               exactLimit,
			CryptPlugin:           exactLimit,
			TablespaceMappingFile: exactLimit,
			TablespacePaths: []RestoreTablespacePath{{
				Name: exactLimit,
				Path: exactLimit,
			}},
			PageBuffers:     maxServiceActionInt32,
			PageSize:        32768,
			ParallelWorkers: maxServiceActionInt32,
		},
	)

	require.NoError(t, err)
	require.True(t, connection.opened)
	require.Equal(t, exactLimit, connection.restoreOptions.FixFSSData)
	require.Equal(t, maxServiceActionInt32, connection.restoreOptions.VerboseInterval)
	require.Equal(t, maxServiceActionInt32, connection.restoreOptions.PageBuffers)
	require.Equal(t, maxServiceActionInt32, connection.restoreOptions.ParallelWorkers)
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

func TestManagerRestoreMapsReplicaMode(t *testing.T) {
	tests := map[string]struct {
		replicaMode RestoreReplicaMode
		want        fbconn.RestoreReplicaMode
	}{
		"from backup": {
			replicaMode: RestoreReplicaModeFromBackup,
			want:        fbconn.RestoreReplicaModeFromBackup,
		},
		"none": {
			replicaMode: RestoreReplicaModeNone,
			want:        fbconn.RestoreReplicaModeNone,
		},
		"read-only": {
			replicaMode: RestoreReplicaModeReadOnly,
			want:        fbconn.RestoreReplicaModeReadOnly,
		},
		"read-write": {
			replicaMode: RestoreReplicaModeReadWrite,
			want:        fbconn.RestoreReplicaModeReadWrite,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)

			err := manager.Restore(
				context.Background(),
				"/backup/employee.fbk",
				"/data/employee.fdb",
				RestoreOptions{ReplicaMode: tt.replicaMode},
			)

			require.NoError(t, err)
			require.Equal(t, tt.want, connection.restoreOptions.ReplicaMode)
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

func TestManagerRestoreRejectsSemanticOptionsBeforeOpeningConnection(t *testing.T) {
	tests := map[string]struct {
		options RestoreOptions
		errText string
	}{
		"verbose and interval": {
			options: RestoreOptions{Verbose: true, VerboseInterval: 100},
			errText: "restore verbose and verbose interval are mutually exclusive",
		},
		"key name without key holder": {
			options: RestoreOptions{KeyName: "KeyName"},
			errText: "restore key name requires key holder",
		},
		"crypt plugin without key holder": {
			options: RestoreOptions{CryptPlugin: "DbCrypt"},
			errText: "restore crypt plugin requires key holder",
		},
		"empty tablespace name": {
			options: RestoreOptions{TablespacePaths: []RestoreTablespacePath{{Path: "/data/ts1"}}},
			errText: "restore tablespace path 1 name must not be empty",
		},
		"empty tablespace path": {
			options: RestoreOptions{TablespacePaths: []RestoreTablespacePath{{Name: "TS1"}}},
			errText: "restore tablespace path 1 path must not be empty",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager, connection := newTestManager(t)

			err := manager.Restore(
				context.Background(),
				"/backup/employee.fbk",
				"/data/employee.fdb",
				tt.options,
			)

			require.EqualError(t, err, tt.errText)
			require.False(t, connection.opened)
		})
	}
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
		"negative verbose interval": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{VerboseInterval: -1},
			errText:    "restore verbose interval must be between 0",
		},
		"verbose interval below minimum": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{VerboseInterval: minRestoreVerboseInterval - 1},
			errText:    "restore verbose interval must be 0 or at least 100",
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
		"invalid replica mode": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ReplicaMode: RestoreReplicaMode(255)},
			errText:    "invalid restore replica mode 255",
		},
		"invalid statistics": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{Statistics: RestoreStatistics(0x10)},
			errText:    "invalid restore statistics flags 0x10",
		},
		"skip data exceeds SPB limit": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options: RestoreOptions{
				SkipData: strings.Repeat("s", maxServiceActionStringLength+1),
			},
			errText: "restore skip data is 65536 bytes; maximum is 65535",
		},
		"include data exceeds SPB limit": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options: RestoreOptions{
				IncludeData: strings.Repeat("i", maxServiceActionStringLength+1),
			},
			errText: "restore include data is 65536 bytes; maximum is 65535",
		},
	}

	tooLong := strings.Repeat("x", maxServiceActionStringLength+1)
	for _, tt := range []struct {
		name      string
		errorName string
		options   func(string) RestoreOptions
	}{
		{"fix FSS data exceeds SPB limit", "fix FSS data", func(value string) RestoreOptions { return RestoreOptions{FixFSSData: value} }},
		{"fix FSS metadata exceeds SPB limit", "fix FSS metadata", func(value string) RestoreOptions { return RestoreOptions{FixFSSMetadata: value} }},
		{"skip schema data exceeds SPB limit", "skip schema data", func(value string) RestoreOptions { return RestoreOptions{SkipSchemaData: value} }},
		{"include schema data exceeds SPB limit", "include schema data", func(value string) RestoreOptions { return RestoreOptions{IncludeSchemaData: value} }},
		{"key holder exceeds SPB limit", "key holder", func(value string) RestoreOptions { return RestoreOptions{KeyHolder: value} }},
		{"key name exceeds SPB limit", "key name", func(value string) RestoreOptions { return RestoreOptions{KeyName: value} }},
		{"crypt plugin exceeds SPB limit", "crypt plugin", func(value string) RestoreOptions {
			return RestoreOptions{KeyHolder: "KeyHolder", CryptPlugin: value}
		}},
		{"tablespace mapping file exceeds SPB limit", "tablespace mapping file", func(value string) RestoreOptions { return RestoreOptions{TablespaceMappingFile: value} }},
		{"tablespace path name exceeds SPB limit", "tablespace path 1 name", func(value string) RestoreOptions {
			return RestoreOptions{TablespacePaths: []RestoreTablespacePath{{Name: value, Path: "/data/ts1"}}}
		}},
		{"tablespace path exceeds SPB limit", "tablespace path 1 path", func(value string) RestoreOptions {
			return RestoreOptions{TablespacePaths: []RestoreTablespacePath{{Name: "TS1", Path: value}}}
		}},
	} {
		tests[tt.name] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    tt.options(tooLong),
			errText:    "restore " + tt.errorName + " is 65536 bytes; maximum is 65535",
		}
	}

	if strconv.IntSize > 32 {
		aboveMaxInt32 := int64(maxServiceActionInt32) + 1
		tests["page buffers exceed int32"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageBuffers: int(aboveMaxInt32)},
			errText:    "restore page buffers must be between 0",
		}
		tests["verbose interval exceeds int32"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{VerboseInterval: int(aboveMaxInt32)},
			errText:    "restore verbose interval must be between 0",
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
