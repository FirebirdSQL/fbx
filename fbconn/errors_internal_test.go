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

func TestTargetSessionAttributeErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"read-only connection", ErrReadOnlyConnection, "read only connection"},
		{"read-write connection", ErrReadWriteConnection, "connection is not read only"},
		{"primary connection", ErrPrimaryConnection, "not replica"},
		{"replica connection", ErrReplicaConnection, "not primary"},
		{"replication not configured", ErrReplicationNotConfigured, "replication is not configured"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.EqualError(t, tt.err, tt.msg)
		})
	}
}

func TestNotPreferredError(t *testing.T) {
	err := &NotPreferredError{err: ErrPrimaryConnection}
	require.ErrorIs(t, err, ErrPrimaryConnection)
	require.NotErrorIs(t, err, ErrReplicaConnection)
	require.EqualError(t, err, "standby server not found: not replica")
}
