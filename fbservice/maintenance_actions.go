package fbservice

import (
	"context"
	"errors"
	"io"
)

// ValidateOptions controls [Manager.Validate]. Output must be set explicitly.
// Validation may repair minor damage unless ReadOnly is set.
// Inspect the report even if the call succeeds.
type ValidateOptions struct {
	// Output receives the report and must not be nil. Use io.Discard to explicitly
	// discard the report. Writes are synchronous; shared writers must be safe
	// for concurrent use. The caller owns Output, which may contain a partial
	// report on error.
	Output io.Writer
	// Full validates records as well as pages.
	Full bool
	// ReadOnly checks the database without repairing it.
	ReadOnly bool
	// IgnoreChecksums ignores checksum errors while reading database pages.
	IgnoreChecksums bool
}

// Validate validates database, a server-side path or alias. It accepts at most
// one [ValidateOptions] value and runs [Manager.Repair] with RepairValidate.
// Options must include a non-nil Output; use io.Discard to discard the report.
// Request validation, connection ownership and cancellation behave as in Repair.
func (m *Manager) Validate(ctx context.Context, database string, options ...ValidateOptions) error {
	if len(options) > 1 {
		return errors.New("validate accepts at most one ValidateOptions value")
	}
	var option ValidateOptions
	if len(options) == 1 {
		option = options[0]
	}
	return m.Repair(ctx, database, RepairOptions{
		Operation:       RepairValidate,
		Output:          option.Output,
		Full:            option.Full,
		ReadOnly:        option.ReadOnly,
		IgnoreChecksums: option.IgnoreChecksums,
	})
}

// SweepOptions controls [Manager.Sweep]. The zero value selects the server's
// sweep defaults and discards service output.
type SweepOptions struct {
	// Output receives service output. Nil discards it. Writes are synchronous;
	// shared writers must be safe for concurrent use. The caller owns Output,
	// which may contain partial output on error.
	Output io.Writer
	// IgnoreChecksums ignores checksum errors while reading database pages.
	IgnoreChecksums bool
	// ParallelWorkers requests 0..32767 workers. Zero omits the parameter.
	// The server configuration may impose a lower limit. Unsupported parallel
	// sweep requests return server errors.
	ParallelWorkers int
	// Timeout limits sweep on the server in whole seconds (0..2147483).
	// Zero omits the limit. This optional extension is sent only on server
	// versions 6+ and is omitted on older versions. It is independent of ctx.
	Timeout int
}

// Sweep removes obsolete record versions from database, a server-side path or
// alias. It accepts at most one [SweepOptions] value and runs [Manager.Repair]
// with RepairSweep. Request validation, connection ownership and cancellation
// behave as in Repair.
func (m *Manager) Sweep(ctx context.Context, database string, options ...SweepOptions) error {
	if len(options) > 1 {
		return errors.New("sweep accepts at most one SweepOptions value")
	}
	var option SweepOptions
	if len(options) == 1 {
		option = options[0]
	}
	return m.Repair(ctx, database, RepairOptions{
		Operation:       RepairSweep,
		Output:          option.Output,
		IgnoreChecksums: option.IgnoreChecksums,
		ParallelWorkers: option.ParallelWorkers,
		SweepTimeout:    option.Timeout,
	})
}

// Shutdown changes database to options.Mode using options.Method. Database is
// a server-side path or alias. Mode and Method must be set explicitly; Timeout
// is a server timeout in seconds, independent of ctx.
// It runs [Manager.SetDatabaseProperties] with only the shutdown property set.
// Validation, connection ownership and cancellation behave as in that method.
// Service output is discarded. To capture it, use SetDatabaseProperties with
// its Output and Shutdown fields.
func (m *Manager) Shutdown(ctx context.Context, database string, options ShutdownOptions) error {
	return m.SetDatabaseProperties(ctx, database, DatabaseProperties{Shutdown: &options})
}
