package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-transactions-rollbackretain

type OpRollbackRetaining struct {
	TransactionHandle int
}

func (op *OpRollbackRetaining) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_rollback_retaining); err != nil {
		return err
	}

	//p_rlse_object
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	return nil
}
