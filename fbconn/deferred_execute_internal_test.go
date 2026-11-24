package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"testing/synctest"
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

const deferredExecuteTestCancelDelay = 50 * time.Millisecond

func newDeferredExecutePipe(t *testing.T) (*FbConn, *Statement, net.Conn, *deferredExecuteWriteConn, <-chan struct{}) {
	t.Helper()
	client, server := net.Pipe()
	wire := &deferredExecuteWriteConn{Conn: client, writeStarted: make(chan struct{})}
	conn := newCancelOperationTestFbConn(wire)
	conn.config = &Config{Autocommit: true}
	handler := &deferredExecuteCancelHandler{
		CancelRequestContextWatcherHandler: &CancelRequestContextWatcherHandler{
			Conn:          conn,
			DeadlineDelay: deferredExecuteTestCancelDelay,
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
	for _, blockedAt := range []string{"flush", "response"} {
		t.Run(blockedAt, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
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
				timer := time.AfterFunc(2*cleanupGracePeriod, func() { _ = server.Close() })
				defer timer.Stop()

				start := time.Now()
				result := rr.readWithoutCommit()
				require.ErrorIs(t, result.Err, context.Canceled)
				require.Equal(t, deferredExecuteTestCancelDelay, time.Since(start))
				require.True(t, conn.IsClosed(), "a missing Execute response prevents reuse")
				require.NoError(t, server.Close())
				waitCleanupDone(t, conn)
			})
		})
	}
}

type arrayPrepareCleanupConn struct {
	net.Conn
	dropped          []int
	rollbackSent     bool
	deadlineResetErr error
	readDeadlineOnly bool
	stallExecute     bool
}

func (c *arrayPrepareCleanupConn) SetDeadline(deadline time.Time) error {
	if deadline.IsZero() && c.deadlineResetErr != nil {
		return c.deadlineResetErr
	}
	if c.readDeadlineOnly {
		return c.SetReadDeadline(deadline)
	}
	return c.Conn.SetDeadline(deadline)
}

func (c *arrayPrepareCleanupConn) Write(p []byte) (int, error) {
	if len(p) == 4 && binary.BigEndian.Uint32(p) == fbproto.Op_ping {
		return c.Conn.Write(p)
	}
	if len(p) == 8 && binary.BigEndian.Uint32(p) == fbproto.Op_rollback {
		c.rollbackSent = true
		return c.Conn.Write(p)
	}
	if c.stallExecute && len(p) >= 4 && binary.BigEndian.Uint32(p) == fbproto.Op_execute {
		return c.Conn.Write(p)
	}
	if len(p) == 12 && binary.BigEndian.Uint32(p) == fbproto.Op_free_statement && binary.BigEndian.Uint32(p[8:]) == fbproto.DSQL_drop {
		handle := int(binary.BigEndian.Uint32(p[4:]))
		c.dropped = append(c.dropped, handle)
		return c.Conn.Write(p)
	}
	return len(p), nil
}

type cancelBeforeArrayMetadataFetch struct {
	context.Context
	conn   *FbConn
	cancel context.CancelFunc
}

func (c cancelBeforeArrayMetadataFetch) Err() error {
	if c.conn.deferredExecutePending.Load() {
		c.cancel()
		synctest.Wait()
	}
	return c.Context.Err()
}

type delayedPrepareCancelHandler struct {
	*CancelRequestContextWatcherHandler
	resume <-chan struct{}
}

func (h *delayedPrepareCancelHandler) HandleCancel(ctx context.Context) {
	<-h.resume
	h.CancelRequestContextWatcherHandler.HandleCancel(ctx)
}

type prepareCancelWithoutDeadline struct{}

func (prepareCancelWithoutDeadline) HandleCancel(context.Context) {}
func (prepareCancelWithoutDeadline) HandleUnwatchAfterCancel()    {}

func TestPrepareBatchCleansUpNewTransactionAfterLockFailure(t *testing.T) {
	conn := newDetachTestConn(&prepareCleanupTestConn{})
	t.Cleanup(func() {
		conn.asyncClose()
		waitCleanupDone(t, conn)
	})
	conn.config.Autocommit = true
	tx := &FBTransaction{Connection: conn, Handle: 1}
	conn.internalTransaction = tx
	// An event operation acquired the connection between Begin and Prepare.
	require.NoError(t, conn.lock())
	defer conn.unlock()

	st, err := conn.prepareTx(context.Background(), tx, "select 1", statementScopeConnection, prepareRollbackAlways)
	require.Nil(t, st)
	require.ErrorContains(t, err, "conn busy")
	require.True(t, SafeToRetry(err))
	require.True(t, conn.IsClosed(), "a failed rollback must discard the temporary transaction")
}

