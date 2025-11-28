package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-statements-information

type OpInfoSQL struct {
	StatementHandle           int
	StatementInformationItems []byte
	MaxBufferLength           int
}

func (op *OpInfoSQL) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_info_sql); err != nil {
		return err
	}

	//p_info_object
	if err = x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	//p_info_incarnation
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	//p_info_items
	if err = x.WriteBuffer(op.StatementInformationItems); err != nil {
		return err
	}

	//p_info_buffer_length
	bufLength := 1024
	if op.MaxBufferLength != 0 {
		bufLength = op.MaxBufferLength
	}
	if err = x.WriteInt32(bufLength); err != nil {
		return err
	}

	return nil
}
