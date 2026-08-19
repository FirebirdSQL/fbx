package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-services-attach

// OpServiceAttach attaches to a Firebird service, usually service_mgr.
type OpServiceAttach struct {
	// Encoding encodes the service object name.
	Encoding *Encoding

	// ServiceName is the service object name sent to the server.
	ServiceName string

	// SPB is the service parameter buffer used for the attachment.
	SPB []byte
}

// Write encodes the service attach operation into the wire protocol.
func (op *OpServiceAttach) Write(x *XDRWriter) error {
	if err := x.WriteInt32(Op_service_attach); err != nil {
		return err
	}
	if err := x.WriteInt32(0); err != nil {
		return err
	}
	if err := x.WriteString(op.ServiceName, op.Encoding); err != nil {
		return err
	}
	return x.WriteBuffer(op.SPB)
}
