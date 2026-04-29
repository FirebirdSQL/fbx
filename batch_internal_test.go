package fbx

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"rdb.red-soft.ru/fbx/fbconn"
)

func skipIfBatchUnsupported(ctx context.Context, t testing.TB, conn *Conn) {
	t.Helper()
	engineVersion := fetchEngineVersion(ctx, t, conn)
	if parseEngineVersionNumber(engineVersion) < 40000 {
		t.Skipf("batch is not supported by server engine version %q", engineVersion)
	}
}

func forceStatementActiveBatch(t testing.TB, st *fbconn.Statement) {
	t.Helper()
	require.NotNil(t, st)

	field := reflect.ValueOf(st).Elem().FieldByName("activeBatch")
	require.True(t, field.IsValid())

	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(&fbconn.Batch{}))
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
	require.False(t, pbDefault.batchCfg.MultiError)
	require.True(t, pbDefault.batchCfg.RecordCounts)
	require.Equal(t, 1, pbDefault.batchCfg.DetailedErrs)
	require.NoError(t, pbDefault.Close(ctx))

	pbConn, err := conn.PrepareBatchWithOptions(ctx, "update "+table+" set note = ? where id = ?", BatchOptions{
		BufferBytes: 2048,
	})
	require.NoError(t, err)
	require.Equal(t, 2048, pbConn.batchCfg.BufferBytes)
	require.False(t, pbConn.batchCfg.MultiError)
	require.True(t, pbConn.batchCfg.RecordCounts)
	require.Equal(t, 1, pbConn.batchCfg.DetailedErrs)
	require.NoError(t, pbConn.Close(ctx))

	pbMulti, err := conn.PrepareBatchWithOptions(ctx, "update "+table+" set note = ? where id = ?", BatchOptions{
		ContinueOnError: true,
	})
	require.NoError(t, err)
	require.True(t, pbMulti.batchCfg.MultiError)
	require.True(t, pbMulti.batchCfg.RecordCounts)
	require.Equal(t, -1, pbMulti.batchCfg.DetailedErrs)
	require.NoError(t, pbMulti.Close(ctx))

	pbMultiDetailed, err := conn.PrepareBatchWithOptions(ctx, "update "+table+" set note = ? where id = ?", BatchOptions{
		ContinueOnError: true,
		DetailedErrors:  2,
	})
	require.NoError(t, err)
	require.True(t, pbMultiDetailed.batchCfg.MultiError)
	require.True(t, pbMultiDetailed.batchCfg.RecordCounts)
	require.Equal(t, 2, pbMultiDetailed.batchCfg.DetailedErrs)
	require.NoError(t, pbMultiDetailed.Close(ctx))

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

