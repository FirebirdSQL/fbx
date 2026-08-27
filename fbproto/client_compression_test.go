package fbproto

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type failOnRead struct {
	t *testing.T
}

func (r failOnRead) Read([]byte) (int, error) {
	r.t.Fatal("compression initialization must not read from the transport")
	return 0, io.ErrUnexpectedEOF
}

type blockingCompressionReader struct {
	readStarted      chan struct{}
	releaseRead      chan struct{}
	closeCalled      chan struct{}
	readStartedOnce  sync.Once
	releaseReadOnce  sync.Once
	closeCalledOnce  sync.Once
	readActive       atomic.Bool
	closedDuringRead atomic.Bool
}

type blockingFirstRead struct {
	reader      io.Reader
	readStarted chan struct{}
	releaseRead chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
}

func newBlockingFirstRead(reader io.Reader) *blockingFirstRead {
	return &blockingFirstRead{
		reader:      reader,
		readStarted: make(chan struct{}),
		releaseRead: make(chan struct{}),
	}
}

func (r *blockingFirstRead) Read(buf []byte) (int, error) {
	r.startOnce.Do(func() {
		close(r.readStarted)
		<-r.releaseRead
	})
	return r.reader.Read(buf)
}

func (r *blockingFirstRead) release() {
	r.releaseOnce.Do(func() { close(r.releaseRead) })
}

type armableBlockingWriter struct {
	writer       io.Writer
	armed        atomic.Bool
	writeStarted chan struct{}
	releaseWrite chan struct{}
	startOnce    sync.Once
	releaseOnce  sync.Once
}

func newArmableBlockingWriter(writer io.Writer) *armableBlockingWriter {
	return &armableBlockingWriter{
		writer:       writer,
		writeStarted: make(chan struct{}),
		releaseWrite: make(chan struct{}),
	}
}

func (w *armableBlockingWriter) Write(buf []byte) (int, error) {
	if w.armed.Load() {
		w.startOnce.Do(func() { close(w.writeStarted) })
		<-w.releaseWrite
	}
	return w.writer.Write(buf)
}

func (w *armableBlockingWriter) arm() {
	w.armed.Store(true)
}

func (w *armableBlockingWriter) release() {
	w.releaseOnce.Do(func() { close(w.releaseWrite) })
}

func newBlockingCompressionReader() *blockingCompressionReader {
	return &blockingCompressionReader{
		readStarted: make(chan struct{}),
		releaseRead: make(chan struct{}),
		closeCalled: make(chan struct{}),
	}
}

func (r *blockingCompressionReader) Read(buf []byte) (int, error) {
	r.readActive.Store(true)
	r.readStartedOnce.Do(func() { close(r.readStarted) })
	<-r.releaseRead
	r.readActive.Store(false)
	buf[0] = 1
	return 1, nil
}

func (r *blockingCompressionReader) Close() error {
	if r.readActive.Load() {
		r.closedDuringRead.Store(true)
	}
	r.closeCalledOnce.Do(func() { close(r.closeCalled) })
	return nil
}

func (r *blockingCompressionReader) release() {
	r.releaseReadOnce.Do(func() { close(r.releaseRead) })
}

type xorStreamCodec struct {
	key []byte
	pos int
}

func (c *xorStreamCodec) Encrypt(dst, src []byte) {
	c.apply(dst, src)
}

func (c *xorStreamCodec) Decrypt(dst, src []byte) {
	c.apply(dst, src)
}

func (c *xorStreamCodec) apply(dst, src []byte) {
	for i := range src {
		dst[i] = src[i] ^ c.key[(c.pos+i)%len(c.key)]
	}
	c.pos += len(src)
}

func TestEnableCompressionDoesNotReadTransport(t *testing.T) {
	client := NewClient(failOnRead{t: t}, io.Discard, nil)

	require.NoError(t, client.EnableCompression())
	require.True(t, client.CompressionEnabled())
}

func TestCompressionRoundTripAcrossFlushes(t *testing.T) {
	var wire bytes.Buffer
	writer := NewClient(nil, &wire, nil)
	require.NoError(t, writer.EnableCompression())

	first := bytes.Repeat([]byte("first payload "), 128)
	second := bytes.Repeat([]byte("second payload "), 256)
	_, err := writer.Write(first)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())
	_, err = writer.Write(second)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())

	reader := NewClient(bytes.NewReader(wire.Bytes()), io.Discard, nil)
	require.NoError(t, reader.EnableCompression())
	got := make([]byte, len(first)+len(second))
	_, err = io.ReadFull(reader, got)
	require.NoError(t, err)
	require.Equal(t, append(first, second...), got)
}

