// PreparedBatch stores rows for a batch DML statement.
// Supported statements: INSERT, UPDATE, and DELETE without result sets.
//
// 1) Object API:
//
//	pb, err := conn.PrepareBatch(ctx, "insert into t(id, note) values (?, ?)")
//	if err != nil { ... }
//	defer pb.Close(ctx)
//
//	_ = pb.Add(1, "one")
//	_ = pb.Add(2, "two")
//
//	res, err := pb.Exec(ctx)
//
// Object batches are manual. Use [PreparedBatch.PendingBytes] and [PreparedBatch.Flush]
// if you want to control when queued rows are sent.
//
// 2) Named API:
//
//	_, err := conn.Prepare(ctx, "b1", "batch insert into t(id, note) values (?, ?)")
//	if err != nil { ... }
//	defer conn.Deallocate(ctx, "b1")
//
//	_, _ = conn.Exec(ctx, "b1", 1, "one")
//	_, _ = conn.Exec(ctx, "b1", 2, "two")
//
//	res, err := conn.ExecBatch(ctx, "b1")
//
// Named batches support built-in auto-flush.

package fbx

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"rdb.red-soft.ru/fbx/fbconn"
)

const (
	batchPreparePrefix            = "batch "
	defaultPreparedBatchFlushSize = 128 * 1024
)

// BatchOptions configures object-style batches created with PrepareBatchWithOptions.
// Zero values keep the default behavior.
type BatchOptions struct {
	// BufferBytes requests the server-side batch buffer size in bytes.
	// Zero uses the server default.
	BufferBytes int

	// ContinueOnError requests Firebird multi-error mode. When enabled, the
	// server continues executing rows after per-row errors and reports all row
	// errors it can return in the batch completion response.
	ContinueOnError bool

	// DetailedErrors requests the number of detailed per-row errors returned by
	// the server. Zero keeps the driver default.
	DetailedErrors int
}

type BatchResult struct {
	Affected int64
}

// BatchError describes a single row error returned by batch execution.
type BatchError struct {
	Row int
	Err error
}

// BatchExecutionError reports one or more row errors returned by batch execution.
type BatchExecutionError struct {
	Errors []BatchError
}

func (e *BatchExecutionError) Error() string {
	if e == nil || len(e.Errors) == 0 {
		return "batch failed"
	}

	first := e.Errors[0]
	msg := fmt.Sprintf("batch failed at row %d", first.Row)
	if first.Err != nil {
		msg = fmt.Sprintf("%s: %v", msg, first.Err)
	}
	if len(e.Errors) > 1 {
		remaining := len(e.Errors) - 1
		if remaining == 1 {
			msg = fmt.Sprintf("%s; 1 more batch error", msg)
		} else {
			msg = fmt.Sprintf("%s; %d more batch errors", msg, remaining)
		}
	}
	return msg
}

func (e *BatchExecutionError) Unwrap() []error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}

	errs := make([]error, 0, len(e.Errors))
	for _, batchErr := range e.Errors {
		if batchErr.Err != nil {
			errs = append(errs, batchErr.Err)
		}
	}
	return errs
}

type PreparedBatch struct {
	conn *Conn
	tx   *dbTx

	name string
	sql  string

	sd *fbconn.StatementDescription
	st *fbconn.Statement

	activeSt                *fbconn.Statement
	releaseActiveSt         bool
	releaseActiveInternalTx bool
	lb                      *fbconn.Batch

	encodedRows    [][]byte
	pendingBytes   int
	autoFlushBytes int
	batchCfg       fbconn.BatchConfig
	cleanupPending bool
	closed         bool

	eqb ExtendedQueryBuilder
}

// Add appends one row to the batch buffer.
func (b *PreparedBatch) Add(args ...any) error {
	return b.addEncoded(context.Background(), args, false)
}

// PendingBytes returns the size of queued encoded rows that have not been sent yet.
func (b *PreparedBatch) PendingBytes() int {
	if b == nil {
		return 0
	}
	return b.pendingBytes
}

func isBatchPrepareSQL(sql string) (normalized string, ok bool) {
	trimmed := strings.TrimSpace(sql)
	if len(trimmed) < len(batchPreparePrefix) || !strings.EqualFold(trimmed[:len(batchPreparePrefix)], batchPreparePrefix) {
		return "", false
	}
	return strings.TrimSpace(trimmed[len(batchPreparePrefix):]), true
}

