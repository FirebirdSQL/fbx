package fbx

import (
	"context"
	"errors"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/internal/operationerror"
)

func skipIfBatchUnsupported(ctx context.Context, t testing.TB, conn *Conn) {
	t.Helper()
	version := conn.ServerVersion()
	if !version.AtLeast(4, 0, 0) {
		t.Skipf("batch is not supported by server engine version %q", version)
	}
}

type cancelAfterBatchPrepare struct {
	cancel context.CancelFunc
}

func (tracer cancelAfterBatchPrepare) TracePrepareStart(ctx context.Context, _ *Conn, _ TracePrepareStartData) context.Context {
	return ctx
}

func (tracer cancelAfterBatchPrepare) TracePrepareEnd(context.Context, *Conn, TracePrepareEndData) {
	tracer.cancel()
}

func TestExecBatchRowsCanceledAfterPrepare(t *testing.T) {
	for _, mode := range []QueryExecMode{QueryExecModeCacheStatement, QueryExecModeExec} {
		t.Run(mode.String(), func(t *testing.T) {
			ctx := context.Background()
			config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
			config.DefaultQueryExecMode = mode
			conn := mustConnect(t, config)
			defer func() { require.NoError(t, conn.Close(ctx)) }()
			skipIfBatchUnsupported(ctx, t, conn)
			_, err := conn.Exec(ctx, "recreate table exec_batch_cancel_test (id integer)")
			require.NoError(t, err)

			canceledCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			conn.prepareTracer = cancelAfterBatchPrepare{cancel: cancel}
			_, err = conn.Exec(canceledCtx, "insert into exec_batch_cancel_test values (?)", BatchRows{{1}})
			require.ErrorIs(t, err, context.Canceled)
			require.False(t, conn.IsClosed())
			require.False(t, conn.fbConn.ActiveTx())
		})
	}
}

func TestBatchInsertRejectsBeforePrepare(t *testing.T) {
	conn := &Conn{config: &ConnConfig{BatchInsertPendingSize: -1}}
	source := BatchInsertFunc(func() ([]any, error) {
		t.Fatal("source must not be read")
		return nil, nil
	})
	_, err := conn.BatchInsert(context.Background(), Identifier{"T"}, []string{"ID"}, source)
	require.ErrorContains(t, err, "pending size must not be negative")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = conn.BatchInsert(ctx, Identifier{"T"}, []string{"ID"}, source)
	require.ErrorIs(t, err, context.Canceled)
}

func TestBatchInsertSourceErrors(t *testing.T) {
	for _, stage := range []string{"before Next", "Next", "Values", "end of source", "first Next error"} {
		t.Run(stage, func(t *testing.T) {
			config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
			config.BatchInsertPendingSize = 0
			conn := mustConnect(t, config)
			defer func() { require.NoError(t, conn.Close(context.Background())) }()
			skipIfBatchUnsupported(context.Background(), t, conn)
			_, err := conn.Exec(context.Background(), "recreate table batch_insert_source (id integer)")
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sourceErr := operationerror.WithRetry(errors.New("source failed"), true)
			calls := 0
			next := func() ([]any, error) {
				calls++
				if stage == "first Next error" {
					return nil, sourceErr
				}
				if calls == 1 {
					return []any{1}, nil
				}
				cancel()
				if stage == "end of source" {
					return nil, sourceErr
				}
				return []any{2}, nil
			}
			source := BatchInsertFunc(next)
			if stage == "Values" {
				source = BatchInsertSlice(3, func(int) ([]any, error) { return next() })
			}
			if stage == "before Next" {
				conn.prepareTracer = cancelAfterBatchPrepare{cancel: cancel}
			}
			count, err := conn.BatchInsert(ctx, Identifier{"BATCH_INSERT_SOURCE"}, []string{"ID"}, source)
			if stage == "first Next error" {
				require.ErrorIs(t, err, sourceErr)
				require.Equal(t, 1, calls)
			} else {
				require.ErrorIs(t, err, context.Canceled)
				if stage == "before Next" {
					require.Zero(t, calls)
				} else {
					require.Equal(t, 2, calls)
				}
			}
			if stage == "end of source" {
				require.ErrorIs(t, err, sourceErr)
			}
			require.Zero(t, count)
			require.False(t, fbconn.SafeToRetry(err))
			require.False(t, conn.IsClosed())
			require.False(t, conn.fbConn.ActiveTx())
			conn.prepareTracer = nil
			require.Zero(t, countStatementHandlesForSQL(t, conn, `insert into "BATCH_INSERT_SOURCE" ("ID") values (?)`))
			var stored int
			require.NoError(t, conn.QueryRow(context.Background(), "select count(*) from batch_insert_source").Scan(&stored))
			require.Zero(t, stored)
		})
	}
}

