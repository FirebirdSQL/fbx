package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-connect-attach

type OpAttach struct {
	File string
	DPB  []byte
}

func (op *OpAttach) Write(x *XDRWriter) error {
	var err error
	//p_operation
	if err = x.WriteInt32(Op_attach); err != nil {
		return err
	}

	//p_atch_database (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	var enc *Encoding
	if enc, err = NewEncoding("UTF8"); err != nil {
		return err
	}

	//p_atch_file
	if err = x.WriteString(op.File, enc); err != nil {
		return err
	}

	//p_atch_dpb
	if err = x.WriteBuffer(op.DPB); err != nil {
		return err
	}

	return nil
}
