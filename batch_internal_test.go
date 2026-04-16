package fbx

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func skipIfBatchUnsupported(ctx context.Context, t testing.TB, conn *Conn) {
	t.Helper()
	engineVersion := fetchEngineVersion(ctx, t, conn)
	if parseEngineVersionNumber(engineVersion) < 40000 {
		t.Skipf("batch is not supported by server engine version %q", engineVersion)
	}
}

func TestPrepareBatchWithOptionsPropagatesConfig(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "prepare_batch_with_options_cfg_test"

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pbDefault, err := conn.PrepareBatch(ctx, "update "+table+" set note = ? where id = ?")
	require.NoError(t, err)
	require.Equal(t, -1, pbDefault.batchCfg.BufferBytes)
	require.True(t, pbDefault.batchCfg.RecordCounts)
	require.Equal(t, 1, pbDefault.batchCfg.DetailedErrs)
	require.NoError(t, pbDefault.Close(ctx))

	pbConn, err := conn.PrepareBatchWithOptions(ctx, "update "+table+" set note = ? where id = ?", BatchOptions{
		BufferBytes: 2048,
	})
	require.NoError(t, err)
	require.Equal(t, 2048, pbConn.batchCfg.BufferBytes)
	require.True(t, pbConn.batchCfg.RecordCounts)
	require.Equal(t, 1, pbConn.batchCfg.DetailedErrs)
	require.NoError(t, pbConn.Close(ctx))

	txi, err := conn.Begin(ctx)
	require.NoError(t, err)
	tx, ok := txi.(*dbTx)
	require.True(t, ok)
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && err != ErrTxClosed {
			t.Fatalf("tx.Rollback unexpectedly failed: %v", err)
		}
	}()

	pbTx, err := tx.PrepareBatchWithOptions(ctx, "update "+table+" set note = ? where id = ?", BatchOptions{
		BufferBytes: 1024,
	})
	require.NoError(t, err)
	require.Equal(t, 1024, pbTx.batchCfg.BufferBytes)
	require.NotNil(t, pbTx.st)
	require.NoError(t, pbTx.Close(ctx))
}
