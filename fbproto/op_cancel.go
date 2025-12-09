package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-databases-cancel

type OpCancel struct {
	Kind int
}

func (op *OpCancel) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_cancel); err != nil {
		return err
	}

	//p_co_kind
	if err = x.WriteInt32(op.Kind); err != nil {
		return err
	}

	return nil
}
