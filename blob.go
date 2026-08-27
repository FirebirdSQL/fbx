package fbx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

var (
	errBlobClosed  = errors.New("blob is closed")
	errBlobNotOpen = errors.New("blob is not open")
	errBlobInput   = errors.New("blob is input")
	errBlobOutput  = errors.New("blob is output")
)

type blobID int64

type blobState uint8

const (
	blobStateNew blobState = iota
	blobStateOpen
	blobStateClosed
)

type blobConfigWriter interface {
	writeOutputConfig(*fbproto.BPBWriter)
	writeInputConfig(*fbproto.BPBWriter)
}

type blobConfigInfo interface {
	blobSubType() int32
	blobStream() bool
	blobTemporary() bool
	blobCharacterSet() string
}

// blobConfigOptions defines optional per-blob settings.
// If nil is passed to create/open methods, connection defaults are used.
type blobConfigOptions struct {
	SubType      int32
	Stream       bool
	Temporary    bool
	CharacterSet string

	charsetID    int32
	hasCharsetID bool
}

func defaultBlobConfig() *blobConfigOptions {
	return &blobConfigOptions{
		SubType:   fbproto.BLOB_SUB_TYPE_BINARY,
		Stream:    true,
		Temporary: false,
	}
}

func defaultBlobConfigForConn(conn *Conn) *blobConfigOptions {
	cfg := defaultBlobConfig()
	if conn != nil && conn.config != nil {
		cfg.Stream = conn.config.UseStreamBlobs
	}
	return cfg
}

func (cfg *blobConfigOptions) blobSubType() int32 {
	if cfg == nil {
		return fbproto.BLOB_SUB_TYPE_BINARY
	}
	return cfg.SubType
}

func (cfg *blobConfigOptions) blobStream() bool {
	if cfg == nil {
		return true
	}
	return cfg.Stream
}

func (cfg *blobConfigOptions) blobTemporary() bool {
	if cfg == nil {
		return false
	}
	return cfg.Temporary
}

func (cfg *blobConfigOptions) blobCharacterSet() string {
	if cfg == nil {
		return ""
	}
	return cfg.CharacterSet
}

func (cfg *blobConfigOptions) writeOutputConfig(bpb *fbproto.BPBWriter) {
	if cfg == nil {
		return
	}
	charsetID := int32(0)
	if cfg.hasCharsetID {
		charsetID = cfg.charsetID
	}
	fbproto.WriteOutputBPB(bpb, fbproto.BlobBPBConfig{
		SubType:   cfg.SubType,
		Stream:    cfg.Stream,
		Temporary: cfg.Temporary,
		CharsetID: charsetID,
	})
}

func (cfg *blobConfigOptions) writeInputConfig(bpb *fbproto.BPBWriter) {
	// Intentionally empty to keep input blobs opened without BPB by default.
}

type blobConfig struct {
	subType   int32
	stream    bool
	temporary bool
	charsetID int32
	encoding  *fbproto.Encoding
	outputBPB []byte
	inputBPB  []byte
}

func resolveBlobConfig(conn *Conn, cfg blobConfigWriter) (blobConfig, error) {
	if cfg == nil {
		cfg = defaultBlobConfigForConn(conn)
	} else if basicCfg, ok := cfg.(*blobConfigOptions); ok && basicCfg == nil {
		cfg = defaultBlobConfigForConn(conn)
	}

	resolved := blobConfig{
		subType:   fbproto.BLOB_SUB_TYPE_BINARY,
		stream:    true,
		temporary: false,
	}

	var charset string
	if info, ok := cfg.(blobConfigInfo); ok {
		resolved.subType = info.blobSubType()
		resolved.stream = info.blobStream()
		resolved.temporary = info.blobTemporary()
		charset = info.blobCharacterSet()
	}

	if resolved.subType == fbproto.BLOB_SUB_TYPE_TEXT {
		if charset == "" {
			return blobConfig{}, errors.New("blob text charset is not configured")
		}
		encoding, err := fbproto.NewEncoding(charset)
		if err != nil {
			return blobConfig{}, err
		}
		resolved.encoding = encoding
		resolved.charsetID = encoding.CharacterSetId
	}

	if basicCfg, ok := cfg.(*blobConfigOptions); ok && basicCfg != nil {
		cfgCopy := *basicCfg
		if resolved.subType == fbproto.BLOB_SUB_TYPE_TEXT {
			cfgCopy.charsetID = resolved.charsetID
			cfgCopy.hasCharsetID = true
		}
		out := fbproto.NewBPBWriter()
		cfgCopy.writeOutputConfig(out)
		resolved.outputBPB = out.Bytes()

		in := fbproto.NewBPBWriter()
		cfgCopy.writeInputConfig(in)
		resolved.inputBPB = in.Bytes()
	} else {
		out := fbproto.NewBPBWriter()
		cfg.writeOutputConfig(out)
		resolved.outputBPB = out.Bytes()

		in := fbproto.NewBPBWriter()
		cfg.writeInputConfig(in)
		resolved.inputBPB = in.Bytes()
	}

	return resolved, nil
}

