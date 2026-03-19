// Package fbconn is a low-level Firebird/RedDatabase database driver.
/*
fbconn provides lower level access to a Firebird connection than a database/sql or fbx connection. It operates at
nearly the same level is the C library libfbclient.

Establishing a Connection

Use Connect to establish a connection. It accepts a connection string in URL or keyword/value format and will read the
environment (see ParseConfig for supported environment variables and parameters)

Executing a Statement

Exec and ExecParams execute a single SQL statement and return a ResultReader that iterates over rows.
Call Close on the ResultReader before reusing the connection. The Read method reads the entire result into memory.

Prepared Statements

Use Prepare / PrepareTx to create a prepared statement and execute it using the returned Statement.

Context Support

All potentially blocking operations take a context.Context. The default behavior when a context is canceled is for the
method to immediately return. In most circumstances, this will also close the underlying connection. This behavior can
be customized by using BuildContextWatcherHandler on the Config to create a ctxwatch.Handler with different behavior.
This can be especially useful when queries that are frequently canceled and the overhead of creating new connections is
a problem. DeadlineContextWatcherHandler and CancelRequestContextWatcherHandler can be used to introduce a delay before
interrupting the query in such a way as to close the connection.

The CancelRequest method may be used to request the server cancel an in-progress query without forcing the
client to abort.
*/
package fbconn
