package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-events

type OpQueEvents struct {
	DatabaseHandle int
	EventItems     []byte
	EventID        int
}

func (op *OpQueEvents) Write(x *XDRWriter) error {
	var err error

	// p_operation
	if err = x.WriteInt32(Op_que_events); err != nil {
		return err
	}

	// p_event_database
	if err = x.WriteInt32(op.DatabaseHandle); err != nil {
		return err
	}

	// p_event_items
	if err = x.WriteBuffer(op.EventItems); err != nil {
		return err
	}

	// p_event_ast + p_event_arg
	if err = x.WriteInt64(0); err != nil {
		return err
	}

	// p_event_rid
	if err = x.WriteInt32(op.EventID); err != nil {
		return err
	}

	return nil
}
