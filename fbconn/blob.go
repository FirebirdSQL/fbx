package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/vax"
)

const blobSegmentSizeMax = 65535
const blobStateEndOfBlob = 2
const putSegmentBatchSize = 8

// FBBlob provides low-level blob operations for an opened blob handle.
type FBBlob interface {
	GetSegmentInto(ctx context.Context, dst []byte) (int, bool, error)
	PutSegments(ctx context.Context, segments [][]byte) (int, error)
	PutSegment(ctx context.Context, segment []byte) error
	Seek(ctx context.Context, offset int, seekMode int) (int64, error)
	Close(ctx context.Context) error
	Cancel(ctx context.Context) error
	Info(ctx context.Context, items []byte, bufferLength int) ([]byte, error)
	TotalLength(ctx context.Context) (int64, error)
}

type inlineBlob struct {
	conn *FbConn
	data []byte
	pos  int
}

type remoteBlob struct {
	conn   *FbConn
	handle int
	locked bool
}

// OpenBlob opens an existing blob and returns its handle. When tx is nil, the internal transaction is used.
// If locked is true, the connection is assumed to be already locked by the caller and no lock is taken.
func (fbConn *FbConn) OpenBlob(ctx context.Context, tx *FBTransaction, blobId int64, bpb []byte, locked bool) (FBBlob, error) {
	if fbConn == nil {
		return nil, errors.New("blob connection is nil")
	}
	if tx == nil {
		var err error
		tx, err = fbConn.ensureInternalTransaction(ctx, nil, locked)
		if err != nil {
			return nil, err
		}
	}
	if data, ok := fbConn.consumeInlineBlob(tx, blobId); ok {
		return &inlineBlob{conn: fbConn, data: data}, nil
	}
	handle, _, err := fbConn.openOrCreateBlob(ctx, tx, &fbproto.OpOpenBlob{
		TransactionHandle: tx.Handle,
		BlobId:            blobId,
		Bpb:               bpb,
	}, locked)
	if err != nil {
		fbConn.rollbackInternalTxIfAutocommitOnError(ctx, tx)
		return nil, err
	}
	return &remoteBlob{conn: fbConn, handle: handle, locked: locked}, nil
}

// CreateBlob creates a new blob and returns its handle and blob id. When tx is nil, the internal transaction is used.
func (fbConn *FbConn) CreateBlob(ctx context.Context, tx *FBTransaction, bpb []byte) (FBBlob, int64, error) {
	if fbConn == nil {
		return nil, 0, errors.New("blob connection is nil")
	}
	if tx == nil {
		var err error
		tx, err = fbConn.ensureInternalTransaction(ctx, nil, false)
		if err != nil {
			return nil, 0, err
		}
	}
	handle, id, err := fbConn.openOrCreateBlob(ctx, tx, &fbproto.OpCreateBlob{
		TransactionHandle: tx.Handle,
		Bpb:               bpb,
	}, false)
	if err != nil {
		fbConn.rollbackInternalTxIfAutocommitOnError(ctx, tx)
		return nil, 0, err
	}
	return &remoteBlob{conn: fbConn, handle: handle}, id, nil
}

func (fbConn *FbConn) openOrCreateBlob(ctx context.Context, tx *FBTransaction, msg fbproto.ClientMessage, locked bool) (int, int64, error) {
	if tx == nil || tx.Closed {
		return 0, 0, ErrTransactionClosed
	}

	if !locked {
		if err := fbConn.lock(); err != nil {
			return 0, 0, err
		}
		defer fbConn.unlock()

		if err := fbConn.beforeRequest(); err != nil {
			return 0, 0, err
		}

		if ctx != context.Background() {
			select {
			case <-ctx.Done():
				return 0, 0, newContextAlreadyDoneError(ctx)
			default:
			}
			fbConn.contextWatcher.Watch(ctx)
			defer fbConn.contextWatcher.Unwatch()
		}
	} else if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return 0, 0, newContextAlreadyDoneError(ctx)
		default:
		}
	}

	if err := fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(msg)
	}); err != nil {
		fbConn.asyncClose()
		return 0, 0, err
	}

	resp, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return 0, 0, err
	}
	switch m := resp.(type) {
	case *fbproto.OpResponse:
		return m.Object, m.BlobId, nil
	case *fbproto.ErrorResponse:
		return 0, 0, ErrorResponseToFbError(m)
	default:
		return 0, 0, fmt.Errorf("unexpected response for blob operation: %T", resp)
	}
}

