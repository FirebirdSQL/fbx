package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-blobs-close

// OpCloseBlob closes a blob handle.
type OpCloseBlob struct {
	BlobHandle int
}

// Write encodes the operation into the wire protocol.
func (op *OpCloseBlob) Write(x *XDRWriter) error {
	// p_operation
	if err := x.WriteInt32(Op_close_blob); err != nil {
		return err
	}
	// p_blob_handle
	return x.WriteInt32(op.BlobHandle)
}
