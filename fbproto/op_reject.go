package fbproto

//https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-reject

type OpReject struct {
}

func (op *OpReject) Read(x *XDRReader) error {
	return nil
}