func TestBatchInsertCleanupFailureClosesConnection(t *testing.T) {
	ctx := context.Background()
	config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	config.BatchInsertPendingSize = 0
	conn := mustConnect(t, config)
	defer func() { require.NoError(t, conn.Close(ctx)) }()
	skipIfBatchUnsupported(ctx, t, conn)
	_, err := conn.Exec(ctx, "recreate table batch_insert_cleanup (id integer)")
	require.NoError(t, err)

	sourceErr := operationerror.WithRetry(errors.New("source failed"), true)
	var rows Rows
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	source := BatchInsertSlice(2, func(i int) ([]any, error) {
		if i == 0 {
			return []any{1}, nil
		}
		// Leave a result open so cleanup cannot acquire the connection.
		rows, err = conn.Query(ctx, "select 1 from rdb$database")
		require.NoError(t, err)
		return nil, sourceErr
	})
	count, err := conn.BatchInsert(ctx, Identifier{"BATCH_INSERT_CLEANUP"}, []string{"ID"}, source)
	require.Zero(t, count)
	require.ErrorIs(t, err, sourceErr)
	require.ErrorContains(t, err, "conn busy")
	require.False(t, fbconn.SafeToRetry(err))
	require.True(t, conn.IsClosed())
}

func TestPrepareBatchRejectsInvalidOptionsBeforePrepare(t *testing.T) {
	type testCase struct {
		name string
		opts BatchOptions
		err  string
	}
	tests := []testCase{
		{"negative buffer", BatchOptions{BufferBytes: -1}, "batch buffer size"},
		{"negative details", BatchOptions{DetailedErrors: -1}, "batch detailed error limit"},
	}
	if strconv.IntSize == 64 {
		overflow := int64(math.MaxInt32) + 1
		tests = append(tests,
			testCase{"buffer overflow", BatchOptions{BufferBytes: int(overflow)}, "batch buffer size"},
			testCase{"details overflow", BatchOptions{DetailedErrors: int(overflow)}, "batch detailed error limit"},
		)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Conn{}
			tx := &dbTx{conn: conn}
			for _, prepare := range []func(context.Context, string, BatchOptions) (*PreparedBatch, error){
				conn.PrepareBatchWithOptions, tx.PrepareBatchWithOptions,
			} {
				batch, err := prepare(context.Background(), "insert into t values (?)", tt.opts)
				require.ErrorContains(t, err, tt.err)
				require.Nil(t, batch)
			}
		})
	}
}

