package fbconn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

type cancelDeadlineErrorConn struct {
	noopConn
	cancel      context.CancelFunc
	deadlineSet chan struct{}
	setErr      error
	resetErr    error
	writeErr    error
}

func (c *cancelDeadlineErrorConn) Write(b []byte) (int, error) {
	c.cancel()
	<-c.deadlineSet
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	return len(b), nil
}

func (c *cancelDeadlineErrorConn) SetDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		return c.resetErr
	}
	close(c.deadlineSet)
	return c.setErr
}

func TestSendCancelOperationDeadlineErrors(t *testing.T) {
	setErr := errors.New("set deadline failed")
	resetErr := errors.New("reset deadline failed")
	writeErr := errors.New("write failed")

	for _, tt := range []struct {
		name     string
		setErr   error
		resetErr error
		writeErr error
	}{
		{name: "success"},
		{name: "set deadline", setErr: setErr},
		{name: "reset deadline", resetErr: resetErr},
		{name: "set and reset deadline", setErr: setErr, resetErr: resetErr},
		{name: "write", writeErr: writeErr},
		{name: "write and reset deadline", writeErr: writeErr, resetErr: resetErr},
		{name: "write and both deadlines", writeErr: writeErr, setErr: setErr, resetErr: resetErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conn := &cancelDeadlineErrorConn{
				cancel:      cancel,
				deadlineSet: make(chan struct{}),
				setErr:      tt.setErr,
				resetErr:    tt.resetErr,
				writeErr:    tt.writeErr,
			}
			fbConn := newCancelOperationTestFbConn(conn)
			t.Cleanup(func() {
				fbConn.asyncClose()
				waitCleanupDone(t, fbConn)
			})

			err := fbConn.sendCancelOperation(ctx, fbproto.Fb_cancel_raise)

			if tt.setErr == nil && tt.resetErr == nil && tt.writeErr == nil {
				require.NoError(t, err)
			}
			for _, wantErr := range []error{tt.setErr, tt.resetErr, tt.writeErr} {
				if wantErr != nil {
					require.ErrorIs(t, err, wantErr)
				}
			}
			wantClosed := tt.setErr != nil || tt.resetErr != nil
			require.Equal(t, wantClosed, fbConn.IsClosed())
			require.Equal(t, wantClosed, conn.closed.Load())
			if wantClosed {
				require.Error(t, fbConn.lock())
				waitCleanupDone(t, fbConn)
			}
		})
	}
}

func BenchmarkSendCancelOperation(b *testing.B) {
	for _, cancellable := range []bool{false, true} {
		name := "Background"
		if cancellable {
			name = "Cancellable"
		}
		b.Run(name, func(b *testing.B) {
			ctx := context.Background()
			if cancellable {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
			}
			fbConn := newCancelOperationTestFbConn(&noopConn{})
			b.Cleanup(func() {
				fbConn.asyncClose()
				<-fbConn.CleanupDone()
			})
			b.ReportAllocs()
			for b.Loop() {
				if err := fbConn.sendCancelOperation(ctx, fbproto.Fb_cancel_raise); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
