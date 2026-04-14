# fbx - Red Database and Firebird Driver for Go

fbx is a pure Go driver for Red Database and Firebird.

## Example Usage

**go.mod**

```txt
module fbxdemo

replace rdb.red-soft.ru/fbx v0.0.0 => go.gitflic.ru/red-soft/fbx main

go 1.24.2

require rdb.red-soft.ru/fbx v0.0.0
```

```shell
go mod tidy
```

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
	url := "firebird://sysdba:masterkey@localhost:3060/employee"
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

## Firebird Events (`POST_EVENT`)

Use package `rdb.red-soft.ru/fbx/fbevent` for high-level event subscriptions.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	fbevent "rdb.red-soft.ru/fbx/fbevent"
)

func main() {
	ctx := context.Background()
	connString := "firebird://sysdba:masterkey@localhost:3050/employee"

	manager, err := fbevent.NewManager(connString)
	if err != nil {
		log.Fatal(err)
	}
	if err := manager.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer manager.Close(context.Background())

	// Wait for one event occurrence. timeout == 0 means wait indefinitely.
	count, err := manager.WaitForEvent(ctx, "TEST_EVENT", 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	if count == -1 {
		fmt.Println("timeout")
		return
	}

	fmt.Printf("TEST_EVENT count delta: %d\n", count)
}
```
