package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-cont-auth

import "encoding/hex"

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
	// p_data (spec: Buffer). This implementation sends hex(AuthData) as XDR string.
	if err := x.WriteString(hex.EncodeToString(op.AuthData), op.Encoding); err != nil {
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
	// p_keys
	return x.WriteString(string(op.Keys), op.Encoding)
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
	// p_data
	dataHex, err := r.ReadString(op.Encoding)
	if err != nil {
		return err
	}
	if dataHex != "" {
		if bs, err := hex.DecodeString(dataHex); err == nil {
			op.AuthData = bs
		} else {
			// if not hex, keep raw bytes
			op.AuthData = []byte(dataHex)
		}
	}
	// p_name
	if op.PluginName, err = r.ReadString(op.Encoding); err != nil {
		return err
	}
	// p_list
	if op.PluginList, err = r.ReadString(op.Encoding); err != nil {
		return err
	}
	// p_keys
	if keys, err := r.ReadString(op.Encoding); err == nil {
		op.Keys = []byte(keys)
	} else {
		return err
	}
	return nil
}
