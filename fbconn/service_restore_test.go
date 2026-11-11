package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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

func TestBuildRestoreActionSPBAllOptionsWireGolden(t *testing.T) {
	// This golden covers wire encoding and order. Manager-level semantic
	// conflicts are intentionally exercised together here.
	options := RestoreOptions{
		Verbose:                    true,
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
		MetadataOnly:               true,
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
		TablespacePaths:            []RestoreTablespacePath{{Name: "TS1", Path: "/data/ts1"}, {Name: "TS2", Path: "/data/ts2"}},
		UseOriginalTablespacePaths: true,
		PageBuffers:                512,
		PageSize:                   16384,
		ParallelWorkers:            4,
	}

	spb, err := buildRestoreActionSPB(
		"/backup/employee.fbk",
		"/data/employee.fdb",
		options,
	)
	require.NoError(t, err)

	// Numeric values make this a wire golden rather than a mirror of fbproto.
	offset := 0
	require.Equal(t, byte(2), spb[offset])
	offset++
	offset = requireServiceActionString(t, spb, offset, 106, "/data/employee.fdb")
	offset = requireServiceActionString(t, spb, offset, 5, "/backup/employee.fbk")
	require.Equal(t, byte(9), spb[offset])
	offset++
	require.Equal(t, uint32(512), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(10), spb[offset])
	offset++
	require.Equal(t, uint32(16384), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(12), spb[offset])
	offset++
	require.Equal(t, byte(39), spb[offset])
	offset++
	require.Equal(t, byte(20), spb[offset])
	offset++
	require.Equal(t, byte(1), spb[offset])
	offset++
	require.Equal(t, byte(107), spb[offset])
	offset++
	require.Equal(t, byte(114), spb[offset])
	offset++
	require.Equal(t, uint32(100), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	offset = requireServiceActionString(t, spb, offset, 15, "TDRW")
	require.Equal(t, byte(21), spb[offset])
	offset++
	require.Equal(t, uint32(4), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	offset = requireServiceActionString(t, spb, offset, 13, "WIN1251")
	offset = requireServiceActionString(t, spb, offset, 14, "WIN1251")
	offset = requireServiceActionString(t, spb, offset, 22, "SKIP_SCHEMA")
	offset = requireServiceActionString(t, spb, offset, 8, "SKIP_%")
	offset = requireServiceActionString(t, spb, offset, 23, "KEEP_SCHEMA")
	offset = requireServiceActionString(t, spb, offset, 19, "KEEP_%")
	offset = requireServiceActionString(t, spb, offset, 16, "KeyHolder")
	offset = requireServiceActionString(t, spb, offset, 17, "KeyName")
	offset = requireServiceActionString(t, spb, offset, 18, "DbCrypt")
	offset = requireServiceActionString(t, spb, offset, 120, "/config/tablespaces.map")
	offset = requireServiceActionString(t, spb, offset, 121, "TS1")
	offset = requireServiceActionString(t, spb, offset, 121, "/data/ts1")
	offset = requireServiceActionString(t, spb, offset, 121, "TS2")
	offset = requireServiceActionString(t, spb, offset, 121, "/data/ts2")
	require.Equal(t, byte(108), spb[offset])
	offset++
	require.Equal(t, uint32(0x825f05), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestRestoreOptionMaskAdditionalFlags(t *testing.T) {
	tests := map[string]struct {
		options RestoreOptions
		want    int32
	}{
		"metadata only": {
			options: RestoreOptions{MetadataOnly: true},
			want:    fbproto.Isc_spb_res_create | fbproto.Isc_spb_res_metadata_only,
		},
		"direct I/O": {
			options: RestoreOptions{DirectIO: true},
			want:    fbproto.Isc_spb_res_create | fbproto.Isc_spb_res_direct_io,
		},
		"ignore BLR errors": {
			options: RestoreOptions{IgnoreBLRErrors: true},
			want:    fbproto.Isc_spb_res_create | fbproto.Isc_spb_bkp_ignore_checksums,
		},
		"original tablespace paths": {
			options: RestoreOptions{UseOriginalTablespacePaths: true},
			want:    fbproto.Isc_spb_res_create | fbproto.Isc_spb_res_ts_original_paths,
		},
		"all additional flags": {
			options: RestoreOptions{
				MetadataOnly:               true,
				DirectIO:                   true,
				IgnoreBLRErrors:            true,
				UseOriginalTablespacePaths: true,
			},
			want: fbproto.Isc_spb_res_create |
				fbproto.Isc_spb_res_metadata_only |
				fbproto.Isc_spb_res_direct_io |
				fbproto.Isc_spb_bkp_ignore_checksums |
				fbproto.Isc_spb_res_ts_original_paths,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.want, restoreOptionMask(tt.options))
		})
	}
}

func TestBuildRestoreActionSPBUsesCreateAndModesFromBackupByDefault(t *testing.T) {
	spb, err := buildRestoreActionSPB(
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{},
	)
	require.NoError(t, err)

	offset := 1
	offset = requireServiceActionString(
		t,
		spb,
		offset,
		fbproto.Isc_spb_dbname,
		"/data/employee.fdb",
	)
	offset = requireServiceActionString(
		t,
		spb,
		offset,
		fbproto.Isc_spb_bkp_file,
		"/backup/employee.fbk",
	)
	require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
	offset++
	require.Equal(
		t,
		uint32(fbproto.Isc_spb_res_create),
		binary.LittleEndian.Uint32(spb[offset:offset+4]),
	)
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestBuildRestoreActionSPBAccessMode(t *testing.T) {
	tests := map[string]struct {
		accessMode RestoreAccessMode
		wireMode   byte
	}{
		"read-only": {
			accessMode: RestoreAccessModeReadOnly,
			wireMode:   fbproto.Isc_spb_res_am_readonly,
		},
		"read-write": {
			accessMode: RestoreAccessModeReadWrite,
			wireMode:   fbproto.Isc_spb_res_am_readwrite,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			spb, err := buildRestoreActionSPB(
				"/backup/employee.fbk",
				"/data/employee.fdb",
				RestoreOptions{AccessMode: tt.accessMode},
			)
			require.NoError(t, err)

			offset := 1
			offset = requireServiceActionString(
				t,
				spb,
				offset,
				fbproto.Isc_spb_dbname,
				"/data/employee.fdb",
			)
			offset = requireServiceActionString(
				t,
				spb,
				offset,
				fbproto.Isc_spb_bkp_file,
				"/backup/employee.fbk",
			)
			require.Equal(t, byte(fbproto.Isc_spb_res_access_mode), spb[offset])
			offset++
			require.Equal(t, tt.wireMode, spb[offset])
			offset++
			require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
		})
	}
}

func TestBuildRestoreActionSPBReplicaMode(t *testing.T) {
	tests := map[string]struct {
		replicaMode RestoreReplicaMode
		wireMode    byte
	}{
		"none": {
			replicaMode: RestoreReplicaModeNone,
			wireMode:    fbproto.Isc_spb_res_rm_none,
		},
		"read-only": {
			replicaMode: RestoreReplicaModeReadOnly,
			wireMode:    fbproto.Isc_spb_res_rm_readonly,
		},
		"read-write": {
			replicaMode: RestoreReplicaModeReadWrite,
			wireMode:    fbproto.Isc_spb_res_rm_readwrite,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			spb, err := buildRestoreActionSPB(
				"/backup/employee.fbk",
				"/data/employee.fdb",
				RestoreOptions{ReplicaMode: tt.replicaMode},
			)
			require.NoError(t, err)

			offset := 1
			offset = requireServiceActionString(
				t,
				spb,
				offset,
				fbproto.Isc_spb_dbname,
				"/data/employee.fdb",
			)
			offset = requireServiceActionString(
				t,
				spb,
				offset,
				fbproto.Isc_spb_bkp_file,
				"/backup/employee.fbk",
			)
			require.Equal(t, byte(fbproto.Isc_spb_res_replica_mode), spb[offset])
			offset++
			require.Equal(t, tt.wireMode, spb[offset])
			offset++
			require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
		})
	}
}

func TestBuildRestoreActionSPBPreservesTablespacePathOrder(t *testing.T) {
	spb, err := buildRestoreActionSPB(
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{TablespacePaths: []RestoreTablespacePath{
			{Name: "TS1", Path: "/data/old-ts1"},
			{Name: "TS2", Path: "/data/ts2"},
			{Name: "TS1", Path: "/data/new-ts1"},
		}},
	)
	require.NoError(t, err)

	offset := 1
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_dbname, "/data/employee.fdb")
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_bkp_file, "/backup/employee.fbk")
	for _, tablespacePath := range []RestoreTablespacePath{
		{Name: "TS1", Path: "/data/old-ts1"},
		{Name: "TS2", Path: "/data/ts2"},
		{Name: "TS1", Path: "/data/new-ts1"},
	} {
		offset = requireServiceActionString(
			t,
			spb,
			offset,
			fbproto.Isc_spb_res_ts_path,
			tablespacePath.Name,
		)
		offset = requireServiceActionString(
			t,
			spb,
			offset,
			fbproto.Isc_spb_res_ts_path,
			tablespacePath.Path,
		)
	}
	require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
	offset++
	require.Equal(t, uint32(fbproto.Isc_spb_res_create), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestRestoreStatisticsString(t *testing.T) {
	tests := map[string]struct {
		statistics RestoreStatistics
		want       string
	}{
		"none": {
			statistics: 0,
			want:       "",
		},
		"elapsed time": {
			statistics: RestoreStatisticsElapsedTime,
			want:       "T",
		},
		"delta time": {
			statistics: RestoreStatisticsDeltaTime,
			want:       "D",
		},
		"page reads": {
			statistics: RestoreStatisticsPageReads,
			want:       "R",
		},
		"page writes": {
			statistics: RestoreStatisticsPageWrites,
			want:       "W",
		},
		"canonical order": {
			statistics: RestoreStatisticsPageWrites |
				RestoreStatisticsElapsedTime |
				RestoreStatisticsPageReads |
				RestoreStatisticsDeltaTime,
			want: "TDRW",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := restoreStatisticsString(tt.statistics)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	_, err := restoreStatisticsString(RestoreStatistics(0x10))
	require.EqualError(t, err, "invalid restore statistics flags 0x10")
	_, err = restoreStatisticsString(RestoreStatisticsElapsedTime | RestoreStatistics(0x10))
	require.EqualError(t, err, "invalid restore statistics flags 0x11")
}

func TestBuildRestoreActionSPBDoesNotApplyManagerSemantics(t *testing.T) {
	tests := map[string]RestoreOptions{
		"verbose and interval": {
			Verbose:         true,
			VerboseInterval: 100,
		},
		"short verbose interval": {
			VerboseInterval: 1,
		},
		"key name without key holder": {
			KeyName: "KeyName",
		},
		"crypt plugin without key holder": {
			CryptPlugin: "DbCrypt",
		},
		"empty tablespace path name": {
			TablespacePaths: []RestoreTablespacePath{{Path: "/data/ts1"}},
		},
	}

	for name, options := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := buildRestoreActionSPB(
				"/backup/employee.fbk",
				"/data/employee.fdb",
				options,
			)
			require.NoError(t, err)
		})
	}
}

