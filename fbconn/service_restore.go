package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"

	"rdb.red-soft.ru/fbx/fbproto"
)

// RestoreOptions controls a logical restore performed by [ServiceConn.Restore].
//
// The zero value creates a new read-write database using the server defaults
// and discards service output.
type RestoreOptions struct {
	// Output receives service output. A nil value discards the output. Calls to
	// [io.Writer.Write] are synchronous, so Output must not block indefinitely.
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
	accessMode := byte(fbproto.Isc_spb_res_am_readwrite)
	if options.ReadOnly {
		accessMode = fbproto.Isc_spb_res_am_readonly
	}
	spb.AppendByte(fbproto.Isc_spb_res_access_mode)
	spb.AppendByte(accessMode)
	if options.Verbose {
		spb.AppendByte(fbproto.Isc_spb_verbose)
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
	spb.AppendByte(fbproto.Isc_spb_options)
	spb.AppendInt32(restoreOptionMask(options))
	return spb.Bytes(), nil
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
	return mask
}
