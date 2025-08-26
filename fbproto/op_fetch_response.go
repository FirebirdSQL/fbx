package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-fetch

type OpFetchResponse struct {
	SQLDataStatus int
	Messages      int
	RowData       []byte
}

func (op *OpFetchResponse) Read(x *XDRReader) error {
	var err error
	// p_sqldata_status
	if op.SQLDataStatus, err = x.ReadInt32(); err != nil {
		return err
	}

	// p_sqldata_messages
	if op.Messages, err = x.ReadInt32(); err != nil {
		return err
	}

	// Row Data
	if op.RowData, err = x.ReadBuffer(); err != nil {
		return err
	}

	return nil
}
