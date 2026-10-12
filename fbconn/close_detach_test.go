package fbconn

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbproto"
)

func newDetachTestConn(conn net.Conn) *FbConn {
	fbConn := newCancelOperationTestFbConn(conn)
	fbConn.config = &Config{}
	fbConn.contextWatcher = ctxwatch.NewContextWatcher(&DeadlineContextWatcherHandler{Conn: conn})
	return fbConn
}

func TestCloseWaitsForDetachResponseBeforeDisconnect(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	fbConn := newDetachTestConn(clientConn)
	serverReady := make(chan struct{})
	respond := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		r := fbproto.NewXDRReader(serverConn)
		w := fbproto.NewXDRWriter(serverConn)
		opcode, err := r.ReadInt32()
		if err != nil {
			serverDone <- err
			return
		}
		if opcode != fbproto.Op_detach {
			serverDone <- errors.New("expected op_detach")
			return
		}
		if _, err := r.ReadInt32(); err != nil {
			serverDone <- err
			return
		}
		close(serverReady)
		<-respond
		if err := writeProbeResponse(w, 0); err != nil {
			serverDone <- err
			return
		}
		opcode, err = r.ReadInt32()
		if err == nil && opcode != fbproto.Op_disconnect {
			err = errors.New("expected op_disconnect after detach response")
		}
		serverDone <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	closeDone := make(chan error, 1)
	go func() { closeDone <- fbConn.Close(ctx) }()
	select {
	case <-serverReady:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case err := <-closeDone:
		t.Fatalf("Close returned before detach response: %v", err)
	default:
	}
	close(respond)
	require.NoError(t, <-serverDone)
	require.NoError(t, <-closeDone)
	require.True(t, fbConn.IsClosed())
}

func TestCloseRollsBackOpenTransactionBeforeDetach(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	fbConn := newDetachTestConn(clientConn)
	tx := &FBTransaction{Connection: fbConn, Handle: 42}
	fbConn.activeTransactions = map[*FBTransaction]struct{}{tx: {}}
	serverDone := make(chan error, 1)
	go func() {
		r := fbproto.NewXDRReader(serverConn)
		w := fbproto.NewXDRWriter(serverConn)
		for _, want := range []struct {
			opcode int
			handle int
		}{
			{fbproto.Op_rollback, 42},
			{fbproto.Op_detach, 0},
		} {
			opcode, err := r.ReadInt32()
			if err != nil {
				serverDone <- err
				return
			}
			handle, err := r.ReadInt32()
			if err != nil {
				serverDone <- err
				return
			}
			if opcode != want.opcode || handle != want.handle {
				serverDone <- errors.New("unexpected close request")
				return
			}
			if err := writeProbeResponse(w, 0); err != nil {
				serverDone <- err
				return
			}
		}
		opcode, err := r.ReadInt32()
		if err == nil && opcode != fbproto.Op_disconnect {
			err = errors.New("expected op_disconnect")
		}
		serverDone <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, fbConn.Close(ctx))
	require.NoError(t, <-serverDone)
	require.True(t, tx.Closed)
	require.Empty(t, fbConn.activeTransactions)
}

func TestCloseReturnsDetachFailureAndClosesTransport(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	fbConn := newDetachTestConn(clientConn)
	serverDone := make(chan error, 1)
	go func() {
		r := fbproto.NewXDRReader(serverConn)
		w := fbproto.NewXDRWriter(serverConn)
		opcode, err := r.ReadInt32()
		if err != nil {
			serverDone <- err
			return
		}
		if opcode != fbproto.Op_detach {
			serverDone <- errors.New("expected op_detach")
			return
		}
		if _, err := r.ReadInt32(); err != nil {
			serverDone <- err
			return
		}
		for _, n := range []int{fbproto.Op_response, 0} {
			if err := w.WriteInt32(n); err != nil {
				serverDone <- err
				return
			}
		}
		if err := w.WriteInt64(0); err != nil {
			serverDone <- err
			return
		}
		if err := w.WriteBuffer(nil); err != nil {
			serverDone <- err
			return
		}
		for _, n := range []int{fbproto.Isc_arg_gds, fbproto.Isc_bad_db_handle, fbproto.Isc_arg_end} {
			if err := w.WriteInt32(n); err != nil {
				serverDone <- err
				return
			}
		}
		_, err = r.ReadInt32()
		if !errors.Is(err, io.EOF) {
			serverDone <- errors.New("expected transport close without op_disconnect")
			return
		}
		serverDone <- nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := fbConn.Close(ctx)
	var fbErr *FbError
	require.ErrorAs(t, err, &fbErr)
	require.Contains(t, fbErr.Message, "invalid database handle")
	require.NoError(t, <-serverDone)
	require.True(t, fbConn.IsClosed())
}

func TestCloseStopsWaitingForDetachWhenContextExpires(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	fbConn := newDetachTestConn(clientConn)
	serverDone := make(chan error, 1)
	go func() {
		r := fbproto.NewXDRReader(serverConn)
		opcode, err := r.ReadInt32()
		if err == nil && opcode != fbproto.Op_detach {
			err = errors.New("expected op_detach")
		}
		if err == nil {
			_, err = r.ReadInt32()
		}
		if err == nil {
			_, err = r.ReadInt32()
			if errors.Is(err, io.EOF) {
				err = nil
			}
		}
		serverDone <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, fbConn.Close(ctx), context.DeadlineExceeded)
	require.NoError(t, <-serverDone)
	require.True(t, fbConn.IsClosed())
}
