package stdlib_test

import (
	"context"
	"database/sql"
	"fmt"

	_ "rdb.red-soft.ru/fbx/stdlib"
)

func Example() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	db, err := sql.Open("fbx", dsn)
	if err != nil {
		return
	}
	defer db.Close()

	var firstName string
	if err := db.QueryRowContext(ctx, "select first_name from employee where emp_no=?", 2).Scan(&firstName); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(firstName)

	// Output:
	// Robert
}
