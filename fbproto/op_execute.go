package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-cont-auth

type OpExecute struct {
	Kind                  int
	StatementHandle       int
	TransactionHandle     int
	SQLDataBlr            []byte
	SQLDataOutBlr         []byte
	Parameters            [][]byte
	ParametersDescriptors []FieldDescriptor
}

func (op *OpExecute) Write(x *XDRWriter) error {
	var err error

	if op.Kind != Op_execute && op.Kind != Op_execute2 {
		panic("wrong execute operation")
	}

	// p_operation
	if err = x.WriteInt32(op.Kind); err != nil {
		return err
	}

	//p_sqldata_statement
	if err = x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	//p_sqldata_transaction
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	//p_sqldata_blr
	if err = x.WriteBuffer(op.SQLDataBlr); err != nil {
		return err
	}

	//p_sqldata_message_number (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	if len(op.Parameters) == 0 {
		//p_sqldata_messages
		if err = x.WriteInt32(0); err != nil {
			return err
		}
	} else {
		//p_sqldata_messages
		if err = x.WriteInt32(1); err != nil {
			return err
		}
		// Row data
		if err = WriteRowData(x, op.ParametersDescriptors, op.Parameters); err != nil {
			return err
		}
	}

	if op.Kind == Op_execute2 {
		//p_sqldata_out_blr
		if err = x.WriteBuffer(op.SQLDataOutBlr); err != nil {
			return err
		}
		//p_sqldata_out_message_number (unused)
		if err = x.WriteInt32(0); err != nil {
			return err
		}
	}

	//FIXME fbx protocol version dependent?
	//statement timeout
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	return nil
}
