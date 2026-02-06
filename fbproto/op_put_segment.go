package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-blobs-putsegment

// OpPutSegment writes a blob segment.
type OpPutSegment struct {
	BlobHandle int
	Segment    []byte
}

// Write encodes the operation into the wire protocol.
func (op *OpPutSegment) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_put_segment); err != nil {
		return err
	}
	// p_blob_handle
	if err = x.WriteInt32(op.BlobHandle); err != nil {
		return err
	}
	// p_seg_length
	if err = x.WriteInt32(len(op.Segment)); err != nil {
		return err
	}
	// p_seg_data
	return x.WriteBuffer(op.Segment)
}
