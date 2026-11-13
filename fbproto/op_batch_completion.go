package fbproto

import "fmt"

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

	// Reuse storage, but grow slices only after successfully reading an element.
	op.UpdateCounts = op.UpdateCounts[:0]
	clear(op.DetailedErrors)
	op.DetailedErrors = op.DetailedErrors[:0]
	op.SimplifiedErrors = op.SimplifiedErrors[:0]

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
	if op.TotalRecords < 0 || updateCountCount < 0 || detailedErrorCount < 0 || simplifiedErrorCount < 0 {
		return fmt.Errorf("invalid batch completion counts: records=%d updates=%d vectors=%d errors=%d",
			op.TotalRecords, updateCountCount, detailedErrorCount, simplifiedErrorCount)
	}
	if updateCountCount > op.TotalRecords || detailedErrorCount > op.TotalRecords ||
		simplifiedErrorCount > op.TotalRecords-detailedErrorCount {
		return fmt.Errorf("batch completion counts exceed record count %d", op.TotalRecords)
	}

	// Update counters
	for i := 0; i < updateCountCount; i++ {
		count, err := x.ReadInt32()
		if err != nil {
			return err
		}
		op.UpdateCounts = append(op.UpdateCounts, count)
	}

	// Detailed info
	for i := 0; i < detailedErrorCount; i++ {
		row, err := x.ReadInt32()
		if err != nil {
			return err
		}
		if row < 0 || row >= op.TotalRecords {
			return fmt.Errorf("invalid batch error row %d for %d records", row, op.TotalRecords)
		}
		sv, err := readStatusVector(x, op.Encoding)
		if err != nil {
			return err
		}
		op.DetailedErrors = append(op.DetailedErrors, BatchDetailedError{Row: row, Error: sv.toErrorResponse()})
	}

	// Simplified error blocks
	for i := 0; i < simplifiedErrorCount; i++ {
		row, err := x.ReadInt32()
		if err != nil {
			return err
		}
		if row < 0 || row >= op.TotalRecords {
			return fmt.Errorf("invalid batch error row %d for %d records", row, op.TotalRecords)
		}
		op.SimplifiedErrors = append(op.SimplifiedErrors, row)
	}

	return nil
}
