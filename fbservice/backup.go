package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"

	"rdb.red-soft.ru/fbx/fbconn"
)

const (
	maxBackupOptionStringLength = 1<<16 - 1
	maxBackupOptionInt32        = 1<<31 - 1
)

// BackupOptions controls a logical backup created by [Manager.Backup].
//
// The zero value selects the server backup defaults and discards service
// output.
type BackupOptions struct {
	// Output receives service output. A nil value discards the output. Calls to
	// [io.Writer.Write] are synchronous, so Output must not block indefinitely.
	Output io.Writer

	// Verbose requests progress output from the backup service.
	Verbose bool

	// IgnoreChecksums ignores checksum errors while reading database pages.
	IgnoreChecksums bool

	// IgnoreLimbo ignores transactions in limbo.
	IgnoreLimbo bool

	// MetadataOnly excludes table data from the backup.
	MetadataOnly bool

	// NoGarbageCollect disables garbage collection during the backup.
	NoGarbageCollect bool

	// OldDescriptions saves metadata descriptions in the legacy format.
	OldDescriptions bool

	// NonTransportable writes a backup in a platform-dependent format.
	NonTransportable bool

	// ConvertExternal converts external tables to regular tables.
	ConvertExternal bool

	// Expand disables backup data compression.
	Expand bool

	// Factor sets the blocking factor. A zero value uses the server default.
	Factor int

	// ParallelWorkers sets the number of workers used by the backup. A zero
	// value uses the server default. This option is available in Firebird 5 and
	// later.
	ParallelWorkers int

	// Replace allows the backup service to overwrite an existing backup file.
	// This option is available in RedDatabase 5.1 and later.
	Replace bool

	// NoDatabaseTriggers prevents database triggers from running during backup.
	NoDatabaseTriggers bool

	// Zip writes the backup in the server's compressed backup format. Firebird
	// 4 and later use zlib.
	Zip bool

	// Compressor specifies the server-side backup compression plugin and its
	// parameters. An empty value uses no explicit compressor. This option is
	// available in RedDatabase 5 and later.
	Compressor string
}

// Backup creates a logical backup of database at backupFile.
//
// The Firebird server interprets both database and backupFile. They can be
// server-side paths or aliases supported by the server. Backup opens a service
// connection, waits for the operation to finish, and closes the connection
// before returning. Backup validates its request before opening a service
// connection. Context cancellation cannot interrupt an [io.Writer.Write] call
// that has already started. Backup accepts at most one [BackupOptions] value.
// Backup does not silently ignore unsupported options or retry without them;
// it returns the server error.
func (m *Manager) Backup(
	ctx context.Context,
	database string,
	backupFile string,
	options ...BackupOptions,
) (err error) {
	option, err := getBackupOptions(options)
	if err != nil {
		return err
	}
	if err := validateBackupRequest(database, backupFile, option); err != nil {
		return err
	}

	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeService(conn))
	}()

	return conn.Backup(ctx, database, backupFile, fbconn.BackupOptions{
		Output:                option.Output,
		Verbose:               option.Verbose,
		IgnoreChecksums:       option.IgnoreChecksums,
		IgnoreLimbo:           option.IgnoreLimbo,
		MetadataOnly:          option.MetadataOnly,
		NoGarbageCollect:      option.NoGarbageCollect,
		OldDescriptions:       option.OldDescriptions,
		NonTransportable:      option.NonTransportable,
		ConvertExternalTables: option.ConvertExternal,
		Expand:                option.Expand,
		Factor:                option.Factor,
		ParallelWorkers:       option.ParallelWorkers,
		Replace:               option.Replace,
		NoDatabaseTriggers:    option.NoDatabaseTriggers,
		Zip:                   option.Zip,
		Compressor:            option.Compressor,
	})
}

func getBackupOptions(options []BackupOptions) (BackupOptions, error) {
	switch len(options) {
	case 0:
		return BackupOptions{}, nil
	case 1:
		return options[0], nil
	default:
		return BackupOptions{}, errors.New("backup accepts at most one BackupOptions value")
	}
}

func validateBackupRequest(database string, backupFile string, options BackupOptions) error {
	if database == "" {
		return errors.New("backup database must not be empty")
	}
	if backupFile == "" {
		return errors.New("backup file must not be empty")
	}
	if err := validateBackupOptionString("database", database); err != nil {
		return err
	}
	if err := validateBackupOptionString("file", backupFile); err != nil {
		return err
	}
	if err := validateBackupOptionString("compressor", options.Compressor); err != nil {
		return err
	}
	if options.Factor < 0 || options.Factor > maxBackupOptionInt32 {
		return fmt.Errorf("backup factor must be between 0 and %d", maxBackupOptionInt32)
	}
	if options.ParallelWorkers < 0 || options.ParallelWorkers > maxBackupOptionInt32 {
		return fmt.Errorf("backup parallel workers must be between 0 and %d", maxBackupOptionInt32)
	}
	return nil
}

func validateBackupOptionString(name string, value string) error {
	if len(value) > maxBackupOptionStringLength {
		return fmt.Errorf(
			"backup %s is %d bytes; maximum is %d",
			name,
			len(value),
			maxBackupOptionStringLength,
		)
	}
	return nil
}
