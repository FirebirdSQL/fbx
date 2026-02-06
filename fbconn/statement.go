package fbconn

import (
	"context"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
)

const (
	StatementSelect = iota + 1
	StatementInsert
	StatementUpdate
	StatementDelete
	StatementDDL
	StatementGetSegment
	StatementPutSegment
	StatementProcedure
	StatementStartTransaction
	StatementCommit
	StatementRollback
	StatementSelectForUpdate
	StatementSetGenerator
	StatementSavepoint
)

type StatementDescription struct {
	Name            string
	SQL             string
	StatementInfo   *StatementInfo
	StatementHandle int
	Fields          []FieldDescription
	Parameters      []FieldDescription
}

func (sd *StatementDescription) GetQueryType() int {
	switch sd.StatementInfo.StatementType {
	case fbproto.Isc_info_sql_stmt_select:
		return StatementSelect
	case fbproto.Isc_info_sql_stmt_insert:
		return StatementInsert
	case fbproto.Isc_info_sql_stmt_update:
		return StatementUpdate
	case fbproto.Isc_info_sql_stmt_delete:
		return StatementDelete
	case fbproto.Isc_info_sql_stmt_ddl:
		return StatementDDL
	case fbproto.Isc_info_sql_stmt_get_segment:
		return StatementGetSegment
	case fbproto.Isc_info_sql_stmt_put_segment:
		return StatementPutSegment
	case fbproto.Isc_info_sql_stmt_exec_procedure:
		return StatementProcedure
	case fbproto.Isc_info_sql_stmt_start_trans:
		return StatementStartTransaction
	case fbproto.Isc_info_sql_stmt_commit:
		return StatementCommit
	case fbproto.Isc_info_sql_stmt_rollback:
		return StatementRollback
	case fbproto.Isc_info_sql_stmt_select_for_upd:
		return StatementSelectForUpdate
	case fbproto.Isc_info_sql_stmt_set_generator:
		return StatementSetGenerator
	case fbproto.Isc_info_sql_stmt_savepoint:
		return StatementSavepoint
	}
	panic("unknown statement type")
}

func (sd *StatementDescription) HaveCursorResult() bool {
	return sd.GetQueryType() == StatementSelect || sd.GetQueryType() == StatementSelectForUpdate
}

func (sd *StatementDescription) HaveSingletonResult() bool {
	return sd.GetQueryType() == StatementProcedure && len(sd.Fields) > 0
}

func (sd *StatementDescription) GetAffectedCount() int64 {
	return sd.StatementInfo.Counts.SelectCount + sd.StatementInfo.Counts.InsertCount + sd.StatementInfo.Counts.UpdateCount + sd.StatementInfo.Counts.DeleteCount
}

type Statement struct {
	conn           *FbConn
	Description    *StatementDescription
	transaction    *FBTransaction
	cursorOpened   bool
	closed         bool
	timeout        int
	cursorFlags    int
}

// SetTimeout sets statement timeout (milliseconds).
func (st *Statement) SetTimeout(timeout int) {
	st.timeout = timeout
}

// SetCursorFlags sets cursor flags for this statement.
func (st *Statement) SetCursorFlags(flags int) {
	st.cursorFlags = flags
}

// SetInlineBlobSize sets inline blob size for this statement.
func (st *Statement) SetInlineBlobSize(size int) {
	st.conn.inlineBlobSize = size
}

// Exec executed a prepared statement inside bound transaction
//
// paramValues are the parameter values. It must be encoded in the binary format
//
// ResultReader must be closed before FbConn can be used again.
func (st *Statement) Exec(ctx context.Context, paramValues [][]byte) *ResultReader {
	var err error
	rr := &ResultReader{
		fbConn: st.conn,
		ctx:    ctx,
	}
	if st.closed {
		err = fmt.Errorf("statement is closed")
		rr.concludeCommand(QueryStatus{}, err)
		rr.closed = true
		return rr
	}

	if err = st.CloseCursor(ctx); err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(QueryStatus{}, err)
		st.conn.contextWatcher.Unwatch()
		rr.closed = true
		return rr
	}

	tx := st.transaction
	if tx == nil {
		if _, err := st.conn.ensureInternalTransaction(ctx, nil, false); err != nil {
			rr := &ResultReader{
				fbConn: st.conn,
				ctx:    ctx,
			}
			rr.concludeCommand(QueryStatus{}, err)
			rr.closed = true
			return rr
		}
		tx = st.conn.internalTransaction
	}
	return st.ExecTx(ctx, tx, paramValues)
}

