package stdlib

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbconn/ctxwatch"
	"github.com/stretchr/testify/require"
)

type rollbackContextTx struct {
	fbx.Tx
	conn     *fbx.Conn
	rollback func(context.Context) error
}

func (tx rollbackContextTx) Conn() *fbx.Conn {
	return tx.conn
}

func (tx rollbackContextTx) Rollback(ctx context.Context) error {
	return tx.rollback(ctx)
}

func TestWrapTxRollbackAfterContextCancellation(t *testing.T) {
	type contextKey struct{}
	for _, expired := range []bool{false, true} {
		name := "canceled"
		if expired {
			name = "deadline_exceeded"
		}
		t.Run(name, func(t *testing.T) {
			parent := context.WithValue(context.Background(), contextKey{}, "transaction")
			var ctx context.Context
			var cancel context.CancelFunc
			if expired {
				ctx, cancel = context.WithDeadline(parent, time.Now().Add(-time.Second))
			} else {
				ctx, cancel = context.WithCancel(parent)
			}
			cancel()
			wantErr := errors.New("rollback response")
			var rollbackCtx context.Context
			tx := wrapTx{ctx: ctx, tx: rollbackContextTx{rollback: func(ctx context.Context) error {
				rollbackCtx = ctx
				require.NoError(t, ctx.Err(), "database/sql rolls back after the begin context is canceled")
				require.Equal(t, "transaction", ctx.Value(contextKey{}))
				deadline, ok := ctx.Deadline()
				require.True(t, ok, "rollback must have a bounded cleanup deadline")
				require.Positive(t, time.Until(deadline))
				require.LessOrEqual(t, time.Until(deadline), 5*time.Second)
				return wantErr
			}}}
			require.ErrorIs(t, tx.Rollback(), wantErr)
			require.ErrorIs(t, rollbackCtx.Err(), context.Canceled)
		})
	}
}

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

func TestCleanupTimeoutScope(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.CleanupTimeout = time.Second
	var handler *cleanupDeadlineHandler
	config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
		handler = &cleanupDeadlineHandler{
			DeadlineContextWatcherHandler: &fbconn.DeadlineContextWatcherHandler{Conn: conn.Conn()},
		}
		return handler
	}
	conn, err := fbx.ConnectConfig(context.Background(), config)
	require.NoError(t, err)
	defer func() { _ = conn.Close(context.Background()) }()

	checkTimeout := func(ctx context.Context, want time.Duration) error {
		t.Helper()
		require.NoError(t, ctx.Err())
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.InDelta(t, float64(want), float64(time.Until(deadline)), float64(100*time.Millisecond))
		return nil
	}
	for _, canceled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if canceled {
			cancel()
		}
		tx := wrapTx{ctx: ctx, tx: rollbackContextTx{conn: conn, rollback: func(ctx context.Context) error {
			return checkTimeout(ctx, config.CleanupTimeout)
		}}}
		err := tx.Rollback()
		cancel()
		require.NoError(t, err)
	}
	c := &Conn{
		conn: conn, close: conn.Close,
		psRefCounts: make(map[*fbconn.StatementDescription]int),
	}
	stmt, err := c.PrepareContext(context.Background(), "select 1 from rdb$database")
	require.NoError(t, err)
	handler.timeout = 0
	require.NoError(t, stmt.Close())
	require.InDelta(t, float64(config.CleanupTimeout), float64(handler.timeout), float64(100*time.Millisecond))
	handler.timeout = 0
	require.NoError(t, c.Close())
	require.InDelta(t, float64(config.CleanupTimeout), float64(handler.timeout), float64(100*time.Millisecond))
}
