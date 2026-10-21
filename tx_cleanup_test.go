package fbx

import (
	"context"
	"errors"
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rollbackFailureTx struct {
	Tx
	rollbackErr error
	contextErr  error
	bounded     bool
}

func (tx *rollbackFailureTx) Rollback(ctx context.Context) error {
	tx.contextErr = ctx.Err()
	_, tx.bounded = ctx.Deadline()
	return tx.rollbackErr
}

func TestBeginFuncPreservesCallbackErrorWhenRollbackFails(t *testing.T) {
	callbackErr := errors.New("callback failed after SQL")
	rollbackErr := errors.New("rollback failed")
	tx := &rollbackFailureTx{rollbackErr: rollbackErr}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := beginFuncExec(ctx, tx, func(Tx) error { return callbackErr })
	require.ErrorIs(t, err, callbackErr)
	require.ErrorIs(t, err, rollbackErr)
	assert.False(t, fbconn.SafeToRetry(err))
	assert.NoError(t, tx.contextErr, "rollback must not inherit the caller's cancellation")
	assert.True(t, tx.bounded, "rollback must have a bounded cleanup deadline")
}
