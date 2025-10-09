package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-crypt

type OpCrypt struct {
	// Encoding used for XDR strings in this packet
	Encoding *Encoding
	// p_name: chosen wire encryption plugin
	Plugin string
}

func (op *OpCrypt) Write(x *XDRWriter) error {
	// p_operation
	if err := x.WriteInt32(Op_crypt); err != nil {
		return err
	}
	// p_name
	if err := x.WriteString(op.Plugin, op.Encoding); err != nil {
		return err
	}
	// p_type (Firebird uses string "Symmetric")
	return x.WriteString("Symmetric", op.Encoding)
}
