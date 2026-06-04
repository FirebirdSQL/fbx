package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-databases-drop

type OpDropDatabase struct {
}

func (op *OpDropDatabase) Write(x *XDRWriter) error {
	var err error
	//p_operation
	if err = x.WriteInt32(Op_drop_database); err != nil {
		return err
	}

	//p_rlse_object (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	return nil
}
