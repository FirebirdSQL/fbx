package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-batches-release

import "fmt"

type OpBatchRelease struct {
	Kind            int
	StatementHandle int
}

func (op *OpBatchRelease) Write(x *XDRWriter) error {
	if op.Kind != Op_batch_cancel && op.Kind != Op_batch_rls {
		return fmt.Errorf("wrong batch release operation: %d", op.Kind)
	}
	// p_operation
	if err := x.WriteInt32(op.Kind); err != nil {
		return err
	}

	// p_batch_statement
	return x.WriteInt32(op.StatementHandle)
}
