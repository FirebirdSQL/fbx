package fbconn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestIsConnectionFatal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		err   *FbError
		fatal bool
	}{
		{name: "nil", err: nil, fatal: false},
		{name: "missing SQLSTATE", err: &FbError{}, fatal: false},
		{name: "undefined table", err: &FbError{Code: "42S02"}, fatal: false},
		{name: "division by zero", err: &FbError{Code: "22012"}, fatal: false},
		{name: "deadlock", err: &FbError{Code: "40001"}, fatal: false},
		{name: "cancelled", err: &FbError{Code: "HY008"}, fatal: false},
		{name: "connection failure", err: &FbError{Code: "08006"}, fatal: true},
		{name: "attachment gone", err: &FbError{Code: "08003"}, fatal: true},
		{name: "memory allocation", err: &FbError{Code: "HY001"}, fatal: true},
		{name: "timeout", err: &FbError{Code: "HYT00"}, fatal: true},
		{name: "malformed SQLSTATE", err: &FbError{Code: "0800"}, fatal: true},
		{
			name:  "shutdown GDS fallback",
			err:   &FbError{Code: "HY000", GdsCodes: []int{fbproto.Isc_shutdown}},
			fatal: true,
		},
		{
			name: "fatal GDS code is not first",
			err: &FbError{
				Code:     "HY000",
				GdsCodes: []int{fbproto.Isc_arith_except, fbproto.Isc_att_shutdown},
			},
			fatal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.fatal, IsConnectionFatal(tt.err))
		})
	}
}

func TestHandleFbErrorPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		err              *FbError
		withHandler      bool
		handlerResult    bool
		wantHandlerCall  bool
		wantKeepOpen     bool
		wantConnectionUp bool
	}{
		{
			name:             "ordinary error without handler stays open",
			err:              &FbError{Code: "42S02"},
			wantKeepOpen:     true,
			wantConnectionUp: true,
		},
		{
			name:             "handler keeps ordinary error open",
			err:              &FbError{Code: "22012"},
			withHandler:      true,
			handlerResult:    true,
			wantHandlerCall:  true,
			wantKeepOpen:     true,
			wantConnectionUp: true,
		},
		{
			name:            "handler closes on ordinary error",
			err:             &FbError{Code: "42S02"},
			withHandler:     true,
			handlerResult:   false,
			wantHandlerCall: true,
		},
		{
			name:             "handler keeps fatal error open",
			err:              &FbError{Code: "08006"},
			withHandler:      true,
			handlerResult:    true,
			wantHandlerCall:  true,
			wantKeepOpen:     true,
			wantConnectionUp: true,
		},
		{
			name:             "fatal error without handler stays open",
			err:              &FbError{Code: "HY000", GdsCodes: []int{fbproto.Isc_att_shutdown}},
			wantKeepOpen:     true,
			wantConnectionUp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rawConn := &noopConn{}
			config := &Config{}
			handlerCalled := false
			if tt.withHandler {
				config.OnFbError = func(conn *FbConn, err *FbError) bool {
					handlerCalled = true
					require.NotNil(t, conn)
					require.Same(t, tt.err, err)
					return tt.handlerResult
				}
			}
			conn := &FbConn{
				conn:        rawConn,
				config:      config,
				status:      connStatusIdle,
				cleanupDone: make(chan struct{}),
			}

			keepOpen := conn.handleFbError(tt.err)

			require.Equal(t, tt.wantKeepOpen, keepOpen)
			require.Equal(t, tt.wantHandlerCall, handlerCalled)
			if tt.wantConnectionUp {
				require.False(t, conn.IsClosed())
				require.False(t, rawConn.closed.Load())
				return
			}

			waitCleanupDone(t, conn)
			require.True(t, conn.IsClosed())
			require.True(t, rawConn.closed.Load())
		})
	}
}

func TestDefaultOnFbError(t *testing.T) {
	t.Parallel()

	require.True(t, defaultOnFbError(nil, &FbError{Code: "42S02"}))
	require.False(t, defaultOnFbError(nil, &FbError{Code: "08006"}))
}

func TestStatementExecTxCloseCursorErrorKeepsConnectionOpen(t *testing.T) {
	t.Parallel()

	rawConn := &noopConn{}
	conn := &FbConn{
		conn:        rawConn,
		status:      connStatusIdle,
		cleanupDone: make(chan struct{}),
	}
	statement := &Statement{
		conn:              conn,
		cursorTransaction: &FBTransaction{Connection: conn},
		Description: &StatementDescription{
			StatementInfo: &StatementInfo{},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := statement.ExecTx(ctx, &FBTransaction{Connection: conn}, nil)

	require.Error(t, result.err)
	require.False(t, conn.IsClosed())
	require.False(t, rawConn.closed.Load())
}

func TestResultReaderNextRowAfterErrorDoesNotUseConnection(t *testing.T) {
	t.Parallel()

	result := &ResultReader{err: &FbError{Code: "HY008"}}

	require.False(t, result.NextRow())
}
