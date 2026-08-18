package fbx

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

// TxIsoLevel is the transaction isolation level ([fbx.Serializable], [fbx.RepeatableRead], [fbx.ReadCommitted])
type TxIsoLevel string

// Transaction isolation levels
const (
	Serializable   TxIsoLevel = "serializable"
	RepeatableRead TxIsoLevel = "repeatable read"
	ReadCommitted  TxIsoLevel = "read committed"
)

// TxAccessMode is the transaction access mode ([fbx.ReadWrite], [fbx.ReadOnly])
type TxAccessMode string

// Transaction access modes
const (
	ReadWrite TxAccessMode = "read write"
	ReadOnly  TxAccessMode = "read only"
)

// TxOptions are transaction modes within a transaction block
type TxOptions struct {
	IsoLevel   TxIsoLevel
	AccessMode TxAccessMode
	// LockTimeout controls lock wait timeout for WAIT transactions.
	// Zero means server/database default behavior.
	// Value must be a positive whole number of seconds in range [1..32767].
	LockTimeout time.Duration
}

var ErrTxClosed = errors.New("tx is closed")

// ErrTxCommitRollback indicates that Commit() did not commit changes and the transaction was rolled back instead
var ErrTxCommitRollback = errors.New("commit unexpectedly resulted in rollback")

// Begin starts a transaction. Unlike database/sql, the context only affects the begin command. i.e. there is no
// auto-rollback on context cancellation.
func (c *Conn) Begin(ctx context.Context) (Tx, error) {
	return c.BeginTx(ctx, TxOptions{})
}

// BeginTx starts a transaction with txOptions determining the transaction mode. Unlike database/sql, the context only
// affects the begin command. i.e. there is no auto-rollback on context cancellation.
func (c *Conn) BeginTx(ctx context.Context, txOptions TxOptions) (Tx, error) {
	tpb, err := txOptions.GetTPB()
	if err != nil {
		return nil, err
	}

	return c.beginTxTPB(ctx, tpb)
}

// BeginTxTPB starts a transaction using a raw Firebird transaction parameter block.
// The TPB is passed to the server as-is.
func (c *Conn) BeginTxTPB(ctx context.Context, tpb []byte) (Tx, error) {
	txTPB := make([]byte, len(tpb))
	copy(txTPB, tpb)

	return c.beginTxTPB(ctx, txTPB)
}

func (c *Conn) beginTxTPB(ctx context.Context, tpb []byte) (Tx, error) {
	fbTx, err := c.fbConn.StartTransactionTPB(ctx, tpb)
	if err != nil {
		return nil, err
	}

	return &dbTx{
		conn: c,
		tx:   fbTx,
	}, nil
}

func (txOptions TxOptions) GetTPB() ([]byte, error) {
	lockTimeoutSeconds, err := normalizeTxLockTimeout(txOptions.LockTimeout)
	if err != nil {
		return nil, err
	}

	// If no options are specified, use server / database defaults. This matches Firebird semantics and keeps
	// database/sql LevelDefault behaving as expected (i.e. do not silently force READ COMMITTED).
	if txOptions.AccessMode == "" && txOptions.IsoLevel == "" && lockTimeoutSeconds == 0 {
		return nil, nil
	}

	tpb := fbproto.NewTPBWriterV3()
	mode := txOptions.AccessMode
	level := txOptions.IsoLevel
	if mode == "" {
		mode = ReadWrite
	}
	if level == "" {
		level = ReadCommitted
	}

	if txOptions.AccessMode != "" || txOptions.IsoLevel != "" {
		switch mode {
		case ReadOnly:
			tpb.PutTag(fbproto.Isc_tpb_read)
		case ReadWrite:
			tpb.PutTag(fbproto.Isc_tpb_write)
		default:
			return nil, fmt.Errorf("unknown transaction access mode: %s", mode)
		}

		switch level {
		case Serializable:
			tpb.PutTag(fbproto.Isc_tpb_consistency)
		case RepeatableRead:
			tpb.PutTag(fbproto.Isc_tpb_concurrency)
		case ReadCommitted:
			tpb.PutTag(fbproto.Isc_tpb_read_committed)
			if txOptions.AccessMode == ReadWrite {
				tpb.PutTag(fbproto.Isc_tpb_no_rec_version)
			} else {
				tpb.PutTag(fbproto.Isc_tpb_rec_version)
			}
		default:
			return nil, fmt.Errorf("unknown transaction iso level: %s", level)
		}
	}

	if lockTimeoutSeconds > 0 {
		tpb.PutTag(fbproto.Isc_tpb_wait)
		tpb.PutInt32(fbproto.Isc_tpb_lock_timeout, int32(lockTimeoutSeconds))
	} else if txOptions.AccessMode != "" || txOptions.IsoLevel != "" {
		tpb.PutTag(fbproto.Isc_tpb_wait)
	}

	return tpb.Bytes(), nil
}

