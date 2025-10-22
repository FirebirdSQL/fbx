package pgconn

import (
	"context"

	"github.com/red-soft/fbx/fbproto"
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
	StatementInfo   *fbproto.StatementInfo
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

type Statement struct {
	conn         *PgConn
	Description  *StatementDescription
	transaction  *FBTransaction
	cursorOpened bool
}

// Exec executed a prepared statement inside bound transaction
//
// paramValues are the parameter values. It must be encoded in the binary format
//
// ResultReader must be closed before PgConn can be used again.
func (st *Statement) Exec(ctx context.Context, paramValues [][]byte) *ResultReader {
	return st.ExecTx(ctx, st.transaction, paramValues)
}

// ExecTx executed a prepared statement inside specified transaction
//
// tx opened transaction
// paramValues are the parameter values. It must be encoded in the binary format
//
// ResultReader must be closed before PgConn can be used again.
func (st *Statement) ExecTx(ctx context.Context, tx *FBTransaction, paramValues [][]byte) *ResultReader {
	rr := &ResultReader{
		pgConn: st.conn,
		ctx:    ctx,
	}

	rr.statementDescription = st.Description
	rr.fieldDescriptions = st.Description.Fields

	if err := st.conn.lock(); err != nil {
		rr.concludeCommand(CommandTag{}, err)
		rr.closed = true
		return rr
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			rr.concludeCommand(CommandTag{}, newContextAlreadyDoneError(ctx))
			rr.closed = true
			st.conn.unlock()
			return rr
		default:
		}
		st.conn.contextWatcher.Watch(ctx)
	}

	var err error

	if st.cursorOpened {
		st.cursorOpened = false
		if err = st.conn.client.WriteMessage(&fbproto.OpFreeStatement{
			StatementHandle: st.Description.StatementHandle,
			Option:          fbproto.DSQL_close,
		}); err != nil {
			st.conn.asyncClose()
			rr.concludeCommand(CommandTag{}, err)
			st.conn.contextWatcher.Unwatch()
			rr.closed = true
			return rr
		}
	}

	if err = st.conn.client.WriteMessage(&fbproto.OpExecute{
		TransactionHandle:     tx.Handle,
		StatementHandle:       st.Description.StatementHandle,
		ParametersDescriptors: st.Description.StatementInfo.ParametersDescriptors,
		SQLDataBlr:            st.Description.StatementInfo.GetParametersBlr(),
		SQLData:               paramValues,
	}); err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(CommandTag{}, err)
		st.conn.contextWatcher.Unwatch()
		rr.closed = true
		return rr
	}

	if err = st.conn.writer.Flush(); err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(CommandTag{}, err)
		st.conn.contextWatcher.Unwatch()
		rr.closed = true
		return rr
	}

	_, err = st.conn.receiveResponse()
	if err != nil {
		st.conn.asyncClose()
		rr.concludeCommand(CommandTag{}, err)
		st.conn.contextWatcher.Unwatch()
		rr.closed = true
		return rr
	}

	st.cursorOpened = st.Description.GetQueryType() == StatementSelect
	return rr
}

// Close frees statement on server
func (st *Statement) Close(ctx context.Context) error {
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

	if err = st.conn.client.WriteMessage(&fbproto.OpFreeStatement{StatementHandle: st.Description.StatementHandle, Option: fbproto.DSQL_drop}); err != nil {
		st.conn.asyncClose()
		return err
	}

	if err = st.conn.writer.Flush(); err != nil {
		st.conn.asyncClose()
		return err
	}

	return nil
}
