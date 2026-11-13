package fbproto

// OpInfoBatch requests information about the batch owned by a statement.
type OpInfoBatch struct {
	StatementHandle int
	Items           []byte
	BufferLength    int
}

func (op *OpInfoBatch) Write(x *XDRWriter) error {
	if err := x.WriteInt32(Op_info_batch); err != nil {
		return err
	}
	if err := x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}
	if err := x.WriteInt32(0); err != nil { // p_info_incarnation
		return err
	}
	if err := x.WriteBuffer(op.Items); err != nil {
		return err
	}
	return x.WriteInt32(op.BufferLength)
}
