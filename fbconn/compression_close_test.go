package fbconn

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type blockingCompressionConn struct {
	compressed      []byte
	offset          int
	readBlocked     chan struct{}
	closed          chan struct{}
	allowReadReturn chan struct{}
	readBlockedOnce sync.Once
	closeOnce       sync.Once
	releaseOnce     sync.Once
}

func newBlockingCompressionConn(t *testing.T, payload []byte) *blockingCompressionConn {
	t.Helper()

	var wire bytes.Buffer
	compressor := zlib.NewWriter(&wire)
	_, err := compressor.Write(payload)
	require.NoError(t, err)
	require.NoError(t, compressor.Flush())
	compressed := append([]byte(nil), wire.Bytes()...)
	require.NoError(t, compressor.Close())

	return &blockingCompressionConn{
		compressed:      compressed,
		readBlocked:     make(chan struct{}),
		closed:          make(chan struct{}),
		allowReadReturn: make(chan struct{}),
	}
}

func (c *blockingCompressionConn) Read(buf []byte) (int, error) {
	if c.offset < len(c.compressed) {
		n := copy(buf, c.compressed[c.offset:])
		c.offset += n
		return n, nil
	}

	c.readBlockedOnce.Do(func() { close(c.readBlocked) })
	<-c.closed
	<-c.allowReadReturn
	return 0, net.ErrClosed
}

func (c *blockingCompressionConn) Write(buf []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
		return len(buf), nil
	}
}

func (c *blockingCompressionConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

func (c *blockingCompressionConn) LocalAddr() net.Addr {
	return compressionTestAddr("local")
}

func (c *blockingCompressionConn) RemoteAddr() net.Addr {
	return compressionTestAddr("remote")
}

func (c *blockingCompressionConn) SetDeadline(time.Time) error {
	return nil
}

func (c *blockingCompressionConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *blockingCompressionConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (c *blockingCompressionConn) releaseRead() {
	c.releaseOnce.Do(func() { close(c.allowReadReturn) })
}

type compressionTestAddr string

func (a compressionTestAddr) Network() string { return string(a) }
func (a compressionTestAddr) String() string  { return string(a) }

func TestAsyncCloseWaitsForCompressionRead(t *testing.T) {
	transport := newBlockingCompressionConn(t, bytes.Repeat([]byte("response "), 1024))
	defer transport.releaseRead()

	fbConn := newCancelOperationTestFbConn(transport)
	require.NoError(t, fbConn.client.EnableCompression())

	readDone := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(fbConn.client)
		readDone <- err
	}()

	select {
	case <-transport.readBlocked:
	case <-time.After(time.Second):
		t.Fatal("compressed read did not block on the transport")
	}

	fbConn.asyncClose()
	select {
	case <-transport.closed:
	case <-time.After(time.Second):
		t.Fatal("asyncClose did not close the transport")
	}

	select {
	case <-fbConn.CleanupDone():
		transport.releaseRead()
		t.Fatal("connection cleanup completed while the compressed read was active")
	case <-time.After(20 * time.Millisecond):
	}

	transport.releaseRead()
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("compressed read did not finish after the transport was released")
	}
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(time.Second):
		t.Fatal("connection cleanup did not finish after the compressed read")
	}
}

func TestCloseUnblocksCompressedReadBeforeCleanup(t *testing.T) {
	testSynchronousCloseUnblocksCompressedReadBeforeCleanup(t, "cannot confirm detach while connection is busy", func(fbConn *FbConn) error {
		return fbConn.Close(context.Background())
	})
}

func TestServiceConnCloseUnblocksCompressedReadBeforeCleanup(t *testing.T) {
	testSynchronousCloseUnblocksCompressedReadBeforeCleanup(t, "", func(fbConn *FbConn) error {
		return (&ServiceConn{conn: fbConn}).Close(context.Background())
	})
}

func testSynchronousCloseUnblocksCompressedReadBeforeCleanup(
	t *testing.T,
	expectedCloseError string,
	closeConnection func(*FbConn) error,
) {
	t.Helper()

	transport := newBlockingCompressionConn(t, bytes.Repeat([]byte("response "), 1024))
	defer transport.releaseRead()

	fbConn := newCancelOperationTestFbConn(transport)
	// Model an unfinished request: its read could consume the detach response.
	fbConn.status = connStatusBusy
	require.NoError(t, fbConn.client.EnableCompression())

	readDone := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(fbConn.client)
		readDone <- err
	}()

	select {
	case <-transport.readBlocked:
	case <-time.After(time.Second):
		t.Fatal("compressed read did not block on the transport")
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- closeConnection(fbConn)
	}()
	select {
	case <-transport.closed:
	case <-time.After(time.Second):
		transport.releaseRead()
		t.Fatal("Close did not close the transport while a compressed read was active")
	}

	select {
	case <-closeDone:
		transport.releaseRead()
		t.Fatal("Close returned while the compressed read was active")
	case <-time.After(20 * time.Millisecond):
	}

	transport.releaseRead()
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("compressed read did not finish after the transport was released")
	}
	select {
	case err := <-closeDone:
		if expectedCloseError == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, expectedCloseError)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not return after the compressed read finished")
	}
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(time.Second):
		t.Fatal("connection cleanup did not finish after Close returned")
	}
}
