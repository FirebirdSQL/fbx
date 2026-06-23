// Package fbx is a driver and toolkit for working with Firebird / RedDatabase.
/*
fbx provides direct connection and transaction APIs, prepared statements, scanning,
and batching.

Establishing a Connection

The primary way of establishing a connection is with [Connect]:

	conn, err := fbx.Connect(context.Background(), os.Getenv("DATABASE_URL"))

The connection string can be in URL or key/value format. A config can also be
created with [ParseConfig] and then passed to [ConnectConfig].

# Batching

fbx exposes Firebird native batch DML for INSERT, UPDATE, and DELETE statements
with parameters and without result sets. Blob and array parameters are rejected
by the batch API. There are two batch styles: object batches and named batches.

# Object Batches

Object batches are created with [Conn.PrepareBatch] or
[Conn.PrepareBatchWithOptions]. Rows are added with [PreparedBatch.Add]. Sending
is manual: [PreparedBatch.Flush] sends queued rows to the server-side batch
buffer without executing them, and [PreparedBatch.Exec] flushes any remaining
rows and executes the batch. [PreparedBatch.PendingBytes] can be used to decide
when to flush queued rows.

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

Always call [PreparedBatch.Close] when the batch is no longer needed.
[PreparedBatch.Cancel] can be used before Exec to discard both locally queued
rows and rows already flushed to the server while keeping the batch reusable.

# Named Batches

Named batches use [Conn.Prepare] with SQL that starts with "batch ". Rows are
added with [Conn.Exec] using the prepared batch name. Adding a row does not
execute the batch; it only encodes the row and may auto-flush queued rows to the
server. Use [Conn.Flush] to send queued rows explicitly and [Conn.ExecBatch] to
execute the batch. Named batches are released with [Conn.Deallocate].

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

# Batch Results and Errors

[PreparedBatch.Exec] and [Conn.ExecBatch] return a [BatchResult]. If the server
reports per-row errors, the error can be inspected as a [BatchExecutionError].
BatchOptions.ContinueOnError asks the server to continue after row errors and
report more of them; it does not change transaction atomicity.

In autocommit mode, a connection-scoped batch is atomic: if any per-row error is
reported, fbx rolls back the implicit internal transaction and no rows from that
execution are committed. Inside an explicit [Tx], Exec does not commit or roll
back; the caller decides what to do after inspecting the result and error.

# Transactions

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
