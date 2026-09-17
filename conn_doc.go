package fbx

/*
Prepared statement model

Three separate decisions govern an ordinary statement:

- Scope: which transaction fbconn.Statement.Exec selects by default.
- Execution mode: how fbx prepares an input that is not an already registered
  statement name.
- Storage and ownership: who retains the statement handle and eventually
  deallocates it.

Preparing a statement needs a transaction to resolve metadata. That transaction
does not necessarily own subsequent executions or the statement's lifetime.

Scope and transactions

fbconn.FbConn.PrepareTx(ctx, tx, sql) always binds Statement.Exec to tx,
including when tx is the connection's current internal transaction. Once tx
ends, Statement.Exec cannot silently switch to another transaction.

fbconn.FbConn.Prepare(ctx, sql) and PrepareTxForConnection(ctx, tx, sql) create
connection-scoped statements. The latter uses tx only for preparation;
Statement.Exec selects the connection's current internal transaction on each
execution. Statement.ExecTx always uses the transaction supplied to that call,
regardless of the statement's default scope.

Scope does not automatically deallocate a handle. A low-level caller must call
Statement.Deallocate when finished with it, even after the preparing transaction
has ended.

Ordinary named statements created by fbx.Conn.Prepare or fbx.Tx.Prepare are
connection-scoped and retained by Conn. Tx.Prepare resolves metadata inside
that transaction, and Tx.Exec / Tx.Query execute inside the calling
transaction. After commit or rollback, the name remains usable through Conn or
another transaction. This requires the prepared metadata to remain valid; for
example, rolling back DDL that introduced a referenced object does not make that
object available to later executions.

Selection and ownership

Exec and Query use the same statement resolver after processing query options
and any query rewriter:

1. Look up the input in Conn.preparedStatements. A registered name wins in
   every execution mode, including when automatic caching is disabled.
2. In QueryExecModeCacheStatement, reuse the SQL entry in the LRU or prepare a
   connection-scoped statement and add it to the cache.
3. In QueryExecModeExec, prepare a temporary statement for this call.
4. Reject an unknown execution mode when the input is not a registered name.

The resolver returns the statement and whether the call owns its cleanup. It
does not execute the statement or close rows.

Storage and cleanup depend on the statement source:

- Explicit nonempty name passed to Conn.Prepare / Tx.Prepare: registered in
  preparedStatements by name; if name equals SQL, the SQL remains the lookup
  key. Not stored in the LRU. Released by Conn.Deallocate,
  DeallocateStatement, DeallocateAll, or connection close.
- Automatic cache entry: registered in preparedStatements under a generated
  name and indexed by SQL in the LRU. Released by connection cache maintenance
  or explicit deallocation.
- Temporary statement for QueryExecModeExec: stored in neither
  preparedStatements nor the LRU. Released by the executing call or returned
  rows.
- Low-level fbconn statement: no high-level registration or LRU entry.
  Released by the low-level caller.

preparedStatements is the registry of retained handles, not a synonym for
explicit preparation. The LRU tracks only automatically cached handles and its
pending invalidation queue. If an automatic cache name already belongs to an
explicit statement with matching SQL, that handle is reused without transferring
ownership to the LRU.

QueryExecModeExec still prepares raw SQL on the server. Its difference here is
that the temporary handle is not retained for the next call. Temporary statements
prepared inside an explicit transaction use transaction scope; their short
lifetime is enforced by the executing call, independently of that scope.

Exec deallocates a temporary statement on return, including execution and
parameter-encoding errors. Query transfers ownership to Rows; Rows.Close
deallocates the temporary statement. Closing rows for a retained statement only
closes its cursor. Exhausting rows, QueryRow.Scan, and fatal row errors also
close rows. Callers that stop reading early must call Rows.Close.

Cache invalidation and deallocation

An LRU eviction or execution error invalidates an automatic cache entry. It moves
out of the active LRU into a pending cleanup queue; it remains registered until
deallocation succeeds. The next Exec or Query drains that queue before
executing more work.

Successful deallocation removes the same handle from both the registry and the
LRU or invalidation queue. Deallocate accepts an explicit name or automatically
cached SQL. If that input identifies both an explicit name and a cached SQL
entry, Deallocate releases both. DeallocateStatement identifies a registered
handle by its returned description and releases only that handle. Cache eviction
also deallocates the exact cached handle, preserving separately named statements
with the same SQL.

DeallocateAll deallocates retained handles before resetting the registry and
cache. It can partially succeed: on an error, successfully deallocated entries
are gone, while the failed and unprocessed entries retain their registry/cache
ownership so cleanup can be retried. If the connection remains open, an error
must not leave a live handle detached from its old LRU.

Named batches use preparedBatches and have their own execution and transaction
ownership rules. In particular, a batch prepared through Tx belongs to that
transaction and is closed when it ends; the ordinary statement rules above do
not change batch ownership.
*/
