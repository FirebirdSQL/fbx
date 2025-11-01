package fbx_test

import (
	"context"
	"os"
	"testing"

	"github.com/red-soft/fbx"
)

func skipCockroachDB(t testing.TB, msg string) {
	conn, err := fbx.Connect(context.Background(), os.Getenv("PGX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())

	if conn.FbConn().ParameterStatus("crdb_version") != "" {
		t.Skip(msg)
	}
}
