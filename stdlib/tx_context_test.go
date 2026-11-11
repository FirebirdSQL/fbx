package stdlib

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/stretchr/testify/require"
)

type rollbackContextTx struct {
	fbx.Tx
	rollback func(context.Context) error
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