func TestNormalizeBatchOptions(t *testing.T) {
	tests := []struct {
		name string
		opts BatchOptions
		want fbconn.BatchConfig
	}{
		{"defaults", BatchOptions{}, fbconn.BatchConfig{RecordCounts: true, BufferBytes: -1, DetailedErrs: 1}},
		{"multi-error defaults", BatchOptions{ContinueOnError: true}, fbconn.BatchConfig{MultiError: true, RecordCounts: true, BufferBytes: -1, DetailedErrs: -1}},
		{"maximum wire values", BatchOptions{BufferBytes: math.MaxInt32, DetailedErrors: math.MaxInt32, ContinueOnError: true}, fbconn.BatchConfig{MultiError: true, RecordCounts: true, BufferBytes: math.MaxInt32, DetailedErrs: math.MaxInt32}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := normalizeBatchOptions(tt.opts)
			require.NoError(t, err)
			require.Equal(t, tt.want, cfg)
		})
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
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_internal_tx_meta"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	require.False(t, conn.fbConn.ActiveTx())

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	require.False(t, conn.fbConn.ActiveTx())

	st := pb.st
	require.NotNil(t, st)
	var serverBatch *fbconn.Batch
	for _, id := range []int{1, 2} {
		require.NoError(t, pb.Add(id, "saved"))
		require.NoError(t, pb.Flush(ctx))
		if serverBatch == nil {
			serverBatch = pb.lb
		}
		require.True(t, pb.lb.HasPending())
		require.False(t, conn.fbConn.ActiveTx(), "Flush must not open a transaction")
		result, err := pb.Exec(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 1, result.Affected)
		require.Same(t, st, pb.st, "Exec must retain the original statement")
		require.Same(t, serverBatch, pb.lb, "Exec must retain the server batch")
		require.False(t, pb.lb.HasPending())
		require.False(t, conn.fbConn.ActiveTx())
	}

	require.NoError(t, pb.Add(3, "discarded"))
	require.NoError(t, pb.Flush(ctx))
	require.NoError(t, pb.Cancel(ctx))
	require.Same(t, st, pb.st, "Cancel must retain the original statement")
	require.Same(t, serverBatch, pb.lb, "Cancel must retain the server batch")
	require.False(t, pb.lb.HasPending())

	require.NoError(t, pb.Close(ctx))
	require.Nil(t, pb.st)
	require.False(t, conn.fbConn.ActiveTx())
	var count int
	require.NoError(t, conn.QueryRow(ctx, "select count(*) from "+table).Scan(&count))
	require.Equal(t, 2, count)
}

func TestPrepareBatchRejectClosesMetadataTransaction(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()

	for _, sql := range []string{"select cast(? as integer) from rdb$database", "invalid batch SQL"} {
		prepareCtx, cancel := context.WithCancel(ctx)
		conn.prepareTracer = cancelAfterBatchPrepare{cancel: cancel}
		_, err := conn.PrepareBatch(prepareCtx, sql)
		cancel()
		require.Error(t, err)
		require.NotErrorIs(t, err, context.Canceled, "cleanup must preserve the preparation or validation error")
		require.False(t, conn.fbConn.ActiveTx())
		require.False(t, conn.IsClosed(), "cleanup after cancellation must preserve a healthy connection")
	}
}

func TestPrepareBatchClosesMetadataTransactionWithAnotherTxActive(t *testing.T) {
	ctx := context.Background()
	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() { require.NoError(t, conn.Close(ctx)) }()

	firstTx, err := conn.Begin(ctx)
	require.NoError(t, err)
	lastTx, err := conn.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, lastTx.Rollback(ctx))

	_, err = conn.PrepareBatch(ctx, "select cast(? as integer) from rdb$database")
	require.Error(t, err)
	require.True(t, conn.fbConn.ActiveTx(), "the first user transaction must remain active")
	require.NoError(t, firstTx.Rollback(ctx))
	require.False(t, conn.fbConn.ActiveTx(), "preparation must not leave its own transaction behind")
}

func TestPreparedBatchClosePreservesLaterTransaction(t *testing.T) {
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
	require.False(t, conn.fbConn.ActiveTx())

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, "insert into "+table+" (id, note) values (2, 'user tx')")
	require.NoError(t, err)
	require.NoError(t, pb.Close(ctx))
	require.True(t, conn.fbConn.ActiveTx(), "Close must not finish the user's transaction")
	var count int
	require.NoError(t, tx.QueryRow(ctx, "select count(*) from "+table).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, tx.Rollback(ctx))
	require.False(t, conn.fbConn.ActiveTx(), "the batch must not leave an old metadata transaction behind")
	require.NoError(t, conn.QueryRow(ctx, "select count(*) from "+table).Scan(&count))
	require.Zero(t, count)
}

