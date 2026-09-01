# fbx - Red Database and Firebird Driver for Go

fbx is a pure Go driver for Red Database and Firebird.

The driver is designed as a richer native Go client than a minimal
`database/sql` adapter: it exposes a high-level `fbx.Conn` API, connection
pooling for production workloads, a `database/sql` compatibility layer, typed
Firebird values, tracing hooks, and modern Red Database / Firebird features.

## Supported Features

- Pure Go implementation of the Red Database / Firebird wire protocol.
- Native `fbx.Conn` API with `context.Context` support for connect, query, transaction, BLOB, and low-level operations.
- `database/sql` compatibility layer registered as the `fbx` driver.
- Concurrency-safe `fbxpool` connection pool for application workloads, with lifecycle hooks, health checks, configurable ping behavior, acquire tracing, release tracing, and pool statistics.
- URL and keyword/value connection strings, environment variable support, and multi-host fallback configuration.
- Database creation and deletion through `CreateDatabase`, `DropDatabase`, and related config helpers.
- Firebird `POST_EVENT` subscriptions through `fbevent.Manager`, including one-shot waits, callback listeners, channel subscriptions, and pgx-like notifications.
- Red Database / Firebird 6 schema-qualified identifiers.
- Prepared statements, statement cache, configurable statement cache size, and configurable query execution mode.
- Batch DML operations through object batches and named batches.
- Positional `?` parameters plus `NamedArgs` query rewriting.
- Transactions with Firebird TPB support, isolation/access mode options, lock timeout, raw TPB mode, helper functions, and pseudo-nested transactions via savepoints.
- Streaming and lazy BLOB support, text and binary BLOB handling, configurable segment buffer size, stream BLOB mode, inline BLOB size, and inline BLOB cache settings.
- Configurable fetch batch size for result sets.
- Optional zlib wire compression for protocol 13 and newer, including combined use with wire encryption.
- Firebird type mapping through `fbtype`, including `DATE`, `TIME`, `TIMESTAMP`, `TIMESTAMP WITH TIME ZONE`, `CHAR`, `VARCHAR`, OCTETS, BOOLEAN, integer types, `INT128`, NUMERIC/DECIMAL, `DECFLOAT(16)`, `DECFLOAT(34)`, arrays, NULL wrappers, and zero-null helpers.
- Advanced type support in the `database/sql` layer, including BLOBs, arrays, `TIME`, `INT128`, NUMERIC/DECIMAL, DECFLOAT, OCTETS, and timestamp with time zone values.
- Firebird array support, including reading, writing, and scanning array values through the native API and the `database/sql` compatibility layer.
- Row collection helpers for scanning rows into values, maps, and structs.
- Query, prepare, connect, acquire, and release tracing hooks, plus the `tracelog` package.

## Planned Features

- Geometric and spatial type support.
- SQL/JSON value mapping and helper APIs.
- ROW and structured value support.
- Services manager API for backup, restore, trace, statistics, online validation, and user-management workflows.
- Bidirectional result-set navigation for server-side scrollable cursors.
- Two-phase commit and distributed transaction support.
- Certificate-based and multifactor authentication using Red Database security policies.

## Example Usage

```go
package main

import (
	"context"
	"fmt"
	"os"

	"rdb.red-soft.ru/fbx"
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

See documentation and more usage examples at https://pkg.go.dev/rdb.red-soft.ru/fbx

Wire compression is disabled by default. Enable it with `wire_compression=true` in the connection string
(`wire_compress` and `wireCompression` are accepted aliases) or with `FBWIRECOMPRESSION=true`.
