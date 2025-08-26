package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-dummy

type OpDummy struct {
}

func (op *OpDummy) Read(x *XDRReader) error {
	return nil
}
