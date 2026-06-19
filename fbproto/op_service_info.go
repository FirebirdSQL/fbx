package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-services-query-service

const defaultServiceInfoBufferLength = 1024

// OpServiceInfo requests information from a service attachment.
type OpServiceInfo struct {
	// ServiceHandle identifies the service attachment to query.
	ServiceHandle int

	// SendItems contains optional data sent with the service info request.
	SendItems []byte

	// ReceiveItems contains the service info items requested from the server.
	ReceiveItems []byte

	// BufferLength is the maximum size of the service info response.
	BufferLength int
}

// Write encodes the service info operation into the wire protocol.
func (op *OpServiceInfo) Write(x *XDRWriter) error {
	if err := x.WriteInt32(Op_service_info); err != nil {
		return err
	}
	if err := x.WriteInt32(op.ServiceHandle); err != nil {
		return err
	}
	if err := x.WriteInt32(0); err != nil {
		return err
	}
	if err := x.WriteBuffer(op.SendItems); err != nil {
		return err
	}
	if err := x.WriteBuffer(op.ReceiveItems); err != nil {
		return err
	}

	bufferLength := op.BufferLength
	if bufferLength == 0 {
		bufferLength = defaultServiceInfoBufferLength
	}
	return x.WriteInt32(bufferLength)
}
