package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
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
	tablespacePaths := []fbservice.RestoreTablespacePath{
		{Name: "TS1", Path: "/data/ts1"},
		{Name: "TS2", Path: "/data/ts2"},
	}

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
		fbservice.RestoreOptions{
			Output:                     &output,
			VerboseInterval:            100,
			Statistics:                 fbservice.RestoreStatisticsAll,
			Replace:                    true,
			AccessMode:                 fbservice.RestoreAccessModeReadOnly,
			ReplicaMode:                fbservice.RestoreReplicaModeReadOnly,
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
		options fbservice.RestoreOptions
		check   func(*testing.T, fbconn.RestoreOptions)
	}{
		"verbose": {
			options: fbservice.RestoreOptions{Verbose: true},
			check: func(t *testing.T, options fbconn.RestoreOptions) {
				require.True(t, options.Verbose)
			},
		},
		"metadata only with data filters": {
			options: fbservice.RestoreOptions{
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
			options: fbservice.RestoreOptions{KeyHolder: "KeyHolder", KeyName: "KeyName"},
			check: func(t *testing.T, options fbconn.RestoreOptions) {
				require.Equal(t, "KeyHolder", options.KeyHolder)
				require.Equal(t, "KeyName", options.KeyName)
				require.Empty(t, options.CryptPlugin)
			},
		},
		"crypt plugin with key holder": {
			options: fbservice.RestoreOptions{KeyHolder: "KeyHolder", CryptPlugin: "DbCrypt"},
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
		statistics fbservice.RestoreStatistics
		want       fbconn.RestoreStatistics
	}{
		"none": {
			statistics: 0,
			want:       0,
		},
		"elapsed time": {
			statistics: fbservice.RestoreStatisticsElapsedTime,
			want:       fbconn.RestoreStatisticsElapsedTime,
		},
		"delta time": {
			statistics: fbservice.RestoreStatisticsDeltaTime,
			want:       fbconn.RestoreStatisticsDeltaTime,
		},
		"page reads": {
			statistics: fbservice.RestoreStatisticsPageReads,
			want:       fbconn.RestoreStatisticsPageReads,
		},
		"page writes": {
			statistics: fbservice.RestoreStatisticsPageWrites,
			want:       fbconn.RestoreStatisticsPageWrites,
		},
		"all": {
			statistics: fbservice.RestoreStatisticsAll,
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
				fbservice.RestoreOptions{Statistics: tt.statistics},
			)

			require.NoError(t, err)
			require.Equal(t, tt.want, connection.restoreOptions.Statistics)
		})
	}
}

func TestManagerRestoreAcceptsMaximumValues(t *testing.T) {
	manager, connection := newTestManager(t)
	exactLimit := strings.Repeat("я", 32767) + "x"
	require.Len(t, []byte(exactLimit), math.MaxUint16)

	err := manager.Restore(
		context.Background(),
		exactLimit,
		exactLimit,
		fbservice.RestoreOptions{
			VerboseInterval:       math.MaxInt32,
			Statistics:            fbservice.RestoreStatisticsAll,
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
			TablespacePaths: []fbservice.RestoreTablespacePath{{
				Name: exactLimit,
				Path: exactLimit,
			}},
			PageBuffers:     math.MaxInt32,
			PageSize:        32768,
			ParallelWorkers: math.MaxInt32,
		},
	)

	require.NoError(t, err)
	require.True(t, connection.opened)
	require.Equal(t, exactLimit, connection.restoreOptions.FixFSSData)
	require.Equal(t, math.MaxInt32, connection.restoreOptions.VerboseInterval)
	require.Equal(t, math.MaxInt32, connection.restoreOptions.PageBuffers)
	require.Equal(t, math.MaxInt32, connection.restoreOptions.ParallelWorkers)
}

