package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"

	"rdb.red-soft.ru/fbx/fbproto"
)

const (
	maxServiceActionStringLength = 1<<16 - 1
	maxServiceActionInt32        = 1<<31 - 1
)

// BackupOptions controls a logical backup created by [ServiceConn.Backup].
//
// The zero value selects the Firebird backup defaults and discards service
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

	// ConvertExternalTables converts external tables to regular tables.
	ConvertExternalTables bool

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
// server-side paths or aliases supported by the server. Backup blocks until
// the service action completes, the context cancels a network operation, or an
// error occurs. When [BackupOptions.Output] is non-nil, Backup writes service
// output to it as the action runs. Context cancellation cannot interrupt an
// [io.Writer.Write] call that has already started. Backup does not silently
// ignore unsupported options or retry without them; it returns the server
// error.
func (c *ServiceConn) Backup(
	ctx context.Context,
	database string,
	backupFile string,
	options BackupOptions,
) error {
	actionSPB, err := buildBackupActionSPB(database, backupFile, options)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, actionSPB, options.Output); err != nil {
		return fmt.Errorf("backup database: %w", err)
	}
	return nil
}

func buildBackupActionSPB(database string, backupFile string, options BackupOptions) ([]byte, error) {
	if database == "" {
		return nil, errors.New("backup database must not be empty")
	}
	if backupFile == "" {
		return nil, errors.New("backup file must not be empty")
	}

	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_action_svc_backup)
	if err := putServiceActionString(spb, fbproto.Isc_spb_dbname, database); err != nil {
		return nil, fmt.Errorf("backup database: %w", err)
	}
	if err := putServiceActionString(spb, fbproto.Isc_spb_bkp_file, backupFile); err != nil {
		return nil, fmt.Errorf("backup file: %w", err)
	}
	if options.Verbose {
		spb.AppendByte(fbproto.Isc_spb_verbose)
	}
	if options.Factor < 0 || options.Factor > maxServiceActionInt32 {
		return nil, fmt.Errorf("backup factor must be between 0 and %d", maxServiceActionInt32)
	}
	if options.Factor > 0 {
		spb.AppendByte(fbproto.Isc_spb_bkp_factor)
		spb.AppendInt32(int32(options.Factor))
	}
	if options.ParallelWorkers < 0 || options.ParallelWorkers > maxServiceActionInt32 {
		return nil, fmt.Errorf(
			"backup parallel workers must be between 0 and %d",
			maxServiceActionInt32,
		)
	}
	if options.ParallelWorkers > 0 {
		spb.AppendByte(fbproto.Isc_spb_bkp_parallel_workers)
		spb.AppendInt32(int32(options.ParallelWorkers))
	}
	if options.Replace {
		spb.AppendByte(fbproto.Isc_spb_bkp_replace)
	}
	if options.Compressor != "" {
		if err := putServiceActionString(spb, fbproto.Isc_spb_bkp_compression, options.Compressor); err != nil {
			return nil, fmt.Errorf("backup compressor: %w", err)
		}
	}
	spb.AppendByte(fbproto.Isc_spb_options)
	spb.AppendInt32(backupOptionMask(options))
	return spb.Bytes(), nil
}

func putServiceActionString(spb *fbproto.XPBWriter, tag byte, value string) error {
	valueBytes := []byte(value)
	if len(valueBytes) > maxServiceActionStringLength {
		return fmt.Errorf("value is %d bytes; maximum is %d", len(valueBytes), maxServiceActionStringLength)
	}
	spb.AppendByte(tag)
	spb.AppendInt16(int16(len(valueBytes)))
	spb.AppendBytes(valueBytes)
	return nil
}

func backupOptionMask(options BackupOptions) int32 {
	var mask int32
	if options.IgnoreChecksums {
		mask |= fbproto.Isc_spb_bkp_ignore_checksums
	}
	if options.IgnoreLimbo {
		mask |= fbproto.Isc_spb_bkp_ignore_limbo
	}
	if options.MetadataOnly {
		mask |= fbproto.Isc_spb_bkp_metadata_only
	}
	if options.NoGarbageCollect {
		mask |= fbproto.Isc_spb_bkp_no_garbage_collect
	}
	if options.OldDescriptions {
		mask |= fbproto.Isc_spb_bkp_old_descriptions
	}
	if options.NonTransportable {
		mask |= fbproto.Isc_spb_bkp_non_transportable
	}
	if options.ConvertExternalTables {
		mask |= fbproto.Isc_spb_bkp_convert
	}
	if options.Expand {
		mask |= fbproto.Isc_spb_bkp_expand
	}
	if options.NoDatabaseTriggers {
		mask |= fbproto.Isc_spb_bkp_no_triggers
	}
	if options.Zip {
		mask |= fbproto.Isc_spb_bkp_zip
	}
	return mask
}
