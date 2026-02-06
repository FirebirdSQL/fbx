package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-requests-info

// OpInfoBlob requests blob information.
type OpInfoBlob struct {
	BlobHandle   int
	Items        []byte
	BufferLength int
}

// Write encodes the operation into the wire protocol.
func (op *OpInfoBlob) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_info_blob); err != nil {
		return err
	}
	// p_info_object
	if err = x.WriteInt32(op.BlobHandle); err != nil {
		return err
	}
	// p_info_incarnation
	if err = x.WriteInt32(0); err != nil {
		return err
	}
	// p_info_items
	if err = x.WriteBuffer(op.Items); err != nil {
		return err
	}
	// p_info_buffer_length
	return x.WriteInt32(op.BufferLength)
}