// ExecTx executed a prepared statement inside specified transaction
//
// tx opened transaction
// paramValues are the parameter values. It must be encoded in the binary format
//
// ResultReader must be closed before FbConn can be used again.
func (st *Statement) ExecTx(ctx context.Context, tx *FBTransaction, paramValues [][]byte) *ResultReader {
	var err error
	rr := &ResultReader{
		fbConn:           st.conn,
		ctx:              ctx,
		boundTransaction: tx,
	}

	// Validate parameter count before any wire interaction.
	if len(paramValues) != len(st.Description.StatementInfo.ParametersDescriptors) {
		err := &FbError{
			Message: fmt.Sprintf("Invalid number of parameters, expected %d, got %d", len(st.Description.StatementInfo.ParametersDescriptors), len(paramValues)),
			Code:    "07008",
		}
		rr.concludeCommand(QueryStatus{}, err)
		rr.closed = true
		return rr
	}

	if err = st.CloseCursor(ctx); err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(QueryStatus{}, err)
		st.conn.contextWatcher.Unwatch()
		rr.closed = true
		return rr
	}

	rr.statementDescription = st.Description
	rr.fieldDescriptions = st.Description.Fields

	if err = st.conn.lock(); err != nil {
		rr.concludeCommand(QueryStatus{}, err)
		rr.closed = true
		return rr
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			rr.concludeCommand(QueryStatus{}, newContextAlreadyDoneError(ctx))
			rr.closed = true
			st.conn.unlock()
			return rr
		default:
		}
		st.conn.contextWatcher.Watch(ctx)
	}

	execKind := fbproto.Op_execute
	var rowBlr []byte
	if st.Description.HaveSingletonResult() {
		execKind = fbproto.Op_execute2
		rowBlr = st.Description.StatementInfo.GetRowBlr()
	}

	var sqlDataBlr []byte
	if len(paramValues) > 0 {
		sqlDataBlr = st.Description.StatementInfo.GetParametersBlr(paramValues)
	}

	if err = st.conn.client.WriteMessage(&fbproto.OpExecute{
		Kind:                  execKind,
		TransactionHandle:     tx.Handle,
		StatementHandle:       st.Description.StatementHandle,
		ParametersDescriptors: st.Description.StatementInfo.ParametersDescriptors,
		SQLDataBlr:            sqlDataBlr,
		SQLDataOutBlr:         rowBlr,
		Parameters:            paramValues,
		Timeout:               st.timeout,
		CursorFlags:           st.cursorFlags,
		InlineBlobSize:        st.conn.inlineBlobSize,
		ProtocolFeatures:      st.conn.protocolFeatures,
	}); err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(QueryStatus{}, err)
		st.conn.contextWatcher.Unwatch()
		st.conn.unlock()
		rr.closed = true
		return rr
	}

	if err = st.conn.flushWithPotentialWriteReadDeadlock(); err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(QueryStatus{}, err)
		st.conn.contextWatcher.Unwatch()
		st.conn.unlock()
		rr.closed = true
		return rr
	}

	msg, err := st.conn.receiveMessage()
	if err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(QueryStatus{}, normalizeTimeoutError(ctx, err))
		st.conn.contextWatcher.Unwatch()
		st.conn.unlock()
		rr.closed = true
		return rr
	}

	if st.Description.HaveSingletonResult() {
		switch r := msg.(type) {
		case *fbproto.OpSqlResponse:
			row, dataResp, err := st.conn.client.ReceiveSQLData(st.Description.StatementInfo.FieldDescriptors)
			if err != nil {
				st.conn.asyncClose()
				rr.concludeCommand(QueryStatus{}, err)
				st.conn.contextWatcher.Unwatch()
				st.conn.unlock()
				rr.closed = true
				return rr
			}
			switch d := dataResp.(type) {
			case *fbproto.OpResponse:
				rr.singleton = row
			case *fbproto.ErrorResponse:
				rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(d))
				st.conn.contextWatcher.Unwatch()
				st.conn.unlock()
				rr.closed = true
				return rr
			}
		case *fbproto.ErrorResponse:
			rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(r))
			st.conn.contextWatcher.Unwatch()
			st.conn.unlock()
			rr.closed = true
			return rr
		default:
			panic(fmt.Sprintf("unexpected response type %T", msg))
		}
	} else {
		switch r := msg.(type) {
		case *fbproto.OpResponse:
		case *fbproto.ErrorResponse:
			rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(r))
			st.conn.contextWatcher.Unwatch()
			st.conn.unlock()
			rr.closed = true
			return rr
		default:
			panic(fmt.Sprintf("unexpected response type %T", msg))
		}
	}

	st.closed = false
	st.cursorOpened = st.Description.HaveCursorResult()
	rr.cursorOpened = st.Description.HaveCursorResult()
	rr.singletonResult = st.Description.HaveSingletonResult()
	rr.queryStatus = st.conn.makeQueryStatus(st.Description.GetQueryType())
	return rr
}

func (st *Statement) internalClose(ctx context.Context, closeOption int) error {
	if st.closed {
		return nil
	}

	var tx *FBTransaction
	if st.transaction == nil && st.conn.internalTransaction != nil {
		tx = st.conn.internalTransaction
	} else {
		tx = st.transaction
	}

	if closeOption == fbproto.DSQL_close && (!st.cursorOpened || tx.Closed) {
		st.cursorOpened = false
		return nil
	}

	st.closed = true

	var err error
	if err = st.conn.lock(); err != nil {
		return err
	}
	defer st.conn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		st.conn.contextWatcher.Watch(ctx)
		defer st.conn.contextWatcher.Unwatch()
	}

	st.cursorOpened = false
	st.closed = closeOption == fbproto.DSQL_drop

	if err = st.conn.client.WriteMessage(&fbproto.OpFreeStatement{StatementHandle: st.Description.StatementHandle, Option: closeOption}); err != nil {
		st.conn.asyncClose()
		return err
	}

	if closeOption == fbproto.DSQL_drop {
		if err = st.conn.flushWithPotentialWriteReadDeadlock(); err != nil {
			st.conn.asyncClose()
			return err
		}
	}

	return nil
}

// Deallocate frees statement on server
func (st *Statement) Deallocate(ctx context.Context) error {
	return st.internalClose(ctx, fbproto.DSQL_drop)
}

// CloseCursor close opened cursor without deallocating statement on server
func (st *Statement) CloseCursor(ctx context.Context) error {
	return st.internalClose(ctx, fbproto.DSQL_close)
}
