package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-services-start

// OpServiceStart starts an action on a service attachment.
type OpServiceStart struct {
	// ServiceHandle identifies the service attachment that runs the action.
	ServiceHandle int

	// ActionSPB is the service request buffer describing the action to start.
	ActionSPB []byte
}

// Write encodes the service start operation into the wire protocol.
func (op *OpServiceStart) Write(x *XDRWriter) error {
	if err := x.WriteInt32(Op_service_start); err != nil {
		return err
	}
	if err := x.WriteInt32(op.ServiceHandle); err != nil {
		return err
	}
	if err := x.WriteInt32(0); err != nil {
		return err
	}
	return x.WriteBuffer(op.ActionSPB)
}