func TestCompressionWrapsEncryption(t *testing.T) {
	payload := bytes.Repeat([]byte("compress before encrypt "), 256)
	key := []byte("test-key")

	var wire bytes.Buffer
	writer := NewClient(nil, &wire, nil)
	require.NoError(t, writer.EnableCompression())
	writer.SetCodec("test", &xorStreamCodec{key: key})
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())

	decrypted := append([]byte(nil), wire.Bytes()...)
	(&xorStreamCodec{key: key}).Decrypt(decrypted, decrypted)
	zr, err := zlib.NewReader(bytes.NewReader(decrypted))
	require.NoError(t, err)
	plain := make([]byte, len(payload))
	_, err = io.ReadFull(zr, plain)
	require.NoError(t, err)
	require.NoError(t, zr.Close())
	require.Equal(t, payload, plain)

	reader := NewClient(bytes.NewReader(wire.Bytes()), io.Discard, nil)
	require.NoError(t, reader.EnableCompression())
	reader.SetCodec("test", &xorStreamCodec{key: key})
	got := make([]byte, len(payload))
	_, err = io.ReadFull(reader, got)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestCompressionPreservesStreamAcrossEncryptionAndRebind(t *testing.T) {
	beforeCrypt := bytes.Repeat([]byte("op_crypt "), 32)
	afterCrypt := bytes.Repeat([]byte("encrypted response "), 64)
	key := []byte("boundary-key")

	var plainWire bytes.Buffer
	writer := NewClient(nil, &plainWire, nil)
	require.NoError(t, writer.EnableCompression())
	_, err := writer.Write(beforeCrypt)
	require.NoError(t, err)
	require.NoError(t, writer.flushCompression())

	var encryptedWire bytes.Buffer
	writer.Rebind(nil, &encryptedWire)
	writer.SetCodec("test", &xorStreamCodec{key: key})
	_, err = writer.Write(afterCrypt)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())

	reader := NewClient(bytes.NewReader(plainWire.Bytes()), io.Discard, nil)
	require.NoError(t, reader.EnableCompression())
	gotBefore := make([]byte, len(beforeCrypt))
	_, err = io.ReadFull(reader, gotBefore)
	require.NoError(t, err)
	require.Equal(t, beforeCrypt, gotBefore)

	reader.Rebind(bytes.NewReader(encryptedWire.Bytes()), io.Discard)
	reader.SetCodec("test", &xorStreamCodec{key: key})
	gotAfter := make([]byte, len(afterCrypt))
	_, err = io.ReadFull(reader, gotAfter)
	require.NoError(t, err)
	require.Equal(t, afterCrypt, gotAfter)
}

func TestCompressionDoesNotEmitEmptyFlushAcrossEncryptionBoundary(t *testing.T) {
	var wire bytes.Buffer
	client := NewClient(nil, &wire, nil)
	require.NoError(t, client.EnableCompression())
	_, err := client.Write([]byte("op_crypt"))
	require.NoError(t, err)
	require.NoError(t, client.Flush())
	plainLength := wire.Len()

	client.SetCodec("test", &xorStreamCodec{key: []byte("key")})
	require.NoError(t, client.Flush())
	require.Equal(t, plainLength, wire.Len())
}

func TestCloseCompressionFinalizesStreamAndFlushesWriter(t *testing.T) {
	payload := bytes.Repeat([]byte("final payload "), 64)
	var wire bytes.Buffer
	bufferedWire := bufio.NewWriter(&wire)
	client := NewClient(nil, bufferedWire, nil)
	require.NoError(t, client.EnableCompression())

	_, err := client.Write(payload)
	require.NoError(t, err)
	require.NoError(t, client.CloseCompression())
	require.False(t, client.CompressionEnabled())
	require.NoError(t, client.CloseCompression())

	zr, err := zlib.NewReader(bytes.NewReader(wire.Bytes()))
	require.NoError(t, err)
	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.NoError(t, zr.Close())
	require.Equal(t, payload, decoded)
}

func TestClosedCompressionDoesNotUseUncompressedTransport(t *testing.T) {
	client := NewClient(bytes.NewReader(nil), io.Discard, nil)
	require.NoError(t, client.EnableCompression())
	require.NoError(t, client.CloseCompression())

	_, err := client.Read(make([]byte, 1))
	require.ErrorIs(t, err, net.ErrClosed)
	_, err = client.Write([]byte("request"))
	require.ErrorIs(t, err, net.ErrClosed)
	require.ErrorIs(t, client.Flush(), net.ErrClosed)
	require.ErrorIs(t, client.EnableCompression(), net.ErrClosed)
	require.NoError(t, client.CloseCompression())
}

