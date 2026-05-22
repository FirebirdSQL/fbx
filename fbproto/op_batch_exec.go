package fbproto

type OpBatchExec struct {
	StatementHandle   int
	TransactionHandle int
}

func (op *OpBatchExec) Write(x *XDRWriter) error {
	if err := x.WriteInt32(Op_batch_exec); err != nil {
		return err
	}
	if err := x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}
	return x.WriteInt32(op.TransactionHandle)
}