func normalizeTxLockTimeout(lockTimeout time.Duration) (int, error) {
	if lockTimeout == 0 {
		return 0, nil
	}
	if lockTimeout < 0 {
		return 0, fmt.Errorf("lock timeout must be >= 0")
	}
	seconds := int(lockTimeout.Seconds())
	if seconds < 1 || seconds > 32767 {
		return 0, fmt.Errorf("lock timeout must be in range [1..32767] seconds")
	}
	return seconds, nil
}

// Tx represents a database transaction.
//
// Tx is an interface instead of a struct to enable connection pools to be implemented without relying on internal fbx
// state, to support pseudo-nested transactions with savepoints, and to allow tests to mock transactions. However,
// adding a method to an interface is technically a breaking change. If new methods are added to Conn it may be
// desirable to add them to Tx as well. Because of this the Tx interface is partially excluded from semantic version
// requirements. Methods will not be removed or changed, but new methods may be added.
type Tx interface {
	// Begin starts a pseudo nested transaction.
	Begin(ctx context.Context) (Tx, error)

	// Commit commits the transaction if this is a real transaction or releases the savepoint if this is a pseudo nested
	// transaction. Commit will return an error where errors.Is(ErrTxClosed) is true if the Tx is already closed, but is
	// otherwise safe to call multiple times. If the commit fails with a rollback status (e.g. the transaction was already
	// in a broken state) then an error where errors.Is(ErrTxCommitRollback) is true will be returned.
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction if this is a real transaction or rolls back to the savepoint if this is a
	// pseudo nested transaction. Rollback will return an error where errors.Is(ErrTxClosed) is true if the Tx is already
	// closed, but is otherwise safe to call multiple times. Hence, a defer tx.Rollback() is safe even if tx.Commit() will
	// be called first in a non-error condition. Any other failure of a real transaction will result in the connection
	// being closed.
	Rollback(ctx context.Context) error

	Prepare(ctx context.Context, name, sql string) (*fbconn.StatementDescription, error)
	// PrepareBatch creates a batch object inside the transaction.
	//
	// Executing the batch does not commit or roll back the transaction.
	PrepareBatch(ctx context.Context, sql string) (*PreparedBatch, error)
	// PrepareBatchWithOptions creates a batch object inside the transaction with
	// additional batch options.
	//
	// Executing the batch does not commit or roll back the transaction.
	PrepareBatchWithOptions(ctx context.Context, sql string, opts BatchOptions) (*PreparedBatch, error)
	// Flush sends queued rows for a named batch without executing it.
	Flush(ctx context.Context, name string) error
	// ExecBatch flushes queued rows for a named batch, executes it, and returns
	// the result.
	//
	// ExecBatch does not commit or roll back the transaction.
	ExecBatch(ctx context.Context, name string) (BatchResult, error)
	// CancelBatch discards every row queued for a named batch — both the local
	// buffer and any rows already flushed to the server. The prepared batch
	// statement is kept and can be reused.
	CancelBatch(ctx context.Context, name string) error

	Exec(ctx context.Context, sql string, arguments ...any) (commandTag fbconn.QueryStatus, err error)
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row

	// Conn returns the underlying *Conn that on which this transaction is executing.
	Conn() *Conn
}

