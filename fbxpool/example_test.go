package fbxpool_test

import (
	"context"
	"fmt"

	"github.com/firebirdsql/fbx/fbxpool"
)

func ExampleNew() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	pool, err := fbxpool.New(ctx, dsn)
	if err != nil {
		return
	}
	defer pool.Close()

	var firstName string
	if err := pool.QueryRow(ctx, "select first_name from employee where emp_no=?", 2).Scan(&firstName); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(firstName)

	// Output:
	// Robert
}
