package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/firebirdsql/fbx/fbproto"
)

const maxNBackupLevel = 1<<15 - 1

var nbackupGUIDPattern = regexp.MustCompile(`^\{[[:xdigit:]]{8}(-[[:xdigit:]]{4}){3}-[[:xdigit:]]{12}\}$`)

// NBackupOptions controls a physical backup created by [ServiceConn.NBackup].
// The zero value creates a full backup and discards service output.
type NBackupOptions struct {
	// Output receives service output synchronously. Nil discards it. A Write
	// call must not block indefinitely and cannot be interrupted by context cancellation.
	Output io.Writer
	// Level is 0 for a full backup or 1..32767 for an incremental backup based
	// on the most recent backup of the preceding level.
	Level int
	// GUID selects a previous backup instead of Level, which must be zero.
	// Use its RDB$BACKUP_HISTORY.RDB$GUID value, including braces. Requires
	// Firebird 4+.
	GUID string
	// NoDatabaseTriggers prevents database triggers from running during backup.
	NoDatabaseTriggers bool
	// DirectIO overrides the server's platform-specific direct I/O default.
	// Nil leaves the default unchanged; false explicitly disables direct I/O.
	DirectIO *bool
	// CleanHistory deletes old RDB$BACKUP_HISTORY records after backup. Set
	// exactly one of KeepDays or KeepRows. Requires Firebird 4.0.3+.
	CleanHistory bool
	// KeepDays retains history from this many recent days; zero omits it.
	KeepDays int
	// KeepRows retains this many recent history rows, including the new backup;
	// zero omits it. Removing a base backup's history can break future backups.
	KeepRows int
}

// NBackup creates a server-side physical backup. Paths are interpreted by the
// server. It waits for completion and returns unsupported-option server errors
// without retrying. Cancellation does not undo a backup already started.
func (c *ServiceConn) NBackup(ctx context.Context, database, backupFile string, options NBackupOptions) error {
	spb, err := buildNBackupActionSPB(database, backupFile, options)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, spb, options.Output); err != nil {
		return fmt.Errorf("nbackup database: %w", err)
	}
	return nil
}

// NRestoreOptions controls [ServiceConn.NRestore]. The zero value restores to a
// new database, resets its GUID and replication sequence, and discards output.
type NRestoreOptions struct {
	// Output receives service output synchronously; nil discards it. Write
	// must not block indefinitely and cannot be interrupted by cancellation.
	Output io.Writer
	// NoDatabaseTriggers suppresses database triggers during in-place restore.
	NoDatabaseTriggers bool
	// InPlace applies incremental files to an existing database. Requires
	// Firebird 4+. An incorrect base or chain can corrupt the target.
	InPlace bool
	// PreserveSequence retains the original database GUID and replication
	// sequence. Requires Firebird 4+.
	PreserveSequence bool
}

// NRestore restores backupFiles in the supplied order: full backup first, then
// its incremental backups. Paths belong to the server. By default the target
// must not exist; InPlace explicitly permits modifying an existing database.
// Cancellation and errors can leave a partially restored database.
func (c *ServiceConn) NRestore(ctx context.Context, backupFiles []string, database string, options NRestoreOptions) error {
	spb, err := buildNRestoreActionSPB(backupFiles, database, options)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, spb, options.Output); err != nil {
		return fmt.Errorf("nrestore database: %w", err)
	}
	return nil
}

// NFixupOptions controls [ServiceConn.NFixup]. The zero value resets the copied
// database's GUID and replication sequence and discards service output.
type NFixupOptions struct {
	// Output receives service output synchronously; nil discards it. Write
	// must not block indefinitely and cannot be interrupted by cancellation.
	Output io.Writer
	// PreserveSequence retains the original database GUID and replication sequence.
	PreserveSequence bool
}

// NFixup makes a physical copy of a database in backup mode usable. It requires
// Firebird 4+. Use it on a copy made after ALTER DATABASE BEGIN BACKUP.
// It does not merge the delta file: applying it to the live source can lose
// changes. Use ALTER DATABASE END BACKUP to unlock the live source instead.
func (c *ServiceConn) NFixup(ctx context.Context, database string, options NFixupOptions) error {
	spb, err := buildNFixupActionSPB(database, options)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, spb, options.Output); err != nil {
		return fmt.Errorf("nfixup database: %w", err)
	}
	return nil
}

