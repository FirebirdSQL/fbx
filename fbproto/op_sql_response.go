package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-sql

type OpSqlResponse struct {
	Messages int
}

func (op *OpSqlResponse) Read(x *XDRReader) error {
	var err error

	// p_sqldata_messages
	if op.Messages, err = x.ReadInt32(); err != nil {
		return err
	}

	return nil
}
