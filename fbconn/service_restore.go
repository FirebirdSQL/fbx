package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/firebirdsql/fbx/fbproto"
)

// RestoreAccessMode specifies the access mode of a restored database.
type RestoreAccessMode uint8

const (
	// RestoreAccessModeFromBackup preserves the access mode recorded in the backup.
	RestoreAccessModeFromBackup RestoreAccessMode = iota
	// RestoreAccessModeReadOnly restores the database in read-only mode.
	RestoreAccessModeReadOnly
	// RestoreAccessModeReadWrite restores the database in read-write mode.
	RestoreAccessModeReadWrite
)

// RestoreReplicaMode specifies the replica mode of a restored database.
type RestoreReplicaMode uint8

const (
	// RestoreReplicaModeFromBackup preserves the replica mode recorded in the backup.
	RestoreReplicaModeFromBackup RestoreReplicaMode = iota
	// RestoreReplicaModeNone disables replica mode for the restored database.
	RestoreReplicaModeNone
	// RestoreReplicaModeReadOnly restores the database as a read-only replica.
	RestoreReplicaModeReadOnly
	// RestoreReplicaModeReadWrite restores the database as a read-write replica.
	RestoreReplicaModeReadWrite
)

// RestoreStatistics selects performance counters included in restore output.
type RestoreStatistics uint8

const (
	// RestoreStatisticsElapsedTime reports time elapsed since the restore started.
	RestoreStatisticsElapsedTime RestoreStatistics = 1 << iota
	// RestoreStatisticsDeltaTime reports time elapsed since the previous output line.
	RestoreStatisticsDeltaTime
	// RestoreStatisticsPageReads reports the number of database page reads.
	RestoreStatisticsPageReads
	// RestoreStatisticsPageWrites reports the number of database page writes.
	RestoreStatisticsPageWrites
	// RestoreStatisticsAll enables every supported restore statistic.
	RestoreStatisticsAll = RestoreStatisticsElapsedTime |
		RestoreStatisticsDeltaTime |
		RestoreStatisticsPageReads |
		RestoreStatisticsPageWrites
)

// RestoreTablespacePath maps a tablespace name to a path during restore.
type RestoreTablespacePath struct {
	// Name is the tablespace name stored in the backup.
	Name string
	// Path is the server-side path to use for the restored tablespace.
	Path string
}

