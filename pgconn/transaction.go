package pgconn

import (
	"context"
	"errors"

	"github.com/red-soft/fbx/fbproto"
)

type FBTransaction struct {
	Connection *PgConn
	Handle     int
	Closed     bool
}

var ErrTransactionClosed = errors.New("transaction is closed")

func (tx *FBTransaction) Commit(ctx context.Context) error {
	if tx.Closed {
		return ErrTransactionClosed
	}
	tx.Closed = true
	var err error

	if err = tx.Connection.lock(); err != nil {
		return err
	}
	defer tx.Connection.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		tx.Connection.contextWatcher.Watch(ctx)
		defer tx.Connection.contextWatcher.Unwatch()
	}

	delete(tx.Connection.activeTransactions, tx)

	opCommit := fbproto.OpCommit{
		TransactionHandle: tx.Handle,
	}
	if err = tx.Connection.client.WriteMessage(&opCommit); err != nil {
		tx.Connection.asyncClose()
		return err
	}

	if err = tx.Connection.writer.Flush(); err != nil {
		tx.Connection.asyncClose()
		return err
	}

	msg, err := tx.Connection.receiveMessage()
	if err != nil {
		tx.Connection.asyncClose()
		return err
	}

	switch m := msg.(type) {
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	}

	return nil
}

func (tx *FBTransaction) Rollback(ctx context.Context) error {
	if tx.Closed {
		return ErrTransactionClosed
	}
	tx.Closed = true
	var err error

	if err = tx.Connection.lock(); err != nil {
		return err
	}
	defer tx.Connection.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		tx.Connection.contextWatcher.Watch(ctx)
		defer tx.Connection.contextWatcher.Unwatch()
	}

	delete(tx.Connection.activeTransactions, tx)

	opRollback := fbproto.OpRollback{
		TransactionHandle: tx.Handle,
	}
	if err = tx.Connection.client.WriteMessage(&opRollback); err != nil {
		tx.Connection.asyncClose()
		return err
	}

	if err = tx.Connection.writer.Flush(); err != nil {
		tx.Connection.asyncClose()
		return err
	}

	msg, err := tx.Connection.receiveMessage()
	if err != nil {
		tx.Connection.asyncClose()
		return err
	}
	switch m := msg.(type) {
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	}

	return nil
}
