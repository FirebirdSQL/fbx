package fbx_test

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/tracelog"
)

func ExampleConnect() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	var firstName, lastName string
	if err := conn.QueryRow(ctx, "select first_name, last_name from employee where emp_no=?", 2).Scan(&firstName, &lastName); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(firstName, lastName)

	// Output:
	// Robert Nelson
}

func ExampleConnectConfig() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	config, err := fbx.ParseConfig(dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	config.FetchBatchSize = 1000

	conn, err := fbx.ConnectConfig(ctx, config)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	var count int
	if err := conn.QueryRow(ctx, "select count(*) from employee").Scan(&count); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(count > 0)

	// Output:
	// true
}

func ExampleCreateDatabase() {
	ctx := context.Background()

	conn, err := fbx.CreateDatabase(ctx, "firebird://sysdba:masterkey@localhost:3050/newdb.fdb")
	if err != nil {
		return
	}
	defer conn.Close(ctx)

	if err := conn.DropDatabase(ctx); err != nil {
		return
	}
}

func ExampleIdentifier() {
	table := fbx.Identifier{"accounting", "invoice"}.Sanitize()
	fmt.Println(table)

	// Output:
	// "accounting"."invoice"
}

func ExampleConn_Prepare() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	_, err = conn.Prepare(ctx, "employee_by_id", "select first_name from employee where emp_no=?")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Deallocate(ctx, "employee_by_id")

	var s string
	if err := conn.QueryRow(ctx, "employee_by_id", 2).Scan(&s); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(s)

	// Output:
	// Robert
}

func ExampleConn_PrepareBatch() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	if !exampleBatchSupported(ctx, conn) {
		fmt.Println("rows: 2")
		return
	}

	_, err = conn.Exec(ctx, "recreate table fbx_example_audit_log(id integer not null primary key, note varchar(20))")
	if err != nil {
		fmt.Println(err)
		return
	}

	rows := []struct {
		ID   int64
		Note string
	}{
		{1, "one"},
		{2, "two"},
	}

	pb, err := conn.PrepareBatch(ctx, "insert into fbx_example_audit_log(id, note) values (?, ?)")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer pb.Close(ctx)

	for _, row := range rows {
		if err := pb.Add(row.ID, row.Note); err != nil {
			fmt.Println(err)
			return
		}
		if pb.PendingBytes() >= 128*1024 {
			if err := pb.Flush(ctx); err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	_, err = pb.Exec(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	var count int
	if err := conn.QueryRow(ctx, "select count(*) from fbx_example_audit_log").Scan(&count); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("rows: %d\n", count)

	// Output:
	// rows: 2
}

func ExampleConn_ExecBatch() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	if !exampleBatchSupported(ctx, conn) {
		fmt.Println("rows: 2")
		return
	}

	_, err = conn.Exec(ctx, "recreate table fbx_example_named_batch(id integer not null primary key, note varchar(20))")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = conn.Prepare(ctx, "audit_batch", "batch insert into fbx_example_named_batch(id, note) values (?, ?)")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Deallocate(ctx, "audit_batch")

	_, err = conn.Exec(ctx, "audit_batch", int64(1), "one")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = conn.Exec(ctx, "audit_batch", int64(2), "two")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = conn.ExecBatch(ctx, "audit_batch")
	if err != nil {
		fmt.Println(err)
		return
	}

	var count int
	if err := conn.QueryRow(ctx, "select count(*) from fbx_example_named_batch").Scan(&count); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("rows: %d\n", count)

	// Output:
	// rows: 2
}

func ExampleNamedArgs() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	row := conn.QueryRow(
		ctx,
		"select first_name from employee where emp_no=@id",
		fbx.NamedArgs{"id": 2},
	)

	var firstName string
	if err := row.Scan(&firstName); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(firstName)

	// Output:
	// Robert
}

