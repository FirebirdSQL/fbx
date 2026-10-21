package fbconn

import (
	"bufio"
	"context"
	"errors"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn/internal/bgreader"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

type noopConn struct {
	closed atomic.Bool
}

func (c *noopConn) Read(_ []byte) (int, error)         { return 0, io.EOF }
func (c *noopConn) Write(b []byte) (int, error)        { return len(b), nil }
func (c *noopConn) Close() error                       { c.closed.Store(true); return nil }
func (c *noopConn) LocalAddr() net.Addr                { return noopAddr("local") }
func (c *noopConn) RemoteAddr() net.Addr               { return noopAddr("remote") }
func (c *noopConn) SetDeadline(_ time.Time) error      { return nil }
func (c *noopConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *noopConn) SetWriteDeadline(_ time.Time) error { return nil }

type noopAddr string

func (a noopAddr) Network() string { return string(a) }
func (a noopAddr) String() string  { return string(a) }

func newWatcherHandlerForWindowTests() (*CancelRequestContextWatcherHandler, *FbConn) {
	done := make(chan struct{})
	close(done)

	fbConn := &FbConn{
		conn:        &noopConn{},
		status:      connStatusIdle,
		cleanupDone: make(chan struct{}),
	}

	h := &CancelRequestContextWatcherHandler{
		Conn:                           fbConn,
		CancelResetTimeout:             10 * time.Millisecond,
		cancelFinishedChan:             done,
		handleUnwatchAfterCancelCalled: func() {},
	}

	return h, fbConn
}

func waitCleanupDone(t *testing.T, fbConn *FbConn) {
	t.Helper()
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for connection cleanup")
	}
}

func TestCancelRequestContextWatcherHandlerUnwatchDisableTimeoutClosesConn(t *testing.T) {
	h, fbConn := newWatcherHandlerForWindowTests()
	h.requestCancelEnabled.Store(true)

	var calls atomic.Int32
	h.sendCancelOperationFunc = func(ctx context.Context, _ int) error {
		calls.Add(1)
		<-ctx.Done()
		return ctx.Err()
	}

	h.HandleUnwatch()

	require.True(t, fbConn.IsClosed())
	waitCleanupDone(t, fbConn)
	require.EqualValues(t, 1, calls.Load())
	require.Error(t, fbConn.lock())
}

func TestCancelRequestContextWatcherHandlerUnwatchDisableErrorClosesConnAndPreventsReuse(t *testing.T) {
	h, fbConn := newWatcherHandlerForWindowTests()
	disableErr := errors.New("disable failed")
	h.requestCancelEnabled.Store(true)

	var calls atomic.Int32
	h.sendCancelOperationFunc = func(_ context.Context, kind int) error {
		calls.Add(1)
		switch kind {
		case fbproto.Fb_cancel_disable:
			return disableErr
		default:
			return nil
		}
	}

	h.HandleUnwatch()

	require.True(t, fbConn.IsClosed())
	waitCleanupDone(t, fbConn)
	require.EqualValues(t, 1, calls.Load())
	require.Error(t, fbConn.lock())
}

func TestCancelRequestContextWatcherHandlerWatchEnableErrorClosesConnAndPreventsReuse(t *testing.T) {
	h, fbConn := newWatcherHandlerForWindowTests()
	enableErr := errors.New("enable failed")

	var calls atomic.Int32
	h.sendCancelOperationFunc = func(_ context.Context, kind int) error {
		calls.Add(1)
		switch kind {
		case fbproto.Fb_cancel_enable:
			return enableErr
		default:
			return nil
		}
	}

	h.HandleWatch(context.Background())

	require.True(t, fbConn.IsClosed())
	waitCleanupDone(t, fbConn)
	require.EqualValues(t, 1, calls.Load())
	require.Error(t, fbConn.lock())
}

func TestCancelRequestContextWatcherHandlerHandleCancelTimeoutDoesNotCloseConn(t *testing.T) {
	h, fbConn := newWatcherHandlerForWindowTests()
	h.CancelRequestDelay = 0
	h.DeadlineDelay = 50 * time.Millisecond

	var calls atomic.Int32
	h.sendCancelOperationFunc = func(_ context.Context, kind int) error {
		require.Equal(t, fbproto.Fb_cancel_raise, kind)
		calls.Add(1)
		return context.DeadlineExceeded
	}

	h.HandleCancel(context.Background())
	h.HandleUnwatchAfterCancel()

	require.EqualValues(t, 1, calls.Load())
	require.False(t, fbConn.IsClosed())
}