// GetSegmentInto reads blob segment data into dst without allocating an intermediate buffer.
//
// The server returns one or more segments in a single response, prefixed with their lengths. This method requests a
// segment buffer sized so that the total returned payload cannot exceed len(dst).
func (b *inlineBlob) GetSegmentInto(_ context.Context, dst []byte) (int, bool, error) {
	if len(dst) == 0 {
		return 0, b.pos >= len(b.data), nil
	}
	if b.pos >= len(b.data) {
		return 0, true, nil
	}
	n := copy(dst, b.data[b.pos:])
	b.pos += n
	return n, b.pos >= len(b.data), nil
}

func (b *remoteBlob) GetSegmentInto(ctx context.Context, dst []byte) (int, bool, error) {
	if len(dst) == 0 {
		return 0, false, nil
	}

	// Request dst + header for a single segment; if multiple segments are returned, additional headers reduce payload.
	reqLen := len(dst) + 2
	if reqLen > blobSegmentSizeMax {
		reqLen = blobSegmentSizeMax
	}

	unlock, err := b.lockConn(ctx)
	if err != nil {
		return 0, false, err
	}
	defer unlock()

	if err := b.conn.withWriteLockAndFlush(func() error {
		return b.conn.client.WriteMessage(&fbproto.OpGetSegment{
			BlobHandle: b.handle,
			Length:     reqLen,
		})
	}); err != nil {
		b.conn.asyncClose()
		return 0, false, err
	}

	copied := 0
	decodeSegments := func(data []byte) error {
		remaining := len(data)
		var hdr [2]byte
		for remaining > 0 {
			if remaining < 2 {
				return fmt.Errorf("invalid blob segment buffer length")
			}
			copy(hdr[:], data[:2])
			data = data[2:]
			remaining -= 2
			segLen := vax.Integer2(hdr[:], 0)
			if segLen < 0 || segLen > remaining {
				return fmt.Errorf("invalid blob segment length %d", segLen)
			}

			need := len(dst) - copied
			if need > 0 {
				toCopy := segLen
				if toCopy > need {
					toCopy = need
				}
				if toCopy > 0 {
					copy(dst[copied:copied+toCopy], data[:toCopy])
					copied += toCopy
				}
			}
			data = data[segLen:]
			remaining -= segLen
		}
		return nil
	}

	msg, err := b.conn.receiveMessage()
	if err != nil {
		return 0, false, normalizeTimeoutError(ctx, err)
	}
	resp, ok := msg.(*fbproto.OpResponse)
	if !ok {
		if errResp, ok := msg.(*fbproto.ErrorResponse); ok {
			return 0, false, ErrorResponseToFbError(errResp)
		}
		return 0, false, fmt.Errorf("unexpected response for blob get segment: %T", msg)
	}
	if err := decodeSegments(resp.Data); err != nil {
		return 0, false, err
	}
	return copied, resp.Object == blobStateEndOfBlob, nil
}

// PutSegments writes blob segments in batches without waiting for a response after each segment.
// It returns the number of bytes successfully written before an error occurred.
func (b *inlineBlob) PutSegments(_ context.Context, segments [][]byte) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}
	return 0, errors.New("invalid blob handle")
}