// dbTx represents a database transaction.
//
// All dbTx methods return ErrTxClosed if Commit or Rollback has already been
// called on the dbTx.
type dbTx struct {
	conn         *Conn
	savepointNum int64
	closed       bool
	tx           *fbconn.FBTransaction
}

// Begin starts a pseudo nested transaction implemented with a savepoint.
func (tx *dbTx) Begin(ctx context.Context) (Tx, error) {
	if tx.closed {
		return nil, ErrTxClosed
	}

	tx.savepointNum++
	_, err := tx.Exec(ctx, "savepoint sp_"+strconv.FormatInt(tx.savepointNum, 10))
	if err != nil {
		return nil, err
	}

	return &dbSimulatedNestedTx{tx: tx, savepointNum: tx.savepointNum}, nil
}

// Commit commits the transaction.
func (tx *dbTx) Commit(ctx context.Context) error {
	if tx.closed {
		return ErrTxClosed
	}

	cleanupErr := tx.conn.closePreparedBatchesForTx(ctx, tx)
	err := tx.tx.Commit(ctx)
	if err != nil {
		tx.closed = tx.tx.Closed || tx.conn.IsClosed()
		return errors.Join(cleanupErr, err)
	}

	tx.closed = true
	return cleanupErr
}

// Rollback rolls back the transaction. Rollback will return ErrTxClosed if the
// Tx is already closed, but is otherwise safe to call multiple times. Hence, a
// defer tx.Rollback() is safe even if tx.Commit() will be called first in a
// non-error condition.
func (tx *dbTx) Rollback(ctx context.Context) error {
	if tx.closed {
		return ErrTxClosed
	}

	cleanupErr := tx.conn.closePreparedBatchesForTx(ctx, tx)
	err := tx.tx.Rollback(ctx)
	tx.closed = true
	if err != nil {
		// A rollback failure leaves the connection in an undefined state.
		tx.conn.die()
		return errors.Join(cleanupErr, err)
	}

	return cleanupErr
}

// Exec delegates to the underlying *Conn
func (tx *dbTx) Exec(ctx context.Context, sql string, arguments ...any) (commandTag fbconn.QueryStatus, err error) {
	if tx.closed {
		return fbconn.QueryStatus{}, ErrTxClosed
	}

	return tx.conn.execTx(ctx, tx, sql, arguments...)
}

// Prepare delegates to the underlying *Conn
func (tx *dbTx) Prepare(ctx context.Context, name, sql string) (*fbconn.StatementDescription, error) {
	if tx.closed {
		return nil, ErrTxClosed
	}

	return tx.conn.prepareTx(ctx, tx, name, sql)
}

// PrepareBatch creates a batch object inside the transaction.
//
// Executing the batch does not commit or roll back the transaction.
func (tx *dbTx) PrepareBatch(ctx context.Context, sql string) (*PreparedBatch, error) {
	return tx.PrepareBatchWithOptions(ctx, sql, BatchOptions{})
}

// PrepareBatchWithOptions creates a batch object inside the transaction with
// additional batch options.
//
// Executing the batch does not commit or roll back the transaction.
func (tx *dbTx) PrepareBatchWithOptions(ctx context.Context, sql string, opts BatchOptions) (*PreparedBatch, error) {
	if tx.closed {
		return nil, ErrTxClosed
	}

	return tx.conn.prepareBatchTx(ctx, tx, sql, opts)
}

// Query delegates to the underlying *Conn
func (tx *dbTx) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	if tx.closed {
		// Because checking for errors can be deferred to the *Rows, build one with the error
		err := ErrTxClosed
		return &baseRows{closed: true, err: err}, err
	}

	return tx.conn.queryTx(ctx, tx, sql, args...)
}