func blobConfigFromField(conn *Conn, fd *fbconn.FieldDescription) (blobConfig, error) {
	var err error
	cfg := defaultBlobConfigForConn(conn)
	cfg.SubType = fd.Subtype
	if cfg.SubType == fbproto.BLOB_SUB_TYPE_TEXT {
		cfg.CharacterSet, err = fbproto.CharacterSetIdIntoName(fd.Scale & 0xFF)
		if err != nil {
			return blobConfig{}, err
		}
	}
	return resolveBlobConfig(conn, cfg)
}

func blobConfigFromParam(conn *Conn, subType int32, scale int32) (blobConfig, error) {
	var err error
	cfg := defaultBlobConfigForConn(conn)
	cfg.SubType = subType
	if cfg.SubType == fbproto.BLOB_SUB_TYPE_TEXT {
		cfg.CharacterSet, err = fbproto.CharacterSetIdIntoName(scale & 0xFF)
		if err != nil {
			return blobConfig{}, err
		}
	}
	return resolveBlobConfig(conn, cfg)
}

type blob struct {
	conn   *Conn
	tx     *fbconn.FBTransaction
	id     blobID
	handle fbconn.FBBlob
	output bool
	state  blobState

	config      blobConfig
	segmentSize int
	// allowBusy permits blob operations while the connection lock is already held
	// by the caller (e.g., during row scanning).
	allowBusy bool
}

func (b *blob) ID() blobID {
	return b.id
}

func (b *blob) IsOutput() bool {
	return b.output
}

func (b *blob) IsOpen() bool {
	return b.state == blobStateOpen
}

func (b *blob) Reader(ctx context.Context) (*blobReader, error) {
	if b.output {
		return nil, errBlobOutput
	}
	if b.state == blobStateClosed {
		return nil, errBlobClosed
	}
	if b.state != blobStateOpen {
		return nil, errBlobNotOpen
	}
	reader := &blobReader{
		blob: b,
		ctx:  ctx,
	}
	return reader, nil
}

func (b *blob) Writer(ctx context.Context) (*blobWriter, error) {
	if !b.output {
		return nil, errBlobInput
	}
	if b.state != blobStateOpen {
		return nil, errBlobClosed
	}
	return &blobWriter{
		blob: b,
		ctx:  ctx,
	}, nil
}

func (b *blob) Close(ctx context.Context) error {
	if b.state == blobStateClosed {
		return errBlobClosed
	}
	if b.state != blobStateOpen {
		b.state = blobStateClosed
		return nil
	}
	if b.conn == nil || b.conn.fbConn == nil {
		return errors.New("blob connection is nil")
	}
	if b.handle == nil {
		b.state = blobStateClosed
		return nil
	}
	if err := b.handle.Close(ctx); err != nil {
		return err
	}
	b.state = blobStateClosed
	b.handle = nil
	return nil
}

func (b *blob) Cancel(ctx context.Context) error {
	if b.state == blobStateClosed {
		return errBlobClosed
	}
	if b.state != blobStateOpen {
		b.state = blobStateClosed
		return nil
	}
	if b.conn == nil || b.conn.fbConn == nil {
		return errors.New("blob connection is nil")
	}
	if b.handle == nil {
		b.state = blobStateClosed
		return nil
	}
	if err := b.handle.Cancel(ctx); err != nil {
		return err
	}
	b.state = blobStateClosed
	b.handle = nil
	return nil
}