func TestCloseCompressionDefersReaderCloseUntilReadCompletes(t *testing.T) {
	client := NewClient(bytes.NewReader(nil), io.Discard, nil)
	require.NoError(t, client.EnableCompression())

	blockingReader := newBlockingCompressionReader()
	defer blockingReader.release()
	client.compression.stateMu.Lock()
	client.compression.decompressor = blockingReader
	client.compression.stateMu.Unlock()

	readDone := make(chan error, 1)
	go func() {
		_, err := client.Read(make([]byte, 1))
		readDone <- err
	}()

	select {
	case <-blockingReader.readStarted:
	case <-time.After(time.Second):
		t.Fatal("compressed read did not start")
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- client.CloseCompression()
	}()

	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		blockingReader.release()
		t.Fatal("CloseCompression blocked on an active read")
	}

	closedBeforeReadCompleted := false
	select {
	case <-blockingReader.closeCalled:
		closedBeforeReadCompleted = true
	default:
	}

	const waiterCount = 2
	waitStarted := make(chan struct{}, waiterCount)
	waitDone := make(chan struct{}, waiterCount)
	for range waiterCount {
		go func() {
			waitStarted <- struct{}{}
			client.WaitForCompressionRead()
			waitDone <- struct{}{}
		}()
	}
	for range waiterCount {
		<-waitStarted
	}
	select {
	case <-waitDone:
		blockingReader.release()
		t.Fatal("WaitForCompressionRead returned before the read completed")
	case <-time.After(20 * time.Millisecond):
	}

	blockingReader.release()
	select {
	case err := <-readDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("compressed read did not finish")
	}

	select {
	case <-blockingReader.closeCalled:
	case <-time.After(time.Second):
		t.Fatal("decompressor was not closed after the read completed")
	}

	for range waiterCount {
		select {
		case <-waitDone:
		case <-time.After(time.Second):
			t.Fatal("WaitForCompressionRead did not return after the read completed")
		}
	}
	require.False(t, closedBeforeReadCompleted)
	require.False(t, blockingReader.closedDuringRead.Load())
	require.False(t, client.CompressionEnabled())
}

func TestCloseCompressionDefersReaderCloseDuringInitialization(t *testing.T) {
	var wire bytes.Buffer
	compressor := zlib.NewWriter(&wire)
	_, err := compressor.Write(bytes.Repeat([]byte("response "), 128))
	require.NoError(t, err)
	require.NoError(t, compressor.Close())

	transport := newBlockingFirstRead(bytes.NewReader(wire.Bytes()))
	defer transport.release()
	client := NewClient(transport, io.Discard, nil)
	require.NoError(t, client.EnableCompression())

	readDone := make(chan error, 1)
	go func() {
		_, err := client.Read(make([]byte, 1))
		readDone <- err
	}()

	select {
	case <-transport.readStarted:
	case <-time.After(time.Second):
		t.Fatal("zlib reader initialization did not start")
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- client.CloseCompression()
	}()
	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("CloseCompression blocked during zlib reader initialization")
	}

	waitStarted := make(chan struct{})
	waitDone := make(chan struct{})
	go func() {
		close(waitStarted)
		client.WaitForCompressionRead()
		close(waitDone)
	}()
	<-waitStarted
	select {
	case <-waitDone:
		transport.release()
		t.Fatal("compression cleanup completed during zlib reader initialization")
	case <-time.After(20 * time.Millisecond):
	}

	transport.release()
	select {
	case err := <-readDone:
		require.ErrorIs(t, err, net.ErrClosed)
	case <-time.After(time.Second):
		t.Fatal("compressed read did not finish")
	}
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("compression cleanup did not finish")
	}
}

func TestCloseCompressionWaitsForActiveCompressedWrite(t *testing.T) {
	transport := newArmableBlockingWriter(io.Discard)
	defer transport.release()
	client := NewClient(nil, transport, nil)
	require.NoError(t, client.EnableCompression())

	_, err := client.Write([]byte("request"))
	require.NoError(t, err)
	transport.arm()

	flushDone := make(chan error, 1)
	go func() {
		flushDone <- client.Flush()
	}()
	select {
	case <-transport.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("compressed write did not block on the transport")
	}

	closeDone := make(chan error, 1)
	closeStarted := make(chan struct{})
	go func() {
		close(closeStarted)
		closeDone <- client.CloseCompression()
	}()
	<-closeStarted
	select {
	case <-closeDone:
		transport.release()
		t.Fatal("CloseCompression completed during an active compressed write")
	case <-time.After(20 * time.Millisecond):
	}

	transport.release()
	select {
	case err := <-flushDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("compressed write did not finish")
	}
	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("CloseCompression did not finish")
	}
}
