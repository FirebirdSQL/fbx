package fbproto

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"net"
	"sync"
)

type Client struct {
	Encoding *Encoding
	reader   io.Reader
	writer   io.Writer
	plugin   string
	codec    StreamCodec
	// Reads and compressed writes use separate locks so a blocked write cannot
	// prevent the reader from draining the peer's response.
	readMu             sync.Mutex
	compressionWriteMu sync.Mutex
	// compressionMu protects compression negotiation and read-side lifecycle.
	// It is never held while reading from the transport.
	compressionMu       sync.Mutex
	compressionEnabled  bool
	compressionClosed   bool
	compressionRead     bool
	compressionReadDone chan struct{}
	compressionDirty    bool
	compressor          *zlib.Writer
	decompressor        io.ReadCloser
	xdrReader           *XDRReader
	xdrWriter           *XDRWriter
	rowScratch          rowScratch
	tracer              *tracer
	lazyCounter         int
	reusable            reusableServerMessages
	inlineBlobHandler   func(*OpInlineBlob)
	inlineBlobAlloc     func(int) []byte
	inlineBlobCache     *inlineBlobCache
	inlineBlobPool      *inlineBlobPool

	writeScratch []byte
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(buf []byte) (int, error) {
	return f(buf)
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(buf []byte) (int, error) {
	return f(buf)
}

type reusableServerMessages struct {
	dummy         OpDummy
	accept        OpAccept
	condAccept    OpCondAccept
	dataAccept    OpDataAccept
	response      OpResponse
	reject        OpReject
	fetchResponse OpFetchResponse
	sqlResponse   OpSqlResponse
	batchResp     OpBatchCompletion
	contAuth      OpContAuth
	inlineBlob    OpInlineBlob
}

// StreamCodec is a minimal interface for symmetric wire encryption.
// Implementations are provided by higher-level code to avoid package cycles.
type StreamCodec interface {
	Encrypt(dst, src []byte)
	Decrypt(dst, src []byte)
}

func NewClient(r io.Reader, w io.Writer, encoding *Encoding) *Client {
	c := Client{}
	c.reader = r
	c.writer = w
	c.xdrReader = NewXDRReader(&c)
	c.xdrWriter = NewXDRWriter(&c)
	c.Encoding = encoding
	c.reusable.condAccept.Encoding = encoding
	c.reusable.dataAccept.Encoding = encoding
	c.reusable.response.Encoding = encoding
	c.reusable.contAuth.Encoding = encoding
	return &c
}

// LazyCounter returns the number of pending lazy messages
// that the server is expected to send.
func (c *Client) LazyCounter() int {
	return c.lazyCounter
}

// Rebind replaces the underlying reader and writer while preserving codec, lazyCounter,
// Encoding and all other Client state.
func (c *Client) Rebind(r io.Reader, w io.Writer) {
	c.reader = r
	c.writer = w
}

// EnableCompression enables streaming zlib compression for subsequent wire data.
func (c *Client) EnableCompression() error {
	c.compressionWriteMu.Lock()
	defer c.compressionWriteMu.Unlock()
	c.compressionMu.Lock()
	defer c.compressionMu.Unlock()

	if c.compressionEnabled {
		return nil
	}
	if c.compressionClosed {
		return net.ErrClosed
	}

	c.compressor = zlib.NewWriter(writerFunc(c.writeTransport))
	c.compressionEnabled = true
	return nil
}

// CompressionEnabled reports whether wire compression is enabled.
func (c *Client) CompressionEnabled() bool {
	if c == nil {
		return false
	}
	c.compressionMu.Lock()
	defer c.compressionMu.Unlock()
	return c.compressionEnabled
}

func (c *Client) flushCompression() error {
	c.compressionWriteMu.Lock()
	defer c.compressionWriteMu.Unlock()
	if c.compressionClosedState() {
		return net.ErrClosed
	}
	return c.flushCompressionLocked()
}

func (c *Client) flushCompressionLocked() error {
	if c.compressor == nil || !c.compressionDirty {
		return nil
	}
	if err := c.compressor.Flush(); err != nil {
		return err
	}
	c.compressionDirty = false
	return nil
}

// Flush emits all compressed data and then flushes the underlying writer, if it supports flushing.
func (c *Client) Flush() error {
	c.compressionWriteMu.Lock()
	defer c.compressionWriteMu.Unlock()

	if c.compressionClosedState() {
		return net.ErrClosed
	}
	if err := c.flushCompressionLocked(); err != nil {
		return err
	}
	if flusher, ok := c.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// CloseCompression finalizes the compressed stream and releases its resources.
// It flushes the underlying writer but does not close the underlying transport.
// If a read is in progress, that read closes the decompressor before returning.
// Further compression or I/O operations on the Client fail with [net.ErrClosed].
func (c *Client) CloseCompression() error {
	if c == nil {
		return nil
	}

	c.compressionWriteMu.Lock()
	c.compressionMu.Lock()
	if !c.compressionEnabled {
		c.compressionMu.Unlock()
		c.compressionWriteMu.Unlock()
		return nil
	}

	compressor := c.compressor
	c.compressor = nil
	c.compressionEnabled = false
	c.compressionClosed = true
	c.compressionDirty = false

	var decompressor io.ReadCloser
	if !c.compressionRead {
		decompressor = c.decompressor
		c.decompressor = nil
	}
	c.compressionMu.Unlock()

	var resultErr error
	if compressor != nil {
		resultErr = compressor.Close()
	}
	if flusher, ok := c.writer.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); resultErr == nil {
			resultErr = err
		}
	}
	c.compressionWriteMu.Unlock()
	if decompressor != nil {
		if err := decompressor.Close(); resultErr == nil {
			resultErr = err
		}
	}
	return resultErr
}

// WaitForCompressionRead waits for an in-progress compressed read and its
// decompressor cleanup. The underlying transport should be closed first so a
// blocked read can return.
func (c *Client) WaitForCompressionRead() {
	if c == nil {
		return
	}

	c.compressionMu.Lock()
	if !c.compressionRead {
		c.compressionMu.Unlock()
		return
	}
	if c.compressionReadDone == nil {
		c.compressionReadDone = make(chan struct{})
	}
	done := c.compressionReadDone
	c.compressionMu.Unlock()
	<-done
}

// SetCodec sets the active wire encryption codec.
func (c *Client) SetCodec(plugin string, codec StreamCodec) {
	c.plugin = plugin
	c.codec = codec
	if codec != nil && cap(c.writeScratch) < cryptoChunkSize {
		c.writeScratch = make([]byte, cryptoChunkSize)
	}
}

// SetInlineBlobHandler sets a handler for op_inline_blob messages.
func (c *Client) SetInlineBlobHandler(handler func(*OpInlineBlob)) {
	c.inlineBlobHandler = handler
}

// SetInlineBlobDataAllocator sets a buffer allocator for inline blob payloads.
func (c *Client) SetInlineBlobDataAllocator(alloc func(int) []byte) {
	c.inlineBlobAlloc = alloc
}

// ConfigureInlineBlobCache enables client-side inline blob caching.
func (c *Client) ConfigureInlineBlobCache(maxInlineBlobSize, maxCacheSize int) {
	c.inlineBlobPool = newInlineBlobPool(maxInlineBlobSize)
	c.inlineBlobCache = newInlineBlobCache(maxCacheSize, c.inlineBlobPool)
	if c.inlineBlobPool != nil {
		c.inlineBlobAlloc = c.inlineBlobPool.Get
	} else {
		c.inlineBlobAlloc = nil
	}
}

func (c *Client) handleInlineBlob(msg *OpInlineBlob) {
	if c.inlineBlobCache == nil {
		if c.inlineBlobPool != nil {
			c.inlineBlobPool.Put(msg.Data)
		}
		if c.inlineBlobHandler != nil {
			c.inlineBlobHandler(msg)
		}
		return
	}
	data := msg.Data
	if data == nil {
		data = []byte{}
	}
	if !c.inlineBlobCache.add(msg.TransactionHandle, msg.BlobId, data) {
		if c.inlineBlobPool != nil {
			c.inlineBlobPool.Put(data)
		}
	}
	if c.inlineBlobHandler != nil {
		c.inlineBlobHandler(msg)
	}
}

// ConsumeInlineBlob returns cached inline blob data for the transaction and removes it from the cache.
func (c *Client) ConsumeInlineBlob(txHandle int, blobId int64) ([]byte, bool) {
	if c == nil {
		panic("client is nil")
	}
	if c.inlineBlobCache == nil {
		return nil, false
	}
	return c.inlineBlobCache.getAndRemove(txHandle, blobId)
}

// ClearInlineBlobCache removes all cached inline blobs for a transaction.
func (c *Client) ClearInlineBlobCache(txHandle int) {
	if c == nil {
		panic("client is nil")
	}
	if c.inlineBlobCache == nil {
		return
	}
	c.inlineBlobCache.clearTx(txHandle)
}

// ClearAllInlineBlobCache removes all cached inline blobs.
func (c *Client) ClearAllInlineBlobCache() {
	if c == nil {
		panic("client is nil")
	}
	if c.inlineBlobCache == nil {
		return
	}
	c.inlineBlobCache.clearAll()
}

// ReleaseInlineBlobData returns inline blob data to the pool.
func (c *Client) ReleaseInlineBlobData(data []byte) {
	if c == nil {
		panic("client is nil")
	}
	if c.inlineBlobPool == nil || data == nil {
		return
	}
	c.inlineBlobPool.Put(data)
}

func (c *Client) Read(buf []byte) (n int, err error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	c.compressionMu.Lock()
	if c.compressionClosed {
		c.compressionMu.Unlock()
		return 0, net.ErrClosed
	}
	if !c.compressionEnabled {
		c.compressionMu.Unlock()
		return c.readTransport(buf)
	}

	c.compressionRead = true
	decompressor := c.decompressor
	c.compressionMu.Unlock()
	defer c.finishCompressionRead()

	if decompressor == nil {
		// zlib.NewReader consumes the stream header, so delay it until the
		// server is expected to have sent compressed data.
		decompressor, err = zlib.NewReader(readerFunc(c.readTransport))
		if err != nil {
			return 0, err
		}

		c.compressionMu.Lock()
		c.decompressor = decompressor
		c.compressionMu.Unlock()
	}
	return decompressor.Read(buf)
}

func (c *Client) finishCompressionRead() {
	c.compressionMu.Lock()
	var decompressor io.ReadCloser
	if c.compressionClosed {
		decompressor = c.decompressor
		c.decompressor = nil
	}
	c.compressionMu.Unlock()

	if decompressor != nil {
		_ = decompressor.Close()
	}

	c.compressionMu.Lock()
	c.compressionRead = false
	if done := c.compressionReadDone; done != nil {
		close(done)
		c.compressionReadDone = nil
	}
	c.compressionMu.Unlock()
}

func (c *Client) readTransport(buf []byte) (n int, err error) {
	if c.codec != nil {
		n, err = c.reader.Read(buf)
		if n > 0 {
			// NOTE: Our wire crypt plugins are stream ciphers and support in-place XOR.
			c.codec.Decrypt(buf[:n], buf[:n])
		}
		return
	}
	return c.reader.Read(buf)
}

func (c *Client) Write(buf []byte) (n int, err error) {
	c.compressionWriteMu.Lock()
	defer c.compressionWriteMu.Unlock()

	if c.compressionClosedState() {
		return 0, net.ErrClosed
	}
	if c.compressor != nil {
		n, err = c.compressor.Write(buf)
		if n > 0 {
			c.compressionDirty = true
		}
		return n, err
	}
	return c.writeTransport(buf)
}

func (c *Client) compressionClosedState() bool {
	c.compressionMu.Lock()
	defer c.compressionMu.Unlock()
	return c.compressionClosed
}

func (c *Client) writeTransport(buf []byte) (n int, err error) {
	if c.codec != nil {
		if cap(c.writeScratch) < cryptoChunkSize {
			c.writeScratch = make([]byte, cryptoChunkSize)
		}
		scratch := c.writeScratch[:cryptoChunkSize]
		totalWritten := 0
		for offset := 0; offset < len(buf); {
			chunkLen := len(buf) - offset
			if chunkLen > len(scratch) {
				chunkLen = len(scratch)
			}
			chunk := scratch[:chunkLen]
			c.codec.Encrypt(chunk, buf[offset:offset+chunkLen])

			written := 0
			for written < chunkLen {
				n, err = c.writer.Write(chunk[written:])
				if err != nil {
					return totalWritten, err
				}
				written += n
				totalWritten += n
			}
			offset += chunkLen
		}
		return totalWritten, nil
	}
	return c.writer.Write(buf)
}

func (c *Client) WriteMessage(msg ClientMessage) error {
	err := msg.Write(c.xdrWriter)
	if err != nil {
		return err
	}

	switch msg.(type) {
	//FIXME: fbx should be implicit or explicit?
	case *OpFreeStatement:
		c.lazyCounter++
	}

	if c.tracer != nil {
		c.tracer.traceRequest('C', msg)
	}
	return nil
}

func (c *Client) ReadPackets(n int) ([]byte, error) {
	buf := make([]byte, n)
	var err error
	read := 0
	totalRead := 0

	padding := n % 4
	if padding > 0 {
		padding = 4 - padding
	}
	paddedLen := n + padding

	for totalRead < paddedLen {
		read, err = c.Read(buf[totalRead:paddedLen])
		if err != nil {
			return buf, err
		}
		totalRead += read
	}
	return buf, nil
}

func (c *Client) ReadMessage() (ServerMessage, error) {
	opcode, err := c.xdrReader.ReadInt32()
	if err != nil {
		return nil, err
	}
	var msg ServerMessage
	switch opcode {
	case Op_dummy:
		return &c.reusable.dummy, nil
	case Op_accept:
		msg = &c.reusable.accept
	case Op_cond_accept:
		c.reusable.condAccept.Encoding = c.Encoding
		msg = &c.reusable.condAccept
	case Op_accept_data:
		c.reusable.dataAccept.Encoding = c.Encoding
		msg = &c.reusable.dataAccept
	case Op_response:
		c.reusable.response.Encoding = c.Encoding
		msg = &c.reusable.response
	case Op_reject:
		msg = &c.reusable.reject
	case Op_fetch_response:
		msg = &c.reusable.fetchResponse
	case Op_sql_response:
		msg = &c.reusable.sqlResponse
	case Op_batch_cs:
		c.reusable.batchResp.Encoding = c.Encoding
		msg = &c.reusable.batchResp
	case Op_cont_auth:
		c.reusable.contAuth.Encoding = c.Encoding
		msg = &c.reusable.contAuth
	case Op_inline_blob:
		inline := &c.reusable.inlineBlob
		err = inline.ReadWithAllocator(c.xdrReader, c.inlineBlobAlloc)
		return inline, err
	default:
		return nil, fmt.Errorf("unknown opcode: %d", opcode)
	}
	err = msg.Read(c.xdrReader)
	return msg, err
}

func (c *Client) ReceiveMessage() (ServerMessage, error) {
	var msg ServerMessage
	var err error
	for {
		msg, err = c.ReadMessage()
		if err != nil {
			return nil, err
		}

		if c.tracer != nil {
			c.tracer.traceResponse('S', msg)
		}

		switch m := msg.(type) {
		case *OpDummy:
			continue
		case *OpInlineBlob:
			c.handleInlineBlob(m)
			continue
		case *OpResponse:
			if c.lazyCounter == 0 {
				if m.Failure {
					return errorResponseFromOpResponse(m), nil
				}
			} else {
				if m.Failure {
					fmt.Printf("Ignoring lazy response with error: %s", m.Message)
				}
				c.lazyCounter--
				continue
			}
		case *OpReject:
			return msg, fmt.Errorf("server rejected connection")
		default:
			if c.lazyCounter > 0 {
				panic(fmt.Sprintf("unexpected response of type %T but expected lazy op_response", m))
			}
		}
		break
	}

	return msg, nil
}

func (c *Client) ReceiveSQLData(row []FieldDescriptor) (*SQLData, ServerMessage, error) {
	sqlData, err := ReadRowData(c.xdrReader, row, &c.rowScratch)
	if err != nil {
		return nil, nil, err
	}

	if c.tracer != nil {
		c.tracer.traceRowData('S', sqlData, len(row))
	}

	msg, err := c.ReceiveMessage()
	if err != nil {
		return nil, nil, err
	}
	return sqlData, msg, nil
}

func (c *Client) Trace(w io.Writer, options TracerOptions) {
	c.tracer = &tracer{
		w:             w,
		buf:           &bytes.Buffer{},
		TracerOptions: options,
	}
}

func (c *Client) Untrace() {
	c.tracer = nil
}

const cryptoChunkSize = 16 * 1024

func (c *Client) ReadSlice(datatype uint32, length int) (ServerMessage, error) {
	opcode, err := c.xdrReader.ReadInt32()
	if err != nil {
		return nil, err
	}
	var msg ServerMessage
	switch opcode {
	case Op_slice_response:
		msg = &OpSliceResponse{DataType: datatype, SliceLengthElements: length}
	case Op_response:
		msg = &OpResponse{Encoding: c.Encoding}
	default:
		return nil, fmt.Errorf("unexpected response opcode: %d", opcode)
	}
	if err = msg.Read(c.xdrReader); err != nil {
		return nil, err
	}
	if op, ok := msg.(*OpResponse); ok && op.Failure {
		return errorResponseFromOpResponse(op), nil
	}
	return msg, nil
}