func (b *blob) Length(ctx context.Context) (int64, error) {
	if b.output {
		return 0, errBlobOutput
	}
	if b.state == blobStateClosed {
		return 0, errBlobClosed
	}
	if b.state != blobStateOpen {
		return 0, errBlobNotOpen
	}
	if b.handle == nil {
		return 0, errBlobClosed
	}
	return b.handle.TotalLength(ctx)
}

func (b *blob) GetBytes(ctx context.Context) ([]byte, error) {
	reader, err := b.Reader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if length, err := b.Length(ctx); err == nil && length > 0 && length < int64(int(^uint(0)>>1)) {
		buf.Grow(int(length))
	}
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *blob) GetString(ctx context.Context) (string, error) {
	data, err := b.GetBytes(ctx)
	if err != nil {
		return "", err
	}
	if b.config.subType == fbproto.BLOB_SUB_TYPE_TEXT && b.config.encoding != nil {
		return b.config.encoding.Decoder.String(string(data))
	}
	return string(data), nil
}

func (b *blob) Seek(ctx context.Context, offset int64, whence int) (int64, error) {
	if b.output {
		return 0, errBlobOutput
	}
	if b.state == blobStateClosed {
		return 0, errBlobClosed
	}
	if b.state != blobStateOpen {
		return 0, errBlobNotOpen
	}
	seekMode, err := blobSeekMode(whence)
	if err != nil {
		return 0, err
	}
	if offset < int64(int32(-1<<31)) || offset > int64(int32(^uint32(0)>>1)) {
		return 0, fmt.Errorf("blob seek offset out of range: %d", offset)
	}
	if b.handle == nil {
		return 0, errBlobClosed
	}
	return b.handle.Seek(ctx, int(offset), seekMode)
}

func (b *blob) openForRead(ctx context.Context) error {
	if b.state == blobStateClosed {
		return errBlobClosed
	}
	if b.state == blobStateOpen {
		return nil
	}
	if b.output {
		return errBlobOutput
	}
	if b.conn == nil || b.conn.fbConn == nil {
		return errors.New("blob connection is nil")
	}
	locked := b.allowBusy && b.conn.fbConn.IsBusy()
	if b.tx != nil && !b.tx.Closed {
		handle, err := b.conn.fbConn.OpenBlob(ctx, b.tx, int64(b.id), b.config.inputBPB, locked)
		if err != nil {
			return err
		}
		b.handle = handle
		b.state = blobStateOpen
		return nil
	}
	handle, err := b.conn.fbConn.OpenBlob(ctx, nil, int64(b.id), b.config.inputBPB, locked)
	if err != nil {
		return err
	}
	b.handle = handle
	b.state = blobStateOpen
	return nil
}

type blobReader struct {
	blob   *blob
	ctx    context.Context
	buf    []byte
	pos    int
	limit  int
	eof    bool
	closed bool
}

type blobLazyReader struct {
	ctx    context.Context
	blob   *blob
	reader *blobReader
	closed bool
}

func newBlobLazyReader(ctx context.Context, blob *blob) *blobLazyReader {
	return &blobLazyReader{
		ctx:  ctx,
		blob: blob,
	}
}

func (r *blobLazyReader) Read(p []byte) (int, error) {
	reader, err := r.open()
	if err != nil {
		return 0, err
	}
	return reader.Read(p)
}

func (r *blobLazyReader) Seek(offset int64, whence int) (int64, error) {
	reader, err := r.open()
	if err != nil {
		return 0, err
	}
	return reader.Seek(offset, whence)
}

func (r *blobLazyReader) Close() error {
	if r.closed {
		return errBlobClosed
	}
	r.closed = true
	if r.reader != nil {
		return r.reader.Close()
	}
	return r.blob.Close(r.ctx)
}

func (r *blobLazyReader) open() (*blobReader, error) {
	if r.closed {
		return nil, errBlobClosed
	}
	if r.reader != nil {
		return r.reader, nil
	}
	if err := r.blob.openForRead(r.ctx); err != nil {
		return nil, err
	}
	reader, err := r.blob.Reader(r.ctx)
	if err != nil {
		return nil, err
	}
	r.reader = reader
	return reader, nil
}

