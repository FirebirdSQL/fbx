package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-blobs-seek

// OpSeekBlob seeks within a blob.
type OpSeekBlob struct {
	BlobHandle int
	SeekMode   int
	Offset     int
}

// Write encodes the operation into the wire protocol.
func (op *OpSeekBlob) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_seek_blob); err != nil {
		return err
	}
	// p_blob_handle
	if err = x.WriteInt32(op.BlobHandle); err != nil {
		return err
	}
	// p_seek_mode
	if err = x.WriteInt32(op.SeekMode); err != nil {
		return err
	}
	// p_offset
	return x.WriteInt32(op.Offset)
}
