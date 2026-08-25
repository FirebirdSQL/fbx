package fbproto

import (
	"compress/zlib"
	"io"
	"net"
	"sync"
)

type readerFunc func([]byte) (int, error)

// Read forwards buf to f.
func (f readerFunc) Read(buf []byte) (int, error) {
	return f(buf)
}

type writerFunc func([]byte) (int, error)

// Write forwards buf to f.
func (f writerFunc) Write(buf []byte) (int, error) {
	return f(buf)
}

// compressionStream owns the complete lifecycle of the protocol's streaming
// zlib reader and writer. Reads and writes use independent locks because a
// blocked network read must not prevent writes or compression shutdown.
type compressionStream struct {
	readTransport  readerFunc
	writeTransport writerFunc
	flushTransport func() error

	readMu  sync.Mutex
	writeMu sync.Mutex
	stateMu sync.Mutex

	enabled  bool
	closed   bool
	reading  bool
	readDone chan struct{}
	dirty    bool

	compressor   *zlib.Writer
	decompressor io.ReadCloser
}

func newCompressionStream(
	readTransport readerFunc,
	writeTransport writerFunc,
	flushTransport func() error,
) *compressionStream {
	return &compressionStream{
		readTransport:  readTransport,
		writeTransport: writeTransport,
		flushTransport: flushTransport,
	}
}

// EnableCompression enables streaming zlib compression for subsequent wire data.
func (c *Client) EnableCompression() error {
	return c.compression.enable()
}

// CompressionEnabled reports whether wire compression is enabled.
func (c *Client) CompressionEnabled() bool {
	if c == nil {
		return false
	}
	return c.compression != nil && c.compression.isEnabled()
}

func (c *Client) flushCompression() error {
	return c.compression.flushCompression()
}

// Flush writes any buffered compressed data and then flushes the underlying
// writer, if it supports flushing.
func (c *Client) Flush() error {
	return c.compression.flush()
}

// CloseCompression finalizes an enabled compressed stream and releases its
// resources.
// It flushes the underlying writer but does not close the underlying transport.
// CloseCompression may run concurrently with [Client.Read] or [Client.Write].
// It waits for an in-progress write, while read-side cleanup completes after
// an in-progress read returns.
// If compression has not been enabled or is already closed, CloseCompression
// returns nil without changing the state.
// After an enabled stream is closed, [Client.EnableCompression], [Client.Read],
// [Client.Write], and [Client.Flush] fail with [net.ErrClosed].
func (c *Client) CloseCompression() error {
	if c == nil || c.compression == nil {
		return nil
	}
	return c.compression.close()
}

// WaitForCompressionRead waits for an in-progress compressed read and cleanup
// deferred by [Client.CloseCompression].
// Call CloseCompression and close the underlying transport first so a blocked
// read can return.
func (c *Client) WaitForCompressionRead() {
	if c == nil || c.compression == nil {
		return
	}
	c.compression.waitRead()
}

func (s *compressionStream) enable() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.enabled {
		return nil
	}
	if s.closed {
		return net.ErrClosed
	}

	s.compressor = zlib.NewWriter(s.writeTransport)
	s.enabled = true
	return nil
}

func (s *compressionStream) isEnabled() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.enabled
}

func (s *compressionStream) read(buf []byte) (n int, err error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	s.stateMu.Lock()
	if s.closed {
		s.stateMu.Unlock()
		return 0, net.ErrClosed
	}
	if !s.enabled {
		s.stateMu.Unlock()
		return s.readTransport(buf)
	}

	s.reading = true
	decompressor := s.decompressor
	s.stateMu.Unlock()
	defer s.finishRead()

	if decompressor == nil {
		// zlib.NewReader consumes the stream header, so delay it until the
		// server is expected to have sent compressed data.
		decompressor, err = zlib.NewReader(s.readTransport)
		if err != nil {
			return 0, err
		}

		s.stateMu.Lock()
		s.decompressor = decompressor
		closed := s.closed
		s.stateMu.Unlock()
		if closed {
			return 0, net.ErrClosed
		}
	}
	return decompressor.Read(buf)
}

func (s *compressionStream) finishRead() {
	s.stateMu.Lock()
	if !s.closed {
		s.finishReadLocked()
		s.stateMu.Unlock()
		return
	}

	decompressor := s.decompressor
	s.decompressor = nil
	s.stateMu.Unlock()
	if decompressor != nil {
		_ = decompressor.Close()
	}

	s.stateMu.Lock()
	s.finishReadLocked()
	s.stateMu.Unlock()
}

func (s *compressionStream) finishReadLocked() {
	s.reading = false
	if s.readDone != nil {
		close(s.readDone)
		s.readDone = nil
	}
}

func (s *compressionStream) write(buf []byte) (n int, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.isClosed() {
		return 0, net.ErrClosed
	}
	if s.compressor == nil {
		return s.writeTransport(buf)
	}

	n, err = s.compressor.Write(buf)
	if n > 0 {
		s.dirty = true
	}
	return n, err
}

func (s *compressionStream) flush() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.isClosed() {
		return net.ErrClosed
	}
	if err := s.flushCompressionLocked(); err != nil {
		return err
	}
	return s.flushTransport()
}

func (s *compressionStream) flushCompression() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.isClosed() {
		return net.ErrClosed
	}
	return s.flushCompressionLocked()
}

func (s *compressionStream) flushCompressionLocked() error {
	if s.compressor == nil || !s.dirty {
		return nil
	}
	if err := s.compressor.Flush(); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

func (s *compressionStream) close() error {
	s.writeMu.Lock()
	s.stateMu.Lock()
	if !s.enabled {
		s.stateMu.Unlock()
		s.writeMu.Unlock()
		return nil
	}

	compressor := s.compressor
	s.compressor = nil
	s.enabled = false
	s.closed = true
	s.dirty = false

	var decompressor io.ReadCloser
	if !s.reading {
		decompressor = s.decompressor
		s.decompressor = nil
	}
	s.stateMu.Unlock()

	var resultErr error
	if compressor != nil {
		resultErr = compressor.Close()
	}
	if err := s.flushTransport(); resultErr == nil {
		resultErr = err
	}
	s.writeMu.Unlock()

	if decompressor != nil {
		if err := decompressor.Close(); resultErr == nil {
			resultErr = err
		}
	}
	return resultErr
}

func (s *compressionStream) waitRead() {
	s.stateMu.Lock()
	if !s.reading {
		s.stateMu.Unlock()
		return
	}
	if s.readDone == nil {
		s.readDone = make(chan struct{})
	}
	done := s.readDone
	s.stateMu.Unlock()
	<-done
}

func (s *compressionStream) isClosed() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.closed
}
