package fbconn

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestConnectWithValidateConnectSessionAttributeErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"read-only connection", ErrReadOnlyConnection, "read only connection"},
		{"read-write connection", ErrReadWriteConnection, "connection is not read only"},
		{"primary", ErrPrimary, "not replica"},
		{"replica", ErrReplica, "not primary"},
		{"replication not configured", ErrReplicationNotConfigured, "replication is not configured"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			config, err := ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
			require.NoError(t, err)
			config.ValidateConnect = func(context.Context, *FbConn) error {
				return tt.err
			}

			conn, err := ConnectConfig(ctx, config)
			if !assert.NotNil(t, err) {
				_ = conn.Close(ctx)
			}
			require.ErrorIs(t, err, tt.err)
			var connectErr *ConnectError
			require.ErrorAs(t, err, &connectErr)
			require.ErrorContains(t, err, "ValidateConnect failed: "+tt.msg)
			require.EqualError(t, tt.err, tt.msg)
		})
	}

	t.Run("prefer-replica", func(t *testing.T) {
		err := &NotPreferredError{err: ErrPrimary}
		require.ErrorIs(t, err, ErrPrimary)
		require.NotErrorIs(t, err, ErrReplica)
		require.EqualError(t, err, "standby server not found: not replica")
	})
}
