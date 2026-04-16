package fbx_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
)

func TestTxPrepareBatchUpdateDeleteCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)

	checkConn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, checkConn)

	table := batchTestTableName(t, "fbx_txpb_commit")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	pbUpdate, err := tx.PrepareBatch(ctx, "update "+table+" set note = ? where id = ?")
	require.NoError(t, err)
	defer pbUpdate.Close(ctx)

	require.NoError(t, pbUpdate.Add("one-tx", 1))
	require.NoError(t, pbUpdate.Add("two-tx", 2))

	updateResult, err := pbUpdate.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, updateResult.Affected)

	pbDelete, err := tx.PrepareBatch(ctx, "delete from "+table+" where id = ?")
	require.NoError(t, err)
	defer pbDelete.Close(ctx)

	require.NoError(t, pbDelete.Add(3))
	require.NoError(t, pbDelete.Add(4))

	deleteResult, err := pbDelete.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, deleteResult.Affected)

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))

	require.NoError(t, tx.Commit(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-tx"},
		{ID: 2, Note: "two-tx"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))
}

func TestTxPrepareBatchUpdateDeleteRollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)

	checkConn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, checkConn)

	table := batchTestTableName(t, "fbx_txpb_rollback")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	pbUpdate, err := tx.PrepareBatch(ctx, "update "+table+" set note = ? where id = ?")
	require.NoError(t, err)
	defer pbUpdate.Close(ctx)

	require.NoError(t, pbUpdate.Add("one-rb", 1))
	require.NoError(t, pbUpdate.Add("two-rb", 2))

	updateResult, err := pbUpdate.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, updateResult.Affected)

	pbDelete, err := tx.PrepareBatch(ctx, "delete from "+table+" where id = ?")
	require.NoError(t, err)
	defer pbDelete.Close(ctx)

	require.NoError(t, pbDelete.Add(3))
	require.NoError(t, pbDelete.Add(4))

	deleteResult, err := pbDelete.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, deleteResult.Affected)

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))

	require.NoError(t, tx.Rollback(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))
}

func TestPreparedBatchMethodsAfterClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)

	table := batchTestTableName(t, "fbx_pb_closed")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)

	require.NoError(t, pb.Close(ctx))
	require.NoError(t, pb.Close(ctx))

	require.ErrorContains(t, pb.Add(1, "one"), "prepared batch is closed")
	require.ErrorContains(t, pb.Flush(ctx), "prepared batch is closed")

	_, err = pb.Exec(ctx)
	require.ErrorContains(t, err, "prepared batch is closed")
}

func TestTxPrepareBatchRejectDoesNotAbortExplicitTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)

	table := batchTestTableName(t, "fbx_txpb_reject")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	_, err = tx.PrepareBatch(ctx, "select cast(? as integer) from rdb$database")
	require.ErrorContains(t, err, "batch supports only insert, update, and delete statements without result sets")

	_, err = tx.Exec(ctx, "insert into "+table+" (id, note) values (?, ?)", 1, "ok")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "ok"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestTxBatchMethodsReturnErrTxClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	_, err = tx.PrepareBatch(ctx, "insert into rdb$database(dummy) values (?)")
	require.ErrorIs(t, err, fbx.ErrTxClosed)

	err = tx.Flush(ctx, "missing")
	require.ErrorIs(t, err, fbx.ErrTxClosed)

	_, err = tx.ExecBatch(ctx, "missing")
	require.ErrorIs(t, err, fbx.ErrTxClosed)
}

func TestPreparedBatchExecWithoutRows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)

	table := batchTestTableName(t, "fbx_pb_empty")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	defer pb.Close(ctx)

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 0, result.Affected)
	require.Empty(t, loadBatchDMLRows(t, ctx, conn, table))
}

func TestTxPrepareBatchClosedTxErrorType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	_, err = tx.PrepareBatch(ctx, "update rdb$database set rdb$description = ?")
	require.ErrorIs(t, err, fbx.ErrTxClosed)
}
