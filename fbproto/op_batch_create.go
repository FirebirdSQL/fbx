package fbproto

type OpBatchCreate struct {
	StatementHandle int
	BatchBLR        []byte
	MessageLength   int
	ParameterBuffer []byte
}

func (op *OpBatchCreate) Write(x *XDRWriter) error {
	if err := x.WriteInt32(Op_batch_create); err != nil {
		return err
	}
	if err := x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}
	if err := x.WriteBuffer(op.BatchBLR); err != nil {
		return err
	}
	if err := x.WriteInt32(op.MessageLength); err != nil {
		return err
	}
	return x.WriteBuffer(op.ParameterBuffer)
}