func (b *remoteBlob) PutSegments(ctx context.Context, segments [][]byte) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}
	for _, segment := range segments {
		if len(segment) == 0 {
			return 0, fmt.Errorf("empty blob segment")
		}
		if len(segment) > blobSegmentSizeMax {
			return 0, fmt.Errorf("blob segment too large: %d", len(segment))
		}
	}

	unlock, err := b.lockConn(ctx)
	if err != nil {
		return 0, err
	}
	defer unlock()

	written := 0
	for batchStart := 0; batchStart < len(segments); {
		batchEnd := batchStart + putSegmentBatchSize
		if batchEnd > len(segments) {
			batchEnd = len(segments)
		}
		batch := segments[batchStart:batchEnd]

		if err := b.conn.withWriteLockAndFlush(func() error {
			for _, segment := range batch {
				if err := b.conn.client.WriteMessage(&fbproto.OpPutSegment{
					BlobHandle: b.handle,
					Segment:    segment,
				}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			b.conn.asyncClose()
			return written, err
		}

		batchWritten, err := b.consumePutSegmentResponses(ctx, batch)
		written += batchWritten
		if err != nil {
			return written, err
		}
		batchStart = batchEnd
	}

	return written, nil
}

func (b *remoteBlob) consumePutSegmentResponses(ctx context.Context, segments [][]byte) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}
	written := 0
	var firstErr error
	counting := true

	for _, segment := range segments {
		resp, err := b.conn.receiveMessage()
		if err != nil {
			b.conn.asyncClose()
			return written, normalizeTimeoutError(ctx, err)
		}
		switch m := resp.(type) {
		case *fbproto.OpResponse:
			if counting {
				written += len(segment)
			}
		case *fbproto.ErrorResponse:
			if firstErr == nil {
				firstErr = ErrorResponseToFbError(m)
				counting = false
			}
		default:
			if firstErr == nil {
				firstErr = fmt.Errorf("unexpected response for blob put segment: %T", resp)
				counting = false
			}
		}
	}

	return written, firstErr
}

// PutSegment writes a blob segment.
func (b *inlineBlob) PutSegment(_ context.Context, _ []byte) error {
	return errors.New("invalid blob handle")
}

