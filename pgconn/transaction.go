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

	opTransaction := fbproto.OpCommit{
		TransactionHandle: tx.Handle,
	}
	if err = tx.Connection.client.WriteMessage(&opTransaction); err != nil {
		return err
	}

	if err = tx.Connection.writer.Flush(); err != nil {
		return err
	}

	if _, err = tx.Connection.receiveResponse(); err == nil {
		return err
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

	opTransaction := fbproto.OpRollback{
		TransactionHandle: tx.Handle,
	}
	if err = tx.Connection.client.WriteMessage(&opTransaction); err != nil {
		return err
	}

	if err = tx.Connection.writer.Flush(); err != nil {
		return err
	}

	if _, err = tx.Connection.receiveResponse(); err == nil {
		return err
	}

	return nil
}