func TestBuildRestoreActionSPBStringLimitUsesBytes(t *testing.T) {
	exactLimit := strings.Repeat("я", 32767) + "x"
	require.Len(t, []byte(exactLimit), maxServiceActionStringLength)

	_, err := buildRestoreActionSPB(
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{FixFSSData: exactLimit},
	)
	require.NoError(t, err)

	_, err = buildRestoreActionSPB(
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{FixFSSData: exactLimit + "y"},
	)
	require.EqualError(
		t,
		err,
		"restore fix FSS data: value is 65536 bytes; maximum is 65535",
	)
}

func TestBuildRestoreActionSPBAcceptsMaximumValues(t *testing.T) {
	maxLengthValue := strings.Repeat("x", maxServiceActionStringLength)

	spb, err := buildRestoreActionSPB(
		maxLengthValue,
		maxLengthValue,
		RestoreOptions{
			VerboseInterval:            maxServiceActionInt32,
			Statistics:                 RestoreStatisticsAll,
			FixFSSData:                 maxLengthValue,
			FixFSSMetadata:             maxLengthValue,
			SkipSchemaData:             maxLengthValue,
			SkipData:                   maxLengthValue,
			IncludeSchemaData:          maxLengthValue,
			IncludeData:                maxLengthValue,
			KeyHolder:                  maxLengthValue,
			KeyName:                    maxLengthValue,
			CryptPlugin:                maxLengthValue,
			TablespaceMappingFile:      maxLengthValue,
			TablespacePaths:            []RestoreTablespacePath{{Name: maxLengthValue, Path: maxLengthValue}},
			UseOriginalTablespacePaths: true,
			PageBuffers:                maxServiceActionInt32,
			PageSize:                   32768,
			ParallelWorkers:            maxServiceActionInt32,
		},
	)
	require.NoError(t, err)

	offset := 1
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_dbname, maxLengthValue)
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_bkp_file, maxLengthValue)
	require.Equal(t, byte(fbproto.Isc_spb_res_buffers), spb[offset])
	offset++
	require.Equal(t, uint32(maxServiceActionInt32), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_res_page_size), spb[offset])
	offset++
	require.Equal(t, uint32(32768), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_verbint), spb[offset])
	offset++
	require.Equal(t, uint32(maxServiceActionInt32), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_res_stat, "TDRW")
	require.Equal(t, byte(fbproto.Isc_spb_res_parallel_workers), spb[offset])
	offset++
	require.Equal(t, uint32(maxServiceActionInt32), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	for _, tag := range []byte{
		fbproto.Isc_spb_res_fix_fss_data,
		fbproto.Isc_spb_res_fix_fss_metadata,
		fbproto.Isc_spb_res_skip_schema_data,
		fbproto.Isc_spb_res_skip_data,
		fbproto.Isc_spb_res_include_schema_data,
		fbproto.Isc_spb_res_include_data,
		fbproto.Isc_spb_res_keyholder,
		fbproto.Isc_spb_res_keyname,
		fbproto.Isc_spb_res_crypt,
		fbproto.Isc_spb_res_ts_mapping_file,
		fbproto.Isc_spb_res_ts_path,
		fbproto.Isc_spb_res_ts_path,
	} {
		offset = requireServiceActionString(t, spb, offset, tag, maxLengthValue)
	}
	require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
	offset++
	require.Equal(
		t,
		uint32(fbproto.Isc_spb_res_create|fbproto.Isc_spb_res_ts_original_paths),
		binary.LittleEndian.Uint32(spb[offset:offset+4]),
	)
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestBuildRestoreActionSPBValidation(t *testing.T) {
	type testCase struct {
		backupFile string
		database   string
		options    RestoreOptions
		wantError  string
	}

	tests := map[string]testCase{
		"empty backup file": {
			database:  "/data/employee.fdb",
			wantError: "restore backup file must not be empty",
		},
		"empty database": {
			backupFile: "/backup/employee.fbk",
			wantError:  "restore database must not be empty",
		},
		"backup file is too long": {
			backupFile: strings.Repeat("b", maxServiceActionStringLength+1),
			database:   "/data/employee.fdb",
			wantError:  "maximum is 65535",
		},
		"database is too long": {
			backupFile: "/backup/employee.fbk",
			database:   strings.Repeat("d", maxServiceActionStringLength+1),
			wantError:  "maximum is 65535",
		},
		"negative page buffers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageBuffers: -1},
			wantError:  "restore page buffers must be between",
		},
		"invalid page size": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageSize: 12345},
			wantError:  "restore page size must be one of",
		},
		"negative verbose interval": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{VerboseInterval: -1},
			wantError:  "restore verbose interval must be between",
		},
		"negative parallel workers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ParallelWorkers: -1},
			wantError:  "restore parallel workers must be between",
		},
		"invalid access mode": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{AccessMode: RestoreAccessMode(255)},
			wantError:  "invalid restore access mode 255",
		},
		"invalid replica mode": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ReplicaMode: RestoreReplicaMode(255)},
			wantError:  "invalid restore replica mode 255",
		},
		"invalid statistics": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{Statistics: RestoreStatistics(0x10)},
			wantError:  "invalid restore statistics flags 0x10",
		},
		"skip data is too long": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options: RestoreOptions{
				SkipData: strings.Repeat("s", maxServiceActionStringLength+1),
			},
			wantError: "restore skip data: value is 65536 bytes; maximum is 65535",
		},
		"include data is too long": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options: RestoreOptions{
				IncludeData: strings.Repeat("i", maxServiceActionStringLength+1),
			},
			wantError: "restore include data: value is 65536 bytes; maximum is 65535",
		},
	}

	tooLong := strings.Repeat("x", maxServiceActionStringLength+1)
	for _, tt := range []struct {
		name      string
		errorName string
		options   func(string) RestoreOptions
	}{
		{"fix FSS data is too long", "fix FSS data", func(value string) RestoreOptions { return RestoreOptions{FixFSSData: value} }},
		{"fix FSS metadata is too long", "fix FSS metadata", func(value string) RestoreOptions { return RestoreOptions{FixFSSMetadata: value} }},
		{"skip schema data is too long", "skip schema data", func(value string) RestoreOptions { return RestoreOptions{SkipSchemaData: value} }},
		{"include schema data is too long", "include schema data", func(value string) RestoreOptions { return RestoreOptions{IncludeSchemaData: value} }},
		{"key holder is too long", "key holder", func(value string) RestoreOptions { return RestoreOptions{KeyHolder: value} }},
		{"key name is too long", "key name", func(value string) RestoreOptions { return RestoreOptions{KeyName: value} }},
		{"crypt plugin is too long", "crypt plugin", func(value string) RestoreOptions { return RestoreOptions{CryptPlugin: value} }},
		{"tablespace mapping file is too long", "tablespace mapping file", func(value string) RestoreOptions { return RestoreOptions{TablespaceMappingFile: value} }},
		{"tablespace path name is too long", "tablespace path 1 name", func(value string) RestoreOptions {
			return RestoreOptions{TablespacePaths: []RestoreTablespacePath{{Name: value, Path: "/data/ts1"}}}
		}},
		{"tablespace path is too long", "tablespace path 1 path", func(value string) RestoreOptions {
			return RestoreOptions{TablespacePaths: []RestoreTablespacePath{{Name: "TS1", Path: value}}}
		}},
	} {
		tests[tt.name] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    tt.options(tooLong),
			wantError:  "restore " + tt.errorName + ": value is 65536 bytes; maximum is 65535",
		}
	}

	if strconv.IntSize > 32 {
		aboveMaxInt32 := int64(maxServiceActionInt32) + 1
		tests["page buffers overflow"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageBuffers: int(aboveMaxInt32)},
			wantError:  "restore page buffers must be between",
		}
		tests["verbose interval overflow"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{VerboseInterval: int(aboveMaxInt32)},
			wantError:  "restore verbose interval must be between",
		}
		tests["parallel workers overflow"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ParallelWorkers: int(aboveMaxInt32)},
			wantError:  "restore parallel workers must be between",
		}
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := buildRestoreActionSPB(tt.backupFile, tt.database, tt.options)
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestServiceConnRestore(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	resultCh := make(chan serviceActionProbeResult, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		resultCh <- runServiceActionProbeServer(serverConn)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := ConnectServiceConfig(ctx, testServiceConfig(config))
	require.NoError(t, err)

	var output bytes.Buffer
	err = conn.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{
			Output:  &output,
			Verbose: true,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "gbak: writing data\n", output.String())
	require.False(t, conn.actionActive)
	require.NoError(t, conn.Close(context.Background()))

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		require.Equal(t, byte(fbproto.Isc_action_svc_restore), result.actionSPB[0])
		require.Equal(t, 42, result.detachedHandle)
		require.Equal(t, fbproto.Op_disconnect, result.finalOpcode)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestServiceConnRestoreCanBeCanceledWhileWaitingForOutput(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	queryStarted := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		serverDone <- runCanceledServiceActionProbeServer(serverConn, queryStarted, fbproto.Isc_action_svc_restore)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}
	config.BuildContextWatcherHandler = func(conn *FbConn) ctxwatch.Handler {
		return &DeadlineContextWatcherHandler{Conn: conn.conn}
	}

	conn, err := ConnectServiceConfig(context.Background(), testServiceConfig(config))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	restoreDone := make(chan error, 1)
	go func() {
		restoreDone <- conn.Restore(
			ctx,
			"/backup/employee.fbk",
			"/data/employee.fdb",
			RestoreOptions{},
		)
	}()

	select {
	case <-queryStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("restore did not start output polling")
	}
	cancel()

	select {
	case err := <-restoreDone:
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "restore database")
	case <-time.After(2 * time.Second):
		t.Fatal("restore did not stop after context cancellation")
	}
	require.True(t, conn.conn.IsClosed())
	select {
	case <-conn.conn.CleanupDone():
	case <-time.After(2 * time.Second):
		t.Fatal("service connection cleanup did not complete")
	}
	require.NoError(t, <-serverDone)
	require.NoError(t, conn.Close(context.Background()))
}

func runCanceledServiceActionProbeServer(conn net.Conn, queryStarted chan<- struct{}, action byte) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return err
	}
	if err := writeAuthenticatedProbeAccept(w, encoding); err != nil {
		return err
	}
	if err := readProbeServiceAttach(r, encoding); err != nil {
		return err
	}
	if err := writeProbeResponse(w, 42); err != nil {
		return err
	}
	actionSPB, err := readProbeServiceStart(r, 42)
	if err != nil {
		return err
	}
	if len(actionSPB) == 0 || actionSPB[0] != action {
		return errors.New("unexpected service action")
	}
	if err := writeProbeResponse(w, 0); err != nil {
		return err
	}
	if _, _, _, err := readProbeServiceInfo(r, 42); err != nil {
		return err
	}
	close(queryStarted)

	_, err = r.ReadInt32()
	if !errors.Is(err, io.EOF) {
		return errors.New("client did not close after context cancellation")
	}
	return nil
}
