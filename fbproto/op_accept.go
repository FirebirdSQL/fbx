package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-op-accept

type AcceptMetadata struct {
	Version int
	Architecture int
	Type int
	Compress bool
}

// OpAccept represents an op_accept response from the server.
type OpAccept struct {
	AcceptMetadata
}

// Read decodes an op_accept response after its operation code has been read.
func (op *OpAccept) Read(r *XDRReader) error {
	return op.read(r)
}

func (metadata *AcceptMetadata) read(r *XDRReader) error {
	var err error
	if metadata.Version, err = r.ReadInt32(); err != nil {
		return err
	}
	if metadata.Version < 0 {
		metadata.Version &= ProtocolMask
	}
	if metadata.Architecture, err = r.ReadInt32(); err != nil {
		return err
	}

	acceptType, err := r.ReadInt32()
	if err != nil {
		return err
	}
	metadata.Type = acceptType & Ptype_MASK
	metadata.Compress = acceptType&Pflag_compress != 0
	return nil
}
