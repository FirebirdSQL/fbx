package fbproto

import (
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto/vax"
)

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-inline-blob

// OpInlineBlob is a server response with inline blob data.
// Data holds the decoded blob payload.
type OpInlineBlob struct {
	TransactionHandle int
	BlobId            int64
	Info              []byte
	Data              []byte
}

// Read decodes the inline blob using the default allocator.
func (op *OpInlineBlob) Read(x *XDRReader) error {
	return op.ReadWithAllocator(x, nil)
}

// ReadWithAllocator decodes the inline blob using a custom allocator.
func (op *OpInlineBlob) ReadWithAllocator(x *XDRReader, alloc func(int) []byte) error {
	var err error
	// p_blob_transaction
	if op.TransactionHandle, err = x.ReadInt32(); err != nil {
		return err
	}
	// p_blob_id
	if op.BlobId, err = x.ReadInt64(); err != nil {
		return err
	}
	// p_blob_info
	if op.Info, err = x.ReadBufferInto(&op.Info); err != nil {
		return err
	}
	// p_blob_data_length
	dataLen, err := x.ReadInt32()
	if err != nil {
		return err
	}
	if dataLen < 0 {
		return fmt.Errorf("invalid inline blob length: %d", dataLen)
	}
	if dataLen == 0 {
		op.Data = nil
		return nil
	}

	var raw []byte
	if alloc != nil {
		raw = alloc(dataLen)
		if cap(raw) < dataLen {
			raw = make([]byte, dataLen)
		} else {
			raw = raw[:dataLen]
		}
	} else {
		if cap(op.Data) < dataLen {
			raw = make([]byte, dataLen)
		} else {
			raw = op.Data[:dataLen]
		}
	}
	// p_blob_data
	if err := x.ReadBytesInto(raw); err != nil {
		return err
	}
	if err := x.SkipPadding(int64(dataLen)); err != nil {
		return err
	}
	op.Data, err = decodeInlineSegmentBuffer(raw)
	return err
}

func decodeInlineSegmentBuffer(buf []byte) ([]byte, error) {
	if len(buf) == 0 {
		return buf[:0], nil
	}
	outPos := 0
	for pos := 0; pos < len(buf); {
		if pos+2 > len(buf) {
			return nil, fmt.Errorf("invalid inline blob segment buffer length")
		}
		segLen := vax.Integer2(buf, pos)
		pos += 2
		if pos+segLen > len(buf) {
			return nil, fmt.Errorf("invalid inline blob segment length %d at %d", segLen, pos)
		}
		if segLen == 0 {
			continue
		}
		copy(buf[outPos:], buf[pos:pos+segLen])
		pos += segLen
		outPos += segLen
	}
	return buf[:outPos], nil
}