func validateBatchStatement(sd *fbconn.StatementDescription) error {
	if sd == nil {
		return fmt.Errorf("prepared batch metadata is not initialized")
	}
	if len(sd.Parameters) == 0 {
		return fmt.Errorf("statement used in batch must have parameters")
	}
	for i, param := range sd.Parameters {
		if param.IsBlob() {
			return fmt.Errorf("batch does not support blob parameters (arg %d)", i)
		}
	}
	if sd.HaveCursorResult() || sd.HaveSingletonResult() {
		return fmt.Errorf("batch supports only insert, update, and delete statements without result sets")
	}

	switch sd.GetQueryType() {
	case fbconn.StatementInsert, fbconn.StatementUpdate, fbconn.StatementDelete:
		return nil
	default:
		return fmt.Errorf("batch supports only insert, update, and delete statements without result sets")
	}
}

func defaultPreparedBatchConfig() fbconn.BatchConfig {
	return fbconn.BatchConfig{
		RecordCounts: true,
		DetailedErrs: 1,
		BufferBytes:  -1,
	}
}

func normalizeBatchOptions(opts BatchOptions) fbconn.BatchConfig {
	cfg := defaultPreparedBatchConfig()
	if opts.BufferBytes > 0 {
		cfg.BufferBytes = opts.BufferBytes
	}
	if opts.ContinueOnError {
		cfg.MultiError = true
		cfg.DetailedErrs = -1
	}
	if opts.DetailedErrors > 0 {
		cfg.DetailedErrs = opts.DetailedErrors
	}
	return cfg
}

func (b *PreparedBatch) addEncoded(ctx context.Context, args []any, allowAutoFlush bool) error {
	if b == nil {
		return fmt.Errorf("prepared batch is nil")
	}
	if b.closed {
		return fmt.Errorf("prepared batch is closed")
	}
	if b.cleanupPending {
		return fmt.Errorf("prepared batch has pending cleanup after execution; close the batch before reusing it")
	}
	if b.sd == nil {
		return fmt.Errorf("prepared batch metadata is not initialized")
	}
	if len(args) != len(b.sd.Parameters) {
		return fmt.Errorf("expected %d arguments, got %d", len(b.sd.Parameters), len(args))
	}
	for i, param := range b.sd.Parameters {
		if param.IsBlob() || isBlobArgument(args[i]) {
			return fmt.Errorf("batch does not support blob parameters (arg %d)", i)
		}
	}

	if err := b.eqb.Build(ctx, b.conn, b.tx, b.conn.typeMap, b.sd, args); err != nil {
		return err
	}

	row, err := b.sd.EncodeBatchRow(b.eqb.ParamValues)
	if err != nil {
		return err
	}

	b.eqb.reset()

	if allowAutoFlush && b.autoFlushBytes > 0 && b.pendingBytes > 0 && b.pendingBytes+len(row) > b.autoFlushBytes {
		if err := b.flushWithSource(ctx, "auto"); err != nil {
			return err
		}
	}

	b.encodedRows = append(b.encodedRows, row)
	b.pendingBytes += len(row)

	return nil
}

// Flush sends queued rows to the server without executing the batch.
func (b *PreparedBatch) Flush(ctx context.Context) error {
	return b.flushWithSource(ctx, "manual")
}

func (b *PreparedBatch) flushWithSource(ctx context.Context, source string) error {
	if b == nil {
		return fmt.Errorf("prepared batch is nil")
	}
	if b.closed {
		return fmt.Errorf("prepared batch is closed")
	}
	if b.cleanupPending {
		return fmt.Errorf("prepared batch has pending cleanup after execution; close the batch before reusing it")
	}
	if len(b.encodedRows) == 0 {
		return nil
	}

	st, err := b.batchStatement(ctx)
	if err != nil {
		return err
	}
	if b.lb == nil {
		b.lb = st.NewBatch(b.batchCfg)
	}

	if err := b.lb.Send(ctx, b.encodedRows); err != nil {
		return errors.Join(err, b.releaseWireResources(ctx))
	}
	b.Reset()
	return nil
}

// Exec flushes queued rows, executes the batch, and returns the result.
func (b *PreparedBatch) Exec(ctx context.Context) (BatchResult, error) {
	if b == nil {
		return BatchResult{}, fmt.Errorf("prepared batch is nil")
	}
	if b.closed {
		return BatchResult{}, fmt.Errorf("prepared batch is closed")
	}
	if b.cleanupPending {
		return BatchResult{}, fmt.Errorf("prepared batch has pending cleanup after execution; close the batch before reusing it")
	}
	if err := b.flushWithSource(ctx, "exec-final"); err != nil {
		return BatchResult{}, err
	}
	if b.lb == nil {
		return BatchResult{}, nil
	}

	var fbTx *fbconn.FBTransaction
	if b.tx != nil {
		fbTx = b.tx.tx
	}
	completion, err := b.lb.Execute(ctx, fbTx)
	b.cleanupPending = true
	releaseErr := b.releaseWireResources(ctx)
	return finalizePreparedBatchExec(completion, err, nil, releaseErr)
}