// RestoreOptions controls a logical restore performed by [ServiceConn.Restore].
//
// The zero value creates a new database using the server defaults, preserves
// the access and replica modes recorded in the backup, and discards service
// output.
type RestoreOptions struct {
	// Output receives service output. A nil value discards the output. Calls to
	// [io.Writer.Write] are synchronous, so Output must not block indefinitely.
	Output io.Writer

	// Verbose requests progress output from the restore service.
	Verbose bool

	// VerboseInterval requests progress output after every specified number of
	// restored records. A zero value disables interval-based output.
	VerboseInterval int

	// Statistics selects performance counters included in service output. The
	// zero value disables statistics output.
	Statistics RestoreStatistics

	// Replace allows the restore service to replace an existing database. When
	// false, Restore creates a new database and fails if it already exists.
	Replace bool

	// AccessMode controls the access mode of the restored database. The zero
	// value preserves the access mode recorded in the backup.
	AccessMode RestoreAccessMode

	// ReplicaMode controls the replica mode of the restored database. The zero
	// value preserves the replica mode recorded in the backup. This option
	// requires Firebird 4 or Red Database 5 or later.
	ReplicaMode RestoreReplicaMode

	// DeactivateIndexes leaves indexes inactive after restoring them.
	DeactivateIndexes bool

	// SkipShadows prevents database shadow files from being restored.
	SkipShadows bool

	// SkipValidityConstraints prevents validity constraints from being
	// restored.
	SkipValidityConstraints bool

	// CommitAfterEachTable commits after restoring each table.
	CommitAfterEachTable bool

	// UseAllPageSpace prevents reserving space on data pages for record
	// versions.
	UseAllPageSpace bool

	// MetadataOnly restores database metadata without table data.
	MetadataOnly bool

	// DirectIO opens the backup file using direct I/O. This option requires
	// Firebird 5 or Red Database 5 or later.
	DirectIO bool

	// IgnoreBLRErrors allows restore to continue when stored object BLR cannot
	// be compiled. This option is available in Red Database.
	IgnoreBLRErrors bool

	// FixFSSData specifies the original character set of malformed UNICODE_FSS
	// table data. An empty value disables data repair.
	FixFSSData string

	// FixFSSMetadata specifies the original character set of malformed
	// UNICODE_FSS metadata. An empty value disables metadata repair.
	FixFSSMetadata string

	// SkipSchemaData is an SQL regular expression matching schemas whose table
	// data is not restored. Schema metadata is still restored. An empty value
	// disables the filter. This option requires restore Service API support for
	// schema filters.
	SkipSchemaData string

	// SkipData is an SQL regular expression matching tables whose data is not
	// restored. Table metadata is still restored. An empty value disables the
	// filter.
	SkipData string

	// IncludeSchemaData is an SQL regular expression matching schemas whose
	// table data is restored. An empty value disables the filter. This option
	// requires restore Service API support for schema filters.
	IncludeSchemaData string

	// IncludeData is an SQL regular expression matching tables whose data is
	// restored. An empty value disables the filter. This option requires
	// Firebird 4 or Red Database 5 or later.
	IncludeData string

	// KeyHolder specifies the key holder plugin used to access encryption keys.
	// Encrypted restore parameters require Firebird 4 or Red Database 5 or later.
	KeyHolder string

	// KeyName overrides the encryption key name stored in the backup.
	KeyName string

	// CryptPlugin specifies the database encryption plugin used by the restored
	// database.
	CryptPlugin string

	// TablespaceMappingFile is a server-side Red Database 5.2 mapping file. It
	// is sent before TablespacePaths so explicit mappings can override file
	// entries. An empty value disables file-based tablespace mapping.
	TablespaceMappingFile string

	// TablespacePaths maps tablespaces to server-side paths. Entries are sent
	// to the service in slice order, so later duplicate names can override
	// earlier mappings. This option is available in Red Database 5.2.
	TablespacePaths []RestoreTablespacePath

	// UseOriginalTablespacePaths allows Red Database 5.2 to reuse paths stored
	// in the backup when no explicit tablespace mapping exists.
	UseOriginalTablespacePaths bool

	// PageBuffers sets the default number of database page buffers. A zero
	// value uses the value recorded in the backup.
	PageBuffers int

	// PageSize sets the restored database page size. A zero value uses the
	// value recorded in the backup.
	PageSize int

	// ParallelWorkers sets the number of workers used by the restore. A zero
	// value uses the server default. This option requires Firebird 5 or Red
	// Database 5 or later.
	ParallelWorkers int
}

// Restore restores backupFile into database.
//
// The Firebird server interprets both backupFile and database. They must be
// server-side paths or aliases supported by the server. Restore blocks until
// the service action completes, the context cancels a network operation, or an
// error occurs. When [RestoreOptions.Output] is non-nil, Restore writes service
// output to it as the action runs. Context cancellation cannot interrupt an
// [io.Writer.Write] call that has already started. Restore does not silently
// ignore unsupported options or retry without them; it returns the server
// error.
func (c *ServiceConn) Restore(
	ctx context.Context,
	backupFile string,
	database string,
	options RestoreOptions,
) error {
	actionSPB, err := buildRestoreActionSPB(backupFile, database, options)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, actionSPB, options.Output); err != nil {
		return fmt.Errorf("restore database: %w", err)
	}
	return nil
}

