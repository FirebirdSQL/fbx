package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-accept-data

type OpCondAccept struct {
	AcceptMetadata
	Encoding	 *Encoding
	Data		 []byte
	PluginName	 string
	AuthComplete bool
	ServerKeys	 []byte
}

// OpDataAccept represents an op_accept_data response from the server.
type OpDataAccept struct {
	OpCondAccept
}

// Read decodes an op_accept_data response after its operation code has been read.
func (op *OpDataAccept) Read(r *XDRReader) error {
	return op.OpCondAccept.Read(r)
}

// Read decodes an op_cond_accept response after its operation code has been read.
func (op *OpCondAccept) Read(r *XDRReader) error {
	if err := op.read(r); err != nil {
		return err
	}

	//p_acpt_data
	var err error
	if op.Data, err = r.ReadBufferInto(&op.Data); err != nil {
		return err
	}

	//p_acpt_plugin
	if op.PluginName, err = r.ReadString(op.Encoding); err != nil {
		return err
	}

	//p_acpt_authenticated
	var authComplete int
	if authComplete, err = r.ReadInt32(); err != nil {
		return err
	}
	op.AuthComplete = authComplete == 1

	//p_acpt_keys
	if op.ServerKeys, err = r.ReadBufferInto(&op.ServerKeys); err != nil {
		return err
	}
	return nil
}
