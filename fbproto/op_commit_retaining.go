package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-transactions-commitretain

type OpCommitRetaining struct {
	TransactionHandle int
}

func (op *OpCommitRetaining) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_commit_retaining); err != nil {
		return err
	}

	//p_rlse_object
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	return nil
}
