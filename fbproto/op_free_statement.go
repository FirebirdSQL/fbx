package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-statements-free

type OpFreeStatement struct {
	StatementHandle int
}

func (op *OpFreeStatement) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_free_statement); err != nil {
		return err
	}

	//p_sqlfree_statement
	if err = x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	//p_sqlfree_option
	if err = x.WriteInt32(DSQL_drop); err != nil {
		return err
	}
	return nil
}
