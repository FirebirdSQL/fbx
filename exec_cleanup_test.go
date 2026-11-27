package fbx

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbconn/ctxwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cleanupDeadlineHandler struct {
	*fbconn.DeadlineContextWatcherHandler
	timeout time.Duration
}

func (h *cleanupDeadlineHandler) HandleWatch(ctx context.Context) {
	if deadline, ok := ctx.Deadline(); ok {
		h.timeout = time.Until(deadline)
	}
}

func (h *cleanupDeadlineHandler) HandleUnwatch() {}

func TestCleanupTimeoutForExecAndRows(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx := context.Background()
	config, err := ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.Autocommit = true
	config.DefaultQueryExecMode = QueryExecModeExec
	config.CleanupTimeout = time.Second
	var handler *cleanupDeadlineHandler
	config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
		handler = &cleanupDeadlineHandler{
			DeadlineContextWatcherHandler: &fbconn.DeadlineContextWatcherHandler{Conn: conn.Conn()},
		}
		return handler
	}
	conn, err := ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()

	for _, operation := range []string{"Exec", "Rows.Scan"} {
		for _, failed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/error=%t", operation, failed), func(t *testing.T) {
				handler.timeout = 0
				var operationErr error
				if operation == "Exec" {
					var value any = 1
					if failed {
						value = struct{}{}
					}
					_, operationErr = conn.Exec(ctx, "select cast(? as integer) from rdb$database", value)
				} else {
					var value int
					var dst any = &value
					if failed {
						dst = struct{}{}
					}
					operationErr = conn.QueryRow(ctx, "select 1 from rdb$database").Scan(dst)
				}
				if failed {
					require.Error(t, operationErr)
				} else {
					require.NoError(t, operationErr)
				}
				require.InDelta(t, float64(config.CleanupTimeout), float64(handler.timeout), float64(100*time.Millisecond))
			})
		}
	}
}

// Fail the next protocol operation after a successful implicit commit. The
// failure happens before cleanup can send anything, but the INSERT is committed.
type cleanupFailureHandler struct {
	*fbconn.DeadlineContextWatcherHandler
	conn           *fbconn.FbConn
	armed          bool
	sawTransaction bool
	fired          bool
}

func (h *cleanupFailureHandler) HandleBeforeRequest() error {
	if !h.armed || h.fired {
		return nil
	}
	if h.conn.ActiveTx() {
		h.sawTransaction = true
		return nil
	}
	if !h.sawTransaction {
		return nil
	}
	h.fired = true
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = h.conn.Close(ctx)
	// This returns the actual retryable conn-lock error without touching the
	// socket. It models cleanup failing before its own command is sent.
	return h.conn.Ping(context.Background())
}

func TestExecCleanupFailureDoesNotAuthorizeRetry(t *testing.T) {
	testCommittedOperationCleanup(t, false)
}

func TestQueryCleanupFailureDoesNotAuthorizeRetry(t *testing.T) {
	testCommittedOperationCleanup(t, true)
}

func testCommittedOperationCleanup(t *testing.T, query bool) {
	t.Helper()
	url := os.Getenv("FBX_TEST_DATABASE")
	if url == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config, err := ParseConfig(url)
	require.NoError(t, err)
	config.Autocommit = true
	config.DefaultQueryExecMode = QueryExecModeExec
	admin, err := ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { _ = admin.Close(context.Background()) }()
	table := fmt.Sprintf("fbx_rev_%d", time.Now().UnixNano()%1000000000000)
	_, err = admin.Exec(ctx, "create table "+table+" (id integer)")
	require.NoError(t, err)
	defer func() {
		_, dropErr := admin.Exec(ctx, "drop table "+table)
		assert.NoError(t, dropErr)
	}()
	var handler *cleanupFailureHandler
	config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
		handler = &cleanupFailureHandler{
			DeadlineContextWatcherHandler: &fbconn.DeadlineContextWatcherHandler{Conn: conn.Conn()},
			conn:                          conn,
		}
		return handler
	}
	conn, err := ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { _ = conn.Close(context.Background()) }()
	handler.armed = true
	if query {
		rows, queryErr := conn.Query(ctx, "insert into "+table+" (id) values (1) returning id")
		require.NoError(t, queryErr)
		require.True(t, rows.Next())
		var id int
		require.NoError(t, rows.Scan(&id))
		require.Equal(t, 1, id)
		rows.Close()
		err = rows.Err()
	} else {
		_, err = conn.Exec(ctx, "insert into "+table+" (id) values (1)")
	}
	require.Error(t, err)
	require.True(t, handler.fired, "the injected failure must occur after the transaction ended")
	assert.False(t, fbconn.SafeToRetry(err), "cleanup's retryable error must not authorize another INSERT")
	assert.True(t, conn.IsClosed())
	var count int
	require.NoError(t, admin.QueryRow(ctx, "select cast(count(*) as integer) from "+table).Scan(&count))
	assert.Equal(t, 1, count, "the INSERT was committed before cleanup failed")
	t.Logf("committed rows=%d; retry=%t; error=%v", count, fbconn.SafeToRetry(err), err)
}

func TestBeginFuncCanceledCommitDoesNotAuthorizeCallbackRetry(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config, err := ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.Autocommit = true
	config.DefaultQueryExecMode = QueryExecModeExec
	conn, err := ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { _ = conn.Close(context.Background()) }()
	table := fmt.Sprintf("fbx_rev_%d", time.Now().UnixNano()%1000000000000)
	_, err = conn.Exec(ctx, "create table "+table+" (id integer)")
	require.NoError(t, err)
	defer func() {
		_, dropErr := conn.Exec(ctx, "drop table "+table)
		assert.NoError(t, dropErr)
	}()
	callCtx, stop := context.WithCancel(ctx)
	defer stop()
	err = BeginFunc(callCtx, conn, func(tx Tx) error {
		_, execErr := tx.Exec(callCtx, "insert into "+table+" (id) values (1)")
		if execErr == nil {
			stop()
		}
		return execErr
	})
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, fbconn.SafeToRetry(err), "commit's pre-send error cannot authorize another callback")
	require.False(t, conn.IsClosed(), "rollback uses its own context after the call is canceled")
	require.False(t, conn.FbConn().ActiveTx())
	_, err = conn.Exec(ctx, "insert into "+table+" (id) values (2)")
	require.NoError(t, err)
	var previousRows, nextRows int
	require.NoError(t, conn.QueryRow(ctx, "select cast(count(*) as integer) from "+table+" where id=1").Scan(&previousRows))
	require.NoError(t, conn.QueryRow(ctx, "select cast(count(*) as integer) from "+table+" where id=2").Scan(&nextRows))
	assert.Zero(t, previousRows)
	assert.Equal(t, 1, nextRows)
}
