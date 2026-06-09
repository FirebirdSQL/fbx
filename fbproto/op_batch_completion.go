package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-batches-execute

type BatchDetailedError struct {
	Row   int
	Error ErrorResponse
}

type OpBatchCompletion struct {
	Encoding         *Encoding
	StatementHandle  int
	TotalRecords     int
	UpdateCounts     []int
	DetailedErrors   []BatchDetailedError
	SimplifiedErrors []int
}

func (op *OpBatchCompletion) Read(x *XDRReader) error {
	var err error

	// p_batch_statement
	if op.StatementHandle, err = x.ReadInt32(); err != nil {
		return err
	}

	// p_batch_reccount
	if op.TotalRecords, err = x.ReadInt32(); err != nil {
		return err
	}

	// p_batch_updates
	updateCountCount, err := x.ReadInt32()
	if err != nil {
		return err
	}

	// p_batch_vectors
	detailedErrorCount, err := x.ReadInt32()
	if err != nil {
		return err
	}

	// p_batch_errors
	simplifiedErrorCount, err := x.ReadInt32()
	if err != nil {
		return err
	}

	if cap(op.UpdateCounts) < updateCountCount {
		op.UpdateCounts = make([]int, updateCountCount)
	} else {
		op.UpdateCounts = op.UpdateCounts[:updateCountCount]
	}

	// Update counters
	for i := 0; i < updateCountCount; i++ {
		if op.UpdateCounts[i], err = x.ReadInt32(); err != nil {
			return err
		}
	}

	if cap(op.DetailedErrors) < detailedErrorCount {
		op.DetailedErrors = make([]BatchDetailedError, detailedErrorCount)
	} else {
		op.DetailedErrors = op.DetailedErrors[:detailedErrorCount]
	}

	// Detailed info
	for i := 0; i < detailedErrorCount; i++ {
		row, err := x.ReadInt32()
		if err != nil {
			return err
		}
		sv, err := readStatusVector(x, op.Encoding)
		if err != nil {
			return err
		}
		op.DetailedErrors[i] = BatchDetailedError{Row: row, Error: sv.toErrorResponse()}
	}

	if cap(op.SimplifiedErrors) < simplifiedErrorCount {
		op.SimplifiedErrors = make([]int, simplifiedErrorCount)
	} else {
		op.SimplifiedErrors = op.SimplifiedErrors[:simplifiedErrorCount]
	}

	// Simplified error blocks
	for i := 0; i < simplifiedErrorCount; i++ {
		if op.SimplifiedErrors[i], err = x.ReadInt32(); err != nil {
			return err
		}
	}

	return nil
}
