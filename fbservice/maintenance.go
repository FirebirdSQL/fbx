package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"rdb.red-soft.ru/fbx/fbconn"
)

// RepairOperation selects a gfix maintenance action. Zero is invalid.
type RepairOperation uint8

const (
	RepairValidate    RepairOperation = iota + 1 // Validate database pages.
	RepairSweep                                  // Remove obsolete record versions.
	RepairMend                                   // Mark corrupt records unavailable, potentially losing data.
	RepairKillShadows                            // Remove references to unavailable shadow files.
	RepairUpgradeODS                             // Upgrade ODS in place when supported by the server.
	RepairFixICU                                 // Update ICU collations and rebuild their indexes.
)

// RepairOptions controls [Manager.Repair]. Operation must be set explicitly.
// Full and ReadOnly apply only to validation. IgnoreChecksums applies to
// validation and mend. ParallelWorkers applies to sweep and ICU repair;
// SweepTimeout applies only to sweep.
// Output belongs to the caller and may contain a partial report on error.
// Shared writers must be safe for concurrent use.
type RepairOptions struct {
	Operation RepairOperation
	// Output receives the report. Nil discards it. Writes are synchronous.
	Output          io.Writer
	Full            bool // Validate records as well as pages.
	ReadOnly        bool // Check without repairing the database.
	IgnoreChecksums bool
	// ParallelWorkers sets the requested worker count (0..32767); zero omits
	// the parameter. The server configuration may impose a lower limit.
	// Sweep uses a legacy parameter on server versions before 5
	// and the standard parameter on 5+. ICU repair requires Firebird 5+ or compatible.
	// Servers without parallel sweep support return an error.
	ParallelWorkers int
	// SweepTimeout limits sweep on the server in whole seconds (0..2147483).
	// Zero omits the limit. This optional extension requires server support and is
	// sent only on server versions 6+; on older versions it is omitted.
	// It is independent of ctx.
	SweepTimeout int
}

// DatabaseAccessMode selects whether a database accepts writes.
type DatabaseAccessMode uint8

const (
	DatabaseAccessModeUnchanged DatabaseAccessMode = iota
	DatabaseAccessModeReadOnly
	DatabaseAccessModeReadWrite
)

// DatabaseWriteMode selects asynchronous or forced writes.
type DatabaseWriteMode uint8

const (
	DatabaseWriteModeUnchanged DatabaseWriteMode = iota
	DatabaseWriteModeAsync
	DatabaseWriteModeSync
)

// DatabaseMode selects the database's shutdown/online state.
type DatabaseMode uint8

const (
	DatabaseModeUnchanged DatabaseMode = iota
	DatabaseModeNormal
	DatabaseModeMulti
	DatabaseModeSingle
	DatabaseModeFull
)

// ShutdownMethod selects which activity to wait for before shutdown.
type ShutdownMethod uint8

const (
	ShutdownForce ShutdownMethod = iota + 1
	ShutdownDenyAttachments
	ShutdownDenyTransactions
)

// ShutdownOptions controls a database shutdown. Timeout is in whole seconds;
// zero requests immediate shutdown. Mode and Method must be set explicitly.
type ShutdownOptions struct {
	Mode    DatabaseMode
	Method  ShutdownMethod
	Timeout int
}

// DatabaseProperties contains gfix property changes. Nil pointers and zero
// enums omit the corresponding setting. Values pointed to must not be changed
// during the call. Exactly one property must be selected per call, matching
// gfix's mutually exclusive actions. No partial multi-property update is attempted.
// PageBuffers must be zero (server default) or at least 50.
// Output belongs to the caller; shared writers must be safe for concurrent use.
type DatabaseProperties struct {
	Output     io.Writer
	AccessMode DatabaseAccessMode
	WriteMode  DatabaseWriteMode
	// PageBuffers sets the database cache in pages; zero restores the server default.
	PageBuffers *int
	// SweepInterval sets the automatic sweep threshold; zero disables it.
	SweepInterval *int
	// SQLDialect is 1 or 3; zero leaves it unchanged.
	SQLDialect int
	// ReserveSpace selects whether to reserve space for record versions.
	ReserveSpace *bool
	Shutdown     *ShutdownOptions
	OnlineMode   DatabaseMode
	// ActivateShadow promotes database (the shadow file path) to a database.
	ActivateShadow bool
}

