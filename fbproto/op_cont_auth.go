package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-cont-auth

type OpContAuth struct {
	// Encoding used for XDR strings in this packet
	Encoding *Encoding
	// p_data: authentication data (spec: Buffer)
	AuthData []byte
	// p_name: name of the current authentication plugin
	PluginName string
	// p_list: remaining plugins (client->server first auth), otherwise empty
	PluginList string
	// p_keys: known server keys / plugin-specific data
	Keys []byte
}

func (op *OpContAuth) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_cont_auth); err != nil {
		return err
	}
	// p_data (Buffer): send raw bytes
	if err := x.WriteBuffer(op.AuthData); err != nil {
		return err
	}
	// p_name
	if err := x.WriteString(op.PluginName, op.Encoding); err != nil {
		return err
	}
	// p_list
	if err := x.WriteString(op.PluginList, op.Encoding); err != nil {
		return err
	}
	// p_keys (Buffer)
	return x.WriteBuffer(op.Keys)
}

// Read parses server -> client op_cont_auth response.
// Layout:
//
//	Int32 p_operation (op_cont_auth = 92)
//	Buffer p_data, String p_name, String p_list, Buffer p_keys
func (op *OpContAuth) Read(r *XDRReader) error {
	if op.Encoding == nil {
		// fallback: default encoding is UTF8 in our stack
		enc, _ := NewEncoding("UTF8")
		op.Encoding = enc
	}
	// p_data (Buffer)
	buf, err := r.ReadBuffer()
	if err != nil {
		return err
	}
	op.AuthData = buf
	// p_name
	if op.PluginName, err = r.ReadString(op.Encoding); err != nil {
		return err
	}
	// p_list
	if op.PluginList, err = r.ReadString(op.Encoding); err != nil {
		return err
	}
	// p_keys (Buffer)
	if keys, err := r.ReadBuffer(); err == nil {
		op.Keys = keys
	} else {
		return err
	}
	return nil
}
