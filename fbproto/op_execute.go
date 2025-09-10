package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-cont-auth

type OpExecute struct {
	StatementHandle       int
	TransactionHandle     int
	SQLDataBlr            []byte
	SQLData               [][]byte
	ParametersDescriptors []FieldDescriptor
}

func (op *OpExecute) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_execute); err != nil {
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

	if op.SQLData == nil {
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
		if err = WriteRowData(x, op.ParametersDescriptors, op.SQLData); err != nil {
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
