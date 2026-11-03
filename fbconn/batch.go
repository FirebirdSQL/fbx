package fbconn

import (
	"context"
	"errors"
	"fmt"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/internal/operationerror"
)

type BatchConfig struct {
	MultiError   bool
	RecordCounts bool
	DetailedErrs int
	BufferBytes  int
	BlobPolicy   byte
}

func (cfg BatchConfig) normalize() BatchConfig {
	if cfg.BlobPolicy == 0 {
		cfg.BlobPolicy = fbproto.BatchBlobIDUser
	}
	return cfg
}

func (cfg BatchConfig) writeParameterBuffer(pb *fbproto.BatchPBWriter) {
	if cfg.MultiError {
		pb.PutInt32(fbproto.BatchTagMultiError, 1)
	}
	if cfg.RecordCounts {
		pb.PutInt32(fbproto.BatchTagRecordCounts, 1)
	}
	if cfg.DetailedErrs >= 0 {
		pb.PutInt32(fbproto.BatchTagDetailedErrors, int32(cfg.DetailedErrs))
	}
	if cfg.BufferBytes >= 0 {
		pb.PutInt32(fbproto.BatchTagBufferBytes, int32(cfg.BufferBytes))
	}
	pb.PutByte(fbproto.BatchTagBlobPolicy, cfg.BlobPolicy)
}

func (cfg BatchConfig) parameterBuffer() []byte {
	pb := fbproto.NewBatchPBWriter()
	cfg.writeParameterBuffer(pb)
	return pb.Bytes()
}

type BatchError struct {
	Row int
	Err error
}

var errBatchExecutionFailed = errors.New("batch execution failed")

type BatchCompletion struct {
	// TotalRecords is the number of processed batch messages, not affected table rows.
	TotalRecords uint32
	UpdateCounts []int64
	Errors       []BatchError
}

func (c *BatchCompletion) HasErrors() bool {
	return c != nil && len(c.Errors) > 0
}

// Affected returns the sum of known update counts, excluding failed messages.
// It returns -1 if counts are missing or unknown, and 0 for an empty or nil completion.
// The result describes this execution, regardless of a later commit or rollback.
func (c *BatchCompletion) Affected() int64 {
	if c == nil {
		return 0
	}
	if uint64(len(c.UpdateCounts)) != uint64(c.TotalRecords) {
		return -1
	}
	var affected int64
	for _, n := range c.UpdateCounts {
		if n >= 0 {
			affected += n
		} else if n != fbproto.BatchExecuteFailed {
			return -1
		}
	}
	return affected
}

type batchState int

const (
	batchStateInit batchState = iota
	batchStateCreated
	batchStateClosed
)

type batchBufferState int

const (
	batchBufferEmpty batchBufferState = iota
	batchBufferPending
	batchBufferNeedsReset
)

type Batch struct {
	st          *Statement
	state       batchState
	cfg         BatchConfig
	err         error
	bufferState batchBufferState // Server queue, independent of the batch's lifetime.
}

// HasPending reports whether the server queue contains rows or its contents
// are uncertain after an unsuccessful exchange.
func (b *Batch) HasPending() bool {
	return b != nil && b.bufferState != batchBufferEmpty
}

// NeedsReset reports an attempted send or execution without a confirmed outcome.
// Call Cancel before sending or executing more rows, or Close to release the batch.
func (b *Batch) NeedsReset() bool {
	return b != nil && b.bufferState == batchBufferNeedsReset
}

