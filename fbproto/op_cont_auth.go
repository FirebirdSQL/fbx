package fbproto

import "github.com/red-soft/fbx/fbproto/auth"

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-cont-auth

type OpContAuth struct {
	AuthPlugin auth.AuthPlugin
	PluginList string
	Keys       []byte
}

func (op *OpContAuth) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_cont_auth); err != nil {
		return err
	}
	//
	//p.packString(hex.EncodeToString(authData))
	//p.packString(authPluginName)
	//p.packString(authPluginList)
	//p.packString(keys)

	return nil
}