func buildRestoreActionSPB(
	backupFile string,
	database string,
	options RestoreOptions,
) ([]byte, error) {
	if backupFile == "" {
		return nil, errors.New("restore backup file must not be empty")
	}
	if database == "" {
		return nil, errors.New("restore database must not be empty")
	}

	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_action_svc_restore)
	if err := putServiceActionString(spb, fbproto.Isc_spb_dbname, database); err != nil {
		return nil, fmt.Errorf("restore database: %w", err)
	}
	if err := putServiceActionString(spb, fbproto.Isc_spb_bkp_file, backupFile); err != nil {
		return nil, fmt.Errorf("restore backup file: %w", err)
	}
	if options.PageBuffers < 0 || options.PageBuffers > maxServiceActionInt32 {
		return nil, fmt.Errorf(
			"restore page buffers must be between 0 and %d",
			maxServiceActionInt32,
		)
	}
	if options.PageBuffers > 0 {
		spb.AppendByte(fbproto.Isc_spb_res_buffers)
		spb.AppendInt32(int32(options.PageBuffers))
	}
	if err := validateRestorePageSize(options.PageSize); err != nil {
		return nil, err
	}
	if options.PageSize > 0 {
		spb.AppendByte(fbproto.Isc_spb_res_page_size)
		spb.AppendInt32(int32(options.PageSize))
	}
	switch options.AccessMode {
	case RestoreAccessModeFromBackup:
	case RestoreAccessModeReadOnly:
		spb.AppendByte(fbproto.Isc_spb_res_access_mode)
		spb.AppendByte(fbproto.Isc_spb_res_am_readonly)
	case RestoreAccessModeReadWrite:
		spb.AppendByte(fbproto.Isc_spb_res_access_mode)
		spb.AppendByte(fbproto.Isc_spb_res_am_readwrite)
	default:
		return nil, fmt.Errorf("invalid restore access mode %d", options.AccessMode)
	}
	switch options.ReplicaMode {
	case RestoreReplicaModeFromBackup:
	case RestoreReplicaModeNone:
		spb.AppendByte(fbproto.Isc_spb_res_replica_mode)
		spb.AppendByte(fbproto.Isc_spb_res_rm_none)
	case RestoreReplicaModeReadOnly:
		spb.AppendByte(fbproto.Isc_spb_res_replica_mode)
		spb.AppendByte(fbproto.Isc_spb_res_rm_readonly)
	case RestoreReplicaModeReadWrite:
		spb.AppendByte(fbproto.Isc_spb_res_replica_mode)
		spb.AppendByte(fbproto.Isc_spb_res_rm_readwrite)
	default:
		return nil, fmt.Errorf("invalid restore replica mode %d", options.ReplicaMode)
	}
	if options.Verbose {
		spb.AppendByte(fbproto.Isc_spb_verbose)
	}
	if options.VerboseInterval < 0 || options.VerboseInterval > maxServiceActionInt32 {
		return nil, fmt.Errorf(
			"restore verbose interval must be between 0 and %d",
			maxServiceActionInt32,
		)
	}
	if options.VerboseInterval > 0 {
		spb.AppendByte(fbproto.Isc_spb_verbint)
		spb.AppendInt32(int32(options.VerboseInterval))
	}
	statistics, err := restoreStatisticsString(options.Statistics)
	if err != nil {
		return nil, err
	}
	if err := putOptionalRestoreActionString(
		spb,
		fbproto.Isc_spb_res_stat,
		"statistics",
		statistics,
	); err != nil {
		return nil, err
	}
	if options.ParallelWorkers < 0 || options.ParallelWorkers > maxServiceActionInt32 {
		return nil, fmt.Errorf(
			"restore parallel workers must be between 0 and %d",
			maxServiceActionInt32,
		)
	}
	if options.ParallelWorkers > 0 {
		spb.AppendByte(fbproto.Isc_spb_res_parallel_workers)
		spb.AppendInt32(int32(options.ParallelWorkers))
	}
	optionalStrings := []struct {
		tag   byte
		name  string
		value string
	}{
		{fbproto.Isc_spb_res_fix_fss_data, "fix FSS data", options.FixFSSData},
		{fbproto.Isc_spb_res_fix_fss_metadata, "fix FSS metadata", options.FixFSSMetadata},
		{fbproto.Isc_spb_res_skip_schema_data, "skip schema data", options.SkipSchemaData},
		{fbproto.Isc_spb_res_skip_data, "skip data", options.SkipData},
		{fbproto.Isc_spb_res_include_schema_data, "include schema data", options.IncludeSchemaData},
		{fbproto.Isc_spb_res_include_data, "include data", options.IncludeData},
		{fbproto.Isc_spb_res_keyholder, "key holder", options.KeyHolder},
		{fbproto.Isc_spb_res_keyname, "key name", options.KeyName},
		{fbproto.Isc_spb_res_crypt, "crypt plugin", options.CryptPlugin},
		{fbproto.Isc_spb_res_ts_mapping_file, "tablespace mapping file", options.TablespaceMappingFile},
	}
	for _, option := range optionalStrings {
		if err := putOptionalRestoreActionString(spb, option.tag, option.name, option.value); err != nil {
			return nil, err
		}
	}
	for i, tablespacePath := range options.TablespacePaths {
		if err := putServiceActionString(spb, fbproto.Isc_spb_res_ts_path, tablespacePath.Name); err != nil {
			return nil, fmt.Errorf("restore tablespace path %d name: %w", i+1, err)
		}
		if err := putServiceActionString(spb, fbproto.Isc_spb_res_ts_path, tablespacePath.Path); err != nil {
			return nil, fmt.Errorf("restore tablespace path %d path: %w", i+1, err)
		}
	}
	spb.AppendByte(fbproto.Isc_spb_options)
	spb.AppendInt32(restoreOptionMask(options))
	return spb.Bytes(), nil
}

