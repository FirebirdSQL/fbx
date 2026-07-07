package fbconn

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnLockErrorUnwrapsErrConnClosed(t *testing.T) {
	require.True(t, errors.Is(&connLockError{status: "conn closed"}, ErrConnClosed))
	require.False(t, errors.Is(&connLockError{status: "conn busy"}, ErrConnClosed))
}

func TestSyncConnClosedConnErrorIsErrConnClosed(t *testing.T) {
	fbConn := &FbConn{status: connStatusClosed}

	err := fbConn.SyncConn(context.Background())

	require.Error(t, err)
	require.ErrorIs(t, err, ErrConnClosed)
}
