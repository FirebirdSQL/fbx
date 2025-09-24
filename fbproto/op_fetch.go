package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-statements-fetch

type OpFetch struct {
	StatementHandle int
	RowBlr          []byte
	Messages        int
}

func (op *OpFetch) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_fetch); err != nil {
		return err
	}

	//p_sqldata_statement
	if err = x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	//p_sqldata_blr
	if err = x.WriteBuffer(op.RowBlr); err != nil {
		return err
	}

	//p_sqldata_message_number (always 0)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	//p_sqldata_messages
	if err = x.WriteInt32(op.Messages); err != nil {
		return err
	}

	return nil
}
