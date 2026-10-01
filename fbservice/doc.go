/*
Package fbservice provides a high-level API for Firebird and RedDatabase
services.

A Manager stores connection settings and opens a separate service connection
for each operation. Create a manager directly from a connection string:

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
*/
package fbservice
