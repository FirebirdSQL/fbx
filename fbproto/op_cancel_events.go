package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-events

type OpCancelEvents struct {
	DatabaseHandle int
	EventID        int
}

func (op *OpCancelEvents) Write(x *XDRWriter) error {
	var err error

	// p_operation
	if err = x.WriteInt32(Op_cancel_events); err != nil {
		return err
	}

	// p_event_database
	if err = x.WriteInt32(op.DatabaseHandle); err != nil {
		return err
	}

	// p_event_rid
	if err = x.WriteInt32(op.EventID); err != nil {
		return err
	}

	return nil
}
