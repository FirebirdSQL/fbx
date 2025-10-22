package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-statements-free

type OpFreeStatement struct {
	StatementHandle int
	Option          int // DSQL_close, DSQL_drop or DSQL_unprepare
}

func (op *OpFreeStatement) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_free_statement); err != nil {
		return err
	}

	//p_sqlfree_statement
	if err = x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	//p_sqlfree_option
	if err = x.WriteInt32(op.Option); err != nil {
		return err
	}
	return nil
}
