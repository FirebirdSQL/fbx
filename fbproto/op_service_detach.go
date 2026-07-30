package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-services-detach

// OpServiceDetach releases a service attachment.
type OpServiceDetach struct {
	// ServiceHandle identifies the service attachment to release.
	ServiceHandle int
}

// Write encodes the service detach operation into the wire protocol.
func (op *OpServiceDetach) Write(x *XDRWriter) error {
	if err := x.WriteInt32(Op_service_detach); err != nil {
		return err
	}
	return x.WriteInt32(op.ServiceHandle)
}
