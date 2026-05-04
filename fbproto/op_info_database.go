package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-databases-information

type OpInfoDatabase struct {
	Parameters   []byte
	BufferLength int
}

func (op *OpInfoDatabase) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_info_database); err != nil {
		return err
	}

	// p_info_object (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	// p_info_incarnation (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	// p_info_items
	if err = x.WriteBuffer(op.Parameters); err != nil {
		return err
	}

	// p_info_object (unused)
	if err = x.WriteInt32(op.BufferLength); err != nil {
		return err
	}

	return nil
}