func TestCancelRequestContextWatcherHandlerHandleCancelZeroDelayStillSendsCancelWithFastUnwatch(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		h, _ := newWatcherHandlerForWindowTests()
		h.CancelRequestDelay = 0
		h.DeadlineDelay = 50 * time.Millisecond

		var calls atomic.Int32
		h.sendCancelOperationFunc = func(_ context.Context, kind int) error {
			require.Equal(t, fbproto.Fb_cancel_raise, kind)
			calls.Add(1)
			return nil
		}

		h.HandleCancel(context.Background())
		h.HandleUnwatchAfterCancel()

		require.EqualValuesf(t, 1, calls.Load(), "iteration=%d", i)
	}
}

func TestCancelRequestContextWatcherHandlerHandleCancelRespectsGraceWindow(t *testing.T) {
	h, _ := newWatcherHandlerForWindowTests()
	h.CancelRequestDelay = 20 * time.Millisecond
	h.DeadlineDelay = 200 * time.Millisecond

	var calls atomic.Int32
	h.sendCancelOperationFunc = func(_ context.Context, kind int) error {
		require.Equal(t, fbproto.Fb_cancel_raise, kind)
		calls.Add(1)
		return nil
	}

	h.HandleCancel(context.Background())
	h.HandleUnwatchAfterCancel()

	require.EqualValues(t, 0, calls.Load())
}

func TestCancelRequestContextWatcherHandlerDelayedCancelNotCanceledByUnwatchAfterSendStarts(t *testing.T) {
	h, fbConn := newWatcherHandlerForWindowTests()
	h.CancelRequestDelay = 5 * time.Millisecond
	h.DeadlineDelay = 5 * time.Second

	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	h.sendCancelOperationFunc = func(ctx context.Context, kind int) error {
		require.Equal(t, fbproto.Fb_cancel_raise, kind)
		calls.Add(1)
		close(started)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return nil
		}
	}

	h.HandleCancel(context.Background())
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cancel request to start")
	}

	unwatchDone := make(chan struct{})
	go func() {
		h.HandleUnwatchAfterCancel()
		close(unwatchDone)
	}()

	select {
	case <-unwatchDone:
		t.Fatal("unwatch canceled an already-started cancel request")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)
	select {
	case <-unwatchDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for unwatch after cancel")
	}

	require.EqualValues(t, 1, calls.Load())
	require.False(t, fbConn.IsClosed())
}

func TestCancelRequestContextWatcherHandlerHandleCancelZeroDelayUsesFallbackDeadline(t *testing.T) {
	h, _ := newWatcherHandlerForWindowTests()
	h.CancelRequestDelay = 0
	h.DeadlineDelay = 0

	h.sendCancelOperationFunc = func(ctx context.Context, kind int) error {
		require.Equal(t, fbproto.Fb_cancel_raise, kind)

		deadline, ok := ctx.Deadline()
		require.True(t, ok)

		remaining := time.Until(deadline)
		require.Greater(t, remaining, 10*time.Millisecond)
		require.Less(t, remaining, 500*time.Millisecond)
		return nil
	}

	h.HandleCancel(context.Background())
	h.HandleUnwatchAfterCancel()
}

func TestCancelRequestContextWatcherHandlerHandleCancelZeroDelayWaitsForWriteCompletionAndStillSends(t *testing.T) {
	h, _ := newWatcherHandlerForWindowTests()
	h.CancelRequestDelay = 0
	h.DeadlineDelay = 200 * time.Millisecond
	h.Conn.writeInProgress.Store(true)

	var calls atomic.Int32
	h.sendCancelOperationFunc = func(ctx context.Context, kind int) error {
		require.Equal(t, fbproto.Fb_cancel_raise, kind)
		calls.Add(1)
		return nil
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		h.Conn.writeInProgress.Store(false)
	}()

	start := time.Now()
	h.HandleCancel(context.Background())
	h.HandleUnwatchAfterCancel()

	require.EqualValues(t, 1, calls.Load())
	require.GreaterOrEqual(t, time.Since(start), 10*time.Millisecond)
	require.Less(t, time.Since(start), 180*time.Millisecond)
}

