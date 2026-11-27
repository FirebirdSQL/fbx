package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn/ctxwatch"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

type deferredExecuteScriptConn struct {
	noopConn
	responses *bytes.Reader
	writes    int
}

func (c *deferredExecuteScriptConn) Read(p []byte) (int, error) {
	return c.responses.Read(p)
}

func (c *deferredExecuteScriptConn) Write(p []byte) (int, error) {
	c.writes++
	return len(p), nil
}

func TestDeferredExecuteFollowsPrepareResponse(t *testing.T) {
	for _, tc := range []struct {
		name          string
		prepareFlags  int
		statementType int
		wantDeferred  bool
	}{
		{"select_with_flag", fbproto.StmtDeferExecute, fbproto.Isc_info_sql_stmt_select, true},
		{"select_without_flag", 0, fbproto.Isc_info_sql_stmt_select, false},
		{"select_for_update", fbproto.StmtDeferExecute, fbproto.Isc_info_sql_stmt_select_for_upd, true},
		{"update_without_flag", 0, fbproto.Isc_info_sql_stmt_update, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var responses bytes.Buffer
			x := fbproto.NewXDRWriter(&responses)
			writeResponse := func(object int, data []byte) {
				require.NoError(t, x.WriteInt32(fbproto.Op_response))
				require.NoError(t, x.WriteInt32(object))
				require.NoError(t, x.WriteInt64(0))
				require.NoError(t, x.WriteBuffer(data))
				require.NoError(t, x.WriteInt32(fbproto.Isc_arg_end))
			}
			writeResponse(1, nil) // Allocate.
			writeResponse(tc.prepareFlags, nil)
			info := tagLengthValue(fbproto.Isc_info_sql_stmt_type, tc.statementType)
			writeResponse(0, append(info, fbproto.Isc_info_end))
			writeResponse(0, nil) // Execute.
			hasCursor := tc.statementType != fbproto.Isc_info_sql_stmt_update
			if hasCursor {
				require.NoError(t, x.WriteInt32(fbproto.Op_fetch_response))
				require.NoError(t, x.WriteInt32(100)) // End of an empty cursor.
				require.NoError(t, x.WriteInt32(0))
			}
			writeResponse(0, []byte{fbproto.Isc_info_end}) // Records on Close.

			wire := &deferredExecuteScriptConn{responses: bytes.NewReader(responses.Bytes())}
			conn := newCancelOperationTestFbConn(wire)
			encoding, err := fbproto.NewEncoding("UTF8")
			require.NoError(t, err)
			conn.Encoding = encoding
			conn.protocolVersion = fbproto.PROTOCOL_VERSION13
			conn.activeStatements = make(map[*Statement]struct{})
			conn.contextWatcher = ctxwatch.NewContextWatcher(&DeadlineContextWatcherHandler{Conn: wire})
			t.Cleanup(func() {
				conn.asyncClose()
				waitCleanupDone(t, conn)
			})
			ctx := context.Background()
			tx := &FBTransaction{Connection: conn, Handle: 1}
			st, err := conn.PrepareTx(ctx, tx, "test statement")
			require.NoError(t, err)

			wire.writes = 0
			rr := st.Exec(ctx, nil)
			wantWrites := 1
			if tc.wantDeferred {
				wantWrites = 0
			}
			require.Equal(t, wantWrites, wire.writes, "Prepare must determine whether Execute is buffered")
			require.False(t, rr.NextRow())
			if hasCursor {
				wantWrites++
			}
			require.Equal(t, wantWrites, wire.writes, "deferred Execute must share the first Fetch write")
			_, err = rr.Close()
			require.NoError(t, err)
			require.Zero(t, wire.responses.Len(), "all responses must be consumed")
			require.False(t, conn.IsClosed())
		})
	}
}

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