func TestManagerRestoreMapsAccessMode(t *testing.T) {
	tests := map[string]struct {
		accessMode fbservice.RestoreAccessMode
		want       fbconn.RestoreAccessMode
	}{
		"from backup": {
			accessMode: fbservice.RestoreAccessModeFromBackup,
			want:       fbconn.RestoreAccessModeFromBackup,
		},
		"read-only": {
			accessMode: fbservice.RestoreAccessModeReadOnly,
			want:       fbconn.RestoreAccessModeReadOnly,
		},
		"read-write": {
			accessMode: fbservice.RestoreAccessModeReadWrite,
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
				fbservice.RestoreOptions{AccessMode: tt.accessMode},
			)

			require.NoError(t, err)
			require.Equal(t, tt.want, connection.restoreOptions.AccessMode)
		})
	}
}

func TestManagerRestoreMapsReplicaMode(t *testing.T) {
	tests := map[string]struct {
		replicaMode fbservice.RestoreReplicaMode
		want        fbconn.RestoreReplicaMode
	}{
		"from backup": {
			replicaMode: fbservice.RestoreReplicaModeFromBackup,
			want:        fbconn.RestoreReplicaModeFromBackup,
		},
		"none": {
			replicaMode: fbservice.RestoreReplicaModeNone,
			want:        fbconn.RestoreReplicaModeNone,
		},
		"read-only": {
			replicaMode: fbservice.RestoreReplicaModeReadOnly,
			want:        fbconn.RestoreReplicaModeReadOnly,
		},
		"read-write": {
			replicaMode: fbservice.RestoreReplicaModeReadWrite,
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
				fbservice.RestoreOptions{ReplicaMode: tt.replicaMode},
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
		fbservice.RestoreOptions{},
		fbservice.RestoreOptions{},
	)

	require.ErrorContains(t, err, "at most one")
	require.False(t, connection.opened)
}

func TestManagerRestoreRejectsSemanticOptionsBeforeOpeningConnection(t *testing.T) {
	tests := map[string]struct {
		options fbservice.RestoreOptions
		errText string
	}{
		"verbose and interval": {
			options: fbservice.RestoreOptions{Verbose: true, VerboseInterval: 100},
			errText: "restore verbose and verbose interval are mutually exclusive",
		},
		"key name without key holder": {
			options: fbservice.RestoreOptions{KeyName: "KeyName"},
			errText: "restore key name requires key holder",
		},
		"crypt plugin without key holder": {
			options: fbservice.RestoreOptions{CryptPlugin: "DbCrypt"},
			errText: "restore crypt plugin requires key holder",
		},
		"empty tablespace name": {
			options: fbservice.RestoreOptions{TablespacePaths: []fbservice.RestoreTablespacePath{{Path: "/data/ts1"}}},
			errText: "restore tablespace path 1 name must not be empty",
		},
		"empty tablespace path": {
			options: fbservice.RestoreOptions{TablespacePaths: []fbservice.RestoreTablespacePath{{Name: "TS1"}}},
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
		options    fbservice.RestoreOptions
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
			backupFile: strings.Repeat("b", math.MaxUint16+1),
			database:   "/data/employee.fdb",
			errText:    "restore backup file is 65536 bytes; maximum is 65535",
		},
		"database exceeds SPB limit": {
			backupFile: "/backup/employee.fbk",
			database:   strings.Repeat("d", math.MaxUint16+1),
			errText:    "restore database is 65536 bytes; maximum is 65535",
		},
		"negative page buffers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{PageBuffers: -1},
			errText:    "restore page buffers must be between 0",
		},
		"invalid page size": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{PageSize: 12345},
			errText:    "restore page size must be one of",
		},
		"negative verbose interval": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{VerboseInterval: -1},
			errText:    "restore verbose interval must be between 0",
		},
		"verbose interval below minimum": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{VerboseInterval: 99},
			errText:    "restore verbose interval must be 0 or at least 100",
		},
		"negative parallel workers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{ParallelWorkers: -1},
			errText:    "restore parallel workers must be between 0",
		},
		"invalid access mode": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{AccessMode: fbservice.RestoreAccessMode(255)},
			errText:    "invalid restore access mode 255",
		},
		"invalid replica mode": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{ReplicaMode: fbservice.RestoreReplicaMode(255)},
			errText:    "invalid restore replica mode 255",
		},
		"invalid statistics": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{Statistics: fbservice.RestoreStatistics(0x10)},
			errText:    "invalid restore statistics flags 0x10",
		},
		"skip data exceeds SPB limit": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options: fbservice.RestoreOptions{
				SkipData: strings.Repeat("s", math.MaxUint16+1),
			},
			errText: "restore skip data is 65536 bytes; maximum is 65535",
		},
		"include data exceeds SPB limit": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options: fbservice.RestoreOptions{
				IncludeData: strings.Repeat("i", math.MaxUint16+1),
			},
			errText: "restore include data is 65536 bytes; maximum is 65535",
		},
	}

	tooLong := strings.Repeat("x", math.MaxUint16+1)
	for _, tt := range []struct {
		name      string
		errorName string
		options   func(string) fbservice.RestoreOptions
	}{
		{"fix FSS data exceeds SPB limit", "fix FSS data", func(value string) fbservice.RestoreOptions { return fbservice.RestoreOptions{FixFSSData: value} }},
		{"fix FSS metadata exceeds SPB limit", "fix FSS metadata", func(value string) fbservice.RestoreOptions { return fbservice.RestoreOptions{FixFSSMetadata: value} }},
		{"skip schema data exceeds SPB limit", "skip schema data", func(value string) fbservice.RestoreOptions { return fbservice.RestoreOptions{SkipSchemaData: value} }},
		{"include schema data exceeds SPB limit", "include schema data", func(value string) fbservice.RestoreOptions { return fbservice.RestoreOptions{IncludeSchemaData: value} }},
		{"key holder exceeds SPB limit", "key holder", func(value string) fbservice.RestoreOptions { return fbservice.RestoreOptions{KeyHolder: value} }},
		{"key name exceeds SPB limit", "key name", func(value string) fbservice.RestoreOptions { return fbservice.RestoreOptions{KeyName: value} }},
		{"crypt plugin exceeds SPB limit", "crypt plugin", func(value string) fbservice.RestoreOptions {
			return fbservice.RestoreOptions{KeyHolder: "KeyHolder", CryptPlugin: value}
		}},
		{"tablespace mapping file exceeds SPB limit", "tablespace mapping file", func(value string) fbservice.RestoreOptions {
			return fbservice.RestoreOptions{TablespaceMappingFile: value}
		}},
		{"tablespace path name exceeds SPB limit", "tablespace path 1 name", func(value string) fbservice.RestoreOptions {
			return fbservice.RestoreOptions{TablespacePaths: []fbservice.RestoreTablespacePath{{Name: value, Path: "/data/ts1"}}}
		}},
		{"tablespace path exceeds SPB limit", "tablespace path 1 path", func(value string) fbservice.RestoreOptions {
			return fbservice.RestoreOptions{TablespacePaths: []fbservice.RestoreTablespacePath{{Name: "TS1", Path: value}}}
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
		aboveMaxInt32 := int64(math.MaxInt32) + 1
		tests["page buffers exceed int32"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{PageBuffers: int(aboveMaxInt32)},
			errText:    "restore page buffers must be between 0",
		}
		tests["verbose interval exceeds int32"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{VerboseInterval: int(aboveMaxInt32)},
			errText:    "restore verbose interval must be between 0",
		}
		tests["parallel workers exceed int32"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    fbservice.RestoreOptions{ParallelWorkers: int(aboveMaxInt32)},
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
	fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		return nil, connectionErr
	})

	err := manager.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
	)

	require.ErrorIs(t, err, connectionErr)
	require.ErrorContains(t, err, "connect to service manager")
}
