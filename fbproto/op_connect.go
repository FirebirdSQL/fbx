package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-connect

import (
	"bytes"
	"strings"

	"github.com/red-soft/fbx/fbproto/auth"
)

type OpConnect struct {
	Database         string
	User             string
	SystemUser       string
	Hostname         string
	Password         string
	AuthPluginName   string
	AuthPluginsList  []string
	AuthSpecificData []auth.SpecificDataChunk
	Options          map[string]string
	WireCrypt        bool
	Charset          string
	Encoding         *Encoding
	Compress         bool
}

func (op *OpConnect) Write(x *XDRWriter) error {
	var err error

	// p_operation
	if err = x.WriteInt32(op_connect); err != nil {
		return err
	}
	// p_cnct_operation (unused)
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	// p_cnct_cversion (3.0+)
	if err = x.WriteInt32(ConnectVersion3); err != nil {
		return err
	}

	//p_cnct_client
	if err = x.WriteInt32(GenericAchitectureClient); err != nil {
		return err
	}

	//p_cnct_file
	if err = x.WriteString(op.Database, op.Encoding); err != nil {
		return err
	}

	//p_cnct_count
	if err = x.WriteInt32(len(SupportedProtocols)); err != nil {
		return err
	}

	sysUserBytes := bytes.NewBufferString(op.SystemUser).Bytes()
	hostnameBytes := bytes.NewBufferString(op.Hostname).Bytes()
	pluginList := strings.Join(op.AuthPluginsList, ",")
	pluginListNameBytes := bytes.NewBufferString(pluginList).Bytes()
	pluginNameBytes := bytes.NewBufferString(op.AuthPluginName).Bytes()
	userBytes := bytes.NewBufferString(strings.ToUpper(op.User)).Bytes()
	var wireCryptByte byte
	if op.WireCrypt {
		wireCryptByte = 1
	}

	//var specific_data []byte
	//if op.AuthPlugin == "Srp" || op.AuthPlugin == "Srp256" {
	//	specific_data = getSrpClientPublicBytes(op.ClientPublic)
	//} else if op.AuthPlugin == "Legacy_Auth" {
	//	b := bytes.NewBufferString(crypt.Crypt(op.Password, "9z")[2:]).Bytes()
	//	specific_data = bytes.Join([][]byte{
	//		{CNCT_specific_data, byte(len(b)) + 1, 0}, b,
	//	}, nil)
	//} else {
	//	panic(fmt.Sprintf("Unknown plugin name:%s", op.AuthPlugin))
	//}

	uid := bytes.Join([][]byte{
		{CNCT_login, byte(len(userBytes))}, userBytes,
		{CNCT_plugin_name, byte(len(pluginNameBytes))}, pluginNameBytes,
		{CNCT_plugin_list, byte(len(pluginListNameBytes))}, pluginListNameBytes}, nil)

	if len(op.AuthSpecificData) > 0 {
		for _, chunk := range op.AuthSpecificData {
			if len(chunk.Data) == 0 {
				continue
			}
			header := []byte{CNCT_specific_data, byte(len(chunk.Data)) + 1, chunk.Part}
			uid = append(uid, header...)
			uid = append(uid, chunk.Data...)
		}
	}

	uid = append(uid, bytes.Join([][]byte{
		{CNCT_client_crypt, 4, wireCryptByte, 0, 0, 0},
		{CNCT_user, byte(len(sysUserBytes))}, sysUserBytes,
		{CNCT_host, byte(len(hostnameBytes))}, hostnameBytes,
		{CNCT_user_verification, 0},
	}, nil)...)

	// p_cnct_user_id
	if err = x.WriteBuffer(uid); err != nil {
		return err
	}

	for _, protocol := range SupportedProtocols {
		//p_cnct_version
		if err = x.WriteInt32(protocol.Version); err != nil {
			return err
		}
		//p_cnct_architecture
		if err = x.WriteInt32(protocol.ArchType); err != nil {
			return err
		}
		//p_cnct_min_type
		if err = x.WriteInt32(protocol.MinType); err != nil {
			return err
		}

		//p_cnct_max_type
		if op.Compress {
			if err = x.WriteInt32(protocol.MaxType | pflag_compress); err != nil {
				return err
			}
		} else {
			if err = x.WriteInt32(protocol.MaxType); err != nil {
				return err
			}
		}

		//p_cnct_weight
		if err = x.WriteInt32(protocol.Weight); err != nil {
			return err
		}
	}

	return nil
}
