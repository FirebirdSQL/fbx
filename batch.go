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

package fbx

import (
	"context"
	"fmt"
	"strings"

	"rdb.red-soft.ru/fbx/fbconn"
)

const (
	batchPreparePrefix            = "batch "
	defaultPreparedBatchFlushSize = 128 * 1024
)

type BatchResult struct {
	Affected int64
}

type PreparedBatch struct {
	conn *Conn
	tx   *dbTx

	name string
	sql  string

	sd *fbconn.StatementDescription
	st *fbconn.Statement

	activeSt        *fbconn.Statement
	releaseActiveSt bool
	lb              *fbconn.Batch

	encodedRows    [][]byte
	pendingBytes   int
	autoFlushBytes int
	closed         bool

	eqb ExtendedQueryBuilder
}

// Add appends one row to the batch buffer.
func (b *PreparedBatch) Add(args ...any) error {
	return b.addEncoded(context.Background(), args, false)
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

func (b *PreparedBatch) addEncoded(ctx context.Context, args []any, allowAutoFlush bool) error {
	if b == nil {
		return fmt.Errorf("prepared batch is nil")
	}
	if b.closed {
		return fmt.Errorf("prepared batch is closed")
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
	if len(b.encodedRows) == 0 {
		return nil
	}

	st, err := b.batchStatement(ctx)
	if err != nil {
		return err
	}
	if b.lb == nil {
		b.lb = st.NewBatch(fbconn.BatchConfig{
			RecordCounts: true,
			DetailedErrs: 1,
		})
	}

	if err := b.lb.Send(ctx, b.encodedRows); err != nil {
		_ = b.releaseWireResources(ctx)
		return err
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
	closeErr := b.lb.Close(ctx)
	b.lb = nil
	if err != nil {
		return BatchResult{}, err
	}
	releaseErr := b.releaseBatchStatement(ctx)
	result := BatchResult{Affected: completion.Affected()}
	if closeErr != nil {
		return result, closeErr
	}
	if releaseErr != nil {
		return result, releaseErr
	}
	if completion.HasErrors() {
		return result, batchCompletionError(completion)
	}
	return result, nil
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

	st, err := b.conn.internalPrepare(ctx, nil, "", b.sql)
	if err != nil {
		return nil, err
	}
	b.activeSt = st
	b.releaseActiveSt = true
	return st, nil
}

func (b *PreparedBatch) releaseBatchStatement(ctx context.Context) error {
	if b.activeSt == nil {
		return nil
	}
	st := b.activeSt
	release := b.releaseActiveSt
	b.activeSt = nil
	b.releaseActiveSt = false
	if release {
		return st.Deallocate(ctx)
	}
	return nil
}

func (b *PreparedBatch) releaseWireResources(ctx context.Context) error {
	var batchErr error
	if b.lb != nil {
		batchErr = b.lb.Close(ctx)
		b.lb = nil
	}
	stErr := b.releaseBatchStatement(ctx)
	if batchErr != nil {
		return batchErr
	}
	return stErr
}

// Reset clears queued rows that have not been sent yet.
func (b *PreparedBatch) Reset() {
	if b == nil {
		return
	}
	b.encodedRows = b.encodedRows[:0]
	b.pendingBytes = 0
}

// Close releases batch resources.
func (b *PreparedBatch) Close(ctx context.Context) error {
	if b == nil || b.closed {
		return nil
	}
	b.closed = true
	b.Reset()

	wireErr := b.releaseWireResources(ctx)

	var stErr error
	if b.st != nil && b.activeSt != b.st {
		stErr = b.st.Deallocate(ctx)
		b.st = nil
	}
	if wireErr != nil {
		return wireErr
	}
	return stErr
}

func batchCompletionError(completion *fbconn.BatchCompletion) error {
	if completion == nil || len(completion.Errors) == 0 {
		return nil
	}
	first := completion.Errors[0]
	if first.Err != nil {
		return fmt.Errorf("batch failed at row %d: %w", first.Row, first.Err)
	}
	return fmt.Errorf("batch failed at row %d", first.Row)
}
