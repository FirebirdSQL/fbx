package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-arrays-putslice

type OpPutSlice struct {
	TransactionHandle int
	ArrayHandle       int64
	SliceLength       int
	SliceDescriptor   []byte
	Slice             []byte
}

func (op *OpPutSlice) Write(x *XDRWriter) error {
	var err error

	// p_operation
	if err = x.WriteInt32(Op_put_slice); err != nil {
		return err
	}

	//p_slc_transaction
	if err = x.WriteInt32(op.TransactionHandle); err != nil {
		return err
	}

	//p_slc_id
	if err = x.WriteInt64(op.ArrayHandle); err != nil {
		return err
	}

	//p_slc_length
	if err = x.WriteInt32(op.SliceLength); err != nil {
		return err
	}

	//p_slc_sdl
	if err = x.WriteBuffer(op.SliceDescriptor); err != nil {
		return err
	}

	//p_slc_parameters
	if err = x.WriteBuffer(nil); err != nil {
		return err
	}

	//p_slc_length
	if err = x.WriteInt32(op.SliceLength); err != nil {
		return err
	}

	//p_slc_slice
	if err = x.WriteBytes(op.Slice); err != nil {
		return err
	}

	return nil
}
