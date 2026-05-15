# fbx Examples

Small examples for the main fbx features. Most snippets assume:

```go
ctx := context.Background()
dsn := "firebird://sysdba:masterkey@localhost:3060/employee"
```

## Native Connection and Query

```go
conn, err := fbx.Connect(ctx, dsn)
if err != nil {
	return err
}
defer conn.Close(ctx)

var name string
err = conn.QueryRow(ctx, "select first_name from employee where emp_no=?", 2).Scan(&name)
```

## Connection Configuration

```go
config, err := fbx.ParseConfig(
	"user=sysdba password=masterkey host=localhost port=3060 database=employee " +
		"target_session_attrs=read-write fetch_batch_size=1000 wire_crypt=true " +
		"auth_plugin_list=Srp256,Srp,Legacy_Auth",
)
if err != nil {
	return err
}

conn, err := fbx.ConnectConfig(ctx, config)
```

## Connection Pool

```go
pool, err := fbxpool.New(ctx, dsn)
if err != nil {
	return err
}
defer pool.Close()

var count int
err = pool.QueryRow(ctx, "select count(*) from employee").Scan(&count)
```

## Database SQL Compatibility

```go
// import _ "rdb.red-soft.ru/fbx/stdlib"

db, err := sql.Open("fbx", dsn)
if err != nil {
	return err
}
defer db.Close()

var count int
err = db.QueryRowContext(ctx, "select count(*) from employee").Scan(&count)
```

## Database Creation

```go
conn, err := fbx.CreateDatabase(ctx, "firebird://sysdba:masterkey@localhost:3060/newdb.fdb")
if err != nil {
	return err
}
defer conn.Close(ctx)
```

## Schemas

```go
config, err := fbx.ParseConfig(dsn + "?search_path=accounting")
if err != nil {
	return err
}

conn, err := fbx.ConnectConfig(ctx, config)
if err != nil {
	return err
}
defer conn.Close(ctx)

table := fbx.Identifier{"accounting", "invoice"}.Sanitize()
rows, err := conn.Query(ctx, "select id, amount from "+table+" where customer_id=?", customerID)
```

## Prepared Statements and Query Modes

```go
config, err := fbx.ParseConfig(dsn + "?statement_cache_capacity=256&default_query_exec_mode=cache_statement")
if err != nil {
	return err
}

conn, err := fbx.ConnectConfig(ctx, config)
if err != nil {
	return err
}
defer conn.Close(ctx)

_, err = conn.Prepare(ctx, "employee_by_id", "select first_name from employee where emp_no=?")
```

## Batch Operations

```go
rows := []struct {
	ID   int64
	Note string
}{
	{1, "one"},
	{2, "two"},
}

pb, err := conn.PrepareBatch(ctx, "insert into audit_log(id, note) values (?, ?)")
if err != nil {
	return err
}
defer pb.Close(ctx)

for _, row := range rows {
	if err := pb.Add(row.ID, row.Note); err != nil {
		return err
	}
	if pb.PendingBytes() >= 128*1024 {
		if err := pb.Flush(ctx); err != nil {
			return err
		}
	}
}

res, err := pb.Exec(ctx)
if err != nil {
	return err
}
_ = res
```

```go
_, err := conn.Prepare(ctx, "audit_batch", "batch insert into audit_log(id, note) values (?, ?)")
if err != nil {
	return err
}
defer conn.Deallocate(ctx, "audit_batch")

_, err = conn.Exec(ctx, "audit_batch", int64(1), "one")
if err != nil {
	return err
}
_, err = conn.Exec(ctx, "audit_batch", int64(2), "two")
if err != nil {
	return err
}

res, err := conn.ExecBatch(ctx, "audit_batch")
if err != nil {
	return err
}
_ = res
```

## Named Arguments

```go
rows, err := conn.Query(
	ctx,
	"select * from employee where emp_no=@id",
	fbx.NamedArgs{"id": 2},
)
```

## Transactions

```go
tx, err := conn.BeginTx(ctx, fbx.TxOptions{
	IsoLevel:    fbx.ReadCommitted,
	AccessMode: fbx.ReadWrite,
	LockTimeout: 5 * time.Second,
})
if err != nil {
	return err
}
defer tx.Rollback(ctx)

_, err = tx.Exec(ctx, "update employee set salary = salary + ? where emp_no=?", 100, 2)
if err != nil {
	return err
}
err = tx.Commit(ctx)
```

## BLOB Values

```go
_, err := conn.Exec(ctx, "insert into files(id, data) values (?, ?)", 1, bytes.NewReader(data))

var out []byte
err = conn.QueryRow(ctx, "select data from files where id=?", 1).Scan(&out)
```

## Type Mapping

```go
var (
	id     fbtype.Int128
	amount fbtype.Numeric
)

err := conn.QueryRow(ctx, "select id, amount from invoice where id=?", invoiceID).Scan(&id, &amount)
```

## Arrays

```go
_, err := conn.Exec(ctx, "insert into samples(id, nums) values (?, ?)", 1, []int16{1, 2, 3})

var nums []int16
err = conn.QueryRow(ctx, "select nums from samples where id=?", 1).Scan(&nums)
```

## Row Collection

```go
type Employee struct {
	EmpNo     int32  `db:"EMP_NO"`
	FirstName string `db:"FIRST_NAME"`
}

rows, err := conn.Query(ctx, "select emp_no, first_name from employee")
if err != nil {
	return err
}
employees, err := fbx.CollectRows(rows, fbx.RowToStructByName[Employee])
```

## Tracing

```go
config, err := fbx.ParseConfig(dsn)
if err != nil {
	return err
}
config.Tracer = &tracelog.TraceLog{
	Logger:   tracelog.LoggerFunc(func(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {}),
	LogLevel: tracelog.LogLevelInfo,
}

conn, err := fbx.ConnectConfig(ctx, config)
```
