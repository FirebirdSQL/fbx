package fbproto

import "rdb.red-soft.ru/fbx/internal/fbio"

// BlobBPBConfig defines BPB settings for blob create/open operations.
type BlobBPBConfig struct {
	SubType   int32
	Stream    bool
	Temporary bool
	CharsetID int32
}

// BPBWriter builds a blob parameter buffer (BPB).
type BPBWriter struct {
	buf []byte
}

// NewBPBWriter creates a new BPBWriter.
func NewBPBWriter() *BPBWriter {
	return &BPBWriter{buf: make([]byte, 0, 16)}
}

// AddArgument appends a BPB argument with a 1-byte length and 4-byte value.
func (bpb *BPBWriter) AddArgument(tag byte, val int32) {
	if bpb == nil {
		return
	}
	if len(bpb.buf) == 0 {
		bpb.buf = append(bpb.buf, byte(Isc_bpb_version1))
	}
	bpb.buf = append(bpb.buf, tag, 4)
	bpb.buf = fbio.AppendInt32LE(bpb.buf, val)
}

// Bytes returns the encoded BPB.
func (bpb *BPBWriter) Bytes() []byte {
	if bpb == nil || len(bpb.buf) == 0 {
		return nil
	}
	return bpb.buf
}

// BuildOutputBPB builds a BPB for create blob operations.
func BuildOutputBPB(cfg BlobBPBConfig) []byte {
	bpb := NewBPBWriter()
	WriteOutputBPB(bpb, cfg)
	return bpb.Bytes()
}

// BuildInputBPB builds a BPB for open blob operations.
func BuildInputBPB(cfg BlobBPBConfig) []byte {
	bpb := NewBPBWriter()
	WriteInputBPB(bpb, cfg)
	return bpb.Bytes()
}

// WriteOutputBPB appends arguments for create blob operations.
func WriteOutputBPB(bpb *BPBWriter, cfg BlobBPBConfig) {
	if bpb == nil {
		return
	}
	blobType := Isc_bpb_type_segmented
	if cfg.Stream {
		blobType = Isc_bpb_type_stream
	}
	if cfg.Temporary {
		blobType |= Isc_bpb_storage_temp
	}
	bpb.AddArgument(Isc_bpb_type, int32(blobType))
	bpb.AddArgument(Isc_bpb_source_type, cfg.SubType)
	bpb.AddArgument(Isc_bpb_target_type, cfg.SubType)
	if cfg.SubType == BLOB_SUB_TYPE_TEXT {
		bpb.AddArgument(Isc_bpb_source_interp, cfg.CharsetID)
		bpb.AddArgument(Isc_bpb_target_interp, cfg.CharsetID)
	}
}

// WriteInputBPB appends arguments for open blob operations.
func WriteInputBPB(bpb *BPBWriter, cfg BlobBPBConfig) {
	if bpb == nil {
		return
	}
	blobType := Isc_bpb_type_segmented
	if cfg.Stream {
		blobType = Isc_bpb_type_stream
	}
	// NOTE: storage/temp only applies to create; don't set it for open.
	bpb.AddArgument(Isc_bpb_type, int32(blobType))
	bpb.AddArgument(Isc_bpb_source_type, cfg.SubType)
	bpb.AddArgument(Isc_bpb_target_type, cfg.SubType)
	if cfg.SubType == BLOB_SUB_TYPE_TEXT {
		bpb.AddArgument(Isc_bpb_source_interp, cfg.CharsetID)
		bpb.AddArgument(Isc_bpb_target_interp, cfg.CharsetID)
	}
}
