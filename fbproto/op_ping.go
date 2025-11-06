package fbproto

type OpPing struct {
}

func (op *OpPing) Write(x *XDRWriter) error {
	var err error

	//p_operation
	if err = x.WriteInt32(Op_ping); err != nil {
		return err
	}

	return nil
}
