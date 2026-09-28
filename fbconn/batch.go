package fbconn

import (
	"context"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
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
	TotalRecords uint32
	UpdateCounts []int64
	Errors       []BatchError
}

func (c *BatchCompletion) HasErrors() bool {
	return c != nil && len(c.Errors) > 0
}

func (c *BatchCompletion) Affected() int64 {
	if c == nil {
		return 0
	}
	if len(c.UpdateCounts) == 0 {
		return int64(c.TotalRecords)
	}
	var affected int64
	for _, n := range c.UpdateCounts {
		if n > 0 {
			affected += n
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

type Batch struct {
	st    *Statement
	state batchState
	cfg   BatchConfig
	err   error
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
			b.st.conn.asyncClose()
			return err
		}

		if err := b.st.conn.client.WriteMessage(&fbproto.OpBatchCreate{
			StatementHandle: b.st.Description.StatementHandle,
			BatchBLR:        batchBLR,
			MessageLength:   fbproto.CalculateBatchMessageLength(b.st.Description.StatementInfo.ParametersDescriptors),
			ParameterBuffer: parameterBuffer,
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		if err := b.st.conn.client.WriteMessage(&fbproto.OpPing{}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		if err := b.st.conn.flushWithPotentialWriteReadDeadlock(); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		return b.receiveTwoGenericResponses(ctx)
	})
}

func (b *Batch) Create(ctx context.Context) error {
	if err := b.checkCreate(); err != nil {
		return err
	}
	if b.state == batchStateCreated {
		return nil
	}

	if err := b.createOnWire(ctx, b.cfg.parameterBuffer()); err != nil {
		return err
	}

	b.state = batchStateCreated
	return nil
}

func (b *Batch) Send(ctx context.Context, rows [][]byte) error {
	if len(rows) == 0 {
		return nil
	}
	if err := b.Create(ctx); err != nil {
		return err
	}

	return b.withConn(ctx, func() error {
		if err := b.st.conn.client.WriteMessage(&fbproto.OpBatchMsg{
			StatementHandle: b.st.Description.StatementHandle,
			Rows:            rows,
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		if err := b.writeBatchSyncMessage(); err != nil {
			b.st.conn.asyncClose()
			return err
		}

		if err := b.st.conn.flushWithPotentialWriteReadDeadlock(); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		return b.receiveTwoGenericResponses(ctx)
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
		if err := b.st.conn.client.WriteMessage(&fbproto.OpBatchExec{
			StatementHandle:   b.st.Description.StatementHandle,
			TransactionHandle: execTx.Handle,
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		if err := b.st.conn.flushWithPotentialWriteReadDeadlock(); err != nil {
			b.st.conn.asyncClose()
			return err
		}

		completion, err = b.receiveCompletion(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	if execTx == b.st.conn.internalTransaction && b.st.conn.config.Autocommit && !b.st.conn.explicitTransaction {
		if completion.HasErrors() {
			if err := b.st.conn.rollbackInternalTxIfAutocommitOnError(ctx, execTx, errBatchExecutionFailed); err != nil {
				return completion, errors.Join(errBatchExecutionFailed, fmt.Errorf("rollback internal tx: %w", err))
			}
		} else if err := b.st.conn.commitInternalTx(context.Background(), execTx); err != nil {
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
// batch context entirely. The [Batch] must not be used after [Batch.Close]
// returns.
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
		if err := b.st.conn.client.WriteMessage(&fbproto.OpBatchRelease{
			Kind:            op,
			StatementHandle: b.st.Description.StatementHandle,
		}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		if err := b.st.conn.client.WriteMessage(&fbproto.OpPing{}); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		if err := b.st.conn.flushWithPotentialWriteReadDeadlock(); err != nil {
			b.st.conn.asyncClose()
			return err
		}
		return b.receiveTwoGenericResponses(ctx)
	})
}

func (b *Batch) receiveGenericResponse(ctx context.Context) error {
	msg, err := b.st.conn.receiveMessage()
	if err != nil {
		b.st.conn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}
	switch m := msg.(type) {
	case *fbproto.OpResponse:
		return nil
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		return fmt.Errorf("received unexpected batch sync response %T", msg)
	}
}

func (b *Batch) receiveTwoGenericResponses(ctx context.Context) error {
	firstErr := b.receiveGenericResponse(ctx)
	secondErr := b.receiveGenericResponse(ctx)
	return errors.Join(firstErr, secondErr)
}

func (b *Batch) withConn(ctx context.Context, fn func() error) error {
	if err := b.st.conn.lock(); err != nil {
		return err
	}
	defer b.st.conn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		b.st.conn.contextWatcher.Watch(ctx)
		defer b.st.conn.contextWatcher.Unwatch()
	}

	return fn()
}

func (b *Batch) receiveCompletion(ctx context.Context) (*BatchCompletion, error) {
	for {
		msg, err := b.st.conn.receiveMessage()
		if err != nil {
			b.st.conn.asyncClose()
			return nil, normalizeTimeoutError(ctx, err)
		}

		switch m := msg.(type) {
		case *fbproto.OpBatchCompletion:
			return b.convertCompletion(m), nil
		case *fbproto.OpResponse:
			continue
		case *fbproto.ErrorResponse:
			return nil, ErrorResponseToFbError(m)
		default:
			return nil, fmt.Errorf("received unexpected batch response %T", msg)
		}
	}
}

func (b *Batch) resolveTx(ctx context.Context, tx *FBTransaction) (*FBTransaction, error) {
	if tx != nil {
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
	if b.st != nil && b.st.activeBatch == b {
		b.st.activeBatch = nil
	}
}
