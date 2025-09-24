package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-statements-prepare

type OpPrepareStatement struct {
	Encoding                       *Encoding
	TransactionHandle              int
	StatementHandle                int
	Dialect                        int
	Query                          string
	StatementInformationItems      []byte
	StatementInformationBufferSize int
}

func (op *OpPrepareStatement) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_prepare_statement); err != nil {
		return err
	}

	// p_sqlst_transaction
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	// p_sqlst_statement
	if err = x.WriteInt32(op.StatementHandle); err != nil {
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
	if err = x.WriteBuffer(op.StatementInformationItems); err != nil {
		return err
	}

	buf := 1024
	if op.StatementInformationBufferSize > 0 {
		buf = op.StatementInformationBufferSize
	}

	// p_sqlst_buffer_length
	if err = x.WriteInt32(buf); err != nil {
		return err
	}

	return nil
}
