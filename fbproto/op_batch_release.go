package fbproto

type OpBatchRelease struct {
	Kind            int
	StatementHandle int
}

func (op *OpBatchRelease) Write(x *XDRWriter) error {
	if op.Kind != Op_batch_cancel && op.Kind != Op_batch_rls {
		panic("wrong batch release operation")
	}
	if err := x.WriteInt32(op.Kind); err != nil {
		return err
	}
	return x.WriteInt32(op.StatementHandle)
}