func (c *Conn) shouldReleaseBatchInternalTx(tx *dbTx) bool {
	return tx == nil && c.config.Autocommit && !c.fbConn.ActiveTx()
}

func (c *Conn) rollbackBatchInternalTx(ctx context.Context, release bool) error {
	if !release {
		return nil
	}
	err := c.fbConn.Rollback(ctx)
	if errors.Is(err, fbconn.ErrTransactionClosed) {
		return nil
	}
	return err
}

func (b *PreparedBatch) batchStatement(ctx context.Context) (*fbconn.Statement, error) {
	if b.activeSt != nil {
		return b.activeSt, nil
	}

	if b.tx != nil {
		if b.tx.closed {
			return nil, ErrTxClosed
		}
		if b.st == nil {
			return nil, fmt.Errorf("prepared batch statement is not initialized")
		}
		b.activeSt = b.st
		b.releaseActiveSt = false
		return b.activeSt, nil
	}

	releaseInternalTx := b.conn.shouldReleaseBatchInternalTx(b.tx)
	st, err := b.conn.internalPrepare(ctx, nil, "", b.sql)
	if err != nil {
		return nil, errors.Join(err, b.conn.rollbackBatchInternalTx(ctx, releaseInternalTx))
	}
	b.activeSt = st
	b.releaseActiveSt = true
	b.releaseActiveInternalTx = releaseInternalTx
	return st, nil
}

func (b *PreparedBatch) releaseBatchStatement(ctx context.Context) error {
	var err error
	if b.activeSt != nil && b.releaseActiveSt {
		if deallocErr := b.activeSt.Deallocate(ctx); deallocErr != nil {
			err = errors.Join(err, deallocErr)
			if b.conn == nil || b.conn.IsClosed() {
				b.activeSt = nil
				b.releaseActiveSt = false
			}
		} else {
			b.activeSt = nil
			b.releaseActiveSt = false
		}
	} else if b.activeSt != nil {
		b.activeSt = nil
	}

	if b.releaseActiveInternalTx {
		if rollbackErr := b.conn.rollbackBatchInternalTx(ctx, true); rollbackErr != nil {
			err = errors.Join(err, rollbackErr)
			if b.conn == nil || b.conn.IsClosed() {
				b.releaseActiveInternalTx = false
			}
		} else {
			b.releaseActiveInternalTx = false
		}
	}

	return err
}

func (b *PreparedBatch) releaseWireResources(ctx context.Context) error {
	var batchErr error
	if b.lb != nil {
		batchErr = b.lb.Close(ctx)
		if batchErr != nil && b.conn != nil && !b.conn.IsClosed() {
			return batchErr
		}
		b.lb = nil
	}
	stErr := b.releaseBatchStatement(ctx)
	err := errors.Join(batchErr, stErr)
	if err == nil {
		b.cleanupPending = false
	}
	return err
}

func (b *PreparedBatch) releaseStoredStatement(ctx context.Context) error {
	if b.st == nil || b.activeSt == b.st {
		return nil
	}
	err := b.st.Deallocate(ctx)
	if err == nil || b.conn == nil || b.conn.IsClosed() {
		b.st = nil
	}
	return err
}

// Reset clears queued rows that have not been sent yet.
func (b *PreparedBatch) Reset() {
	if b == nil {
		return
	}
	clear(b.encodedRows)
	b.encodedRows = b.encodedRows[:0]
	b.pendingBytes = 0
}

// Close releases batch resources.
func (b *PreparedBatch) Close(ctx context.Context) error {
	if b == nil || b.closed {
		return nil
	}
	b.Reset()

	err := b.releaseWireResources(ctx)
	if err == nil {
		err = errors.Join(err, b.releaseStoredStatement(ctx))
	}
	if err == nil || b.conn == nil || b.conn.IsClosed() {
		b.closed = true
	}
	return err
}

func batchCompletionError(completion *fbconn.BatchCompletion) error {
	if completion == nil || len(completion.Errors) == 0 {
		return nil
	}

	batchErr := &BatchExecutionError{
		Errors: make([]BatchError, len(completion.Errors)),
	}
	for i, completionErr := range completion.Errors {
		batchErr.Errors[i] = BatchError{
			Row: completionErr.Row,
			Err: completionErr.Err,
		}
	}
	return batchErr
}

func finalizePreparedBatchExec(completion *fbconn.BatchCompletion, execErr, closeErr, releaseErr error) (BatchResult, error) {
	if execErr != nil {
		return BatchResult{}, errors.Join(execErr, closeErr, releaseErr)
	}

	result := BatchResult{}
	if completion != nil {
		result.Affected = completion.Affected()
	}

	return result, errors.Join(batchCompletionError(completion), closeErr, releaseErr)
}
