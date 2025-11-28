package fbproto

type OpExecuteImmediate struct {
	Encoding          *Encoding
	Query             string
	TransactionHandle int
	Dialect           int
}

func (op *OpExecuteImmediate) Write(x *XDRWriter) error {
	var err error

	// p_operation
	if err = x.WriteInt32(Op_exec_immediate); err != nil {
		return err
	}

	//p_sqlst_transaction
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	//p_sqlst_statement
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	// p_sqlst_SQL_dialect
	if err = x.WriteInt32(op.Dialect); err != nil {
		return err
	}

	// p_sqlst_SQL_str
	if err = x.WriteString(op.Query, op.Encoding); err != nil {
		return err
	}

	// p_sqlst_items
	if err = x.WriteBuffer(nil); err != nil {
		return err
	}

	// p_sqlst_buffer_length
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	return nil
}
