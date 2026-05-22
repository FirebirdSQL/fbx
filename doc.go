// Package fbx is a driver and toolkit for working with Firebird / RedDatabase.
/*
fbx provides direct connection and transaction APIs, prepared statements, scanning,
and batching.

Establishing a Connection

The primary way of establishing a connection is with [Connect]:

	conn, err := fbx.Connect(context.Background(), os.Getenv("DATABASE_URL"))

The connection string can be in URL or key/value format. A config can also be
created with [ParseConfig] and then passed to [ConnectConfig].

Batching

fbx has two batch styles.

Object batches use [Conn.PrepareBatch] or [Conn.PrepareBatchWithOptions]. Rows are
added with [PreparedBatch.Add]. Sending is manual with [PreparedBatch.Flush] and
[PreparedBatch.Exec]. [PreparedBatch.PendingBytes] can be used to decide when to
flush queued rows.

	pb, err := conn.PrepareBatch(context.Background(), "insert into t(id, note) values (?, ?)")
	if err != nil {
		return err
	}
	defer pb.Close(context.Background())

	for _, row := range rows {
		if err := pb.Add(row.ID, row.Note); err != nil {
			return err
		}
		if pb.PendingBytes() >= 128*1024 {
			if err := pb.Flush(context.Background()); err != nil {
				return err
			}
		}
	}

	res, err := pb.Exec(context.Background())
	if err != nil {
		return err
	}
	_ = res

Named batches use [Conn.Prepare] with SQL that starts with "batch ". Rows are added
with [Conn.Exec]. They can be sent with [Conn.Flush] and executed with [Conn.ExecBatch].
Named batches support built-in auto-flush.

	_, err := conn.Prepare(context.Background(), "b1", "batch insert into t(id, note) values (?, ?)")
	if err != nil {
		return err
	}
	defer conn.Deallocate(context.Background(), "b1")

	for _, row := range rows {
		if _, err := conn.Exec(context.Background(), "b1", row.ID, row.Note); err != nil {
			return err
		}
	}

	res, err := conn.ExecBatch(context.Background(), "b1")
	if err != nil {
		return err
	}
	_ = res

Transactions

Transactions are started by calling [Conn.Begin].

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), "insert into t(id) values (?)", 1)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}
*/
package fbx
