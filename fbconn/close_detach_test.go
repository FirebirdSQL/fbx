package fbconn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbconn/internal/bgreader"
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

func TestAsyncCloseAfterReadDisconnectsAndDrains(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		t.Run(fmt.Sprintf("compressed=%t", compressed), func(t *testing.T) {
			clientConn, serverConn := net.Pipe()
			defer func() { _ = clientConn.Close() }()
			defer func() { _ = serverConn.Close() }()
			require.NoError(t, serverConn.SetDeadline(time.Now().Add(time.Second)))
			conn := newDetachTestConn(clientConn)
			peer := fbproto.NewClient(serverConn, serverConn, nil)
			if compressed {
				require.NoError(t, conn.client.EnableCompression())
				require.NoError(t, peer.EnableCompression())
			}

			ctx, cancel := context.WithCancel(context.Background())
			conn.contextWatcher.Watch(ctx)
			cancel()
			_, err := conn.receiveMessage()
			require.ErrorIs(t, err, os.ErrDeadlineExceeded)
			require.NoError(t, clientConn.SetReadDeadline(time.Time{}))
			conn.bgReader.Start()
			serverDone := make(chan error, 1)
			go func() {
				defer func() { _ = serverConn.Close() }()
				opcode, err := fbproto.NewXDRReader(peer).ReadInt32()
				if err == nil && opcode != fbproto.Op_disconnect {
					err = fmt.Errorf("expected op_disconnect, got %d", opcode)
				}
				if err == nil {
					_, err = peer.Write(bytes.Repeat([]byte("remaining response"), 4096))
				}
				if err == nil {
					err = peer.Flush()
				}
				serverDone <- err
			}()

			conn.asyncCloseAfterRead(err)
			require.True(t, conn.IsClosed())
			conn.contextWatcher.Unwatch()
			require.NoError(t, <-serverDone)
			waitCleanupDone(t, conn)
			require.EqualValues(t, bgreader.StatusStopped, conn.bgReader.Status())
			require.False(t, conn.WireCompressionEnabled())
		})
	}
}

func TestAsyncCloseAfterReadTCPDoesNotResetPeer(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		t.Run(fmt.Sprintf("compressed=%t", compressed), func(t *testing.T) {
			listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
			require.NoError(t, err)
			defer func() { _ = listener.Close() }()
			require.NoError(t, listener.SetDeadline(time.Now().Add(3*time.Second)))

			clientConn, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
			require.NoError(t, err)
			defer func() { _ = clientConn.Close() }()
			serverConn, err := listener.AcceptTCP()
			require.NoError(t, err)
			defer func() { _ = serverConn.Close() }()
			require.NoError(t, serverConn.SetDeadline(time.Now().Add(3*time.Second)))

			conn := newDetachTestConn(clientConn)
			peer := fbproto.NewClient(serverConn, serverConn, nil)
			if compressed {
				require.NoError(t, conn.client.EnableCompression())
				require.NoError(t, peer.EnableCompression())
			}

			ctx, cancel := context.WithCancel(context.Background())
			conn.contextWatcher.Watch(ctx)
			defer conn.contextWatcher.Unwatch()
			cancel()
			_, readErr := conn.receiveMessage()
			require.ErrorIs(t, readErr, os.ErrDeadlineExceeded)

			_, err = peer.Write(bytes.Repeat([]byte("remaining response"), 4096))
			require.NoError(t, err)
			require.NoError(t, peer.Flush())
			conn.asyncCloseAfterRead(readErr)
			require.True(t, conn.IsClosed())

			opcode, err := fbproto.NewXDRReader(peer).ReadInt32()
			require.NoError(t, err)
			require.Equal(t, fbproto.Op_disconnect, opcode)
			require.NoError(t, peer.CloseCompression())
			require.NoError(t, serverConn.CloseWrite())
			n, err := serverConn.Read(make([]byte, 1))
			require.Zero(t, n)
			require.ErrorIs(t, err, io.EOF)
			waitCleanupDone(t, conn)
		})
	}
}

func TestAsyncCloseDrainTimeout(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()
	conn := newDetachTestConn(clientConn)
	require.NoError(t, serverConn.SetDeadline(time.Now().Add(time.Second)))
	conn.asyncCloseWithTimeout(100 * time.Millisecond)
	opcode, err := fbproto.NewXDRReader(serverConn).ReadInt32()
	require.NoError(t, err)
	require.Equal(t, fbproto.Op_disconnect, opcode)
	waitCleanupDone(t, conn)
	_, err = serverConn.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
}

func TestAsyncCloseAfterReadDoesNotFlushFailedWrite(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()
	conn := newDetachTestConn(clientConn)
	err := conn.withWriteLockAndFlush(func() error {
		_, err := conn.client.Write([]byte{0, 0})
		require.NoError(t, err)
		return os.ErrDeadlineExceeded
	})
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	conn.asyncCloseAfterRead(os.ErrDeadlineExceeded)
	require.NoError(t, serverConn.SetReadDeadline(time.Now().Add(time.Second)))
	_, err = serverConn.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
	waitCleanupDone(t, conn)
}
