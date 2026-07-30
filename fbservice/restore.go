package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"

	"rdb.red-soft.ru/fbx/fbconn"
)

// RestoreOptions controls a logical restore performed by [Manager.Restore].
//
// The zero value creates a new read-write database using the server defaults
// and discards service output.
type RestoreOptions struct {
	// Output receives service output. A nil value discards the output. Calls to
	// [io.Writer.Write] are synchronous, so Output must not block indefinitely.
	// A writer shared by concurrent operations must be safe for concurrent use.
	Output io.Writer

	// Verbose requests progress output from the restore service.
	Verbose bool

	// Replace allows the restore service to replace an existing database. When
	// false, Restore creates a new database and fails if it already exists.
	Replace bool

	// ReadOnly marks the restored database as read-only.
	ReadOnly bool

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

	// PageBuffers sets the default number of database page buffers. A zero
	// value uses the value recorded in the backup.
	PageBuffers int

	// PageSize sets the restored database page size. A zero value uses the
	// value recorded in the backup.
	PageSize int

	// ParallelWorkers sets the number of workers used by the restore. A zero
	// value uses the server default. This option is available in Firebird 5 and
	// later.
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

	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeService(conn))
	}()

	return conn.Restore(ctx, backupFile, database, fbconn.RestoreOptions{
		Output:                  option.Output,
		Verbose:                 option.Verbose,
		Replace:                 option.Replace,
		ReadOnly:                option.ReadOnly,
		DeactivateIndexes:       option.DeactivateIndexes,
		SkipShadows:             option.SkipShadows,
		SkipValidityConstraints: option.SkipValidityConstraints,
		CommitAfterEachTable:    option.CommitAfterEachTable,
		UseAllPageSpace:         option.UseAllPageSpace,
		PageBuffers:             option.PageBuffers,
		PageSize:                option.PageSize,
		ParallelWorkers:         option.ParallelWorkers,
	})
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
