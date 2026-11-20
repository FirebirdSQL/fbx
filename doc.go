// Package fbx is a driver and toolkit for working with Firebird / RedDatabase.
/*
fbx provides direct connection and transaction APIs, prepared statements, scanning,
and batching.

Establishing a Connection

The primary way of establishing a connection is with [Connect]:

	conn, err := fbx.Connect(context.Background(), os.Getenv("DATABASE_URL"))

The connection string can be in URL or key/value format. A config can also be
created with [ParseConfig] and then passed to [ConnectConfig].

# Closing Query Results

[Conn.QueryRow] and [Tx.QueryRow] return a [Row]. Always call [Row.Scan] on it.
Scan reads the first row and closes the underlying [Rows] before returning,
including on [ErrNoRows] or another error. No separate Close call is needed.
Use [CollectExactlyOneRow] to require exactly one row.

The same rules apply to nested transactions and QueryRow on Pool, Conn, and Tx
in [github.com/firebirdsql/fbx/fbxpool]. Pool.QueryRow also releases the connection
it acquired when Scan completes. A connection acquired explicitly with
Pool.Acquire still requires Release. Finish an explicit transaction with
[Tx.Commit] or [Tx.Rollback]. Close any BLOB stream returned as an [io.ReadCloser]
separately.

[ForEachRow], [AppendRows], [CollectRows], [CollectOneRow], and
[CollectExactlyOneRow] close the Rows passed to them before returning, even when
the callback returns an error. Converters such as [RowTo] and [RowToStructByName]
read one row and leave result cleanup to their caller.

With [Conn.Query] or [Tx.Query], [Rows.Next] closes the result when it returns
false. Check [Rows.Err] after iteration or after closing the result. Use
defer rows.Close() after a successful Query to cover early returns. If you stop
reading early and need the connection again before returning, call [Rows.Close]
before the next query. Pool.Query releases the connection when the result closes
and closes the result on errors from [Rows.Scan] or [Rows.Values].

With [database/sql], [sql.Row.Scan] closes the result of QueryRow or
QueryRowContext on [sql.DB], [sql.Tx], and [sql.Stmt], and of QueryRowContext on
[sql.Conn]. A conversion error from [sql.Rows.Scan] can leave a result open.
Call [sql.Rows.Close] when leaving the iteration early and [sql.Conn.Close]
when finished with a dedicated connection.

[fbconn.ResultReader.Read] and [fbconn.ResultReader.ReadAll] close the reader
before returning. When iterating with [fbconn.ResultReader.NextRow], call
[fbconn.ResultReader.Close] even after NextRow returns false. Closing the Rows
returned by [RowsFromResultReader] also closes its reader.

# Batching

fbx exposes Firebird native batch DML for INSERT, UPDATE, and DELETE statements
with parameters and without result sets. Blob and array parameters are rejected
by the batch API.

Pass one [BatchRows] argument to [Conn.Exec] or [Tx.Exec] to execute a complete
batch with default options. This also works through database/sql ExecContext,
including prepared statements.

	_, err := conn.Exec(ctx, "insert into t(id, note) values (?, ?)",
		BatchRows{{1, "one"}, {2, "two"}})

Each call sends and executes the whole set, then releases its server batch.
No rows remain queued for another call. Encoded rows must fit in the server
batch buffer; the driver does not split executions into chunks. For larger
loads, submit portions in an explicit transaction and commit or roll back
the transaction yourself. Batches use the same transaction rules described below.

Use [Conn.PrepareBatch] or [Tx.PrepareBatch] to create a batch object, or their
PrepareBatchWithOptions variants to configure it. Rows are added with
[PreparedBatch.Add]. [PreparedBatch.Flush] sends rows to the server-side batch
buffer without executing them, and [PreparedBatch.Exec] flushes any remaining
rows and executes the batch. [PreparedBatch.PendingBytes] can be used to decide
when to flush queued rows.

[PreparedBatch.Info] queries the server's buffer capacity and current data sizes
(protocol 17 or later). It does not flush or execute rows. These sizes describe
server storage and can differ from the local encoded size in PendingBytes.

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
[PreparedBatch.Cancel] can be used before [PreparedBatch.Exec] to discard both
locally queued rows and rows already flushed to the server while keeping the
batch reusable.

# Batch Results and Errors

[PreparedBatch.Exec] returns a [BatchResult]. If the server reports per-row
errors, the error can be inspected as a [BatchExecutionError].
BatchOptions.ContinueOnError asks the server to continue after row errors and
report more of them; it does not change transaction atomicity.

In autocommit mode, a connection-scoped batch is atomic: if any per-row error is
reported, fbx rolls back the implicit internal transaction and no rows from that
execution are committed. Inside an explicit [Tx], [PreparedBatch.Exec] does not
commit or roll back; the caller decides what to do after inspecting the result
and error.

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

A native transaction remains active after a confirmed server rejection of commit
on a usable connection, so it can still be rolled back. A transport error while
waiting for commit leaves its outcome unknown and closes the connection.

# Errors and Retries

[fbconn.SafeToRetry] applies to the complete call, including SQL execution,
commit, and resource cleanup. Once SQL may have been sent, a later error does not
authorize repeating the call. A false result means that safe retry is not
guaranteed; it does not mean that the SQL succeeded.

A non-nil error does not prove that changes were rolled back. A lost commit
response leaves the outcome unknown. An error freeing a statement after a
confirmed commit does not undo the saved changes. Cleanup errors preserve the
original error and cannot turn an unsafe retry into a safe one.

A failed autocommit resolves the driver's internal transaction before another
query can use it. Rollback uses a separate, bounded context after cancellation;
if it cannot be confirmed, the connection is closed. User-managed transactions
remain the caller's responsibility.
*/
package fbx
