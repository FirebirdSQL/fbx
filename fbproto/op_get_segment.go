package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-blobs-getsegment

// OpGetSegment requests blob segment data.
type OpGetSegment struct {
	BlobHandle int
	Length     int
}

// Write encodes the operation into the wire protocol.
func (op *OpGetSegment) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_get_segment); err != nil {
		return err
	}
	// p_blob_handle
	if err = x.WriteInt32(op.BlobHandle); err != nil {
		return err
	}
	// p_seg_length
	if err = x.WriteInt32(op.Length); err != nil {
		return err
	}
	// p_blob_offset
	return x.WriteInt32(0)
}
