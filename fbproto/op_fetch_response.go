package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-fetch

type OpFetchResponse struct {
	SQLDataStatus int
	Messages      int
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

	return nil
}

func (op *OpFetchResponse) MoreRows() bool {
	return op.SQLDataStatus == 0 && op.Messages > 0
}

func (op *OpFetchResponse) EndOfCursor() bool {
	return op.SQLDataStatus == 100 && op.Messages == 0
}
