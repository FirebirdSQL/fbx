package stdlib_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxpool"
	"rdb.red-soft.ru/fbx/stdlib"
)

func Example() {
	ctx := context.Background()

	db, err := sql.Open("fbx", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var firstName string
	err = db.QueryRowContext(ctx, "select first_name from employee where emp_no = ?", 2).Scan(&firstName)
	if err != nil {
		panic(err)
	}

	fmt.Println(firstName)
}

func ExampleOpenDB() {
	ctx := context.Background()

	connConfig, err := fbx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	connConfig.DefaultQueryExecMode = fbx.QueryExecModeExec

	db := stdlib.OpenDB(*connConfig)
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
}

func ExampleOpenDB_withOptions() {
	ctx := context.Background()

	connConfig, err := fbx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	db := stdlib.OpenDB(*connConfig,
		stdlib.OptionBeforeConnect(func(_ context.Context, config *fbx.ConnConfig) error {
			config.ConnectTimeout = 5 * time.Second
			return nil
		}),
		stdlib.OptionAfterConnect(func(ctx context.Context, conn *fbx.Conn) error {
			return conn.Ping(ctx)
		}),
		stdlib.OptionShouldPing(func(_ context.Context, params stdlib.ShouldPingParams) bool {
			return params.IdleDuration > 30*time.Second
		}),
		stdlib.OptionResetSession(func(_ context.Context, conn *fbx.Conn) error {
			if conn.IsClosed() {
				return driver.ErrBadConn
			}
			return nil
		}),
	)
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
}

func ExampleGetConnector() {
	ctx := context.Background()

	connConfig, err := fbx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	connector := stdlib.GetConnector(*connConfig)
	db := sql.OpenDB(connector)
	defer db.Close()

	var one int
	if err := db.QueryRowContext(ctx, "select 1 from rdb$database").Scan(&one); err != nil {
		panic(err)
	}
}

func ExampleRandomizeHostOrderFunc() {
	ctx := context.Background()

	connConfig, err := fbx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	db := stdlib.OpenDB(*connConfig,
		stdlib.OptionBeforeConnect(stdlib.RandomizeHostOrderFunc),
	)
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
}

func ExampleOpenDBFromPool() {
	ctx := context.Background()

	pool, err := fbxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	var one int
	if err := db.QueryRowContext(ctx, "select 1 from rdb$database").Scan(&one); err != nil {
		panic(err)
	}
}

func ExampleRegisterConnConfig() {
	ctx := context.Background()

	connConfig, err := fbx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	connConfig.DefaultQueryExecMode = fbx.QueryExecModeExec

	connString := stdlib.RegisterConnConfig(connConfig)
	defer stdlib.UnregisterConnConfig(connString)

	db, err := sql.Open("fbx", connString)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var one int
	if err := db.QueryRowContext(ctx, "select 1 from rdb$database").Scan(&one); err != nil {
		panic(err)
	}
}

func ExampleConn_Conn() {
	ctx := context.Background()

	db, err := sql.Open("fbx", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sqlConn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	defer sqlConn.Close()

	err = sqlConn.Raw(func(driverConn any) error {
		nativeConn := driverConn.(*stdlib.Conn).Conn()
		version := nativeConn.ServerVersion()
		fmt.Println(version.Major)
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func Example_firebirdTypes() {
	ctx := context.Background()

	db, err := sql.Open("fbx", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	arrayInput := []int64{1, 2, 3}
	_, err = db.ExecContext(ctx, "insert into array_example(id, values_array) values (?, ?)", 1, arrayInput)
	if err != nil {
		panic(err)
	}

	m := fbtype.NewMap()
	var arrayOutput []int64
	err = db.QueryRowContext(ctx, "select values_array from array_example where id = ?", 1).Scan(m.SQLScanner(&arrayOutput))
	if err != nil {
		panic(err)
	}

	blobInput := bytes.NewReader([]byte("document body"))
	_, err = db.ExecContext(ctx, "insert into document_example(id, body) values (?, ?)", 1, blobInput)
	if err != nil {
		panic(err)
	}

	var blob stdlib.Blob
	err = db.QueryRowContext(ctx, "select body from document_example where id = ?", 1).Scan(&blob)
	if err != nil {
		panic(err)
	}
	defer blob.Close()
}

func ExampleBlob_Text() {
	ctx := context.Background()

	db, err := sql.Open("fbx", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var blob stdlib.Blob
	err = db.QueryRowContext(ctx, "select body from document_example where id = ?", 1).Scan(&blob)
	if err != nil {
		panic(err)
	}

	text, err := blob.Text()
	if err != nil {
		panic(err)
	}

	fmt.Println(text)
}

func ExampleBlob_Read() {
	ctx := context.Background()

	db, err := sql.Open("fbx", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var blob stdlib.Blob
	err = db.QueryRowContext(ctx, "select body from document_example where id = ?", 1).Scan(&blob)
	if err != nil {
		panic(err)
	}
	defer blob.Close()

	data, err := io.ReadAll(blob)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))
}
