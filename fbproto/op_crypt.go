package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-crypt

type OpCrypt struct {
	Encoding *Encoding
	Plugin   string
}

func (op *OpCrypt) Write(x *XDRWriter) error {
	if err := x.WriteInt32(op_crypt); err != nil {
		return err
	}
	if err := x.WriteString(op.Plugin, op.Encoding); err != nil {
		return err
	}
	return x.WriteString("Symmetric", op.Encoding)
}