// QueryRow delegates to the underlying *Conn
func (tx *dbTx) QueryRow(ctx context.Context, sql string, args ...any) Row {
	rows, _ := tx.Query(ctx, sql, args...)
	return (*connRow)(rows.(*baseRows))
}

func (tx *dbTx) Conn() *Conn {
	return tx.conn
}

// Flush sends queued rows for a named batch without executing it.
func (tx *dbTx) Flush(ctx context.Context, name string) error {
	if tx.closed {
		return ErrTxClosed
	}
	return tx.conn.flushTx(ctx, tx, name)
}

// ExecBatch flushes queued rows for a named batch, executes it, and returns the result.
//
// ExecBatch does not commit or roll back the transaction.
func (tx *dbTx) ExecBatch(ctx context.Context, name string) (BatchResult, error) {
	if tx.closed {
		return BatchResult{}, ErrTxClosed
	}
	return tx.conn.execBatchTx(ctx, tx, name)
}

// CancelBatch discards every row queued for a named batch — both the local
// buffer and any rows already flushed to the server. The prepared batch
// statement is kept and can be reused.
func (tx *dbTx) CancelBatch(ctx context.Context, name string) error {
	if tx.closed {
		return ErrTxClosed
	}
	return tx.conn.cancelBatchTx(ctx, tx, name)
}

// dbSimulatedNestedTx represents a simulated nested transaction implemented by a savepoint.
type dbSimulatedNestedTx struct {
	tx           Tx
	savepointNum int64
	closed       bool
}

// Begin starts a pseudo nested transaction implemented with a savepoint.
func (sp *dbSimulatedNestedTx) Begin(ctx context.Context) (Tx, error) {
	if sp.closed {
		return nil, ErrTxClosed
	}

	return sp.tx.Begin(ctx)
}

// Commit releases the savepoint essentially committing the pseudo nested transaction.
func (sp *dbSimulatedNestedTx) Commit(ctx context.Context) error {
	if sp.closed {
		return ErrTxClosed
	}

	_, err := sp.Exec(ctx, "release savepoint sp_"+strconv.FormatInt(sp.savepointNum, 10))
	sp.closed = true
	return err
}

// Rollback rolls back to the savepoint essentially rolling back the pseudo nested transaction. Rollback will return
// ErrTxClosed if the dbSavepoint is already closed, but is otherwise safe to call multiple times. Hence, a defer sp.Rollback()
// is safe even if sp.Commit() will be called first in a non-error condition.
func (sp *dbSimulatedNestedTx) Rollback(ctx context.Context) error {
	if sp.closed {
		return ErrTxClosed
	}

	_, err := sp.Exec(ctx, "rollback to savepoint sp_"+strconv.FormatInt(sp.savepointNum, 10))
	sp.closed = true
	return err
}

// Exec delegates to the underlying Tx
func (sp *dbSimulatedNestedTx) Exec(ctx context.Context, sql string, arguments ...any) (commandTag fbconn.QueryStatus, err error) {
	if sp.closed {
		return fbconn.QueryStatus{}, ErrTxClosed
	}

	return sp.tx.Exec(ctx, sql, arguments...)
}

// Prepare delegates to the underlying Tx
func (sp *dbSimulatedNestedTx) Prepare(ctx context.Context, name, sql string) (*fbconn.StatementDescription, error) {
	if sp.closed {
		return nil, ErrTxClosed
	}

	return sp.tx.Prepare(ctx, name, sql)
}

// PrepareBatch creates a batch object inside the transaction.
//
// Executing the batch does not commit or roll back the transaction.
func (sp *dbSimulatedNestedTx) PrepareBatch(ctx context.Context, sql string) (*PreparedBatch, error) {
	return sp.PrepareBatchWithOptions(ctx, sql, BatchOptions{})
}