// PrepareBatchStatement prepares a connection-scoped statement for batching.
// In autocommit mode, a transaction started only for this preparation is rolled
// back before returning. An existing transaction is left to its owner.
func (fbConn *FbConn) PrepareBatchStatement(ctx context.Context, sql string) (*Statement, error) {
	startedTx := fbConn.internalTransaction == nil || fbConn.internalTransaction.Closed
	tx, err := fbConn.ensureInternalTransaction(ctx, nil, false)
	if err != nil {
		return nil, err
	}
	st, err := fbConn.prepareTx(ctx, tx, sql, statementScopeConnection)
	if startedTx {
		err = operationerror.Join(err, fbConn.rollbackOwnedInternalTx(ctx, tx))
	}
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (b *Batch) isClosed() bool {
	return b == nil || b.state == batchStateClosed
}

func (b *Batch) checkCreate() error {
	if b == nil {
		return fmt.Errorf("batch is nil")
	}
	if b.err != nil {
		return b.err
	}
	if b.state == batchStateClosed {
		return fmt.Errorf("batch is closed")
	}
	if b.NeedsReset() {
		return fmt.Errorf("batch requires Cancel after an unsuccessful exchange")
	}
	if b.state == batchStateCreated {
		return nil
	}
	if b.st == nil || b.st.conn == nil || b.st.Description == nil {
		return fmt.Errorf("batch statement is not initialized")
	}
	if b.st.conn.protocolVersion < fbproto.PROTOCOL_VERSION16 {
		return fmt.Errorf("batch is not supported by protocol version %d", b.st.conn.protocolVersion)
	}
	if len(b.st.Description.StatementInfo.ParametersDescriptors) == 0 {
		return fmt.Errorf("statement used in batch must have parameters")
	}
	return nil
}

func (b *Batch) createOnWire(ctx context.Context, parameterBuffer []byte) error {
	return b.withConn(ctx, func() error {
		batchBLR, err := b.st.Description.StatementInfo.GetParametersBlr(nil)
		if err != nil {
			return err
		}

		if err := b.st.conn.withWriteLockAndFlush(func() error {
			if err := b.st.conn.client.WriteMessage(&fbproto.OpBatchCreate{
				StatementHandle: b.st.Description.StatementHandle,
				BatchBLR:        batchBLR,
				MessageLength:   fbproto.CalculateBatchMessageLength(b.st.Description.StatementInfo.ParametersDescriptors),
				ParameterBuffer: parameterBuffer,
			}); err != nil {
				return err
			}
			return b.st.conn.client.WriteMessage(&fbproto.OpPing{})
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		created, err := b.receiveTwoGenericResponses(ctx)
		if created {
			b.state = batchStateCreated
		}
		return err
	})
}

func (b *Batch) Create(ctx context.Context) error {
	if err := b.checkCreate(); err != nil {
		return err
	}
	if b.state == batchStateCreated {
		return nil
	}

	return b.createOnWire(ctx, b.cfg.parameterBuffer())
}

func (b *Batch) Send(ctx context.Context, rows [][]byte) error {
	if len(rows) == 0 {
		return nil
	}
	if err := b.Create(ctx); err != nil {
		return err
	}

	return b.withConn(ctx, func() error {
		b.bufferState = batchBufferNeedsReset
		if err := b.st.conn.withWriteLockAndFlush(func() error {
			if err := b.st.conn.client.WriteMessage(&fbproto.OpBatchMsg{
				StatementHandle: b.st.Description.StatementHandle,
				Rows:            rows,
			}); err != nil {
				return err
			}
			return b.writeBatchSyncMessage()
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		_, err := b.receiveTwoGenericResponses(ctx)
		if err == nil {
			b.bufferState = batchBufferPending
		}
		return err
	})
}

func (b *Batch) writeBatchSyncMessage() error {
	if b.st.conn.protocolVersion >= fbproto.PROTOCOL_VERSION17 {
		return b.st.conn.client.WriteMessage(&fbproto.OpBatchSync{})
	}
	return b.st.conn.client.WriteMessage(&fbproto.OpPing{})
}

func (b *Batch) Execute(ctx context.Context, tx *FBTransaction) (*BatchCompletion, error) {
	if err := b.Create(ctx); err != nil {
		return nil, err
	}

	execTx, err := b.resolveTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	var completion *BatchCompletion
	err = b.withConn(ctx, func() error {
		b.bufferState = batchBufferNeedsReset
		if err := b.st.conn.withWriteLockAndFlush(func() error {
			return b.st.conn.client.WriteMessage(&fbproto.OpBatchExec{
				StatementHandle:   b.st.Description.StatementHandle,
				TransactionHandle: execTx.Handle,
			})
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}

		completion, err = b.receiveCompletion(ctx)
		if completion != nil {
			// A completion consumes the entire queue, including skipped rows.
			// A later rollback or commit failure does not restore that queue.
			b.bufferState = batchBufferEmpty
		}
		return err
	})
	if completion.HasErrors() && ctx.Err() != nil {
		err = operationerror.Join(errBatchExecutionFailed, normalizeTimeoutError(ctx, ctx.Err()))
	}
	if err != nil || completion.HasErrors() {
		if rbErr := b.st.conn.rollbackOwnedInternalTx(ctx, execTx); rbErr != nil {
			if err == nil {
				err = errBatchExecutionFailed
			}
			err = operationerror.Join(err, fmt.Errorf("rollback internal tx: %w", rbErr))
		}
		return completion, err
	}

	if execTx == b.st.conn.internalTransaction && b.st.conn.config.Autocommit && !b.st.conn.explicitTransaction {
		commitCtx, cancel := cleanupContext(ctx)
		defer cancel()
		if err := b.st.conn.commitInternalTx(commitCtx, execTx); err != nil {
			return completion, err
		}
	}

	return completion, nil
}

// Cancel sends [fbproto.Op_batch_cancel] to the server. Per the Firebird protocol
// this discards every message that the server has accumulated for this
// batch but keeps the server-side batch context alive, so the same [Batch]
// object can be reused for new [Batch.Send] and [Batch.Execute] calls.
func (b *Batch) Cancel(ctx context.Context) error {
	if b == nil || b.state == batchStateClosed {
		return nil
	}
	if b.state != batchStateCreated {
		// Nothing was sent yet — there is nothing to cancel server-side.
		return nil
	}
	return b.sendReleaseOp(ctx, fbproto.Op_batch_cancel)
}

// Close sends [fbproto.Op_batch_rls] to the server, releasing the server-side
// batch context entirely. A confirmed release closes the batch even if the
// following sync fails. An unconfirmed Close can be retried while the connection
// remains usable.
func (b *Batch) Close(ctx context.Context) error {
	if b == nil || b.state == batchStateClosed {
		return nil
	}
	if b.state != batchStateCreated {
		b.markClosed()
		return nil
	}

	err := b.sendReleaseOp(ctx, fbproto.Op_batch_rls)
	if err == nil || b.st.conn.IsClosed() {
		b.markClosed()
	}
	return err
}

func (b *Batch) sendReleaseOp(ctx context.Context, op int) error {
	return b.withConn(ctx, func() error {
		if op == fbproto.Op_batch_cancel {
			b.bufferState = batchBufferNeedsReset
		}
		if err := b.st.conn.withWriteLockAndFlush(func() error {
			if err := b.st.conn.client.WriteMessage(&fbproto.OpBatchRelease{
				Kind:            op,
				StatementHandle: b.st.Description.StatementHandle,
			}); err != nil {
				return err
			}
			return b.st.conn.client.WriteMessage(&fbproto.OpPing{})
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		released, err := b.receiveTwoGenericResponses(ctx)
		if released {
			b.bufferState = batchBufferEmpty
			if op == fbproto.Op_batch_rls {
				b.markClosed()
			}
		}
		return err
	})
}

func (b *Batch) receiveGenericResponse(ctx context.Context) error {
	msg, err := b.st.conn.receiveMessage()
	if err != nil {
		b.st.conn.asyncCloseAfterRead(err)
		return normalizeTimeoutError(ctx, err)
	}
	switch m := msg.(type) {
	case *fbproto.OpResponse:
		return nil
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		b.st.conn.asyncClose()
		return fmt.Errorf("received unexpected batch sync response %T", msg)
	}
}

func (b *Batch) receiveTwoGenericResponses(ctx context.Context) (bool, error) {
	firstErr := b.receiveGenericResponse(ctx)

	if b.st.conn.IsClosed() {
		return firstErr == nil, firstErr
	}
	secondErr := b.receiveGenericResponse(ctx)
	return firstErr == nil, errors.Join(firstErr, secondErr)
}

func (b *Batch) withConn(ctx context.Context, fn func() error) (err error) {
	if err := b.st.conn.lock(); err != nil {
		return err
	}
	defer b.st.conn.unlock()
	defer func() {
		if err != nil && ctx.Err() != nil && !errors.Is(err, ctx.Err()) {
			err = operationerror.Join(err, normalizeTimeoutError(ctx, ctx.Err()))
		}
	}()

	if err := b.st.conn.beforeRequest(); err != nil {
		return err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		b.st.conn.contextWatcher.Watch(ctx)
		defer b.st.conn.contextWatcher.Unwatch()
	}

	return operationerror.WithRetry(normalizeTimeoutError(ctx, fn()), false)
}

func (b *Batch) receiveCompletion(ctx context.Context) (*BatchCompletion, error) {
	msg, err := b.st.conn.receiveMessage()
	if err != nil {
		b.st.conn.asyncCloseAfterRead(err)
		return nil, normalizeTimeoutError(ctx, err)
	}

	switch m := msg.(type) {
	case *fbproto.OpBatchCompletion:
		if m.StatementHandle != b.st.Description.StatementHandle {
			b.st.conn.asyncClose()
			return nil, fmt.Errorf("batch response statement handle %d, expected %d", m.StatementHandle, b.st.Description.StatementHandle)
		}
		return b.convertCompletion(m), nil
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	default:
		b.st.conn.asyncClose()
		return nil, fmt.Errorf("received unexpected batch response %T", msg)
	}
}

func (b *Batch) resolveTx(ctx context.Context, tx *FBTransaction) (*FBTransaction, error) {
	if tx != nil {
		if tx.Connection != b.st.conn {
			return nil, fmt.Errorf("batch transaction belongs to another connection")
		}
		if tx.Closed {
			return nil, ErrTransactionClosed
		}
		return tx, nil
	}
	if b.st.transaction != nil {
		if b.st.transaction.Closed {
			return nil, ErrTransactionClosed
		}
		return b.st.transaction, nil
	}
	return b.st.conn.ensureInternalTransaction(ctx, nil, false)
}

func (b *Batch) convertCompletion(c *fbproto.OpBatchCompletion) *BatchCompletion {
	res := &BatchCompletion{
		TotalRecords: uint32(c.TotalRecords),
		UpdateCounts: make([]int64, len(c.UpdateCounts)),
	}
	for i, n := range c.UpdateCounts {
		res.UpdateCounts[i] = int64(n)
	}
	for _, detailed := range c.DetailedErrors {
		err := ErrorResponseToFbError(&detailed.Error)
		res.Errors = append(res.Errors, BatchError{Row: detailed.Row, Err: err})
	}
	for _, row := range c.SimplifiedErrors {
		res.Errors = append(res.Errors, BatchError{Row: row})
	}
	return res
}

func (b *Batch) markClosed() {
	b.state = batchStateClosed
	b.bufferState = batchBufferEmpty
	if b.st != nil && b.st.activeBatch == b {
		b.st.activeBatch = nil
	}
}