func (b *remoteBlob) PutSegment(ctx context.Context, segment []byte) error {
	if len(segment) == 0 {
		return fmt.Errorf("empty blob segment")
	}
	if len(segment) > blobSegmentSizeMax {
		return fmt.Errorf("blob segment too large: %d", len(segment))
	}

	unlock, err := b.lockConn(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	if err := b.conn.withWriteLockAndFlush(func() error {
		return b.conn.client.WriteMessage(&fbproto.OpPutSegment{
			BlobHandle: b.handle,
			Segment:    segment,
		})
	}); err != nil {
		b.conn.asyncClose()
		return err
	}

	resp, err := b.conn.receiveMessage()
	if err != nil {
		b.conn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}
	switch m := resp.(type) {
	case *fbproto.OpResponse:
		return nil
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		return fmt.Errorf("unexpected response for blob put segment: %T", resp)
	}
}

// Seek seeks to the given offset. Returns the resulting position.
func (b *inlineBlob) Seek(_ context.Context, offset int, seekMode int) (int64, error) {
	var base int
	switch seekMode {
	case 0:
		base = 0
	case 1:
		base = b.pos
	case 2:
		base = len(b.data)
	default:
		return 0, fmt.Errorf("invalid blob seek mode: %d", seekMode)
	}
	pos := int64(base) + int64(offset)
	if pos < 0 {
		return 0, fmt.Errorf("blob seek offset out of range: %d", offset)
	}
	if pos > int64(len(b.data)) {
		pos = int64(len(b.data))
	}
	b.pos = int(pos)
	return pos, nil
}

func (b *remoteBlob) Seek(ctx context.Context, offset int, seekMode int) (int64, error) {
	unlock, err := b.lockConn(ctx)
	if err != nil {
		return 0, err
	}
	defer unlock()

	if err := b.conn.withWriteLockAndFlush(func() error {
		return b.conn.client.WriteMessage(&fbproto.OpSeekBlob{
			BlobHandle: b.handle,
			SeekMode:   seekMode,
			Offset:     offset,
		})
	}); err != nil {
		b.conn.asyncClose()
		return 0, err
	}

	resp, err := b.conn.receiveMessage()
	if err != nil {
		b.conn.asyncClose()
		return 0, normalizeTimeoutError(ctx, err)
	}
	switch m := resp.(type) {
	case *fbproto.OpResponse:
		return int64(m.Object), nil
	case *fbproto.ErrorResponse:
		return 0, ErrorResponseToFbError(m)
	default:
		return 0, fmt.Errorf("unexpected response for blob seek: %T", resp)
	}
}

// Close closes a blob handle.
func (b *inlineBlob) Close(_ context.Context) error {
	if b.conn != nil && b.data != nil {
		b.conn.releaseInlineBlobData(b.data)
	}
	b.data = nil
	b.pos = 0
	return nil
}

func (b *remoteBlob) Close(ctx context.Context) error {
	return b.release(ctx, &fbproto.OpCloseBlob{BlobHandle: b.handle})
}

// Cancel cancels a blob handle.
func (b *inlineBlob) Cancel(_ context.Context) error {
	if b.conn != nil && b.data != nil {
		b.conn.releaseInlineBlobData(b.data)
	}
	b.data = nil
	b.pos = 0
	return nil
}

func (b *remoteBlob) Cancel(ctx context.Context) error {
	return b.release(ctx, &fbproto.OpCancelBlob{BlobHandle: b.handle})
}

func (b *remoteBlob) release(ctx context.Context, msg fbproto.ClientMessage) error {
	unlock, err := b.lockConn(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	if err := b.conn.withWriteLockAndFlush(func() error {
		return b.conn.client.WriteMessage(msg)
	}); err != nil {
		b.conn.asyncClose()
		return err
	}

	resp, err := b.conn.receiveMessage()
	if err != nil {
		b.conn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}
	switch m := resp.(type) {
	case *fbproto.OpResponse:
		return nil
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		return fmt.Errorf("unexpected response for blob release: %T", resp)
	}
}

// Info retrieves blob information using op_info_blob.
func (b *inlineBlob) Info(_ context.Context, items []byte, bufferLength int) ([]byte, error) {
	return inlineInfo(items, len(b.data), bufferLength), nil
}

func (b *inlineBlob) TotalLength(ctx context.Context) (int64, error) {
	return blobTotalLength(ctx, b)
}

func (b *remoteBlob) Info(ctx context.Context, items []byte, bufferLength int) ([]byte, error) {
	if bufferLength <= 0 {
		bufferLength = 512
	}

	unlock, err := b.lockConn(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	var data []byte
	msgBuilder := func() fbproto.ClientMessage {
		return &fbproto.OpInfoBlob{
			BlobHandle:   b.handle,
			Items:        items,
			BufferLength: bufferLength,
		}
	}
	truncationHandler := func(InfoTruncationError) {}
	err = b.conn.getServerInfo(msgBuilder, truncationHandler, func(buf []byte) error {
		data = buf
		return nil
	})
	if err != nil {
		return nil, normalizeTimeoutError(ctx, err)
	}
	return data, nil
}

func (b *remoteBlob) TotalLength(ctx context.Context) (int64, error) {
	return blobTotalLength(ctx, b)
}

func blobTotalLength(ctx context.Context, blob FBBlob) (int64, error) {
	if blob == nil {
		return 0, errors.New("blob is nil")
	}
	items := []byte{fbproto.Isc_info_blob_total_length, fbproto.Isc_info_end}
	data, err := blob.Info(ctx, items, 32)
	if err != nil {
		return 0, err
	}
	reader := NewServerInfoBufferReader(data)
	item, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	if item != fbproto.Isc_info_blob_total_length {
		return 0, fmt.Errorf("unexpected blob info item: %d", item)
	}
	return reader.ReadInt64Value()
}

// EncodeBlobID encodes the blob id into wire format (big-endian 8 bytes).
func EncodeBlobID(id int64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(id))
	return buf[:]
}

// DecodeBlobID decodes the blob id from wire format (big-endian 8 bytes).
func DecodeBlobID(buf []byte) (int64, error) {
	if len(buf) != 8 {
		return 0, fmt.Errorf("invalid blob id length: %d", len(buf))
	}
	return int64(binary.BigEndian.Uint64(buf)), nil
}

func (b *remoteBlob) lockConn(ctx context.Context) (func(), error) {
	if b.conn == nil {
		return nil, errors.New("blob connection is nil")
	}
	if b.locked && b.conn.IsBusy() {
		if ctx != context.Background() {
			select {
			case <-ctx.Done():
				return nil, newContextAlreadyDoneError(ctx)
			default:
			}
		}
		return func() {}, nil
	}

	if err := b.conn.lock(); err != nil {
		return nil, err
	}

	if err := b.conn.beforeRequest(); err != nil {
		b.conn.unlock()
		return nil, err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			b.conn.unlock()
			return nil, newContextAlreadyDoneError(ctx)
		default:
		}
		b.conn.contextWatcher.Watch(ctx)
		return func() {
			b.conn.contextWatcher.Unwatch()
			b.conn.unlock()
		}, nil
	}

	return func() { b.conn.unlock() }, nil
}

func decodeSegmentBuffer(buf []byte) ([]byte, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	total := 0
	for pos := 0; pos < len(buf); {
		if pos+2 > len(buf) {
			return nil, fmt.Errorf("invalid blob segment buffer length")
		}
		segLen := vax.Integer2(buf, pos)
		pos += 2
		if pos+segLen > len(buf) {
			return nil, fmt.Errorf("invalid blob segment length %d at %d", segLen, pos)
		}
		total += segLen
		pos += segLen
	}

	out := make([]byte, total)
	outPos := 0
	for pos := 0; pos < len(buf); {
		segLen := vax.Integer2(buf, pos)
		pos += 2
		copy(out[outPos:], buf[pos:pos+segLen])
		pos += segLen
		outPos += segLen
	}
	return out, nil
}

func copySegmentBufferInto(dst []byte, segmentBuffer []byte) (int, error) {
	if len(dst) == 0 || len(segmentBuffer) == 0 {
		return 0, nil
	}
	outPos := 0
	for pos := 0; pos < len(segmentBuffer) && outPos < len(dst); {
		if pos+2 > len(segmentBuffer) {
			return outPos, fmt.Errorf("invalid blob segment buffer length")
		}
		segLen := vax.Integer2(segmentBuffer, pos)
		pos += 2
		if pos+segLen > len(segmentBuffer) {
			return outPos, fmt.Errorf("invalid blob segment length %d at %d", segLen, pos)
		}
		if segLen == 0 {
			continue
		}
		n := segLen
		if n > len(dst)-outPos {
			n = len(dst) - outPos
		}
		copy(dst[outPos:], segmentBuffer[pos:pos+n])
		outPos += n
		pos += segLen
	}
	return outPos, nil
}

func inlineInfo(items []byte, totalLength int, bufferLength int) []byte {
	if len(items) == 0 {
		return []byte{fbproto.Isc_info_end}
	}
	var buf bytes.Buffer
	for _, item := range items {
		if item == fbproto.Isc_info_end {
			buf.WriteByte(item)
			break
		}
		switch item {
		case fbproto.Isc_info_blob_total_length:
			buf.WriteByte(item)
			writeInfoInt64(&buf, int64(totalLength))
		}
	}
	if buf.Len() == 0 || buf.Bytes()[buf.Len()-1] != fbproto.Isc_info_end {
		buf.WriteByte(fbproto.Isc_info_end)
	}
	data := buf.Bytes()
	if bufferLength > 0 && len(data) > bufferLength {
		return data[:bufferLength]
	}
	return data
}

func writeInfoInt64(buf *bytes.Buffer, value int64) {
	var tmp [8]byte
	for i := 0; i < 8; i++ {
		tmp[i] = byte(value >> (8 * i))
	}
	length := uint16(len(tmp))
	buf.WriteByte(byte(length))
	buf.WriteByte(byte(length >> 8))
	buf.Write(tmp[:])
}
