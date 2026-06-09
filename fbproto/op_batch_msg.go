package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-batches-msg

type OpBatchMsg struct {
	StatementHandle int
	RowCount        int
	Rows            [][]byte
}

func (op *OpBatchMsg) Write(x *XDRWriter) error {
	rowCount := op.RowCount
	if len(op.Rows) > 0 {
		rowCount = len(op.Rows)
	}

	// p_operation
	if err := x.WriteInt32(Op_batch_msg); err != nil {
		return err
	}

	// p_batch_statement
	if err := x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	// p_batch_messages
	if err := x.WriteInt32(rowCount); err != nil {
		return err
	}

	// p_batch_data
	for _, row := range op.Rows {
		if err := x.WriteBytes(row); err != nil {
			return err
		}
		if err := x.WritePadding((4 - len(row)) & 3); err != nil {
			return err
		}
	}
	return nil
}
