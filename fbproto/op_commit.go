package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-transactions-commit

type OpCommit struct {
	TransactionHandle int
}

func (op *OpCommit) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_commit); err != nil {
		return err
	}

	//p_rlse_object
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	return nil
}
