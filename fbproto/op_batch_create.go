package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-batches-create

type OpBatchCreate struct {
	StatementHandle int
	BatchBLR        []byte
	MessageLength   int
	ParameterBuffer []byte
}

func (op *OpBatchCreate) Write(x *XDRWriter) error {
	// p_operation
	if err := x.WriteInt32(Op_batch_create); err != nil {
		return err
	}

	// p_batch_statement
	if err := x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	// p_batch_blr
	if err := x.WriteBuffer(op.BatchBLR); err != nil {
		return err
	}

	// p_batch_msglen
	if err := x.WriteInt32(op.MessageLength); err != nil {
		return err
	}

	// p_batch_pb
	return x.WriteBuffer(op.ParameterBuffer)
}
