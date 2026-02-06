package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-blobs-cancel

// OpCancelBlob cancels a blob handle.
type OpCancelBlob struct {
	BlobHandle int
}

// Write encodes the operation into the wire protocol.
func (op *OpCancelBlob) Write(x *XDRWriter) error {
	// p_operation
	if err := x.WriteInt32(Op_cancel_blob); err != nil {
		return err
	}
	// p_blob_handle
	return x.WriteInt32(op.BlobHandle)
}
