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

Operations without special settings can omit their options:

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

Manager does not write to standard output unless an operation explicitly
receives os.Stdout as its output.
*/
package fbservice
