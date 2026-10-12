/*
Package fbservice provides a high-level API for Firebird services.

A Manager stores connection settings and opens a separate service connection
for each service action. Create a manager directly from a connection string:

	manager, err := fbservice.NewManager(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}

To customize settings, parse the configuration before creating the manager:

	config, err := fbservice.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	manager, err := fbservice.NewManagerConfig(config)
	if err != nil {
		return err
	}

Backup and Restore without special settings can omit their options:

	err = manager.Backup(ctx, database, backupFile)

Pass BackupOptions to request verbose output or change backup behavior:

	err = manager.Backup(ctx, database, backupFile, fbservice.BackupOptions{
		Output:          os.Stdout,
		Verbose:         true,
		IgnoreLimbo:     true,
		ParallelWorkers: 4,
	})

Restore a server-side backup into a new database:

	err = manager.Restore(ctx, backupFile, database, fbservice.RestoreOptions{
		Output:  os.Stdout,
		Verbose: true,
	})

Service output can also be captured in memory:

	var output bytes.Buffer
	err = manager.Restore(ctx, backupFile, database, fbservice.RestoreOptions{
		Output:  &output,
		Verbose: true,
	})

Set Replace explicitly to replace an existing database:

	err = manager.Restore(ctx, backupFile, database, fbservice.RestoreOptions{
		Replace: true,
	})

Set AccessMode to override the access mode recorded in the backup:

	err = manager.Restore(ctx, backupFile, database, fbservice.RestoreOptions{
		AccessMode: fbservice.RestoreAccessModeReadOnly,
	})

Collect a gstat report with the same Manager:

	err = manager.Statistics(ctx, database, fbservice.StatisticsOptions{
		Output:         os.Stdout,
		DataPages:      true,
		RecordVersions: true,
		Tables:         []string{"EMPLOYEE"},
	})

Statistics supports Output, DataPages, HeaderPages, IndexPages,
SystemRelations, RecordVersions, Tables, Schemas, NoCreation, Encryption and
ParallelWorkers. Unsupported server options return errors. Output is required;
use io.Discard to explicitly discard the report. Leaving all other options at
their zero values selects the server's default data and index analysis.

Start a server trace session using configuration text and stream its output:

	err = manager.TraceStart(ctx, "application trace", configuration, fbservice.TraceOptions{
		Output: os.Stdout,
	})

TraceStart blocks until the session stops, the context is canceled, or an error
occurs. Run it in a goroutine when managing the session concurrently. It closes
its connection before returning, which ends the user trace session. The writer
remains open and receives the server output unchanged, including the session ID.

To start reading in the background and obtain the session ID directly:

	session, err := manager.TraceStartSession(ctx, "application trace", configuration, fbservice.TraceOptions{
		Output: os.Stdout,
	})
	if err != nil {
		return err
	}
	defer session.Cancel()

	// Run application work while the session collects events.
	if err := manager.TraceStop(ctx, session.ID()); err != nil {
		session.Cancel()
		return errors.Join(err, session.Wait())
	}
	return session.Wait()

TraceStartSession waits for the startup message before returning. The context
controls the entire session lifetime. Cancel requests cancellation; Wait waits
for connection cleanup and returns background errors, including context errors
on cancellation. Done supports waiting with select. The output writer remains
open; do not close it or inspect an unsynchronized buffer until Wait completes.
Each returned session has its own ID, even when session names are identical.

List sessions, then use the desired session ID to control one:

	err = manager.TraceList(ctx, fbservice.TraceOptions{Output: os.Stdout})
	err = manager.TraceSuspend(ctx, sessionID, fbservice.TraceOptions{Output: os.Stdout})
	err = manager.TraceResume(ctx, sessionID, fbservice.TraceOptions{Output: os.Stdout})
	err = manager.TraceStop(ctx, sessionID, fbservice.TraceOptions{Output: os.Stdout})

Each call uses a separate service connection. TraceList returns the server's
textual report, including IDs and state flags. Output is required for TraceStart,
TraceStartSession and TraceList; use io.Discard to explicitly discard it. Output is optional for
the control operations. Some server diagnostics, such as a missing session or
insufficient permissions, are returned as text instead of an error.

Trace operations require a single host and port and reject fallback servers
before connecting. Use a separate manager for each server: trace session IDs
are local to that server, and the host must identify the same server for every
operation. Other service operations continue to support fallback servers.
The complete encoded trace start parameter block must not exceed 65535 bytes.
This includes 4 bytes of overhead plus, for a nonempty name, another 3 bytes.

Managers do not write to standard output unless an operation explicitly
receives os.Stdout as its output.

Validate a database with Validate:

	err = manager.Validate(ctx, database, fbservice.ValidateOptions{
		Full:     true,
		ReadOnly: true,
		Output:   os.Stdout,
	})

Validate and Sweep accept at most one options value. Validate requires a
non-nil Output; use io.Discard to explicitly discard the report. Sweep without
options selects the server defaults and discards service output. Validation may
repair minor damage unless ReadOnly is set. Inspect validation reports even if
the call succeeds.

Repair remains available for all maintenance actions, including RepairValidate,
RepairSweep, RepairMend, RepairKillShadows,
RepairUpgradeODS and RepairFixICU. Mend can discard damaged records.
ODS upgrades require server support (Firebird 5+); ICU repair requires
Firebird 3+. Unsupported actions return server errors.
RepairValidate and RepairMend require a non-nil Output; use io.Discard to
explicitly discard their reports. IgnoreChecksums is accepted for validation,
sweep and mend, including through ValidateOptions and SweepOptions.

Sweep supports parallel execution on Firebird 5+:

	err = manager.Sweep(ctx, database, fbservice.SweepOptions{
		ParallelWorkers: 4,
	})

For a sweep using the server defaults, call manager.Sweep(ctx, database).

ParallelWorkers accepts 0..32767; zero omits the parameter. Server configuration
may impose a lower limit. For compatibility, sweep uses a legacy parameter on
server versions before 5 and the standard parameter on 5+.

SweepOptions.Timeout (RepairOptions.SweepTimeout) is an optional server extension
that limits sweep in whole seconds.
It requires server support and is sent only on server versions 6+; on older
servers it is omitted. Its range is 0..2147483 seconds, with zero omitting the
limit. This is independent of the context deadline.

Change one database property per call. Pointer values distinguish an omitted
property from an explicit zero, which disables automatic sweep or resets the
database cache to its server default:

	err = manager.SetDatabaseProperties(ctx, database, fbservice.DatabaseProperties{
		SweepInterval: new(0),
	})

Other properties include AccessMode, WriteMode (asynchronous or forced writes),
PageBuffers, SQLDialect, ReserveSpace, Shutdown and OnlineMode. ActivateShadow
uses the shadow file path as the database argument. For example, shut down a
database immediately and later return it to normal operation:

	err = manager.Shutdown(ctx, database, fbservice.ShutdownOptions{
		Mode:   fbservice.DatabaseModeFull,
		Method: fbservice.ShutdownForce,
	})
	err = manager.SetDatabaseProperties(ctx, database, fbservice.DatabaseProperties{
		OnlineMode: fbservice.DatabaseModeNormal,
	})

LimboTransactions returns transaction IDs (including 64-bit IDs).
LimboTransactionDetails returns structured information about all transactions:
their top-level ID, the Distributed flag, Participants and recovery Advice.
Each participant has an ID, State, HostSite, RemoteSite and DatabasePath.
Missing or unavailable states are LimboStateUnknown; missing or unavailable
advice is LimboAdviceUnknown. Missing participant IDs and strings are zero values.
The driver does not infer committed or rolled-back states from missing fields.
Advice is the server's recommendation, not confirmation of a completed action.

After deciding the appropriate outcome, pass an ID to ResolveLimboTransaction
with LimboCommit, LimboRollback or LimboRecover. LimboRecover requires a non-nil
output writer; use io.Discard to explicitly discard the report. Output is
optional for LimboCommit and LimboRollback. The same rule applies to
ResolveLimboTransactionChecked.
A nil error confirms that the service action completed without an error.
Automatic recovery uses other participants' states and can leave a transaction
unresolved.

ResolveLimboTransactionChecked runs the action once, closes its service
connection, then obtains a fresh limbo list using another service connection:

	result, err := manager.ResolveLimboTransactionChecked(ctx, database, id, LimboRecover, io.Discard)

On a nil error, result.Status is LimboResolutionNotListed if the ID is absent
from the returned limbo list, or LimboResolutionUnresolved if it remains.
An unresolved result includes the remaining details in result.Transaction.
Servers may limit the list without signalling truncation. Absence therefore
does not prove commit, rollback or completion in this or other participant
databases. The list is a snapshot and can change when another administrator
resolves a transaction.

If the action or subsequent check fails, Status is LimboResolutionUnknown and
Transaction is nil. In particular, an error during the check does not imply
that the preceding action failed or that it is safe to repeat. Context
cancellation does not roll back maintenance actions already performed by the
server. Errors preserve their causes for errors.Is and errors.As, but
fbconn.SafeToRetry is false for this compound call.
*/
package fbservice
