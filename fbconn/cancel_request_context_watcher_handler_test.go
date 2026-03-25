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
