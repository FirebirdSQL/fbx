package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-batches-sync

type OpBatchSync struct {
}

func (op *OpBatchSync) Write(x *XDRWriter) error {
	// p_operation
	return x.WriteInt32(Op_batch_sync)
}