func putOptionalRestoreActionString(
	spb *fbproto.XPBWriter,
	tag byte,
	name string,
	value string,
) error {
	if value == "" {
		return nil
	}
	if err := putServiceActionString(spb, tag, value); err != nil {
		return fmt.Errorf("restore %s: %w", name, err)
	}
	return nil
}

func restoreStatisticsString(statistics RestoreStatistics) (string, error) {
	if statistics&^RestoreStatisticsAll != 0 {
		return "", fmt.Errorf("invalid restore statistics flags %#x", uint8(statistics))
	}

	result := make([]byte, 0, 4)
	if statistics&RestoreStatisticsElapsedTime != 0 {
		result = append(result, 'T')
	}
	if statistics&RestoreStatisticsDeltaTime != 0 {
		result = append(result, 'D')
	}
	if statistics&RestoreStatisticsPageReads != 0 {
		result = append(result, 'R')
	}
	if statistics&RestoreStatisticsPageWrites != 0 {
		result = append(result, 'W')
	}
	return string(result), nil
}

func validateRestorePageSize(pageSize int) error {
	switch pageSize {
	case 0, 1024, 2048, 4096, 8192, 16384, 32768:
		return nil
	default:
		return errors.New(
			"restore page size must be one of 0, 1024, 2048, 4096, 8192, 16384, or 32768",
		)
	}
}

func restoreOptionMask(options RestoreOptions) int32 {
	var mask int32
	if options.IgnoreBLRErrors {
		// Red Database reuses the backup ignore-checksums bit for gbak -IGNORE
		// during restore, where it means to ignore errors in object BLR.
		mask |= fbproto.Isc_spb_bkp_ignore_checksums
	}
	if options.Replace {
		mask |= fbproto.Isc_spb_res_replace
	} else {
		mask |= fbproto.Isc_spb_res_create
	}
	if options.DeactivateIndexes {
		mask |= fbproto.Isc_spb_res_deactivate_idx
	}
	if options.SkipShadows {
		mask |= fbproto.Isc_spb_res_no_shadow
	}
	if options.SkipValidityConstraints {
		mask |= fbproto.Isc_spb_res_no_validity
	}
	if options.CommitAfterEachTable {
		mask |= fbproto.Isc_spb_res_one_at_a_time
	}
	if options.UseAllPageSpace {
		mask |= fbproto.Isc_spb_res_use_all_space
	}
	if options.MetadataOnly {
		mask |= fbproto.Isc_spb_res_metadata_only
	}
	if options.DirectIO {
		mask |= fbproto.Isc_spb_res_direct_io
	}
	if options.UseOriginalTablespacePaths {
		mask |= fbproto.Isc_spb_res_ts_original_paths
	}
	return mask
}
