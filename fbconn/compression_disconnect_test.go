package fbconn

import (
	"bufio"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestCloseCompressedDisconnect(t *testing.T) {
	for _, buffered := range []bool{false, true} {
		for _, tc := range []struct {
			name                  string
			closeBeforeDisconnect bool
			waitForTransportClose bool
		}{
			{name: "peer_closes_after_disconnect"},
			{name: "peer_waits_for_transport_close", waitForTransportClose: true},
			{name: "disconnect_write_error", closeBeforeDisconnect: true},
		} {
			t.Run(fmt.Sprintf("%s/buffered=%t", tc.name, buffered), func(t *testing.T) {
				clientConn, serverConn := net.Pipe()
				t.Cleanup(func() { _ = clientConn.Close() })
				t.Cleanup(func() { _ = serverConn.Close() })
				require.NoError(t, serverConn.SetDeadline(time.Now().Add(2*time.Second)))

				transport := &closeTrackingConn{Conn: clientConn}
				fbConn := newDetachTestConn(transport)
				if !buffered {
					fbConn.client.Rebind(fbConn.bgReader, transport)
				}
				require.NoError(t, fbConn.client.EnableCompression())

				serverDone := make(chan error, 1)
				go func() {
					defer func() { _ = serverConn.Close() }()
					serverDone <- runCompressedDisconnectServer(serverConn, tc.closeBeforeDisconnect, tc.waitForTransportClose)
				}()

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				err := fbConn.Close(ctx)
				if tc.closeBeforeDisconnect {
					require.ErrorContains(t, err, "send disconnect")
					require.ErrorIs(t, err, io.ErrClosedPipe)
				} else {
					require.NoError(t, err)
				}
				require.NoError(t, <-serverDone)
				require.True(t, transport.closed)
				require.True(t, fbConn.IsClosed())
				require.False(t, fbConn.WireCompressionEnabled())
				select {
				case <-fbConn.CleanupDone():
				default:
					t.Fatal("Close returned before compression cleanup completed")
				}
				require.NoError(t, fbConn.Close(context.Background()))
			})
		}
	}
}

func runCompressedDisconnectServer(conn net.Conn, closeBeforeDisconnect, waitForTransportClose bool) error {
	compressedReader, err := zlib.NewReader(bufio.NewReader(conn))
	if err != nil {
		return err
	}
	defer func() { _ = compressedReader.Close() }()
	compressedWriter := zlib.NewWriter(conn)
	// The server closes the transport after disconnect, without a zlib trailer.
	r := fbproto.NewXDRReader(compressedReader)
	w := fbproto.NewXDRWriter(compressedWriter)
	opcode, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if opcode != fbproto.Op_detach {
		return fmt.Errorf("expected op_detach, got %d", opcode)
	}
	if _, err := r.ReadInt32(); err != nil {
		return err
	}
	if err := writeProbeResponse(w, 0); err != nil {
		return err
	}
	if err := compressedWriter.Flush(); err != nil {
		return err
	}
	if closeBeforeDisconnect {
		return nil
	}
	opcode, err = r.ReadInt32()
	if err != nil {
		return err
	}
	if opcode != fbproto.Op_disconnect {
		return fmt.Errorf("expected op_disconnect, got %d", opcode)
	}
	if waitForTransportClose {
		var tail [1]byte
		n, err := conn.Read(tail[:])
		if n != 0 || !errors.Is(err, io.EOF) {
			return fmt.Errorf("expected transport EOF after disconnect, got %d bytes and %v", n, err)
		}
	}
	return nil
}
