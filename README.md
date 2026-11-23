# fbx - Firebird Driver for Go

fbx is a pure Go driver for Firebird.

The driver is designed as a richer native Go client than a minimal
`database/sql` adapter: it exposes a high-level `fbx.Conn` API, connection
pooling for production workloads, a `database/sql` compatibility layer, typed
Firebird values, tracing hooks, and modern Firebird features.

## Supported Features

- Pure Go implementation of the Firebird wire protocol
- Native `fbx.Conn` API with `context.Context` support for connect, query, transaction, BLOB, and low-level operations
- `database/sql` compatibility layer registered as the `fbx` driver
- Concurrency-safe `fbxpool` connection pool for application workloads, with lifecycle hooks, health checks, configurable ping behavior, acquire tracing, release tracing, and pool statistics
- URL and keyword/value connection strings, environment variable support, and multi-host fallback configuration
- Database creation and deletion through `CreateDatabase`, `DropDatabase`, and related config helpers
- Services manager API for server-side logical and physical backup/restore (nbackup), database statistics, gfix maintenance, and trace session management through `fbservice.Manager`, including background streaming with `TraceStartSession` and access to the session ID and completion errors
- Firebird `POST_EVENT` subscriptions through `fbevent.Manager`, including one-shot waits, callback listeners, channel subscriptions, and pgx-like notifications
- Firebird 6 schema-qualified identifiers
- Prepared statements, statement cache, configurable statement cache size, and configurable query execution mode
- Batch DML operations through PreparedBatch or Exec with BatchRows
- Positional `?` parameters plus `NamedArgs` query rewriting
- Transactions with Firebird TPB support, isolation/access mode options, lock timeout, raw TPB mode, helper functions, and pseudo-nested transactions via savepoints
- Streaming and lazy BLOB support, text and binary BLOB handling, configurable segment buffer size, stream BLOB mode, inline BLOB size, and inline BLOB cache settings
- Configurable fetch batch size for result sets
- Optional _zlib_ wire compression for protocol 13 and newer, including combined use with wire encryption
- Firebird type mapping through `fbtype`, including `DATE`, `TIME`, `TIMESTAMP`, `TIMESTAMP WITH TIME ZONE`, `CHAR`, `VARCHAR`, `OCTETS`, `BOOLEAN`, integer types, `INT128`, `NUMERIC/DECIMAL`, `DECFLOAT(16)`, `DECFLOAT(34)`, arrays, NULL wrappers, and zero-null helpers
- Advanced type support in the `database/sql` layer, including BLOBs, arrays, `TIME`, `INT128`, `NUMERIC/DECIMAL`, `DECFLOAT`, `OCTETS`, and timestamp with time zone values
- Firebird array support, including reading, writing, and scanning array values through the native API and the `database/sql` compatibility layer
- Row collection helpers for scanning rows into values, maps, and structs
- Query, prepare, connect, acquire, and release tracing hooks, plus the `tracelog` package
- Multi-factor authentication using Red Database security policies

## Planned Features

- Geometric and spatial type support
- SQL/JSON value mapping and helper APIs
- ROW and structured value support
- Services manager API for online validation and user-management workflows
- Bidirectional result-set navigation for server-side scrollable cursors
- Two-phase commit and distributed transaction support

## Related projects

### Authentication plugins

- fbx-gss - https://pkg.go.dev/rdb.red-soft.ru/fbx-gss
- fbx-cryptopro - https://pkg.go.dev/rdb.red-soft.ru/fbx-cryptopro

## Example Usage

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/firebirdsql/fbx"
)

func main() {
	url := "firebird://sysdba:masterkey@localhost:3050/employee"
	conn, err := fbx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var name string
	var phone string
	var ext int32
	err = conn.QueryRow(context.Background(), "select first_name, phone_no, phone_ext from phone_list where emp_no=?", 144).Scan(&name, &phone, &ext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(name, phone, ext)
}
```

See documentation and more usage examples at https://pkg.go.dev/github.com/firebirdsql/fbx

Wire compression is disabled by default. Enable it with `wire_compression=true` in the connection string
(`wire_compress` and `wireCompression` are accepted aliases) or with `FBWIRECOMPRESSION=true`.

Set `default_transaction_isolation=read_committed` in a keyword/value connection
string, or `?default_transaction_isolation=read_committed` in a URL, to choose
the default transaction isolation. Supported values are `read_committed`,
`repeatable_read`, and `serializable`. In connection strings, the aliases
`read committed` and `repeatable read` are also accepted and normalized to the
underscore values; an empty value preserves the server default.
The setting also applies to autocommit, `Begin`, and `database/sql` transactions
using `sql.LevelDefault`. An explicit `TxOptions.IsoLevel` or `sql.TxOptions.Isolation`
overrides it. Raw TPBs passed to `BeginTxTPB` or `StartTransactionTPB` are unchanged.
SQL `SET TRANSACTION` statements use the options in the SQL and server defaults
for any omitted options.
The same setting is available as `Config.DefaultTransactionIsolation`, using
`fbconn.TxIsoLevel` and its constants (`fbx.TxIsoLevel` is an alias).

For cursor queries, the driver automatically uses the server's deferred-execute flag from
`Prepare`. When supported, `Exec` buffers `Execute`; the first row read sends
`Execute` and `Fetch` together. A result closed before reading sends `Execute`
alone and reads its response. Errors from a deferred `Execute` are returned by
the first row read or close. This saves one network round trip while retaining
both protocol commands. Servers without this flag use separate requests.
To reduce the number of later `Fetch` commands, increase `fetch_batch_size`
(default: 400); larger batches can use more client memory.
