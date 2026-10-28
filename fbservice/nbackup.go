package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/firebirdsql/fbx/fbconn"
)

const maxNBackupLevel = 1<<15 - 1

var nbackupGUIDPattern = regexp.MustCompile(`^\{[[:xdigit:]]{8}(-[[:xdigit:]]{4}){3}-[[:xdigit:]]{12}\}$`)

// NBackupOptions controls a physical backup created by [Manager.NBackup].
// The zero value creates a full backup and discards service output.
type NBackupOptions struct {
	// Output receives service output synchronously. Nil discards it. A Write
	// call must not block indefinitely and cannot be interrupted by cancellation.
	// A writer shared by concurrent calls must be safe for concurrent use.
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

// NBackup creates a physical backup of database at backupFile. The server
// interprets both paths and refuses to overwrite an existing backup file.
// It validates the request before connecting, opens a separate service connection,
// waits for completion and closes the connection. It accepts at most one options
// value. Unsupported options return server errors; the operation is not retried.
// A valid request with an already canceled context returns before connecting.
// Cancellation does not undo a backup already started.
func (m *Manager) NBackup(ctx context.Context, database, backupFile string, options ...NBackupOptions) (err error) {
	var option NBackupOptions
	switch len(options) {
	case 0:
	case 1:
		option = options[0]
	default:
		return errors.New("nbackup accepts at most one NBackupOptions value")
	}
	if err := validateNBackupRequest(database, backupFile, option); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return conn.NBackup(ctx, database, backupFile, fbconn.NBackupOptions{
		Output:             option.Output,
		Level:              option.Level,
		GUID:               option.GUID,
		NoDatabaseTriggers: option.NoDatabaseTriggers,
		DirectIO:           option.DirectIO,
		CleanHistory:       option.CleanHistory,
		KeepDays:           option.KeepDays,
		KeepRows:           option.KeepRows,
	})
}

// NRestoreOptions controls [Manager.NRestore]. The zero value restores to a
// new database, resets its GUID and replication sequence, and discards output.
type NRestoreOptions struct {
	// Output receives service output synchronously; nil discards it. Write
	// must not block indefinitely and cannot be interrupted by cancellation.
	// A writer shared by concurrent calls must be safe for concurrent use.
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
// Cancellation and errors can leave a partially restored database. NRestore
// validates before connecting, waits for completion and closes its connection.
// It accepts at most one options value and does not retry unsupported options.
// A valid request with an already canceled context returns before connecting.
// The caller must not modify backupFiles until NRestore returns.
func (m *Manager) NRestore(ctx context.Context, backupFiles []string, database string, options ...NRestoreOptions) (err error) {
	var option NRestoreOptions
	switch len(options) {
	case 0:
	case 1:
		option = options[0]
	default:
		return errors.New("nrestore accepts at most one NRestoreOptions value")
	}
	if err := validateNRestoreRequest(backupFiles, database, option); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return conn.NRestore(ctx, backupFiles, database, fbconn.NRestoreOptions{
		Output:             option.Output,
		NoDatabaseTriggers: option.NoDatabaseTriggers,
		InPlace:            option.InPlace,
		PreserveSequence:   option.PreserveSequence,
	})
}

// NFixupOptions controls [Manager.NFixup]. The zero value resets the copied
// database's GUID and replication sequence and discards service output.
type NFixupOptions struct {
	// Output receives service output synchronously; nil discards it. Write
	// must not block indefinitely and cannot be interrupted by cancellation.
	// A writer shared by concurrent calls must be safe for concurrent use.
	Output io.Writer
	// PreserveSequence retains the original database GUID and replication sequence.
	PreserveSequence bool
}

// NFixup makes a physical copy of a database in backup mode usable. It requires
// Firebird 4+. Use it on a copy made after ALTER DATABASE BEGIN BACKUP.
// It does not merge the delta file: applying it to the live source can lose
// changes. Use ALTER DATABASE END BACKUP to unlock the live source instead.
// NFixup validates before connecting, waits for completion and closes its
// connection. It accepts at most one options value and never retries the action.
// A valid request with an already canceled context returns before connecting.
func (m *Manager) NFixup(ctx context.Context, database string, options ...NFixupOptions) (err error) {
	var option NFixupOptions
	switch len(options) {
	case 0:
	case 1:
		option = options[0]
	default:
		return errors.New("nfixup accepts at most one NFixupOptions value")
	}
	if err := validateNBackupString("database", database); err != nil {
		return err
	}
	length := 4 + len(database)
	if option.PreserveSequence {
		length += 5
	}
	if err := validateNBackupSPBLength(length); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return conn.NFixup(ctx, database, fbconn.NFixupOptions{
		Output:           option.Output,
		PreserveSequence: option.PreserveSequence,
	})
}

func validateNBackupRequest(database, backupFile string, options NBackupOptions) error {
	if err := validateNBackupString("database", database); err != nil {
		return err
	}
	if err := validateNBackupString("backup file", backupFile); err != nil {
		return err
	}
	if backupFile == "stdout" {
		return errors.New("nbackup requires a server-side backup file, not stdout")
	}
	if options.Level < 0 || options.Level > maxNBackupLevel {
		return fmt.Errorf("nbackup level must be between 0 and %d", maxNBackupLevel)
	}
	if options.GUID != "" {
		if options.Level != 0 {
			return errors.New("nbackup level and GUID are mutually exclusive")
		}
		if !nbackupGUIDPattern.MatchString(options.GUID) {
			return errors.New("nbackup GUID must have the form {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}")
		}
	}
	if options.KeepDays < 0 || options.KeepDays > maxServiceActionInt32 ||
		options.KeepRows < 0 || options.KeepRows > maxServiceActionInt32 {
		return fmt.Errorf("nbackup keep days and keep rows must be between 0 and %d", maxServiceActionInt32)
	}
	if options.KeepDays > 0 && options.KeepRows > 0 {
		return errors.New("nbackup keep days and keep rows are mutually exclusive")
	}
	if options.CleanHistory != (options.KeepDays > 0 || options.KeepRows > 0) {
		return errors.New("nbackup clean history requires exactly one of keep days or keep rows, and vice versa")
	}

	// Action byte, two string headers and the level tag with an int32 value.
	length := 12 + len(database) + len(backupFile)
	if options.GUID != "" {
		length += len(options.GUID) - 2 // GUID string replaces the level parameter.
	}
	if options.NoDatabaseTriggers {
		length += 5
	}
	if options.DirectIO != nil {
		length += 5 // String header and "ON".
		if !*options.DirectIO {
			length++
		}
	}
	if options.CleanHistory {
		length += 6
	}
	return validateNBackupSPBLength(length)
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

func validateNRestoreRequest(backupFiles []string, database string, options NRestoreOptions) error {
	if err := validateNBackupString("database", database); err != nil {
		return err
	}
	if len(backupFiles) == 0 {
		return errors.New("nrestore requires at least one backup file")
	}
	length := 4 + len(database)
	if options.NoDatabaseTriggers || options.InPlace || options.PreserveSequence {
		length += 5
	}
	for i, file := range backupFiles {
		if err := validateNBackupString("backup file", file); err != nil {
			return fmt.Errorf("nrestore file %d: %w", i+1, err)
		}
		if file == "stdin" {
			return errors.New("nrestore requires server-side backup files, not stdin")
		}
		length += 3 + len(file)
		if err := validateNBackupSPBLength(length); err != nil {
			return err
		}
	}
	return nil
}

func validateNBackupSPBLength(length int) error {
	if length > maxServiceActionStringLength {
		return fmt.Errorf("nbackup action SPB exceeds %d bytes", maxServiceActionStringLength)
	}
	return nil
}
