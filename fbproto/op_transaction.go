package fbproto

// http://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-transactions-start

type OpTransaction struct {
	TPB []byte
}

func (op *OpTransaction) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_transaction); err != nil {
		return err
	}

	// p_sttr_database (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	// p_sttr_tpb
	if err = x.WriteBuffer(op.TPB); err != nil {
		return err
	}
	return nil
}
