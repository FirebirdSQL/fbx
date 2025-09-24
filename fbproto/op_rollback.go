package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-transactions-rollback

type OpRollback struct {
	TransactionHandle int
}

func (op *OpRollback) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_rollback); err != nil {
		return err
	}

	//p_rlse_object
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	return nil
}