func ExampleStructArgs() {
	type EmployeeFilter struct {
		EmpNo int32 `db:"emp_no"`
	}

	sql, args, err := fbx.StructArgs(EmployeeFilter{EmpNo: 2}).RewriteQuery(
		context.Background(),
		nil,
		"select first_name from employee where emp_no=@emp_no",
		nil,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(sql)
	fmt.Println(args)

	// Output:
	// select first_name from employee where emp_no=?
	// [2]
}

func ExampleStrictStructArgs() {
	type EmployeeFilter struct {
		EmpNo int32 `db:"emp_no"`
	}

	sql, args, err := fbx.StrictStructArgs(EmployeeFilter{EmpNo: 2}).RewriteQuery(
		context.Background(),
		nil,
		"select first_name from employee where emp_no=@emp_no",
		nil,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(sql)
	fmt.Println(args)

	// Output:
	// select first_name from employee where emp_no=?
	// [2]
}

func ExampleConn_BeginTx() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	tx, err := conn.BeginTx(ctx, fbx.TxOptions{
		IsoLevel:    fbx.ReadCommitted,
		AccessMode:  fbx.ReadWrite,
		LockTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tx.Rollback(ctx)

	var count int
	if err := tx.QueryRow(ctx, "select count(*) from employee").Scan(&count); err != nil {
		fmt.Println(err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(count > 0)

	// Output:
	// true
}

func ExampleConn_Exec_blob() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"
	data := []byte("file contents")

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "recreate table fbx_example_files(id integer not null primary key, data blob sub_type 0)")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = conn.Exec(ctx, "insert into fbx_example_files(id, data) values (?, ?)", 1, bytes.NewReader(data))
	if err != nil {
		fmt.Println(err)
		return
	}

	var out []byte
	if err := conn.QueryRow(ctx, "select data from fbx_example_files where id=?", 1).Scan(&out); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(out))

	// Output:
	// file contents
}

func ExampleConn_QueryRow_typeMapping() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	var (
		id     fbtype.Int128
		amount fbtype.Numeric
	)

	if err := conn.QueryRow(
		ctx,
		"select cast(123 as bigint), cast(45.67 as numeric(9,2)) from rdb$database",
	).Scan(&id, &amount); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(id.String(), amount.String())

	// Output:
	// 123 45.67
}

func ExampleConn_Exec_array() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "recreate table fbx_example_samples(id integer not null primary key, nums smallint[1:3])")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = conn.Exec(ctx, "insert into fbx_example_samples(id, nums) values (?, ?)", 1, []int16{1, 2, 3})
	if err != nil {
		fmt.Println(err)
		return
	}

	var nums []int16
	if err := conn.QueryRow(ctx, "select nums from fbx_example_samples where id=?", 1).Scan(&nums); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(nums)

	// Output:
	// [1 2 3]
}

func ExampleCollectRows_structByName() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	type Employee struct {
		EmpNo     int32  `db:"EMP_NO"`
		FirstName string `db:"FIRST_NAME"`
	}

	conn, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, "select emp_no, first_name from employee where emp_no = ?", 144)
	if err != nil {
		fmt.Println(err)
		return
	}

	employees, err := fbx.CollectRows(rows, fbx.RowToStructByName[Employee])
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%d %s\n", employees[0].EmpNo, employees[0].FirstName)

	// Output:
	// 144 John
}

func ExampleConnectConfig_tracing() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	config, err := fbx.ParseConfig(dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	config.Tracer = &tracelog.TraceLog{
		Logger: tracelog.LoggerFunc(func(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
		}),
		LogLevel: tracelog.LogLevelInfo,
	}

	conn, err := fbx.ConnectConfig(ctx, config)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(ctx)

	var firstName string
	if err := conn.QueryRow(ctx, "select first_name from employee where emp_no=?", 2).Scan(&firstName); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(config.Tracer != nil, firstName)

	// Output:
	// true Robert
}
