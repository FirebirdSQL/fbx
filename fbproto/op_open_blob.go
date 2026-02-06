package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-blobs-create

// OpOpenBlob opens an existing blob.
type OpOpenBlob struct {
	TransactionHandle int
	BlobId            int64
	Bpb               []byte
}

// Write encodes the operation into the wire protocol.
func (op *OpOpenBlob) Write(x *XDRWriter) error {
	var err error
	kind := Op_open_blob
	if len(op.Bpb) > 0 {
		kind = Op_open_blob2
	}
	// p_operation
	if err = x.WriteInt32(kind); err != nil {
		return err
	}
	if kind == Op_open_blob2 {
		// p_bpb
		if err = x.WriteBuffer(op.Bpb); err != nil {
			return err
		}
	}
	// p_blob_transaction
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}
	// p_blob_id
	return x.WriteInt64(op.BlobId)
}
