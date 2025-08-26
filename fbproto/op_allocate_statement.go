package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-statements-allocate

type OpAllocateStatement struct {
}

func (op *OpAllocateStatement) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_allocate_statement); err != nil {
		return err
	}

	// p_rlse_object (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	return nil
}
