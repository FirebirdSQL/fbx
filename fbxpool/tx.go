package fbxpool

import (
	"context"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
)

// Tx represents a database transaction acquired from a Pool.
type Tx struct {
	t fbx.Tx
	c *Conn
}

// Begin starts a pseudo nested transaction implemented with a savepoint.
func (tx *Tx) Begin(ctx context.Context) (fbx.Tx, error) {
	return tx.t.Begin(ctx)
}

// Commit commits the transaction and returns the associated connection back to the Pool. Commit will return an error
// where errors.Is(ErrTxClosed) is true if the Tx is already closed, but is otherwise safe to call multiple times. If
// the commit fails with a rollback status (e.g. the transaction was already in a broken state) then ErrTxCommitRollback
// will be returned.
func (tx *Tx) Commit(ctx context.Context) error {
	err := tx.t.Commit(ctx)
	if tx.c != nil {
		tx.c.Release()
		tx.c = nil
	}
	return err
}

// Rollback rolls back the transaction and returns the associated connection back to the Pool. Rollback will return
// where an error where errors.Is(ErrTxClosed) is true if the Tx is already closed, but is otherwise safe to call
// multiple times. Hence, defer tx.Rollback() is safe even if tx.Commit() will be called first in a non-error condition.
func (tx *Tx) Rollback(ctx context.Context) error {
	err := tx.t.Rollback(ctx)
	if tx.c != nil {
		tx.c.Release()
		tx.c = nil
	}
	return err
}

// Prepare creates a prepared statement with name and sql. If the name is empty,
// an anonymous prepared statement will be used. sql can contain placeholders
// for bound parameters. These placeholders are referenced positionally as $1, $2, etc.
//
// Prepare is idempotent; i.e. it is safe to call Prepare multiple times with the same
// name and sql arguments. This allows a code path to Prepare and Query/Exec without
// needing to first check whether the statement has already been prepared.
func (tx *Tx) Prepare(ctx context.Context, name, sql string) (*fbconn.StatementDescription, error) {
	return tx.t.Prepare(ctx, name, sql)
}

// PrepareBatch creates a batch object inside the transaction.
func (tx *Tx) PrepareBatch(ctx context.Context, sql string) (*fbx.PreparedBatch, error) {
	return tx.t.PrepareBatch(ctx, sql)
}

// PrepareBatchWithOptions creates a batch object inside the transaction with additional batch options.
func (tx *Tx) PrepareBatchWithOptions(ctx context.Context, sql string, opts fbx.BatchOptions) (*fbx.PreparedBatch, error) {
	return tx.t.PrepareBatchWithOptions(ctx, sql, opts)
}

// Flush sends queued rows for a named batch without executing it.
func (tx *Tx) Flush(ctx context.Context, name string) error {
	return tx.t.Flush(ctx, name)
}

// ExecBatch flushes queued rows for a named batch, executes it, and returns the result.
func (tx *Tx) ExecBatch(ctx context.Context, name string) (fbx.BatchResult, error) {
	return tx.t.ExecBatch(ctx, name)
}

// CancelBatch discards every row queued for a named batch — both the local
// buffer and any rows already flushed to the server.
func (tx *Tx) CancelBatch(ctx context.Context, name string) error {
	return tx.t.CancelBatch(ctx, name)
}

func (tx *Tx) Exec(ctx context.Context, sql string, arguments ...any) (fbconn.QueryStatus, error) {
	return tx.t.Exec(ctx, sql, arguments...)
}

func (tx *Tx) Query(ctx context.Context, sql string, args ...any) (fbx.Rows, error) {
	return tx.t.Query(ctx, sql, args...)
}

func (tx *Tx) QueryRow(ctx context.Context, sql string, args ...any) fbx.Row {
	return tx.t.QueryRow(ctx, sql, args...)
}

func (tx *Tx) Conn() *fbx.Conn {
	return tx.t.Conn()
}
