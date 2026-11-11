package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/firebirdsql/fbx/fbconn"
)

const minRestoreVerboseInterval = 100

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

// RestoreOptions controls a logical restore performed by [Manager.Restore].
//
// The zero value creates a new database using the server defaults, preserves
// the access and replica modes recorded in the backup, and discards service
// output.
type RestoreOptions struct {
	// Output receives service output. A nil value discards the output. Calls to
	// [io.Writer.Write] are synchronous, so Output must not block indefinitely.
	// A writer shared by concurrent operations must be safe for concurrent use.
	Output io.Writer

	// Verbose requests progress output from the restore service.
	Verbose bool

	// VerboseInterval requests progress output after every specified number of
	// restored records. A zero value disables interval-based output. Positive
	// values must be at least 100 and cannot be combined with Verbose.
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

	// KeyName overrides the encryption key name stored in the backup. KeyName
	// requires KeyHolder.
	KeyName string

	// CryptPlugin specifies the database encryption plugin used by the restored
	// database. CryptPlugin requires KeyHolder.
	CryptPlugin string

	// TablespaceMappingFile is a server-side Red Database 5.2 mapping file. It
	// is applied before TablespacePaths so explicit mappings can override file
	// entries. An empty value disables file-based tablespace mapping.
	TablespaceMappingFile string

	// TablespacePaths maps tablespaces to server-side paths. Names and paths
	// must not be empty. Entries are applied in slice order, so later duplicate
	// names can override earlier mappings. This option is available in Red
	// Database 5.2.
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
// server-side paths or aliases supported by the server. Restore opens a service
// connection, waits for the operation to finish, and closes the connection
// before returning. Restore validates its request before opening a service
// connection. Context cancellation cannot interrupt an [io.Writer.Write] call
// that has already started. Restore accepts at most one [RestoreOptions] value.
// Restore does not silently ignore unsupported options or retry without them;
// it returns the server error.
func (m *Manager) Restore(
	ctx context.Context,
	backupFile string,
	database string,
	options ...RestoreOptions,
) (err error) {
	option, err := getRestoreOptions(options)
	if err != nil {
		return err
	}
	if err := validateRestoreRequest(backupFile, database, option); err != nil {
		return err
	}
	connOptions, err := toFbconnRestoreOptions(option)
	if err != nil {
		return err
	}

	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeService(conn))
	}()

	return conn.Restore(ctx, backupFile, database, connOptions)
}

func getRestoreOptions(options []RestoreOptions) (RestoreOptions, error) {
	switch len(options) {
	case 0:
		return RestoreOptions{}, nil
	case 1:
		return options[0], nil
	default:
		return RestoreOptions{}, errors.New("restore accepts at most one RestoreOptions value")
	}
}

func toFbconnRestoreOptions(options RestoreOptions) (fbconn.RestoreOptions, error) {
	accessMode, err := toFbconnRestoreAccessMode(options.AccessMode)
	if err != nil {
		return fbconn.RestoreOptions{}, err
	}
	replicaMode, err := toFbconnRestoreReplicaMode(options.ReplicaMode)
	if err != nil {
		return fbconn.RestoreOptions{}, err
	}
	statistics, err := toFbconnRestoreStatistics(options.Statistics)
	if err != nil {
		return fbconn.RestoreOptions{}, err
	}

	var tablespacePaths []fbconn.RestoreTablespacePath
	if len(options.TablespacePaths) > 0 {
		tablespacePaths = make([]fbconn.RestoreTablespacePath, len(options.TablespacePaths))
		for i, tablespacePath := range options.TablespacePaths {
			tablespacePaths[i] = fbconn.RestoreTablespacePath{
				Name: tablespacePath.Name,
				Path: tablespacePath.Path,
			}
		}
	}

	return fbconn.RestoreOptions{
		Output:                     options.Output,
		Verbose:                    options.Verbose,
		VerboseInterval:            options.VerboseInterval,
		Statistics:                 statistics,
		Replace:                    options.Replace,
		AccessMode:                 accessMode,
		ReplicaMode:                replicaMode,
		DeactivateIndexes:          options.DeactivateIndexes,
		SkipShadows:                options.SkipShadows,
		SkipValidityConstraints:    options.SkipValidityConstraints,
		CommitAfterEachTable:       options.CommitAfterEachTable,
		UseAllPageSpace:            options.UseAllPageSpace,
		MetadataOnly:               options.MetadataOnly,
		DirectIO:                   options.DirectIO,
		IgnoreBLRErrors:            options.IgnoreBLRErrors,
		FixFSSData:                 options.FixFSSData,
		FixFSSMetadata:             options.FixFSSMetadata,
		SkipSchemaData:             options.SkipSchemaData,
		SkipData:                   options.SkipData,
		IncludeSchemaData:          options.IncludeSchemaData,
		IncludeData:                options.IncludeData,
		KeyHolder:                  options.KeyHolder,
		KeyName:                    options.KeyName,
		CryptPlugin:                options.CryptPlugin,
		TablespaceMappingFile:      options.TablespaceMappingFile,
		TablespacePaths:            tablespacePaths,
		UseOriginalTablespacePaths: options.UseOriginalTablespacePaths,
		PageBuffers:                options.PageBuffers,
		PageSize:                   options.PageSize,
		ParallelWorkers:            options.ParallelWorkers,
	}, nil
}