func buildNBackupActionSPB(database, backupFile string, options NBackupOptions) ([]byte, error) {
	if err := validateNBackupString("database", database); err != nil {
		return nil, err
	}
	if err := validateNBackupString("backup file", backupFile); err != nil {
		return nil, err
	}
	if backupFile == "stdout" {
		return nil, errors.New("nbackup requires a server-side backup file, not stdout")
	}
	if options.Level < 0 || options.Level > maxNBackupLevel {
		return nil, fmt.Errorf("nbackup level must be between 0 and %d", maxNBackupLevel)
	}
	if options.GUID != "" {
		if options.Level != 0 {
			return nil, errors.New("nbackup level and GUID are mutually exclusive")
		}
		if !nbackupGUIDPattern.MatchString(options.GUID) {
			return nil, errors.New("nbackup GUID must have the form {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}")
		}
	}
	if options.KeepDays < 0 || options.KeepDays > maxServiceActionInt32 ||
		options.KeepRows < 0 || options.KeepRows > maxServiceActionInt32 {
		return nil, fmt.Errorf("nbackup keep days and keep rows must be between 0 and %d", maxServiceActionInt32)
	}
	if options.KeepDays > 0 && options.KeepRows > 0 {
		return nil, errors.New("nbackup keep days and keep rows are mutually exclusive")
	}
	if options.CleanHistory != (options.KeepDays > 0 || options.KeepRows > 0) {
		return nil, errors.New("nbackup clean history requires exactly one of keep days or keep rows, and vice versa")
	}

	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_action_svc_nbak)
	if err := putServiceActionString(spb, fbproto.Isc_spb_dbname, database); err != nil {
		return nil, err
	}
	if err := putServiceActionString(spb, fbproto.Isc_spb_nbk_file, backupFile); err != nil {
		return nil, err
	}
	if options.GUID == "" {
		spb.AppendByte(fbproto.Isc_spb_nbk_level).AppendInt32(int32(options.Level))
	} else {
		if err := putServiceActionString(spb, fbproto.Isc_spb_nbk_guid, options.GUID); err != nil {
			return nil, err
		}
	}
	if options.NoDatabaseTriggers {
		spb.AppendByte(fbproto.Isc_spb_options).AppendInt32(fbproto.Isc_spb_nbk_no_triggers)
	}
	if options.DirectIO != nil {
		value := "OFF"
		if *options.DirectIO {
			value = "ON"
		}
		if err := putServiceActionString(spb, fbproto.Isc_spb_nbk_direct, value); err != nil {
			return nil, err
		}
	}
	if options.CleanHistory {
		spb.AppendByte(fbproto.Isc_spb_nbk_clean_history)
		if options.KeepDays > 0 {
			spb.AppendByte(fbproto.Isc_spb_nbk_keep_days).AppendInt32(int32(options.KeepDays))
		} else {
			spb.AppendByte(fbproto.Isc_spb_nbk_keep_rows).AppendInt32(int32(options.KeepRows))
		}
	}
	if err := validateNBackupSPBLength(len(spb.Bytes())); err != nil {
		return nil, err
	}
	return spb.Bytes(), nil
}

func validateNBackupString(name, value string) error {
	if value == "" || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("nbackup %s must not be empty or contain NUL", name)
	}
	if len(value) > maxServiceActionStringLength {
		return fmt.Errorf("nbackup %s exceeds %d bytes", name, maxServiceActionStringLength)
	}
	return nil
}

func buildNRestoreActionSPB(backupFiles []string, database string, options NRestoreOptions) ([]byte, error) {
	if err := validateNBackupString("database", database); err != nil {
		return nil, err
	}
	if len(backupFiles) == 0 {
		return nil, errors.New("nrestore requires at least one backup file")
	}
	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_action_svc_nrest)
	if err := putServiceActionString(spb, fbproto.Isc_spb_dbname, database); err != nil {
		return nil, err
	}
	for i, file := range backupFiles {
		if err := validateNBackupString("backup file", file); err != nil {
			return nil, fmt.Errorf("nrestore file %d: %w", i+1, err)
		}
		if file == "stdin" {
			return nil, errors.New("nrestore requires server-side backup files, not stdin")
		}
		if err := putServiceActionString(spb, fbproto.Isc_spb_nbk_file, file); err != nil {
			return nil, err
		}
		if err := validateNBackupSPBLength(len(spb.Bytes())); err != nil {
			return nil, err
		}
	}
	var mask int32
	if options.NoDatabaseTriggers {
		mask |= fbproto.Isc_spb_nbk_no_triggers
	}
	if options.InPlace {
		mask |= fbproto.Isc_spb_nbk_inplace
	}
	if options.PreserveSequence {
		mask |= fbproto.Isc_spb_nbk_sequence
	}
	if mask != 0 {
		spb.AppendByte(fbproto.Isc_spb_options).AppendInt32(mask)
	}
	if err := validateNBackupSPBLength(len(spb.Bytes())); err != nil {
		return nil, err
	}
	return spb.Bytes(), nil
}

func buildNFixupActionSPB(database string, options NFixupOptions) ([]byte, error) {
	if err := validateNBackupString("database", database); err != nil {
		return nil, err
	}
	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_action_svc_nfix)
	if err := putServiceActionString(spb, fbproto.Isc_spb_dbname, database); err != nil {
		return nil, err
	}
	if options.PreserveSequence {
		spb.AppendByte(fbproto.Isc_spb_options).AppendInt32(fbproto.Isc_spb_nbk_sequence)
	}
	if err := validateNBackupSPBLength(len(spb.Bytes())); err != nil {
		return nil, err
	}
	return spb.Bytes(), nil
}

func validateNBackupSPBLength(length int) error {
	if length > maxServiceActionStringLength {
		return fmt.Errorf("nbackup action SPB exceeds %d bytes", maxServiceActionStringLength)
	}
	return nil
}
