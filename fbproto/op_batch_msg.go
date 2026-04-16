package fbproto

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

	if err := x.WriteInt32(Op_batch_msg); err != nil {
		return err
	}
	if err := x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}
	if err := x.WriteInt32(rowCount); err != nil {
		return err
	}

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