func validateRestoreRequest(
	backupFile string,
	database string,
	options RestoreOptions,
) error {
	if backupFile == "" {
		return errors.New("restore backup file must not be empty")
	}
	if database == "" {
		return errors.New("restore database must not be empty")
	}
	if err := validateRestoreOptionString("backup file", backupFile); err != nil {
		return err
	}
	if err := validateRestoreOptionString("database", database); err != nil {
		return err
	}
	optionStrings := []struct {
		name  string
		value string
	}{
		{"fix FSS data", options.FixFSSData},
		{"fix FSS metadata", options.FixFSSMetadata},
		{"skip schema data", options.SkipSchemaData},
		{"skip data", options.SkipData},
		{"include schema data", options.IncludeSchemaData},
		{"include data", options.IncludeData},
		{"key holder", options.KeyHolder},
		{"key name", options.KeyName},
		{"crypt plugin", options.CryptPlugin},
		{"tablespace mapping file", options.TablespaceMappingFile},
	}
	for _, option := range optionStrings {
		if err := validateRestoreOptionString(option.name, option.value); err != nil {
			return err
		}
	}
	for i, tablespacePath := range options.TablespacePaths {
		if tablespacePath.Name == "" {
			return fmt.Errorf("restore tablespace path %d name must not be empty", i+1)
		}
		if tablespacePath.Path == "" {
			return fmt.Errorf("restore tablespace path %d path must not be empty", i+1)
		}
		if err := validateRestoreOptionString(
			fmt.Sprintf("tablespace path %d name", i+1),
			tablespacePath.Name,
		); err != nil {
			return err
		}
		if err := validateRestoreOptionString(
			fmt.Sprintf("tablespace path %d path", i+1),
			tablespacePath.Path,
		); err != nil {
			return err
		}
	}
	if options.VerboseInterval < 0 || options.VerboseInterval > maxServiceActionInt32 {
		return fmt.Errorf(
			"restore verbose interval must be between 0 and %d",
			maxServiceActionInt32,
		)
	}
	if options.VerboseInterval > 0 && options.VerboseInterval < minRestoreVerboseInterval {
		return fmt.Errorf(
			"restore verbose interval must be 0 or at least %d",
			minRestoreVerboseInterval,
		)
	}
	if options.Verbose && options.VerboseInterval > 0 {
		return errors.New("restore verbose and verbose interval are mutually exclusive")
	}
	if options.KeyHolder == "" {
		if options.KeyName != "" {
			return errors.New("restore key name requires key holder")
		}
		if options.CryptPlugin != "" {
			return errors.New("restore crypt plugin requires key holder")
		}
	}
	if options.PageBuffers < 0 || options.PageBuffers > maxServiceActionInt32 {
		return fmt.Errorf(
			"restore page buffers must be between 0 and %d",
			maxServiceActionInt32,
		)
	}
	if err := validatePublicRestorePageSize(options.PageSize); err != nil {
		return err
	}
	if options.ParallelWorkers < 0 || options.ParallelWorkers > maxServiceActionInt32 {
		return fmt.Errorf(
			"restore parallel workers must be between 0 and %d",
			maxServiceActionInt32,
		)
	}
	return nil
}

func toFbconnRestoreAccessMode(mode RestoreAccessMode) (fbconn.RestoreAccessMode, error) {
	switch mode {
	case RestoreAccessModeFromBackup:
		return fbconn.RestoreAccessModeFromBackup, nil
	case RestoreAccessModeReadOnly:
		return fbconn.RestoreAccessModeReadOnly, nil
	case RestoreAccessModeReadWrite:
		return fbconn.RestoreAccessModeReadWrite, nil
	default:
		return 0, fmt.Errorf("invalid restore access mode %d", mode)
	}
}

func toFbconnRestoreReplicaMode(mode RestoreReplicaMode) (fbconn.RestoreReplicaMode, error) {
	switch mode {
	case RestoreReplicaModeFromBackup:
		return fbconn.RestoreReplicaModeFromBackup, nil
	case RestoreReplicaModeNone:
		return fbconn.RestoreReplicaModeNone, nil
	case RestoreReplicaModeReadOnly:
		return fbconn.RestoreReplicaModeReadOnly, nil
	case RestoreReplicaModeReadWrite:
		return fbconn.RestoreReplicaModeReadWrite, nil
	default:
		return 0, fmt.Errorf("invalid restore replica mode %d", mode)
	}
}

func toFbconnRestoreStatistics(statistics RestoreStatistics) (fbconn.RestoreStatistics, error) {
	if statistics&^RestoreStatisticsAll != 0 {
		return 0, fmt.Errorf("invalid restore statistics flags %#x", uint8(statistics))
	}

	var result fbconn.RestoreStatistics
	if statistics&RestoreStatisticsElapsedTime != 0 {
		result |= fbconn.RestoreStatisticsElapsedTime
	}
	if statistics&RestoreStatisticsDeltaTime != 0 {
		result |= fbconn.RestoreStatisticsDeltaTime
	}
	if statistics&RestoreStatisticsPageReads != 0 {
		result |= fbconn.RestoreStatisticsPageReads
	}
	if statistics&RestoreStatisticsPageWrites != 0 {
		result |= fbconn.RestoreStatisticsPageWrites
	}
	return result, nil
}

func validateRestoreOptionString(name string, value string) error {
	if len(value) > maxServiceActionStringLength {
		return fmt.Errorf(
			"restore %s is %d bytes; maximum is %d",
			name,
			len(value),
			maxServiceActionStringLength,
		)
	}
	return nil
}

func validatePublicRestorePageSize(pageSize int) error {
	switch pageSize {
	case 0, 1024, 2048, 4096, 8192, 16384, 32768:
		return nil
	default:
		return errors.New(
			"restore page size must be one of 0, 1024, 2048, 4096, 8192, 16384, or 32768",
		)
	}
}
