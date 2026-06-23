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

// BatchOptions configures batches created with [Conn.PrepareBatchWithOptions]
// or [Tx.PrepareBatchWithOptions].
//
// Zero values keep the driver defaults.
type BatchOptions struct {
	// BufferBytes requests the server-side batch buffer size in bytes.
	// Zero uses the server default.
	BufferBytes int

	// ContinueOnError requests Firebird multi-error mode. When enabled, the
	// server continues executing rows after per-row errors and reports all row
	// errors it can return in the batch completion response.
	//
	// This affects error reporting only, not transaction atomicity: in
	// autocommit mode a batch with any per-row error is still rolled back as a
	// whole (see [PreparedBatch.Exec]). Use an explicit transaction if you want
	// to inspect the result and decide whether to commit rows that were applied
	// successfully.
	ContinueOnError bool

	// DetailedErrors requests the number of detailed per-row errors returned by
	// the server.
	//
	// Zero keeps the driver default: one detailed error in stop-on-first-error
	// mode, or the server default when ContinueOnError is enabled.
	DetailedErrors int
}

// BatchResult describes the server completion data returned by
// [PreparedBatch.Exec] and [Conn.ExecBatch].
//
// When execution returns a [BatchExecutionError], BatchResult still describes
// what the server processed before the error response. In autocommit mode those
// successful rows are rolled back by the driver when any per-row error is
// reported.
type BatchResult struct {
	// Affected is the sum of positive per-row update counts. If the server
	// returned no per-row counts (RecordCounts disabled), Affected falls back
	// to the server-reported total record count.
	Affected int64

	// UpdateCounts holds per-row update counts in submission order, when
	// RecordCounts is enabled (the default). Negative values are Firebird
	// sentinels for failed/skipped rows; use BatchExecutionError.Errors for
	// the cause.
	//
	// Length: with BatchOptions.ContinueOnError there is one entry per
	// submitted row; in the default stop-on-first-error mode the slice may
	// be shorter (only rows processed before the failure are reported).
	//
	// Nil when the server returned no per-row counts.
	UpdateCounts []int64
}

// ErrBatchRowFailed marks a per-row batch failure that the server reported
// without a detailed status vector. It is wrapped into BatchError.Err when
// MultiError mode is in effect and the server exceeds the DetailedErrors
// budget — Firebird then sends only the row number for the remaining errors.
// Use errors.Is to detect such rows from a [BatchExecutionError].
var ErrBatchRowFailed = errors.New("batch row failed")

// BatchError describes a single row error returned by batch execution.
type BatchError struct {
	// Row is the zero-based row index in submission order.
	Row int

	// Err is the detailed row error returned by the server, or an error wrapping
	// [ErrBatchRowFailed] when only simplified row information was returned.
	Err error
}

