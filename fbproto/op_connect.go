package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-connect

import (
	"bytes"
	"strings"

	"rdb.red-soft.ru/fbx/fbproto/auth"
)

type OpConnect struct {
	Operation        int
	Database         string
	User             string
	SystemUser       string
	SystemHost       string
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
	if err = x.WriteInt32(Op_connect); err != nil {
		return err
	}
	operation := op.Operation
	if operation == 0 {
		operation = Op_attach
	}
	// p_cnct_operation is ignored by modern servers, but older protocol paths
	// can use it to identify the database operation that follows connect.
	if err = x.WriteInt32(operation); err != nil {
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
	hostnameBytes := bytes.NewBufferString(op.SystemHost).Bytes()
	pluginList := strings.Join(op.AuthPluginsList, ",")
	pluginListNameBytes := bytes.NewBufferString(pluginList).Bytes()
	pluginNameBytes := bytes.NewBufferString(op.AuthPluginName).Bytes()
	userBytes := bytes.NewBufferString(strings.ToUpper(op.User)).Bytes()
	var wireCryptByte byte
	if op.WireCrypt {
		wireCryptByte = 1
	}

	uid := make([]byte, 0, 256)
	// Only include CNCT_login if username is non-empty; presence of an empty
	// login can influence GSS negotiation on some servers/clients.
	if len(userBytes) > 0 {
		uid = append(uid, []byte{CNCT_login, byte(len(userBytes))}...)
		uid = append(uid, userBytes...)
	}
	uid = append(uid, []byte{CNCT_plugin_name, byte(len(pluginNameBytes))}...)
	uid = append(uid, pluginNameBytes...)
	uid = append(uid, []byte{CNCT_plugin_list, byte(len(pluginListNameBytes))}...)
	uid = append(uid, pluginListNameBytes...)

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
			if err = x.WriteInt32(protocol.MaxType | Pflag_compress); err != nil {
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
