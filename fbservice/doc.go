/*
Package fbservice provides a high-level API for Firebird services.

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

Managers do not write to standard output unless an operation explicitly
receives os.Stdout as its output.

Run gfix maintenance with Repair:

	err = manager.Repair(ctx, database, fbservice.RepairOptions{
		Operation: fbservice.RepairValidate,
		Full:      true,
		ReadOnly:  true,
		Output:    os.Stdout,
	})

Repair also supports RepairSweep, RepairMend, RepairKillShadows,
RepairUpgradeODS and RepairFixICU. Mend can discard damaged records.
ODS upgrades require server support (Firebird 5+); ICU repair requires
Firebird 3+. Unsupported actions return server errors. Validation reports should
be inspected even if the call succeeds.

Sweep supports parallel execution on Firebird 5+:

	err = manager.Repair(ctx, database, fbservice.RepairOptions{
		Operation:       fbservice.RepairSweep,
		ParallelWorkers: 4,
	})

ParallelWorkers accepts 0..32767; zero omits the parameter. Server configuration
may impose a lower limit. For compatibility, sweep uses a legacy parameter on
server versions before 5 and the standard parameter on 5+.

SweepTimeout is an optional server extension that limits sweep in whole seconds.
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

	err = manager.SetDatabaseProperties(ctx, database, fbservice.DatabaseProperties{
		Shutdown: &fbservice.ShutdownOptions{
			Mode:   fbservice.DatabaseModeFull,
			Method: fbservice.ShutdownForce,
		},
	})
	err = manager.SetDatabaseProperties(ctx, database, fbservice.DatabaseProperties{
		OnlineMode: fbservice.DatabaseModeNormal,
	})

LimboTransactions returns transaction IDs (including 64-bit IDs). After deciding
the appropriate outcome, pass an ID to ResolveLimboTransaction with LimboCommit,
LimboRollback or LimboRecover and an optional output writer. Automatic recovery
uses other participants' states and can leave a transaction unresolved. Inspect
the report and query the list again. Context cancellation does not roll back
maintenance actions already performed by the server.
*/
package fbservice