func TestPrepareArrayMetadataCleanup(t *testing.T) {
	const statementHandle, metadataHandle = 42, 43
	const slowDropDelay = cleanupGracePeriod + time.Second
	const cancelDeadlineDelay = 2 * time.Second
	const requestTimeout = 2 * cleanupGracePeriod
	bothDrops := []int{metadataHandle, statementHandle}
	resetErr := errors.New("reset deadline failed")
	for _, tc := range []struct {
		name             string
		dropsRead        int
		readDelay        time.Duration
		lateCancel       bool
		resetErr         error
		readDeadlineOnly bool
		stallAt          string
		wantContextErr   error
		wantDropped      []int
		wantElapsed      time.Duration
	}{
		{
			name: "successful cleanup after cancellation", dropsRead: 2,
			wantContextErr: context.Canceled, wantDropped: bothDrops,
		},
		{
			name: "slow drop outlasts the read deadline", dropsRead: 2, readDeadlineOnly: true,
			readDelay: cancelDeadlineDelay + time.Second, wantElapsed: cancelDeadlineDelay + time.Second,
			wantContextErr: context.Canceled, wantDropped: bothDrops,
		},
		{
			name: "late cancel after dropping both statements", dropsRead: 2, lateCancel: true,
			wantContextErr: context.Canceled, wantDropped: bothDrops,
		},
		{
			name:           "metadata statement blocked",
			wantContextErr: context.Canceled, wantDropped: []int{metadataHandle}, wantElapsed: cancelDeadlineDelay,
		},
		{
			name: "original statement blocked after metadata drop", dropsRead: 1, readDelay: time.Second,
			wantContextErr: context.Canceled, wantDropped: bothDrops, wantElapsed: cancelDeadlineDelay,
		},
		{
			name:           "caller deadline during metadata drop",
			wantContextErr: context.DeadlineExceeded, wantDropped: []int{metadataHandle}, wantElapsed: requestTimeout + cancelDeadlineDelay,
		},
		{
			name: "deadline reset fails after both drops", dropsRead: 2, resetErr: resetErr,
			wantContextErr: context.Canceled, wantDropped: bothDrops,
		},
		{
			name: "slow drop within caller deadline", dropsRead: 2, readDelay: slowDropDelay,
			wantDropped: bothDrops, wantElapsed: slowDropDelay,
		},
		{
			name: "cleanup grace bounds deferred execute write", stallAt: "execute write",
			wantContextErr: context.Canceled, wantElapsed: cleanupGracePeriod,
		},
		{
			name: "cleanup grace bounds deferred execute response", stallAt: "execute response",
			wantContextErr: context.Canceled, wantElapsed: cleanupGracePeriod,
		},
		{
			name: "both drops share the cleanup grace", stallAt: "drop", dropsRead: 1,
			readDelay:      cleanupGracePeriod - time.Second,
			wantContextErr: context.Canceled, wantDropped: bothDrops, wantElapsed: cleanupGracePeriod,
		},
		{
			name: "owned rollback shares the cleanup grace", stallAt: "rollback", dropsRead: 2,
			readDelay:      cleanupGracePeriod - time.Second,
			wantContextErr: context.Canceled, wantDropped: bothDrops, wantElapsed: cleanupGracePeriod,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var responses bytes.Buffer
				w := fbproto.NewXDRWriter(&responses)
				arrayInfo := tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)
				arrayInfo = append(arrayInfo, fbproto.Isc_info_sql_select)
				arrayInfo = append(arrayInfo, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 1)...)
				arrayInfo = append(arrayInfo, tagLengthValue(fbproto.Isc_info_sql_sqlda_seq, 1)...)
				arrayInfo = append(arrayInfo, tagLengthValue(fbproto.Isc_info_sql_type, fbproto.SQL_TYPE_ARRAY)...)
				arrayInfo = append(arrayInfo, tagBufferValue(fbproto.Isc_info_sql_field, []byte("VAL"))...)
				arrayInfo = append(arrayInfo, tagBufferValue(fbproto.Isc_info_sql_relation, []byte("T"))...)
				arrayInfo = append(arrayInfo, fbproto.Isc_info_sql_describe_end, fbproto.Isc_info_end)
				selectInfo := append(tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select), fbproto.Isc_info_end)
				require.NoError(t, writeProbeResponse(w, statementHandle))
				require.NoError(t, writeProbeResponse(w, 0))
				require.NoError(t, writeProbeResponseData(w, 0, arrayInfo))
				require.NoError(t, writeProbeResponse(w, metadataHandle))
				require.NoError(t, writeProbeResponse(w, fbproto.StmtDeferExecute))
				require.NoError(t, writeProbeResponseData(w, 0, selectInfo))
				if tc.stallAt != "execute write" && tc.stallAt != "execute response" {
					require.NoError(t, writeProbeResponse(w, 0)) // Deferred Execute completed.
				}
				if tc.stallAt == "rollback" {
					for range tc.dropsRead {
						require.NoError(t, writeProbeResponse(w, 0))
					}
				}
				cancelBeforeFetch := tc.wantContextErr == context.Canceled
				if !cancelBeforeFetch {
					require.NoError(t, w.WriteInt32(fbproto.Op_fetch_response))
					require.NoError(t, w.WriteInt32(100)) // End of an empty cursor.
					require.NoError(t, w.WriteInt32(0))
					require.NoError(t, writeProbeResponseData(w, 0, []byte{fbproto.Isc_info_end})) // Records.
				}

				client, server := net.Pipe()
				wire := &arrayPrepareCleanupConn{
					Conn: client, deadlineResetErr: tc.resetErr, stallExecute: tc.stallAt == "execute write",
					readDeadlineOnly: tc.readDeadlineOnly,
				}
				conn := newDetachTestConn(wire)
				conn.config.Autocommit = tc.stallAt == "rollback"
				peerDone := make(chan struct{})
				go func() {
					defer close(peerDone)
					time.Sleep(tc.readDelay)
					_, _ = io.CopyN(io.Discard, server, int64(tc.dropsRead)*12)
					if tc.stallAt == "rollback" {
						_, _ = io.CopyN(io.Discard, server, 8)
					}
				}()
				defer func() {
					require.NoError(t, server.Close())
					<-peerDone
					conn.asyncClose()
					waitCleanupDone(t, conn)
				}()
				var cancelKinds []int
				conn.contextHandler = &DeadlineContextWatcherHandler{Conn: wire, DeadlineDelay: cancelDeadlineDelay}
				switch {
				case tc.stallAt != "":
					conn.contextHandler = prepareCancelWithoutDeadline{}
				case tc.lateCancel:
					conn.contextHandler = &delayedPrepareCancelHandler{
						CancelRequestContextWatcherHandler: &CancelRequestContextWatcherHandler{
							Conn: conn, DeadlineDelay: cancelDeadlineDelay,
							sendCancelOperationFunc: func(_ context.Context, kind int) error {
								cancelKinds = append(cancelKinds, kind)
								return nil
							},
						},
						resume: peerDone,
					}
				}
				conn.contextWatcher = ctxwatch.NewContextWatcher(conn.contextHandler)
				encoding, err := fbproto.NewEncoding("UTF8")
				require.NoError(t, err)
				conn.Encoding = encoding
				conn.client = fbproto.NewClient(io.MultiReader(&responses, conn.bgReader), conn.writer, encoding)
				conn.protocolVersion = fbproto.PROTOCOL_VERSION13
				conn.activeStatements = make(map[*Statement]struct{})
				conn.internalTransaction = &FBTransaction{Connection: conn, Handle: 1}
				ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
				defer cancel()
				if cancelBeforeFetch {
					ctx = cancelBeforeArrayMetadataFetch{ctx, conn, cancel}
				}
				// Release stalled I/O so the test can report a regression.
				timer := time.AfterFunc(3*cleanupGracePeriod, func() { _ = server.Close() })
				defer timer.Stop()

				start := time.Now()
				st, err := conn.Prepare(ctx, "select VAL from T")
				require.Equal(t, tc.wantElapsed, time.Since(start), "Prepare cleanup duration")
				require.Nil(t, st)
				require.NotContains(t, cancelKinds, fbproto.Fb_cancel_raise)
				if tc.wantContextErr != nil {
					require.ErrorIs(t, err, tc.wantContextErr)
				} else {
					require.ErrorContains(t, err, "cannot resolve array metadata")
				}
				require.False(t, SafeToRetry(err))
				require.False(t, conn.IsBusy())
				require.False(t, conn.deferredExecutePending.Load())
				require.Empty(t, conn.activeStatements)
				require.Equal(t, tc.wantDropped, wire.dropped)
				if tc.stallAt == "rollback" {
					require.True(t, wire.rollbackSent)
					require.ErrorContains(t, err, "rollback internal tx")
				}
				require.Zero(t, responses.Len())
				switch {
				case tc.stallAt != "":
					require.True(t, conn.IsClosed())
					require.ErrorIs(t, err, os.ErrDeadlineExceeded)
				case tc.resetErr != nil:
					require.True(t, conn.IsClosed())
					require.ErrorIs(t, err, tc.resetErr)
				case tc.dropsRead < len(tc.wantDropped):
					var timeoutErr net.Error
					require.ErrorAs(t, err, &timeoutErr)
					require.True(t, timeoutErr.Timeout())
					require.True(t, conn.IsClosed())
					require.ErrorIs(t, err, os.ErrDeadlineExceeded)
					if tc.wantContextErr == context.DeadlineExceeded {
						require.True(t, Timeout(err))
					}
				default:
					cancel()
					time.Sleep(cleanupGracePeriod)
					require.False(t, conn.IsClosed())
					go func() {
						if _, err := io.CopyN(io.Discard, server, 4); err != nil {
							return
						}
						w := fbproto.NewXDRWriter(server)
						for range tc.dropsRead + 1 {
							if err := writeProbeResponse(w, 0); err != nil {
								return
							}
						}
					}()
					require.NoError(t, conn.Ping(context.Background()))
				}
			})
		})
	}
}