func TestPreparedBatchPreservesImplicitTransaction(t *testing.T) {
	ctx := context.Background()
	config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	config.Autocommit = false
	conn := mustConnect(t, config)
	defer func() { require.NoError(t, conn.Close(ctx)) }()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_manual_commit"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer)")
	require.NoError(t, err)
	require.NoError(t, conn.Commit(ctx))
	_, err = conn.Exec(ctx, "insert into "+table+" values (1)")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" values (?)")
	require.NoError(t, err)
	require.NoError(t, pb.Add(2))
	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Affected)
	require.NoError(t, pb.Close(ctx))
	require.True(t, conn.fbConn.ActiveTx())
	_, err = conn.Exec(ctx, "insert into "+table+" values (?)", BatchRows{{3}})
	require.NoError(t, err)
	require.True(t, conn.fbConn.ActiveTx(), "BatchRows must preserve the manually managed transaction")
	var count int
	require.NoError(t, conn.QueryRow(ctx, "select count(*) from "+table).Scan(&count))
	require.Equal(t, 3, count, "both batch APIs must preserve earlier uncommitted work")
	require.NoError(t, conn.Rollback(ctx))
	require.False(t, conn.fbConn.ActiveTx())
	require.NoError(t, conn.QueryRow(ctx, "select count(*) from "+table).Scan(&count))
	require.Zero(t, count, "Exec must not commit when Autocommit is disabled")
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
	require.False(t, conn.fbConn.ActiveTx())

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = pb.Close(canceledCtx)
	require.Error(t, err)
	require.False(t, pb.closed)
	require.NotNil(t, pb.lb)
	require.False(t, conn.fbConn.ActiveTx())
	require.ErrorContains(t, pb.Add(2, "blocked"), "closing")
	require.ErrorContains(t, pb.Flush(ctx), "closing")
	_, err = pb.Exec(ctx)
	require.ErrorContains(t, err, "closing")
	require.ErrorContains(t, pb.Cancel(ctx), "closing")

	require.NoError(t, pb.Close(ctx))
	require.True(t, pb.closed)
	require.Nil(t, pb.lb)
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPreparedBatchExecCanceledBeforeSendingCanRetry(t *testing.T) {
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
	require.False(t, conn.fbConn.ActiveTx())

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = pb.Exec(canceledCtx)
	require.Error(t, err)
	require.False(t, pb.closed)
	require.NotNil(t, pb.lb)
	require.False(t, conn.fbConn.ActiveTx(), "failed execution must release the owned internal transaction")
	require.True(t, pb.lb.HasPending())
	require.False(t, pb.lb.NeedsReset())
	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Affected)

	require.NoError(t, pb.Close(ctx))
	require.True(t, pb.closed)
	require.Nil(t, pb.lb)
	require.False(t, conn.fbConn.ActiveTx())
}

func TestPreparedBatchEmptyExecDoesNotReexecute(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_exec_cleanup_no_reexecute"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, n integer not null)")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "insert into "+table+" (id, n) values (?, ?)", 1, 0)
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "update "+table+" set n = n + 1 where id = ?")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, pb.Close(ctx))
	}()

	require.NoError(t, pb.Add(1))
	require.NoError(t, pb.Flush(ctx))

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Affected)
	require.NotNil(t, pb.lb)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	result, err = pb.Exec(canceledCtx)
	require.NoError(t, err, "empty Exec must not start a transaction or send a command")
	require.Zero(t, result.Affected)
	require.False(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Close(ctx))

	var n int
	err = conn.QueryRow(ctx, "select n from "+table+" where id = ?", 1).Scan(&n)
	require.NoError(t, err)
	require.Equal(t, 1, n)
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
	defer func() {
		require.NoError(t, pb.Close(ctx))
	}()

	require.NoError(t, pb.Add(1, "first"))
	require.NoError(t, pb.Flush(ctx))
	require.False(t, conn.fbConn.ActiveTx())

	require.NoError(t, pb.Add(2, "second"))
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = pb.Flush(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)
	require.NotNil(t, pb.lb)
	require.NotZero(t, pb.PendingBytes())
	require.False(t, conn.fbConn.ActiveTx())

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