// Repair performs a gfix action on database, a server-side path or alias.
// It validates options before connecting and closes the service connection
// before returning. Validation may repair minor damage unless ReadOnly is set.
// Mend may lose data by marking corrupt records unavailable. UpgradeODS and
// FixICU require server support; unsupported actions return the server error.
// SweepTimeout is omitted on server versions before 6.
// Cancellation does not undo server effects or interrupt a blocked Output.Write.
// Database must not start with '-'; use ./-name.fdb for such a relative filename.
func (m *Manager) Repair(ctx context.Context, database string, options RepairOptions) (err error) {
	if err := validateMaintenanceDatabase(database); err != nil {
		return err
	}
	if options.Operation < RepairValidate || options.Operation > RepairFixICU {
		return errors.New("invalid repair operation")
	}
	if (options.Full || options.ReadOnly) && options.Operation != RepairValidate {
		return errors.New("full and read-only repair options require validation")
	}
	if options.IgnoreChecksums && options.Operation != RepairValidate && options.Operation != RepairMend {
		return errors.New("ignore checksums requires validation or mend")
	}
	// gfix stores the worker count in a signed 16-bit field.
	if options.ParallelWorkers < 0 || options.ParallelWorkers > 32767 {
		return errors.New("repair parallel workers must be between 0 and 32767")
	}
	if options.ParallelWorkers != 0 && options.Operation != RepairSweep && options.Operation != RepairFixICU {
		return errors.New("parallel workers require sweep or ICU repair")
	}
	if options.SweepTimeout < 0 || options.SweepTimeout > maxServiceActionInt32/1000 {
		return errors.New("sweep timeout must be between 0 and 2147483 seconds")
	}
	if options.SweepTimeout != 0 && options.Operation != RepairSweep {
		return errors.New("sweep timeout requires sweep")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return conn.Repair(ctx, database, fbconn.RepairOptions{
		Operation:       fbconn.RepairOperation(options.Operation),
		Output:          options.Output,
		Full:            options.Full,
		ReadOnly:        options.ReadOnly,
		IgnoreChecksums: options.IgnoreChecksums,
		ParallelWorkers: options.ParallelWorkers,
		SweepTimeout:    options.SweepTimeout,
	})
}

// SetDatabaseProperties changes one database property through gfix. The target
// is a server-side path or alias (the shadow path when ActivateShadow is set).
// It validates the request before connecting and closes its service connection.
// Unsupported options return server errors. Output and cancellation behave as
// in Repair. Shutdown.Timeout is a server timeout, independent of ctx.
func (m *Manager) SetDatabaseProperties(ctx context.Context, database string, options DatabaseProperties) (err error) {
	if err := validateMaintenanceDatabase(database); err != nil {
		return err
	}
	if err := validateDatabaseProperties(options); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	low := fbconn.DatabaseProperties{
		Output:         options.Output,
		AccessMode:     fbconn.DatabaseAccessMode(options.AccessMode),
		WriteMode:      fbconn.DatabaseWriteMode(options.WriteMode),
		PageBuffers:    options.PageBuffers,
		SweepInterval:  options.SweepInterval,
		SQLDialect:     options.SQLDialect,
		ReserveSpace:   options.ReserveSpace,
		OnlineMode:     fbconn.DatabaseMode(options.OnlineMode),
		ActivateShadow: options.ActivateShadow,
	}
	if s := options.Shutdown; s != nil {
		low.Shutdown = &fbconn.ShutdownOptions{Mode: fbconn.DatabaseMode(s.Mode), Method: fbconn.ShutdownMethod(s.Method), Timeout: s.Timeout}
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return conn.SetDatabaseProperties(ctx, database, low)
}

func validateMaintenanceDatabase(database string) error {
	if database == "" || strings.HasPrefix(database, "-") {
		return errors.New("maintenance database must not be empty or start with '-'")
	}
	if strings.IndexByte(database, 0) >= 0 {
		return errors.New("maintenance database must not contain NUL")
	}
	if len(database) > maxServiceActionStringLength {
		return errors.New("maintenance database exceeds maximum SPB string length")
	}
	return nil
}

func validateDatabaseProperties(options DatabaseProperties) error {
	if options.AccessMode > DatabaseAccessModeReadWrite {
		return errors.New("invalid database access mode")
	}
	if options.WriteMode > DatabaseWriteModeSync {
		return errors.New("invalid database write mode")
	}
	if options.OnlineMode > DatabaseModeSingle {
		return errors.New("invalid online database mode")
	}
	if options.SQLDialect != 0 && options.SQLDialect != 1 && options.SQLDialect != 3 {
		return errors.New("database SQL dialect must be 1 or 3")
	}
	if n := options.PageBuffers; n != nil && (*n < 0 || *n > maxServiceActionInt32 || (*n > 0 && *n < 50)) {
		return errors.New("database page buffers must be zero or between 50 and 2147483647")
	}
	if n := options.SweepInterval; n != nil && (*n < 0 || *n > maxServiceActionInt32) {
		return errors.New("database sweep interval must fit a nonnegative int32")
	}
	if s := options.Shutdown; s != nil {
		if s.Mode < DatabaseModeMulti || s.Mode > DatabaseModeFull {
			return errors.New("invalid shutdown database mode")
		}
		if s.Method < ShutdownForce || s.Method > ShutdownDenyTransactions {
			return errors.New("invalid database shutdown method")
		}
		if s.Timeout < 0 || s.Timeout > maxServiceActionInt32 {
			return errors.New("shutdown timeout must fit a nonnegative int32")
		}
	}
	count := 0
	for _, set := range []bool{
		options.AccessMode != DatabaseAccessModeUnchanged,
		options.WriteMode != DatabaseWriteModeUnchanged,
		options.PageBuffers != nil,
		options.SweepInterval != nil,
		options.SQLDialect != 0,
		options.ReserveSpace != nil,
		options.Shutdown != nil,
		options.OnlineMode != DatabaseModeUnchanged,
		options.ActivateShadow,
	} {
		if set {
			count++
		}
	}
	if count != 1 {
		return errors.New("exactly one database property must be selected")
	}
	return nil
}

// LimboTransactionAction selects how to finish a prepared transaction.
type LimboTransactionAction uint8

const (
	LimboCommit LimboTransactionAction = iota + 1
	LimboRollback
	// LimboRecover uses the states of other participants in a two-phase commit.
	LimboRecover
)

// LimboTransactions returns top-level IDs of transactions in limbo, including
// 64-bit IDs. Participant IDs are not returned as separate transactions.
// Use [Manager.LimboTransactionDetails] for participant states and recovery advice.
// On error no list is returned. The service connection is closed before return.
func (m *Manager) LimboTransactions(ctx context.Context, database string) (ids []int64, err error) {
	if err := validateMaintenanceDatabase(database); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeService(conn))
		if err != nil {
			ids = nil
		}
	}()
	return conn.LimboTransactions(ctx, database)
}

// ResolveLimboTransaction commits, rolls back or automatically recovers a
// prepared transaction. ID must be positive. Output may be nil to discard the
// report. Recovery can leave a transaction unresolved; inspect the report and
// query LimboTransactions afterwards. Unsupported actions return server errors.
// Resource ownership and cancellation behave as in Repair.
func (m *Manager) ResolveLimboTransaction(ctx context.Context, database string, id int64, action LimboTransactionAction, output io.Writer) (err error) {
	if err := validateMaintenanceDatabase(database); err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("limbo transaction ID must be positive")
	}
	if action < LimboCommit || action > LimboRecover {
		return errors.New("invalid limbo transaction action")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return conn.ResolveLimboTransaction(ctx, database, id, fbconn.LimboTransactionAction(action), output)
}