func TestCancelRequestContextWatcherHandlerHandleUnwatchAfterCancelDefersCancelStateResetWhenConnOpen(t *testing.T) {
	h, _ := newWatcherHandlerForWindowTests()
	h.requestCancelEnabled.Store(true)

	h.HandleUnwatchAfterCancel()

	require.True(t, h.pendingCancelReset.Load())
}

func TestCancelRequestContextWatcherHandlerHandleUnwatchAfterCancelSkipsResetWhenConnClosed(t *testing.T) {
	h, fbConn := newWatcherHandlerForWindowTests()
	h.requestCancelEnabled.Store(true)
	fbConn.asyncClose()
	waitCleanupDone(t, fbConn)

	var calls atomic.Int32
	h.sendCancelOperationFunc = func(_ context.Context, _ int) error {
		calls.Add(1)
		return nil
	}

	h.HandleUnwatchAfterCancel()

	require.EqualValues(t, 0, calls.Load())
	require.False(t, h.pendingCancelReset.Load())
}

func TestCancelRequestContextWatcherHandlerHandleBeforeRequestResetsPendingCancelState(t *testing.T) {
	h, _ := newWatcherHandlerForWindowTests()
	h.pendingCancelReset.Store(true)

	var calls []int
	h.sendCancelOperationFunc = func(_ context.Context, kind int) error {
		calls = append(calls, kind)
		return nil
	}

	err := h.HandleBeforeRequest()

	require.NoError(t, err)
	require.Equal(t, []int{fbproto.Fb_cancel_disable, fbproto.Fb_cancel_enable}, calls)
	require.False(t, h.pendingCancelReset.Load())
}

type blockFirstWriteConn struct {
	noopConn
	writeStarted chan struct{}
	releaseWrite chan struct{}
	blockOnce    sync.Once
}

func (c *blockFirstWriteConn) Write(b []byte) (int, error) {
	c.blockOnce.Do(func() {
		close(c.writeStarted)
		<-c.releaseWrite
	})
	return len(b), nil
}

type writeErrorConn struct {
	noopConn
	err error
}

func (c *writeErrorConn) Write(_ []byte) (int, error) {
	return 0, c.err
}

func newCancelOperationTestFbConn(conn net.Conn) *FbConn {
	fbConn := &FbConn{
		conn:        conn,
		writer:      bufio.NewWriter(conn),
		status:      connStatusIdle,
		cleanupDone: make(chan struct{}),
	}
	fbConn.bgReader = bgreader.New(conn)
	fbConn.bgReaderStarted = make(chan struct{})
	fbConn.slowWriteTimer = time.AfterFunc(time.Duration(math.MaxInt64), func() {
		fbConn.bgReader.Start()
		fbConn.bgReaderStarted <- struct{}{}
	})
	fbConn.slowWriteTimer.Stop()
	fbConn.client = fbproto.NewClient(fbConn.bgReader, fbConn.writer, nil)
	return fbConn
}

func TestSendCancelOperationWriteErrorDoesNotCloseConn(t *testing.T) {
	writeErr := errors.New("write failed")
	fbConn := newCancelOperationTestFbConn(&writeErrorConn{err: writeErr})

	err := fbConn.sendCancelOperation(context.Background(), fbproto.Fb_cancel_raise)

	require.ErrorIs(t, err, writeErr)
	require.False(t, fbConn.IsClosed())
}

func TestSendCancelOperationConcurrentFlushDoesNotPanic(t *testing.T) {
	conn := &blockFirstWriteConn{
		writeStarted: make(chan struct{}),
		releaseWrite: make(chan struct{}),
	}

	fbConn := newCancelOperationTestFbConn(conn)

	type callResult struct {
		err error
		pan any
	}
	runCancel := func() <-chan callResult {
		done := make(chan callResult, 1)
		go func() {
			res := callResult{}
			defer func() {
				if r := recover(); r != nil {
					res.pan = r
				}
				done <- res
			}()
			res.err = fbConn.sendCancelOperation(context.Background(), fbproto.Fb_cancel_raise)
		}()
		return done
	}

	first := runCancel()
	select {
	case <-conn.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first flush write to start")
	}

	second := runCancel()
	close(conn.releaseWrite)

	var r1, r2 callResult
	select {
	case r1 = <-first:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first cancel request")
	}
	select {
	case r2 = <-second:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second cancel request")
	}

	require.Nil(t, r1.pan)
	require.Nil(t, r2.pan)
	require.NoError(t, r1.err)
	require.NoError(t, r2.err)
}
