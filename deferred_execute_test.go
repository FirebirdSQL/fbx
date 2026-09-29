package fbx_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
)

func TestDeferredExecuteHighLevel(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}
	ctx := context.Background()
	for _, mode := range []fbx.QueryExecMode{fbx.QueryExecModeCacheStatement, fbx.QueryExecModeExec} {
		t.Run(mode.String(), func(t *testing.T) {
			config, err := fbx.ParseConfig(connString)
			require.NoError(t, err)
			config.DefaultQueryExecMode = mode
			conn, err := fbx.ConnectConfig(ctx, config)
			require.NoError(t, err)
			defer func() { require.NoError(t, conn.Close(ctx)) }()

			var got int32
			require.NoError(t, conn.QueryRow(ctx, "SELECT 42 FROM RDB$DATABASE").Scan(&got))
			require.Equal(t, int32(42), got)

			rows, err := conn.Query(ctx, "SELECT 1 FROM RDB$DATABASE")
			require.NoError(t, err)
			rows.Close()
			require.NoError(t, rows.Err())
			require.NoError(t, conn.QueryRow(ctx, "SELECT 43 FROM RDB$DATABASE").Scan(&got))
			require.Equal(t, int32(43), got)
		})
	}
}