func TestDeferredExecuteCallerErrorDisallowsRetry(t *testing.T) {
	conn, st, server, _, _ := newDeferredExecutePipe(t)
	rr := st.Exec(context.Background(), nil)
	callerErr := &fbconnError{msg: "caller error", safeToRetry: true}
	rr.SetCallerErr(callerErr)

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- respondToDeferredExecuteAndRollback(server)
	}()
	_, err := rr.Close()
	require.ErrorIs(t, err, callerErr)
	require.False(t, SafeToRetry(err), "a buffered Execute may already have reached the server")
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

func TestDeferredExecuteInternalReadUsesRequestDeadline(t *testing.T) {
	const cleanupTimeout = 250 * time.Millisecond
	for _, blockedAt := range []string{"flush", "response"} {
		t.Run(blockedAt, func(t *testing.T) {
			conn, st, server, _, canceled := newDeferredExecutePipe(t)
			conn.config.CleanupTimeout = cleanupTimeout
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
				require.Less(t, time.Since(start), cleanupTimeout,
					"internal reads must retain the request cancellation deadline")
				require.True(t, conn.IsClosed(), "a missing Execute response prevents reuse")
				require.NoError(t, server.Close())
				waitCleanupDone(t, conn)
			case <-time.After(cleanupTimeout + 2*time.Second):
				t.Fatal("internal read cleanup exceeded its timeout")
			}
		})
	}
}

func TestDeferredExecuteCloseCleanupTimeout(t *testing.T) {
	const cleanupTimeout = 150 * time.Millisecond
	for _, blockedAt := range []string{"flush", "response"} {
		t.Run(blockedAt, func(t *testing.T) {
			conn, st, server, _, _ := newDeferredExecutePipe(t)
			conn.config.CleanupTimeout = cleanupTimeout
			require.NoError(t, conn.lock())
			defer conn.unlock()
			ctx := context.Background()
			rr := st.internalExecTx(ctx, st.transaction, &ResultReader{fbConn: conn, ctx: ctx}, nil)
			if blockedAt == "response" {
				go func() { _, _ = io.Copy(io.Discard, server) }()
			}
			start := time.Now()
			done := make(chan error, 1)
			go func() {
				_, err := rr.closeWithoutCommit(true)
				done <- err
			}()
			select {
			case err := <-done:
				require.Error(t, err)
				require.True(t, Timeout(err))
				require.GreaterOrEqual(t, time.Since(start), cleanupTimeout)
				require.True(t, conn.IsClosed())
			case <-time.After(cleanupTimeout + 2*time.Second):
				_ = server.Close()
				<-done
				t.Fatal("Close with Background did not enforce CleanupTimeout")
			}
			require.NoError(t, server.Close())
			waitCleanupDone(t, conn)
		})
	}
}

func TestInternalCleanupKeepsRequestWatcher(t *testing.T) {
	var responses bytes.Buffer
	require.NoError(t, writeProbeResponse(fbproto.NewXDRWriter(&responses), 0))
	wire := &deferredExecuteScriptConn{responses: bytes.NewReader(responses.Bytes())}
	conn := newDetachTestConn(wire)
	handler := &deferredExecuteCancelHandler{
		CancelRequestContextWatcherHandler: &CancelRequestContextWatcherHandler{
			Conn:                    conn,
			sendCancelOperationFunc: func(context.Context, int) error { return nil },
		},
		canceled: make(chan struct{}),
	}
	conn.contextWatcher = ctxwatch.NewContextWatcher(handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn.contextWatcher.Watch(ctx)
	defer func() {
		conn.contextWatcher.Unwatch()
		conn.asyncClose()
		waitCleanupDone(t, conn)
	}()
	require.NoError(t, conn.lock())
	defer conn.unlock()
	conn.deferredExecutePending.Store(true)
	rr := &ResultReader{fbConn: conn, ctx: ctx, deferredExecute: true, rowsInfoPopulated: true}
	_, err := rr.closeWithoutCommit(false)
	require.NoError(t, err)
	require.False(t, conn.IsClosed())
	cancel()
	select {
	case <-handler.canceled:
	case <-time.After(time.Second):
		t.Fatal("internal cleanup stopped the outer request watcher")
	}
}