func TestTxCommitCanceledPreservesPreparedBatch(t *testing.T) {
	ctx := context.Background()

	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	defer func() {
		err := conn.Close(ctx)
		require.NoError(t, err)
	}()
	skipIfBatchUnsupported(ctx, t, conn)

	table := "batch_tx_commit_retry"
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
	defer func() { require.NoError(t, pb.Close(ctx)) }()
	require.NoError(t, pb.Add(1, "pending"))
	require.NoError(t, pb.Flush(ctx))
	require.True(t, conn.fbConn.ActiveTx())

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = tx.Commit(canceledCtx)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, tx.closed)
	require.False(t, pb.closing)
	require.True(t, pb.lb.HasPending())
	require.True(t, conn.fbConn.ActiveTx())

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Affected)
	require.NoError(t, tx.Commit(ctx))
	require.True(t, tx.closed)
	require.False(t, conn.fbConn.ActiveTx())
	require.ErrorIs(t, pb.Add(2, "after commit"), ErrTxClosed)
	require.ErrorIs(t, pb.Flush(ctx), ErrTxClosed)
	_, err = pb.Exec(ctx)
	require.ErrorIs(t, err, ErrTxClosed)
	require.NoError(t, pb.Close(ctx))
	var count int
	require.NoError(t, conn.QueryRow(ctx, "select count(*) from "+table).Scan(&count))
	require.Equal(t, 1, count)
}

func TestPreparedBatchClearPendingClearsBufferedRows(t *testing.T) {
	row1 := []byte("row-one")
	row2 := []byte("row-two")

	pb := &PreparedBatch{
		encodedRows:  [][]byte{row1, row2},
		pendingBytes: len(row1) + len(row2),
	}

	backing := pb.encodedRows[:len(pb.encodedRows)]

	pb.ClearPending()

	require.Zero(t, pb.pendingBytes)
	require.Len(t, pb.encodedRows, 0)
	require.Nil(t, backing[0])
	require.Nil(t, backing[1])
}

func TestBatchExecutionErrorSurfacesSimplifiedRows(t *testing.T) {
	// Server returns mixed detailed + simplified row errors in MultiError mode.
	completion := &fbconn.BatchCompletion{
		UpdateCounts: []int64{-1, 1, -1, 1, -1},
		Errors: []fbconn.BatchError{
			{Row: 0, Err: errors.New("detailed: unique violation")},
			{Row: 2}, // simplified (server exhausted DetailedErrors budget)
			{Row: 4}, // simplified
		},
	}

	err := batchCompletionError(completion)
	require.Error(t, err)

	// errors.As reaches BatchExecutionError, errors are not silently dropped.
	var batchErr *BatchExecutionError
	require.ErrorAs(t, err, &batchErr)
	require.Len(t, batchErr.Errors, 3)

	// Detailed row keeps its original error verbatim.
	require.EqualError(t, batchErr.Errors[0].Err, "detailed: unique violation")
	require.NotErrorIs(t, batchErr.Errors[0].Err, ErrBatchRowFailed,
		"detailed errors must not be tagged as ErrBatchRowFailed")

	// Simplified rows now carry the sentinel so errors.Is finds them.
	require.ErrorIs(t, batchErr.Errors[1].Err, ErrBatchRowFailed)
	require.ErrorIs(t, batchErr.Errors[2].Err, ErrBatchRowFailed)
	require.Contains(t, batchErr.Errors[1].Err.Error(), "row 2",
		"the wrapped sentinel should mention the originating row")

	// Unwrap exposes every per-row error to errors.Is traversal.
	unwrapped := batchErr.Unwrap()
	require.Len(t, unwrapped, 3,
		"Unwrap must include every row error, including simplified ones")

	// And errors.Is on the whole BatchExecutionError finds the sentinel.
	require.ErrorIs(t, err, ErrBatchRowFailed,
		"errors.Is on the outer BatchExecutionError should find ErrBatchRowFailed")
}

func TestPreparedBatchCancelGuards(t *testing.T) {
	ctx := context.Background()

	t.Run("nil receiver returns descriptive error", func(t *testing.T) {
		var pb *PreparedBatch
		require.EqualError(t, pb.Cancel(ctx), "prepared batch is nil")
	})

	t.Run("closed batch refuses cancel", func(t *testing.T) {
		pb := &PreparedBatch{closed: true}
		require.EqualError(t, pb.Cancel(ctx), "prepared batch is closed")
	})

	t.Run("closing batch refuses cancel", func(t *testing.T) {
		pb := &PreparedBatch{closing: true}
		require.ErrorContains(t, pb.Cancel(ctx), "prepared batch is closing")
	})

	t.Run("cancel completes pending cleanup", func(t *testing.T) {
		pb := &PreparedBatch{cleanupPending: true}
		require.NoError(t, pb.Cancel(ctx))
		require.False(t, pb.cleanupPending)
	})

	t.Run("no wire state — Cancel clears local buffer and returns nil", func(t *testing.T) {
		row1 := []byte("row-one")
		row2 := []byte("row-two")
		pb := &PreparedBatch{
			encodedRows:  [][]byte{row1, row2},
			pendingBytes: len(row1) + len(row2),
		}

		require.NoError(t, pb.Cancel(ctx))
		require.Zero(t, pb.pendingBytes)
		require.Len(t, pb.encodedRows, 0)
		require.Nil(t, pb.lb)
	})
}

