package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-connect-disconnect

type OpDisconnect struct {
}

func (op *OpDisconnect) Write(x *XDRWriter) error {
	var err error
	//p_operation
	if err = x.WriteInt32(Op_disconnect); err != nil {
		return err
	}

	return nil
}
