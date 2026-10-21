package fbconn

import (
	"context"
	"errors"
	"fmt"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/internal/operationerror"
)

type FBTransaction struct {
	Connection *FbConn
	Handle     int
	Closed     bool
}

var ErrTransactionClosed = errors.New("transaction is closed")

func (tx *FBTransaction) Commit(ctx context.Context) error {
	if tx.Closed {
		return ErrTransactionClosed
	}
	var err error

	if err = tx.Connection.lock(); err != nil {
		return err
	}
	defer tx.Connection.unlock()

	if err = tx.Connection.beforeRequest(); err != nil {
		return err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		tx.Connection.contextWatcher.Watch(ctx)
		defer tx.Connection.contextWatcher.Unwatch()
	}

	opCommit := fbproto.OpCommit{
		TransactionHandle: tx.Handle,
	}
	if err = tx.Connection.withWriteLockAndFlush(func() error {
		return tx.Connection.client.WriteMessage(&opCommit)
	}); err != nil {
		tx.Connection.asyncClose()
		return operationerror.WithRetry(err, false)
	}

	msg, err := tx.Connection.receiveMessage()
	if err != nil {
		tx.Connection.asyncCloseAfterRead(err)
		return operationerror.WithRetry(normalizeTimeoutError(ctx, err), false)
	}

	switch m := msg.(type) {
	case *fbproto.OpResponse:
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		tx.Connection.asyncClose()
		return fmt.Errorf("unexpected transaction response: %T", msg)
	}

	// A rejected commit leaves the server transaction active. Keep its handle
	// until success so callers or Close can still roll it back.
	tx.markClosed()
	return nil
}

func (tx *FBTransaction) Rollback(ctx context.Context) error {
	if tx.Closed {
		return ErrTransactionClosed
	}
	var err error

	if err = tx.Connection.lock(); err != nil {
		return err
	}
	defer tx.Connection.unlock()

	if err = tx.Connection.beforeRequest(); err != nil {
		return err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		tx.Connection.contextWatcher.Watch(ctx)
		defer tx.Connection.contextWatcher.Unwatch()
	}

	opRollback := fbproto.OpRollback{
		TransactionHandle: tx.Handle,
	}
	if err = tx.Connection.withWriteLockAndFlush(func() error {
		return tx.Connection.client.WriteMessage(&opRollback)
	}); err != nil {
		tx.Connection.asyncClose()
		return operationerror.WithRetry(err, false)
	}

	msg, err := tx.Connection.receiveMessage()
	if err != nil {
		tx.Connection.asyncCloseAfterRead(err)
		return operationerror.WithRetry(normalizeTimeoutError(ctx, err), false)
	}
	switch m := msg.(type) {
	case *fbproto.OpResponse:
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		tx.Connection.asyncClose()
		return fmt.Errorf("unexpected transaction response: %T", msg)
	}

	tx.markClosed()
	return nil
}

func (tx *FBTransaction) markClosed() {
	tx.Closed = true
	tx.Connection.clearInlineBlobCache(tx.Handle)
	delete(tx.Connection.activeTransactions, tx)
}

// recordSQLTransaction applies the handle returned by a successful Execute.
// A zero handle ends the supplied transaction; a new handle without an input
// transaction must be retained for pool validation and connection cleanup.
func (fbConn *FbConn) recordSQLTransaction(tx *FBTransaction, handle int) {
	if tx != nil {
		if handle == 0 {
			tx.markClosed()
		}
		return
	}
	if handle != 0 {
		sqlTx := &FBTransaction{Connection: fbConn, Handle: handle}
		fbConn.activeTransactions[sqlTx] = struct{}{}
	}
}