func TestFinalizePreparedBatchExecJoinsErrors(t *testing.T) {
	t.Run("unknown affected count is not an execution error", func(t *testing.T) {
		completion := &fbconn.BatchCompletion{TotalRecords: 2}

		result, err := finalizePreparedBatchExec(completion, nil)

		require.NoError(t, err)
		require.EqualValues(t, -1, result.Affected)
		require.Nil(t, result.UpdateCounts)
	})

	t.Run("completion survives cancellation and retryable cleanup", func(t *testing.T) {
		rowErr := errors.New("row cancelled")
		completion := &fbconn.BatchCompletion{
			TotalRecords: 2,
			UpdateCounts: []int64{1, -1},
			Errors:       []fbconn.BatchError{{Row: 1, Err: rowErr}},
		}
		cleanupErr := operationerror.WithRetry(errors.New("cleanup failed"), true)
		result, err := finalizePreparedBatchExec(completion, operationerror.Join(context.DeadlineExceeded, cleanupErr))

		require.EqualValues(t, 1, result.Affected)
		require.Equal(t, completion.UpdateCounts, result.UpdateCounts)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, rowErr)
		require.ErrorIs(t, err, cleanupErr)
		require.False(t, fbconn.SafeToRetry(err))
		var batchErr *BatchExecutionError
		require.ErrorAs(t, err, &batchErr)
	})

	t.Run("execute error keeps cleanup errors", func(t *testing.T) {
		execErr := errors.New("execute failed")
		cleanupErr := errors.New("cancel failed")

		result, err := finalizePreparedBatchExec(nil, operationerror.Join(execErr, cleanupErr))

		require.Zero(t, result.Affected)
		require.ErrorContains(t, err, "execute failed")
		require.ErrorIs(t, err, cleanupErr)
	})

	t.Run("batch row error keeps affected and cleanup errors", func(t *testing.T) {
		rollbackErr := errors.New("rollback failed")

		completion := &fbconn.BatchCompletion{
			TotalRecords: 3,
			UpdateCounts: []int64{1, 0, -1},
			Errors: []fbconn.BatchError{
				{Row: 2, Err: errors.New("duplicate key")},
			},
		}

		result, err := finalizePreparedBatchExec(completion, rollbackErr)

		require.EqualValues(t, 1, result.Affected)
		require.Equal(t, completion.UpdateCounts, result.UpdateCounts)
		require.ErrorContains(t, err, "batch failed at row 2")
		require.ErrorContains(t, err, "duplicate key")
		require.ErrorIs(t, err, rollbackErr)

		var batchErr *BatchExecutionError
		require.ErrorAs(t, err, &batchErr)
		require.Equal(t, []BatchError{{Row: 2, Err: completion.Errors[0].Err}}, batchErr.Errors)
	})

	t.Run("multiple batch row errors are exposed", func(t *testing.T) {
		completion := &fbconn.BatchCompletion{
			TotalRecords: 6,
			UpdateCounts: []int64{1, -1, 1, -1, 0, -1},
			Errors: []fbconn.BatchError{
				{Row: 1, Err: errors.New("first duplicate")},
				{Row: 3, Err: errors.New("second duplicate")},
				{Row: 5},
			},
		}

		result, err := finalizePreparedBatchExec(completion, nil)

		require.EqualValues(t, 2, result.Affected)
		require.Equal(t, completion.UpdateCounts, result.UpdateCounts)
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
		require.ErrorIs(t, batchErr.Errors[2].Err, ErrBatchRowFailed,
			"simplified errors must wrap ErrBatchRowFailed so errors.Is can see them")
	})
}
