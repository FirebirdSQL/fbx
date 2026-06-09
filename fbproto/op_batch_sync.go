package fbproto

type OpBatchSync struct {
}

func (op *OpBatchSync) Write(x *XDRWriter) error {
	// p_operation
	return x.WriteInt32(Op_batch_sync)
}
