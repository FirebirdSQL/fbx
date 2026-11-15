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
	cancel         context.CancelFunc
	deadlineSet    chan struct{}
	setErr         error
	resetErr       error
	writeErr       error
	deadlineResets int
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
		c.deadlineResets++
		return c.resetErr
	}
	defer close(c.deadlineSet)
	return c.setErr
}

func TestSendCancelOperationDeadlineErrors(t *testing.T) {
	setErr := errors.New("set deadline failed")
	resetErr := errors.New("reset deadline failed")
	writeErr := errors.New("cancel write failed")
	for _, tt := range []struct {
		name     string
		setErr   error
		resetErr error
		writeErr error
	}{
		{name: "success"},
		{name: "set_error", setErr: setErr},
		{name: "reset_error", resetErr: resetErr},
		{name: "set_and_reset_errors", setErr: setErr, resetErr: resetErr},
		{name: "write_error", writeErr: writeErr},
		{name: "write_and_set_errors", writeErr: writeErr, setErr: setErr},
		{name: "write_and_reset_errors", writeErr: writeErr, resetErr: resetErr},
		{name: "all_errors", writeErr: writeErr, setErr: setErr, resetErr: resetErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

			for _, wantErr := range []error{tt.setErr, tt.resetErr, tt.writeErr} {
				if wantErr != nil {
					require.ErrorIs(t, err, wantErr)
				}
			}
			if tt.setErr == nil && tt.resetErr == nil && tt.writeErr == nil {
				require.NoError(t, err)
			}
			require.Equal(t, 1, conn.deadlineResets)
			if tt.setErr != nil || tt.resetErr != nil {
				require.True(t, conn.closed.Load())
				require.True(t, fbConn.IsClosed())
				require.ErrorIs(t, fbConn.lock(), ErrConnClosed)
				waitCleanupDone(t, fbConn)
			} else {
				require.False(t, fbConn.IsClosed())
				require.False(t, conn.closed.Load())
			}
		})
	}
}
