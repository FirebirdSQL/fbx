# fbx - Red Database and Firebird Driver for Go

fbx is a pure Go driver for Red Database and Firebird.

The driver is designed as a richer native Go client than a minimal
`database/sql` adapter: it exposes a high-level `fbx.Conn` API, connection
pooling for production workloads, a `database/sql` compatibility layer, typed
Firebird values, tracing hooks, and modern Red Database / Firebird features.

## Supported Features

- Pure Go implementation of the Red Database / Firebird wire protocol.
- Native `fbx.Conn` API with `context.Context` support for connect, query, transaction, BLOB, and low-level operations. See [Native Connection and Query](devdoc/examples.md#native-connection-and-query).
- `database/sql` compatibility layer registered as the `fbx` driver. See [Database SQL Compatibility](devdoc/examples.md#database-sql-compatibility).
- Concurrency-safe `fbxpool` connection pool for application workloads, with lifecycle hooks, health checks, configurable ping behavior, acquire tracing, release tracing, and pool statistics. See [Connection Pool](devdoc/examples.md#connection-pool).
- URL and keyword/value connection strings, environment variable support, multi-host fallback configuration, and `target_session_attrs` validation for primary, replica, prefer-replica, read-only, and read-write sessions. See [Connection Configuration](devdoc/examples.md#connection-configuration).
- Database creation through `CreateDatabase`, `CreateDatabaseConfig`, and the `create_database=true` connection setting. See [Database Creation](devdoc/examples.md#database-creation).
- Firebird `POST_EVENT` subscriptions through `fbevent.Manager`, including one-shot waits, callback listeners, channel subscriptions, and pgx-like notifications. See [Firebird Events](devdoc/examples.md#firebird-events).
- Red Database / Firebird 6 schema support, including `search_path` connection settings and schema-qualified identifiers. See [Schemas](devdoc/examples.md#schemas).
- Authentication with SRP, SRP256, SRP224, SRP384, SRP512, GSS/Kerberos, and Legacy_Auth plugins. See [Connection Configuration](devdoc/examples.md#connection-configuration).
- Wire encryption support. See [Connection Configuration](devdoc/examples.md#connection-configuration).
- Prepared statements, statement cache, configurable statement cache size, and configurable query execution mode. See [Prepared Statements and Query Modes](devdoc/examples.md#prepared-statements-and-query-modes).
- Batch DML operations through object batches and named batches. See [Batch Operations](devdoc/examples.md#batch-operations).
- Positional `?` parameters plus `NamedArgs` and `StrictNamedArgs` query rewriting. See [Named Arguments](devdoc/examples.md#named-arguments).
- Transactions with Firebird TPB support, isolation/access mode options, lock timeout, raw TPB mode, helper functions, and pseudo-nested transactions via savepoints. See [Transactions](devdoc/examples.md#transactions).
- Streaming and lazy BLOB support, text and binary BLOB handling, configurable segment buffer size, stream BLOB mode, inline BLOB size, and inline BLOB cache settings. See [BLOB Values](devdoc/examples.md#blob-values).
- Configurable fetch batch size for result sets. See [Connection Configuration](devdoc/examples.md#connection-configuration).
- Firebird type mapping through `fbtype`, including `DATE`, `TIME`, `TIMESTAMP`, `TIMESTAMP WITH TIME ZONE`, text/varchar, OCTETS, BOOLEAN, integer types, `INT128`, NUMERIC/DECIMAL, `DECFLOAT(16)`, `DECFLOAT(34)`, arrays, NULL wrappers, and zero-null helpers. See [Type Mapping](devdoc/examples.md#type-mapping).
- Advanced type support in the `database/sql` layer, including BLOBs, arrays, `TIME`, `INT128`, NUMERIC/DECIMAL, DECFLOAT, OCTETS, and timestamp with time zone values. See [Database SQL Compatibility](devdoc/examples.md#database-sql-compatibility).
- Firebird array support, including reading, writing, and scanning array values through the native API and the `database/sql` compatibility layer. See [Arrays](devdoc/examples.md#arrays).
- Row collection helpers for scanning rows into values, maps, and structs. See [Row Collection](devdoc/examples.md#row-collection).
- Query, prepare, connect, acquire, and release tracing hooks, plus the `tracelog` package. See [Tracing](devdoc/examples.md#tracing).

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

See [devdoc/examples.md](devdoc/examples.md) for focused feature snippets.
