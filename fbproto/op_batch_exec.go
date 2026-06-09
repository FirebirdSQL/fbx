package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-batches-execute

type OpBatchExec struct {
	StatementHandle   int
	TransactionHandle int
}

func (op *OpBatchExec) Write(x *XDRWriter) error {
	// p_operation
	if err := x.WriteInt32(Op_batch_exec); err != nil {
		return err
	}

	// p_batch_statement
	if err := x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	// p_batch_transaction
	return x.WriteInt32(op.TransactionHandle)
}
