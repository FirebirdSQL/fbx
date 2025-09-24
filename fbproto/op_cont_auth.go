package fbproto

import "encoding/hex"

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-cont-auth

type OpContAuth struct {
	Encoding   *Encoding
	AuthData   []byte
	PluginName string
	PluginList string
	Keys       []byte
}

func (op *OpContAuth) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(Op_cont_auth); err != nil {
		return err
	}
	if err := x.WriteString(hex.EncodeToString(op.AuthData), op.Encoding); err != nil {
		return err
	}
	if err := x.WriteString(op.PluginName, op.Encoding); err != nil {
		return err
	}
	if err := x.WriteString(op.PluginList, op.Encoding); err != nil {
		return err
	}
	return x.WriteString(string(op.Keys), op.Encoding)
}

// Read parses server -> client op_cont_auth response.
// Layout: XDR string hex(authData), XDR string pluginName, XDR string pluginList, XDR string keys
func (op *OpContAuth) Read(r *XDRReader) error {
	if op.Encoding == nil {
		// fallback: default encoding is UTF8 in our stack
		enc, _ := NewEncoding("UTF8")
		op.Encoding = enc
	}
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
	if op.PluginName, err = r.ReadString(op.Encoding); err != nil {
		return err
	}
	if op.PluginList, err = r.ReadString(op.Encoding); err != nil {
		return err
	}
	if keys, err := r.ReadString(op.Encoding); err == nil {
		op.Keys = []byte(keys)
	} else {
		return err
	}
	return nil
}
