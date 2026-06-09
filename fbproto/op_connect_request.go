package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-connections-auxiliary

// OpConnectRequest requests auxiliary connection information (eg event channel port).
type OpConnectRequest struct{}

func (op *OpConnectRequest) Write(x *XDRWriter) error {
	var err error

	// p_operation
	if err = x.WriteInt32(Op_connect_request); err != nil {
		return err
	}

	// p_req_type
	if err = x.WriteInt32(P_req_async); err != nil {
		return err
	}

	// p_req_object + p_req_partner
	if err = x.WriteInt64(0); err != nil {
		return err
	}

	return nil
}
