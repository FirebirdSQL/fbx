package fbconn

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
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

func TestCancelRequestContextWatcherHandlerHandleWatchResetsPendingCancelState(t *testing.T) {
	h, _ := newWatcherHandlerForWindowTests()
	h.pendingCancelReset.Store(true)

	var calls []int
	h.sendCancelOperationFunc = func(_ context.Context, kind int) error {
		calls = append(calls, kind)
		return nil
	}

	h.HandleWatch(context.Background())

	require.Equal(t, []int{fbproto.Fb_cancel_disable, fbproto.Fb_cancel_enable}, calls)
	require.False(t, h.pendingCancelReset.Load())
	require.True(t, h.requestCancelEnabled.Load())
}
