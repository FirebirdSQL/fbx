# fbx - Red Database and Firebird Driver for Go

fbx is a pure Go driver for Red Database and Firebird.

The driver is designed as a richer native Go client than a minimal
`database/sql` adapter: it exposes a high-level `fbx.Conn` API, connection
pooling for production workloads, a `database/sql` compatibility layer, typed
Firebird values, tracing hooks, and modern Red Database / Firebird features.

## Supported Features

- Pure Go implementation of the Red Database / Firebird wire protocol.
- Native `fbx.Conn` API with `context.Context` support for connect, query, transaction, BLOB, and low-level operations. See [Go examples](examples_test.go#L14).
- `database/sql` compatibility layer registered as the `fbx` driver. See [database/sql example](stdlib/example_test.go#L11).
- Concurrency-safe `fbxpool` connection pool for application workloads, with lifecycle hooks, health checks, configurable ping behavior, acquire tracing, release tracing, and pool statistics. See [pool example](fbxpool/example_test.go#L10).
- URL and keyword/value connection strings, environment variable support, and multi-host fallback configuration.
- Database creation and deletion through `CreateDatabase`, `DropDatabase`, and related config helpers. See [Go examples](examples_test.go#L65).
- Firebird `POST_EVENT` subscriptions through `fbevent.Manager`, including one-shot waits, callback listeners, channel subscriptions, and pgx-like notifications. See [event examples](fbevent/example_test.go#L19).
- Red Database / Firebird 6 schema-qualified identifiers. See [Go examples](examples_test.go#L79).
- Prepared statements, statement cache, configurable statement cache size, and configurable query execution mode. See [Go examples](examples_test.go#L87).
- Batch DML operations through object batches and named batches. See [batch examples](batch_insert_test.go#L723).
- Positional `?` parameters plus `NamedArgs` query rewriting. See [Go examples](examples_test.go#L240).
- Transactions with Firebird TPB support, isolation/access mode options, lock timeout, raw TPB mode, helper functions, and pseudo-nested transactions via savepoints. See [Go examples](examples_test.go#L268).
- Streaming and lazy BLOB support, text and binary BLOB handling, configurable segment buffer size, stream BLOB mode, inline BLOB size, and inline BLOB cache settings. See [Go examples](examples_test.go#L305).
- Configurable fetch batch size for result sets. See [Go examples](examples_test.go#L36).
- Firebird type mapping through `fbtype`, including `DATE`, `TIME`, `TIMESTAMP`, `TIMESTAMP WITH TIME ZONE`, `CHAR`, `VARCHAR`, OCTETS, BOOLEAN, integer types, `INT128`, NUMERIC/DECIMAL, `DECFLOAT(16)`, `DECFLOAT(34)`, arrays, NULL wrappers, and zero-null helpers. See [Go examples](examples_test.go#L340).
- Advanced type support in the `database/sql` layer, including BLOBs, arrays, `TIME`, `INT128`, NUMERIC/DECIMAL, DECFLOAT, OCTETS, and timestamp with time zone values. See [database/sql example](stdlib/example_test.go#L11).
- Firebird array support, including reading, writing, and scanning array values through the native API and the `database/sql` compatibility layer. See [Go examples](examples_test.go#L369).
- Row collection helpers for scanning rows into values, maps, and structs. See [row examples](rows_test.go#L155).
- Query, prepare, connect, acquire, and release tracing hooks, plus the `tracelog` package. See [Go examples](examples_test.go#L436).

## Planned Features

- Geometric and spatial type support.
- SQL/JSON value mapping and helper APIs.
- ROW and structured value support.
- Services manager API for backup, restore, trace, statistics, online validation, and user-management workflows.
- Bidirectional result-set navigation for server-side scrollable cursors.
- Two-phase commit and distributed transaction support.
- Certificate-based and multifactor authentication using Red Database security policies.

## Example Usage

**main.go**

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

```shell
go mod init fbxdemo
go mod tidy
```

See the focused Go examples in [examples_test.go](examples_test.go),
[fbevent/example_test.go](fbevent/example_test.go),
[fbxpool/example_test.go](fbxpool/example_test.go), and
[stdlib/example_test.go](stdlib/example_test.go).
