package fbconn

import (
	"context"
	"errors"

	"rdb.red-soft.ru/fbx/fbproto"
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
		return err
	}

	msg, err := tx.Connection.receiveMessage()
	if err != nil {
		tx.Connection.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}

	switch m := msg.(type) {
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	}

	// A rejected commit leaves the server transaction active. Keep its handle
	// until success so callers or Close can still roll it back.
	tx.Closed = true
	tx.Connection.clearInlineBlobCache(tx.Handle)
	delete(tx.Connection.activeTransactions, tx)
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
		return err
	}

	msg, err := tx.Connection.receiveMessage()
	if err != nil {
		tx.Connection.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}
	switch m := msg.(type) {
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	}

	tx.Closed = true
	tx.Connection.clearInlineBlobCache(tx.Handle)
	delete(tx.Connection.activeTransactions, tx)
	return nil
}
