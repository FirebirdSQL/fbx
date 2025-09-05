package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-cont-auth

type OpExecute struct {
	StatementHandle   int
	TransactionHandle int
	SQLData           []byte
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
	if err = x.WriteBuffer(op.SQLData); err != nil {
		return err
	}

	//p_sqldata_message_number (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	parameters := 0
	//p_sqldata_messages
	if err = x.WriteInt32(parameters); err != nil {
		return err
	}

	/*
		if parameters != 0 {
			// Row data
			if err = x.WriteBuffer(values); err != nil {
				return err
			}
		}
	*/

	//FIXME fbx protocol version dependent?
	//statement timeout
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	return nil
}