func (r *blobReader) Read(p []byte) (int, error) {
	if r.closed {
		return 0, errBlobClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	// Inline blob fast-path (no segment framing).
	if r.pos < r.limit {
		n := copy(p, r.buf[r.pos:r.limit])
		r.pos += n
		return n, nil
	}
	if r.eof {
		return 0, io.EOF
	}
	segmentSize := r.blob.segmentSize
	reqBuf := p
	if len(reqBuf) > segmentSize {
		reqBuf = reqBuf[:segmentSize]
	}
	if r.blob.handle == nil {
		return 0, errBlobClosed
	}
	n, eof, err := r.blob.handle.GetSegmentInto(r.ctx, reqBuf)
	if err != nil {
		return 0, err
	}
	if n == 0 && eof {
		r.eof = true
		return 0, io.EOF
	}
	r.eof = eof
	return n, nil
}

func (r *blobReader) Seek(offset int64, whence int) (int64, error) {
	if r.closed {
		return 0, errBlobClosed
	}
	pos, err := r.blob.Seek(r.ctx, offset, whence)
	if err != nil {
		return 0, err
	}
	r.buf = nil
	r.pos = 0
	r.limit = 0
	r.eof = false
	return pos, nil
}

func (r *blobReader) Close() error {
	if r.closed {
		return errBlobClosed
	}
	r.closed = true
	return r.blob.Close(r.ctx)
}

type blobWriter struct {
	blob   *blob
	ctx    context.Context
	closed bool
}

func (w *blobWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, errBlobClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	segmentSize := w.blob.segmentSize
	written := 0
	const putBatchSize = 8
	offset := 0
	batch := make([][]byte, 0, putBatchSize)
	for offset < len(p) {
		batch = batch[:0]
		batchBytes := 0
		nextOffset := offset
		for nextOffset < len(p) && len(batch) < putBatchSize {
			n := min(len(p)-nextOffset, segmentSize)
			batch = append(batch, p[nextOffset:nextOffset+n])
			batchBytes += n
			nextOffset += n
		}

		if w.blob.handle == nil {
			return written, errBlobClosed
		}
		nWritten, err := w.blob.handle.PutSegments(w.ctx, batch)
		written += nWritten
		if err != nil {
			return written, err
		}
		if nWritten < batchBytes {
			return written, io.ErrShortWrite
		}
		offset = nextOffset
	}
	return written, nil
}

func (w *blobWriter) WriteString(s string) (int, error) {
	if w.closed {
		return 0, errBlobClosed
	}
	if w.blob.config.subType == fbproto.BLOB_SUB_TYPE_TEXT && w.blob.config.encoding != nil {
		data, err := w.blob.config.encoding.Encoder.Bytes([]byte(s))
		if err != nil {
			return 0, err
		}
		return w.Write(data)
	}
	return w.Write([]byte(s))
}

func (w *blobWriter) Close() error {
	if w.closed {
		return errBlobClosed
	}
	w.closed = true
	return w.blob.Close(w.ctx)
}

func (w *blobWriter) Cancel() error {
	if w.closed {
		return errBlobClosed
	}
	w.closed = true
	return w.blob.Cancel(w.ctx)
}

func blobSeekMode(whence int) (int, error) {
	switch whence {
	case io.SeekStart:
		return 0, nil
	case io.SeekCurrent:
		return 1, nil
	case io.SeekEnd:
		return 2, nil
	default:
		return 0, fmt.Errorf("invalid blob seek whence: %d", whence)
	}
}

func newOutputBlob(ctx context.Context, conn *Conn, tx *fbconn.FBTransaction, cfg blobConfig) (*blob, error) {
	if conn == nil || conn.fbConn == nil {
		return nil, errors.New("blob connection is nil")
	}
	if tx != nil && tx.Closed {
		return nil, fbconn.ErrTransactionClosed
	}
	handle, id, err := conn.fbConn.CreateBlob(ctx, tx, cfg.outputBPB)
	if err != nil {
		return nil, err
	}
	return &blob{
		conn:        conn,
		tx:          tx,
		id:          blobID(id),
		handle:      handle,
		output:      true,
		state:       blobStateOpen,
		config:      cfg,
		segmentSize: conn.config.BlobBufferSize,
	}, nil
}

func newInputBlob(conn *Conn, tx *fbconn.FBTransaction, cfg blobConfig, id blobID) *blob {
	return &blob{
		conn:        conn,
		tx:          tx,
		id:          id,
		output:      false,
		state:       blobStateNew,
		config:      cfg,
		segmentSize: conn.config.BlobBufferSize,
	}
}