// BatchExecutionError reports one or more row errors returned by batch execution.
type BatchExecutionError struct {
	// Errors contains the row errors reported by the server.
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

// PreparedBatch stores encoded rows for one Firebird batch DML statement.
//
// Create a connection-scoped batch with [Conn.PrepareBatch] or
// [Conn.PrepareBatchWithOptions]. Create a transaction-scoped batch with
// [Tx.PrepareBatch] or [Tx.PrepareBatchWithOptions].
//
// [PreparedBatch.Add] encodes one row into the local buffer. [PreparedBatch.Flush]
// sends locally buffered rows to the server-side batch buffer without executing
// them. [PreparedBatch.Exec] flushes remaining rows, executes the server-side
// batch, and returns a [BatchResult].
//
// Connection-scoped batches use the connection's internal transaction. In
// autocommit mode [PreparedBatch.Exec] is atomic: if the server reports any
// per-row error, the driver rolls back the internal transaction and no rows from
// that execution are committed. Transaction-scoped batches never commit or roll
// back the user's transaction; call [Tx.Commit] or [Tx.Rollback] yourself after
// inspecting the result and error.
//
// Always call [PreparedBatch.Close] when the batch is no longer needed.
// [PreparedBatch.Cancel] can be used before [PreparedBatch.Exec] to discard both
// local rows and rows already flushed to the server while keeping the
// PreparedBatch reusable.
// PreparedBatch is not safe for concurrent use.
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

// Add appends one encoded row to the local batch buffer.
//
// Add does not send batch data to the server and does not execute the batch.
// Object-style batches do not auto-flush; call [PreparedBatch.Flush] yourself
// when [PreparedBatch.PendingBytes] reaches a useful threshold.
func (b *PreparedBatch) Add(args ...any) error {
	return b.addEncoded(context.Background(), args, false)
}

// PendingBytes returns the size of locally queued encoded rows that have not
// been sent to the server yet.
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
		if param.IsArray() {
			return fmt.Errorf("batch does not support array parameters (arg %d)", i)
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

// Flush sends locally queued rows to the server-side batch buffer without
// executing the batch.
//
// Rows sent by Flush stay pending on the server until [PreparedBatch.Exec],
// [PreparedBatch.Cancel], or [PreparedBatch.Close]. Flush is a no-op when there
// are no locally queued rows.
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
	b.ClearPending()
	return nil
}

// Exec flushes queued rows, executes the batch, and returns the result.
//
// Exec returns a [BatchExecutionError] when the server reports per-row errors.
// The returned [BatchResult] still contains server completion data such as
// update counts.
//
// In autocommit mode on a connection-scoped batch, Exec is atomic: any per-row
// error rolls back the internal transaction and no rows from that execution are
// committed. In an explicit transaction, Exec leaves commit and rollback
// decisions to the caller.
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

// ClearPending discards rows that are still buffered locally and have not been
// transmitted to the server yet. Rows already sent by a prior
// [PreparedBatch.Flush] remain in the server-side batch buffer and will still be
// applied on the next [PreparedBatch.Exec]; use [PreparedBatch.Cancel] to
// discard those too.
//
// ClearPending performs no wire I/O and never returns an error.
func (b *PreparedBatch) ClearPending() {
	if b == nil {
		return
	}
	clear(b.encodedRows)
	b.encodedRows = b.encodedRows[:0]
	b.pendingBytes = 0
}

// Cancel performs a full reset: drops queued rows, releases the server-side
// batch context, and rolls back any internally owned statement and
// transaction. PreparedBatch stays valid — the next [PreparedBatch.Add],
// [PreparedBatch.Flush], or [PreparedBatch.Exec] re-prepares what it needs. For
// tx-scoped batches the user's statement and tx are left alone.
//
// Returns an error if the batch is closed or has pending cleanup from a prior
// [PreparedBatch.Exec] (call [PreparedBatch.Close] instead). On context
// cancellation, retry with a fresh context to free server-side resources.
func (b *PreparedBatch) Cancel(ctx context.Context) error {
	if b == nil {
		return fmt.Errorf("prepared batch is nil")
	}
	if b.closed {
		return fmt.Errorf("prepared batch is closed")
	}
	if b.cleanupPending {
		return fmt.Errorf("prepared batch has pending cleanup after execution; close the batch before reusing it")
	}
	b.ClearPending()
	return b.releaseWireResources(ctx)
}

// Close releases every resource owned by the batch and marks it unusable.
// Idempotent. Must be called even after [PreparedBatch.Exec] returns an error,
// otherwise the prepared statement and any internal transaction stay allocated.
// On context cancellation, retry with a fresh context to free server-side
// resources.
func (b *PreparedBatch) Close(ctx context.Context) error {
	if b == nil || b.closed {
		return nil
	}
	b.ClearPending()

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
		err := completionErr.Err
		if err == nil {
			// Simplified errors (no detailed status vector) — synthesize a
			// sentinel so errors.Is can recognize them and so Unwrap does
			// not silently drop these rows.
			err = fmt.Errorf("%w at row %d", ErrBatchRowFailed, completionErr.Row)
		}
		batchErr.Errors[i] = BatchError{
			Row: completionErr.Row,
			Err: err,
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
		if len(completion.UpdateCounts) > 0 {
			result.UpdateCounts = append(make([]int64, 0, len(completion.UpdateCounts)), completion.UpdateCounts...)
		}
	}

	return result, errors.Join(batchCompletionError(completion), closeErr, releaseErr)
}