// PrepareBatchWithOptions creates a batch object inside the transaction with
// additional batch options.
//
// Executing the batch does not commit or roll back the transaction.
func (sp *dbSimulatedNestedTx) PrepareBatchWithOptions(ctx context.Context, sql string, opts BatchOptions) (*PreparedBatch, error) {
	if sp.closed {
		return nil, ErrTxClosed
	}

	return sp.tx.PrepareBatchWithOptions(ctx, sql, opts)
}

// Flush sends queued rows for a named batch without executing it.
func (sp *dbSimulatedNestedTx) Flush(ctx context.Context, name string) error {
	if sp.closed {
		return ErrTxClosed
	}
	return sp.tx.Flush(ctx, name)
}

// ExecBatch flushes queued rows for a named batch, executes it, and returns the result.
//
// ExecBatch does not commit or roll back the transaction.
func (sp *dbSimulatedNestedTx) ExecBatch(ctx context.Context, name string) (BatchResult, error) {
	if sp.closed {
		return BatchResult{}, ErrTxClosed
	}
	return sp.tx.ExecBatch(ctx, name)
}

// CancelBatch discards every row queued for a named batch — both the local
// buffer and any rows already flushed to the server. The prepared batch
// statement is kept and can be reused.
func (sp *dbSimulatedNestedTx) CancelBatch(ctx context.Context, name string) error {
	if sp.closed {
		return ErrTxClosed
	}
	return sp.tx.CancelBatch(ctx, name)
}

// Query delegates to the underlying Tx
func (sp *dbSimulatedNestedTx) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	if sp.closed {
		// Because checking for errors can be deferred to the *Rows, build one with the error
		err := ErrTxClosed
		return &baseRows{closed: true, err: err}, err
	}

	return sp.tx.Query(ctx, sql, args...)
}

// QueryRow delegates to the underlying Tx
func (sp *dbSimulatedNestedTx) QueryRow(ctx context.Context, sql string, args ...any) Row {
	rows, _ := sp.Query(ctx, sql, args...)
	return (*connRow)(rows.(*baseRows))
}

func (sp *dbSimulatedNestedTx) Conn() *Conn {
	return sp.tx.Conn()
}

// BeginFunc calls Begin on db and then calls fn. If fn does not return an error then it calls Commit on db. If fn
// returns an error it calls Rollback on db. The context will be used when executing the transaction control statements
// (BEGIN, ROLLBACK, and COMMIT) but does not otherwise affect the execution of fn.
func BeginFunc(
	ctx context.Context,
	db interface {
		Begin(ctx context.Context) (Tx, error)
	},
	fn func(Tx) error,
) (err error) {
	var tx Tx
	tx, err = db.Begin(ctx)
	if err != nil {
		return err
	}

	return beginFuncExec(ctx, tx, fn)
}

// BeginTxFunc calls BeginTx on db and then calls fn. If fn does not return an error then it calls Commit on db. If fn
// returns an error it calls Rollback on db. The context will be used when executing the transaction control statements
// (BEGIN, ROLLBACK, and COMMIT) but does not otherwise affect the execution of fn.
func BeginTxFunc(
	ctx context.Context,
	db interface {
		BeginTx(ctx context.Context, txOptions TxOptions) (Tx, error)
	},
	txOptions TxOptions,
	fn func(Tx) error,
) (err error) {
	var tx Tx
	tx, err = db.BeginTx(ctx, txOptions)
	if err != nil {
		return err
	}

	return beginFuncExec(ctx, tx, fn)
}

func beginFuncExec(ctx context.Context, tx Tx, fn func(Tx) error) (err error) {
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, ErrTxClosed) {
			err = rollbackErr
		}
	}()

	fErr := fn(tx)
	if fErr != nil {
		_ = tx.Rollback(ctx) // ignore rollback error as there is already an error to return
		return fErr
	}

	return tx.Commit(ctx)
}
