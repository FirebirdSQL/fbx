package fbconn

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbproto"
)

type deferredExecuteWriteConn struct {
	net.Conn
	writeStarted chan struct{}
	once         sync.Once
}

func (c *deferredExecuteWriteConn) Write(p []byte) (int, error) {
	c.once.Do(func() { close(c.writeStarted) })
	return c.Conn.Write(p)
}

type deferredExecuteCancelHandler struct {
	*CancelRequestContextWatcherHandler
	canceled chan struct{}
}

func (h *deferredExecuteCancelHandler) HandleCancel(ctx context.Context) {
	h.CancelRequestContextWatcherHandler.HandleCancel(ctx)
	close(h.canceled)
}

func newDeferredExecutePipe(t *testing.T) (*FbConn, *Statement, net.Conn, *deferredExecuteWriteConn, <-chan struct{}) {
	t.Helper()
	client, server := net.Pipe()
	wire := &deferredExecuteWriteConn{Conn: client, writeStarted: make(chan struct{})}
	conn := newCancelOperationTestFbConn(wire)
	conn.config = &Config{Autocommit: true}
	handler := &deferredExecuteCancelHandler{
		CancelRequestContextWatcherHandler: &CancelRequestContextWatcherHandler{
			Conn:          conn,
			DeadlineDelay: 50 * time.Millisecond,
			// Keep cancel state operations out of the scripted query exchange.
			sendCancelOperationFunc: func(context.Context, int) error { return nil },
		},
		canceled: make(chan struct{}),
	}
	conn.contextHandler = handler
	conn.contextWatcher = ctxwatch.NewContextWatcher(handler)
	tx := &FBTransaction{Connection: conn, Handle: 1}
	conn.internalTransaction = tx
	conn.activeTransactions = map[*FBTransaction]struct{}{tx: {}}
	st := &Statement{
		conn:         conn,
		transaction:  tx,
		deferExecute: true,
		Description: &StatementDescription{
			StatementHandle: 1,
			StatementInfo: &StatementInfo{
				StatementType: fbproto.Isc_info_sql_stmt_select,
			},
		},
	}
	t.Cleanup(func() {
		require.NoError(t, server.Close())
		conn.contextWatcher.Unwatch()
		conn.asyncClose()
		waitCleanupDone(t, conn)
	})
	return conn, st, server, wire, handler.canceled
}

func TestDeferredExecuteCancelInterruptsLargeWrite(t *testing.T) {
	conn, st, _, wire, _ := newDeferredExecutePipe(t)
	const parameterSize = 8192
	st.Description.StatementInfo.ParametersDescriptors = []fbproto.FieldDescriptor{
		{Type: fbproto.SQL_TYPE_TEXT, Length: parameterSize},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan *ResultReader, 1)
	go func() {
		done <- st.ExecTx(ctx, st.transaction, [][]byte{make([]byte, parameterSize)})
	}()
	select {
	case <-wire.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("Execute did not overflow the writer buffer")
	}
	cancel()
	select {
	case rr := <-done:
		_, err := rr.Close()
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, SafeToRetry(err))
		require.True(t, conn.IsClosed(), "a partially written Execute cannot be reused")
		waitCleanupDone(t, conn)
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt the blocked Execute write")
	}
}

func TestDeferredExecuteCanceledFetchRollsBack(t *testing.T) {
	conn, st, server, _, canceled := newDeferredExecutePipe(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rr := st.Exec(ctx, nil)
	cancel()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("context cancellation was not observed")
	}
	require.False(t, rr.NextRow())

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- respondToDeferredExecuteAndRollback(server)
	}()
	_, err := rr.Close()
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, SafeToRetry(err))
	require.False(t, conn.ActiveTx(), "Close must roll back the implicit transaction")
	require.False(t, conn.IsClosed())
	require.False(t, conn.IsBusy())
	select {
	case err := <-serverDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("server did not receive the rollback")
	}
}

func respondToDeferredExecuteAndRollback(server net.Conn) error {
	response := make([]byte, 24)
	binary.BigEndian.PutUint32(response, fbproto.Op_response)
	for _, request := range []struct {
		op   uint32
		size int
	}{
		{fbproto.Op_execute, 24},
		{fbproto.Op_rollback, 8},
	} {
		packet := make([]byte, request.size)
		if _, err := io.ReadFull(server, packet); err != nil {
			return err
		}
		if op := binary.BigEndian.Uint32(packet); op != request.op {
			return fmt.Errorf("received operation %d, expected %d", op, request.op)
		}
		if _, err := server.Write(response); err != nil {
			return err
		}
	}
	return nil
}

func TestDeferredExecuteInternalReadCleanupTimeout(t *testing.T) {
	for _, blockedAt := range []string{"flush", "response"} {
		t.Run(blockedAt, func(t *testing.T) {
			conn, st, server, _, canceled := newDeferredExecutePipe(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			require.NoError(t, conn.lock())
			defer conn.unlock()
			conn.contextWatcher.Watch(ctx)
			rr := st.internalExecTx(ctx, st.transaction, &ResultReader{fbConn: conn, ctx: ctx}, nil)
			cancel()
			select {
			case <-canceled:
			case <-time.After(time.Second):
				t.Fatal("context cancellation was not observed")
			}
			if blockedAt == "response" {
				go func() { _, _ = io.Copy(io.Discard, server) }()
			}
			start := time.Now()
			done := make(chan *Result, 1)
			go func() { done <- rr.readWithoutCommit() }()
			select {
			case result := <-done:
				require.ErrorIs(t, result.Err, context.Canceled)
				require.GreaterOrEqual(t, time.Since(start), cleanupGracePeriod,
					"cleanup must replace the original short cancellation deadline")
				require.True(t, conn.IsClosed(), "a missing Execute response prevents reuse")
				waitCleanupDone(t, conn)
			case <-time.After(cleanupGracePeriod + 2*time.Second):
				t.Fatal("internal read cleanup exceeded its timeout")
			}
		})
	}
}
