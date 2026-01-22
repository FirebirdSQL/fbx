package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-accept-data

type OpCondAccept struct {
	Encoding     *Encoding
	Version      int
	Architecture int
	Type         int
	Data         []byte
	PluginName   string
	AuthComplete bool
	ServerKeys   []byte
	Compress     bool
}
type OpDataAccept struct {
	OpCondAccept
}

func (op *OpDataAccept) Read(r *XDRReader) error {
	return op.OpCondAccept.Read(r)
}

func (op *OpCondAccept) Read(r *XDRReader) error {
	var err error

	//p_acpt_version
	if op.Version, err = r.ReadInt32(); err != nil {
		return err
	}
	if op.Version < 0 {
		op.Version = op.Version & ProtocolMask
	}

	//p_acpt_architecture
	if op.Architecture, err = r.ReadInt32(); err != nil {
		return err
	}

	//p_acpt_type
	var acceptType int
	if acceptType, err = r.ReadInt32(); err != nil {
		return err
	}
	op.Type = acceptType & Ptype_MASK
	op.Compress = (acceptType & Pflag_compress) != 0

	//p_acpt_data
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