func TestPrepareBatchConnectionScopeClosesMetadataTransaction(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()

	table := "batch_internal_tx_meta"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	require.False(t, conn.fbConn.ActiveTx())

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	require.False(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Close(ctx))
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPrepareBatchRejectClosesMetadataTransaction(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()

	_, err := conn.PrepareBatch(ctx, "select cast(? as integer) from rdb$database")
	require.Error(t, err)
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPreparedBatchFlushCloseReleasesOwnedInternalTransaction(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_internal_tx_flush_obj"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	require.False(t, conn.fbConn.ActiveTx())

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	require.NoError(t, pb.Add(1, "pending"))
	require.NoError(t, pb.Flush(ctx))
	require.True(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Close(ctx))
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPreparedBatchCloseCanceledContextCanRetry(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_close_retry"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	require.NoError(t, pb.Add(1, "pending"))
	require.NoError(t, pb.Flush(ctx))
	require.True(t, conn.fbConn.ActiveTx())

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = pb.Close(canceledCtx)
	require.Error(t, err)
	require.False(t, pb.closed)
	require.NotNil(t, pb.lb)
	require.True(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Close(ctx))
	require.True(t, pb.closed)
	require.Nil(t, pb.lb)
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPreparedBatchExecCanceledCleanupCanRetry(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_exec_retry"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	require.NoError(t, pb.Add(1, "pending"))
	require.NoError(t, pb.Flush(ctx))
	require.True(t, conn.fbConn.ActiveTx())

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = pb.Exec(canceledCtx)
	require.Error(t, err)
	require.False(t, pb.closed)
	require.NotNil(t, pb.lb)
	require.True(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Close(ctx))
	require.True(t, pb.closed)
	require.Nil(t, pb.lb)
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPreparedBatchFlushCanceledCanRetry(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_flush_retry"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	defer pb.Close(ctx)

	require.NoError(t, pb.Add(1, "first"))
	require.NoError(t, pb.Flush(ctx))
	require.True(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Add(2, "second"))
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = pb.Flush(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)
	require.NotNil(t, pb.lb)
	require.NotZero(t, pb.PendingBytes())
	require.True(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Flush(ctx))
	require.Zero(t, pb.PendingBytes())

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Affected)
	require.False(t, conn.fbConn.ActiveTx())
}

func TestTxPreparedBatchCloseCanceledStatementCanRetry(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_close_retry_tx"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

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

	pb, err := tx.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	require.NotNil(t, pb.st)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = pb.Close(canceledCtx)
	require.Error(t, err)
	require.False(t, pb.closed)
	require.NotNil(t, pb.st)

	require.NoError(t, pb.Close(ctx))
	require.True(t, pb.closed)
	require.Nil(t, pb.st)
}

func TestTxCommitCanceledKeepsNamedBatchForRetry(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_tx_commit_retry"
	name := "batch_tx_commit_retry_named"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

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

	_, err = tx.Prepare(ctx, name, "batch insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	_, err = tx.Exec(ctx, name, 1, "pending")
	require.NoError(t, err)
	require.NoError(t, tx.Flush(ctx, name))
	require.Contains(t, conn.preparedBatches, name)
	require.True(t, conn.fbConn.ActiveTx())

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = tx.Commit(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, tx.closed)
	require.Contains(t, conn.preparedBatches, name)
	require.True(t, conn.fbConn.ActiveTx())

	require.NoError(t, tx.Commit(ctx))
	require.True(t, tx.closed)
	require.NotContains(t, conn.preparedBatches, name)
	require.False(t, conn.fbConn.ActiveTx())
}

func TestNamedBatchFlushDeallocateReleasesOwnedInternalTransaction(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_internal_tx_flush_named"
	name := "batch_internal_tx_named"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	require.False(t, conn.fbConn.ActiveTx())

	_, err = conn.Prepare(ctx, name, "batch insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, name, 1, "pending")
	require.NoError(t, err)
	require.NoError(t, conn.Flush(ctx, name))
	require.True(t, conn.fbConn.ActiveTx())

	require.NoError(t, conn.Deallocate(ctx, name))
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPreparedBatchResetClearsBufferedRows(t *testing.T) {
	row1 := []byte("row-one")
	row2 := []byte("row-two")

	pb := &PreparedBatch{
		encodedRows:  [][]byte{row1, row2},
		pendingBytes: len(row1) + len(row2),
	}

	backing := pb.encodedRows[:len(pb.encodedRows)]

	pb.Reset()

	require.Zero(t, pb.pendingBytes)
	require.Len(t, pb.encodedRows, 0)
	require.Nil(t, backing[0])
	require.Nil(t, backing[1])
}

func TestFinalizePreparedBatchExecJoinsErrors(t *testing.T) {
	t.Run("execute error keeps cleanup errors", func(t *testing.T) {
		execErr := errors.New("execute failed")
		closeErr := errors.New("close failed")
		releaseErr := errors.New("release failed")

		result, err := finalizePreparedBatchExec(nil, execErr, closeErr, releaseErr)

		require.Zero(t, result.Affected)
		require.ErrorContains(t, err, "execute failed")
		require.ErrorContains(t, err, "close failed")
		require.ErrorContains(t, err, "release failed")
	})

	t.Run("batch row error keeps affected and cleanup errors", func(t *testing.T) {
		closeErr := errors.New("close failed")
		releaseErr := errors.New("release failed")

		completion := &fbconn.BatchCompletion{
			UpdateCounts: []int64{1},
			Errors: []fbconn.BatchError{
				{Row: 2, Err: errors.New("duplicate key")},
			},
		}

		result, err := finalizePreparedBatchExec(completion, nil, closeErr, releaseErr)

		require.EqualValues(t, 1, result.Affected)
		require.ErrorContains(t, err, "batch failed at row 2")
		require.ErrorContains(t, err, "duplicate key")
		require.ErrorContains(t, err, "close failed")
		require.ErrorContains(t, err, "release failed")

		var batchErr *BatchExecutionError
		require.ErrorAs(t, err, &batchErr)
		require.Equal(t, []BatchError{{Row: 2, Err: completion.Errors[0].Err}}, batchErr.Errors)
	})

	t.Run("multiple batch row errors are exposed", func(t *testing.T) {
		completion := &fbconn.BatchCompletion{
			UpdateCounts: []int64{1, -1, 1, -1},
			Errors: []fbconn.BatchError{
				{Row: 1, Err: errors.New("first duplicate")},
				{Row: 3, Err: errors.New("second duplicate")},
				{Row: 5},
			},
		}

		result, err := finalizePreparedBatchExec(completion, nil, nil, nil)

		require.EqualValues(t, 2, result.Affected)
		require.ErrorContains(t, err, "batch failed at row 1")
		require.ErrorContains(t, err, "2 more batch errors")

		var batchErr *BatchExecutionError
		require.ErrorAs(t, err, &batchErr)
		require.Len(t, batchErr.Errors, 3)
		require.Equal(t, 1, batchErr.Errors[0].Row)
		require.EqualError(t, batchErr.Errors[0].Err, "first duplicate")
		require.Equal(t, 3, batchErr.Errors[1].Row)
		require.EqualError(t, batchErr.Errors[1].Err, "second duplicate")
		require.Equal(t, 5, batchErr.Errors[2].Row)
		require.Nil(t, batchErr.Errors[2].Err)
	})
}

func TestTxCommitDoesNotStopOnNamedBatchCloseError(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	checkConn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := checkConn.Close(ctx)
		require.NoError(t, err)
	}()

	table := "tx_commit_named_batch_close_err_test"

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	txi, err := conn.Begin(ctx)
	require.NoError(t, err)
	tx, ok := txi.(*dbTx)
	require.True(t, ok)

	_, err = tx.Exec(ctx, "insert into "+table+"(id, note) values (?, ?)", 1, "committed")
	require.NoError(t, err)

	_, err = tx.Prepare(ctx, "commit_close_err_batch", "batch update "+table+" set note = ? where id = ?")
	require.NoError(t, err)

	pb := conn.preparedBatches["commit_close_err_batch"]
	require.NotNil(t, pb)
	require.NotNil(t, pb.st)

	forceStatementActiveBatch(t, pb.st)

	err = tx.Commit(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "statement has active batch")
	require.True(t, tx.closed)
	require.NotContains(t, conn.preparedBatches, "commit_close_err_batch")

	var note string
	err = checkConn.QueryRow(ctx, "select note from "+table+" where id = ?", 1).Scan(&note)
	require.NoError(t, err)
	require.Equal(t, "committed", note)
}

func TestTxRollbackDoesNotStopOnNamedBatchCloseError(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	checkConn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := checkConn.Close(ctx)
		require.NoError(t, err)
	}()

	table := "tx_rollback_named_batch_close_err_test"

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	txi, err := conn.Begin(ctx)
	require.NoError(t, err)
	tx, ok := txi.(*dbTx)
	require.True(t, ok)

	_, err = tx.Exec(ctx, "insert into "+table+"(id, note) values (?, ?)", 1, "rolled back")
	require.NoError(t, err)

	_, err = tx.Prepare(ctx, "rollback_close_err_batch", "batch update "+table+" set note = ? where id = ?")
	require.NoError(t, err)

	pb := conn.preparedBatches["rollback_close_err_batch"]
	require.NotNil(t, pb)
	require.NotNil(t, pb.st)

	forceStatementActiveBatch(t, pb.st)

	err = tx.Rollback(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "statement has active batch")
	require.True(t, tx.closed)
	require.NotContains(t, conn.preparedBatches, "rollback_close_err_batch")

	var count int
	err = checkConn.QueryRow(ctx, "select count(*) from "+table).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}
