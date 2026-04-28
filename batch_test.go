package fbx_test

import (
	"context"
	"strings"
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
	skipIfBatchUnsupported(ctx, t, conn)

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
	skipIfBatchUnsupported(ctx, t, conn)

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
	skipIfBatchUnsupported(ctx, t, conn)

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
	skipIfBatchUnsupported(ctx, t, conn)

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
	skipIfBatchUnsupported(ctx, t, conn)

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
	skipIfBatchUnsupported(ctx, t, conn)

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

func TestPreparedBatchPendingBytes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_pb_pending")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	defer pb.Close(ctx)

	require.Equal(t, 0, pb.PendingBytes())

	require.NoError(t, pb.Add(1, "one"))
	firstSize := pb.PendingBytes()
	require.Greater(t, firstSize, 0)

	require.NoError(t, pb.Add(2, "two"))
	require.Greater(t, pb.PendingBytes(), firstSize)

	require.NoError(t, pb.Flush(ctx))
	require.Equal(t, 0, pb.PendingBytes())

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Affected)
	require.Equal(t, 0, pb.PendingBytes())
}

func TestPreparedBatchPendingBytesReset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_pb_pending_reset")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	defer pb.Close(ctx)

	require.NoError(t, pb.Add(1, "drop me"))
	require.Greater(t, pb.PendingBytes(), 0)

	pb.Reset()
	require.Equal(t, 0, pb.PendingBytes())

	require.NoError(t, pb.Add(10, "kept"))
	require.Greater(t, pb.PendingBytes(), 0)

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Affected)
	require.Equal(t, 0, pb.PendingBytes())
	require.Equal(t, []batchDMLRow{
		{ID: 10, Note: "kept"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestPrepareBatchWithOptionsDefaultBehavior(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_pb_opts_default")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatchWithOptions(ctx, "insert into "+table+" (id, note) values (?, ?)", fbx.BatchOptions{})
	require.NoError(t, err)
	defer pb.Close(ctx)

	require.NoError(t, pb.Add(1, "one"))
	require.NoError(t, pb.Add(2, "two"))

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Affected)
	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one"},
		{ID: 2, Note: "two"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestPrepareBatchWithOptionsBufferBytes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_pb_opts")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	pb, err := conn.PrepareBatchWithOptions(ctx, "update "+table+" set note = ? where id = ?", fbx.BatchOptions{
		BufferBytes: 4096,
	})
	require.NoError(t, err)
	defer pb.Close(ctx)

	require.NoError(t, pb.Add("one-buffered", 1))

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Affected)
	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-buffered"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestPrepareBatchWithOptionsBufferBytesOverflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	inspect := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, inspect)

	table := batchTestTableName(t, "fbx_pb_opts_overflow")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(4096))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatchWithOptions(ctx, "insert into "+table+" (id, note) values (?, ?)", fbx.BatchOptions{
		BufferBytes: 4096,
	})
	require.NoError(t, err)
	defer pb.Close(ctx)

	require.NoError(t, pb.Add(1, strings.Repeat("a", 2048)))
	require.NoError(t, pb.Add(2, strings.Repeat("b", 2048)))
	require.NoError(t, pb.Add(3, strings.Repeat("c", 2048)))
	require.Greater(t, pb.PendingBytes(), 0)

	result, err := pb.Exec(ctx)
	require.Error(t, err)
	require.EqualValues(t, 0, result.Affected)
	require.ErrorContains(t, err, "Internal buffer overflow")
	require.Empty(t, loadBatchDMLRows(t, ctx, inspect, table))
}

func TestTxPrepareBatchWithOptionsBufferBytesCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	checkConn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, checkConn)

	table := batchTestTableName(t, "fbx_txpb_opts")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	pb, err := tx.PrepareBatchWithOptions(ctx, "insert into "+table+" (id, note) values (?, ?)", fbx.BatchOptions{
		BufferBytes: 2048,
	})
	require.NoError(t, err)
	defer pb.Close(ctx)

	require.NoError(t, pb.Add(1, "inside tx / one"))
	require.NoError(t, pb.Add(2, "inside tx / two"))

	result, err := pb.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Affected)

	require.Empty(t, loadBatchDMLRows(t, ctx, checkConn, table))
	require.NoError(t, tx.Commit(ctx))
	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "inside tx / one"},
		{ID: 2, Note: "inside tx / two"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))
}

func TestTxPrepareBatchClosedTxErrorType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	_, err = tx.PrepareBatch(ctx, "update rdb$database set rdb$description = ?")
	require.ErrorIs(t, err, fbx.ErrTxClosed)
}

func TestTxPrepareNamedBatchUpdateDeleteCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	checkConn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, checkConn)

	table := batchTestTableName(t, "fbx_txnb_commit")
	updateName := batchTestTableName(t, "upd_batch")
	deleteName := batchTestTableName(t, "del_batch")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	_, err = tx.Prepare(ctx, updateName, "batch update "+table+" set note = ? where id = ?")
	require.NoError(t, err)

	_, err = tx.Exec(ctx, updateName, "one-tx-named", 1)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, updateName, "two-tx-named", 2)
	require.NoError(t, err)

	updateResult, err := tx.ExecBatch(ctx, updateName)
	require.NoError(t, err)
	require.EqualValues(t, 2, updateResult.Affected)

	_, err = tx.Prepare(ctx, deleteName, "batch delete from "+table+" where id = ?")
	require.NoError(t, err)

	_, err = tx.Exec(ctx, deleteName, 3)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, deleteName, 4)
	require.NoError(t, err)

	deleteResult, err := tx.ExecBatch(ctx, deleteName)
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
		{ID: 1, Note: "one-tx-named"},
		{ID: 2, Note: "two-tx-named"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))
}

func TestPreparedBatchReuseUpdateDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_pb_reuse_ud")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	pbUpdate, err := conn.PrepareBatch(ctx, "update "+table+" set note = ? where id = ?")
	require.NoError(t, err)
	defer pbUpdate.Close(ctx)

	require.NoError(t, pbUpdate.Add("one-reuse", 1))
	updateResult, err := pbUpdate.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, updateResult.Affected)

	require.NoError(t, pbUpdate.Add("two-reuse", 2))
	updateResult, err = pbUpdate.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, updateResult.Affected)

	pbDelete, err := conn.PrepareBatch(ctx, "delete from "+table+" where id = ?")
	require.NoError(t, err)
	defer pbDelete.Close(ctx)

	require.NoError(t, pbDelete.Add(3))
	deleteResult, err := pbDelete.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleteResult.Affected)

	require.NoError(t, pbDelete.Add(4))
	deleteResult, err = pbDelete.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleteResult.Affected)

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-reuse"},
		{ID: 2, Note: "two-reuse"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestNamedBatchReuseUpdateDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_nb_reuse_ud")
	updateName := batchTestTableName(t, "upd_batch")
	deleteName := batchTestTableName(t, "del_batch")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	_, err = conn.Prepare(ctx, updateName, "batch update "+table+" set note = ? where id = ?")
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Deallocate(ctx, updateName)) }()

	_, err = conn.Exec(ctx, updateName, "one-named-reuse", 1)
	require.NoError(t, err)
	updateResult, err := conn.ExecBatch(ctx, updateName)
	require.NoError(t, err)
	require.EqualValues(t, 1, updateResult.Affected)

	_, err = conn.Exec(ctx, updateName, "two-named-reuse", 2)
	require.NoError(t, err)
	updateResult, err = conn.ExecBatch(ctx, updateName)
	require.NoError(t, err)
	require.EqualValues(t, 1, updateResult.Affected)

	_, err = conn.Prepare(ctx, deleteName, "batch delete from "+table+" where id = ?")
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Deallocate(ctx, deleteName)) }()

	_, err = conn.Exec(ctx, deleteName, 3)
	require.NoError(t, err)
	deleteResult, err := conn.ExecBatch(ctx, deleteName)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleteResult.Affected)

	_, err = conn.Exec(ctx, deleteName, 4)
	require.NoError(t, err)
	deleteResult, err = conn.ExecBatch(ctx, deleteName)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleteResult.Affected)

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-named-reuse"},
		{ID: 2, Note: "two-named-reuse"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestNamedBatchExecConsumesOptionsAndQueryRewriter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_nb_options")
	name := batchTestTableName(t, "batch_options")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	_, err = conn.Prepare(ctx, name, "batch insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Deallocate(ctx, name)) }()

	_, err = conn.Exec(ctx, name, fbx.QueryExecModeExec, 1, "one")
	require.NoError(t, err)

	qr := testQueryRewriter{
		sql:  name,
		args: []any{2, "two"},
	}
	_, err = conn.Exec(ctx, "rewritten_batch_name", &qr)
	require.NoError(t, err)

	result, err := conn.ExecBatch(ctx, name)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Affected)

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one"},
		{ID: 2, Note: "two"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestPreparedBatchFlushUpdateDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	checkConn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, checkConn)

	table := batchTestTableName(t, "fbx_pb_flush_ud")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	pbUpdate, err := conn.PrepareBatch(ctx, "update "+table+" set note = ? where id = ?")
	require.NoError(t, err)
	defer pbUpdate.Close(ctx)

	require.NoError(t, pbUpdate.Add("one-flush", 1))
	require.NoError(t, pbUpdate.Flush(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))

	updateResult, err := pbUpdate.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, updateResult.Affected)

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-flush"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	pbDelete, err := tx.PrepareBatch(ctx, "delete from "+table+" where id = ?")
	require.NoError(t, err)
	defer pbDelete.Close(ctx)

	require.NoError(t, pbDelete.Add(4))
	require.NoError(t, pbDelete.Flush(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-flush"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))

	deleteResult, err := pbDelete.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleteResult.Affected)

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-flush"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))

	require.NoError(t, tx.Commit(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-flush"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
	}, loadBatchDMLRows(t, ctx, checkConn, table))
}

func TestPreparedBatchFlushCloseKeepsConnectionUsable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_pb_flush_close")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	pb, err := conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?)")
	require.NoError(t, err)

	require.NoError(t, pb.Add(1, "flushed"))
	require.NoError(t, pb.Flush(ctx))
	require.NoError(t, pb.Close(ctx))

	ensureConnValid(t, conn)
	require.Empty(t, loadBatchDMLRows(t, ctx, conn, table))
}

func TestPreparedBatchPartialFailureUpdateDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	updateTable := batchTestTableName(t, "fbx_pb_pf_upd")
	deleteParentTable := batchTestTableName(t, "fbx_pb_pf_parent")
	deleteChildTable := batchTestTableName(t, "fbx_pb_pf_child")

	_, err := conn.Exec(ctx, "recreate table "+updateTable+" (id integer not null primary key, note varchar(50) unique)")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, updateTable)

	pbUpdate, err := conn.PrepareBatch(ctx, "update "+updateTable+" set note = ? where id = ?")
	require.NoError(t, err)
	defer pbUpdate.Close(ctx)

	require.NoError(t, pbUpdate.Add("same-note", 1))
	require.NoError(t, pbUpdate.Add("same-note", 2))

	updateResult, err := pbUpdate.Exec(ctx)
	require.ErrorContains(t, err, "batch failed at row 1")
	require.EqualValues(t, 1, updateResult.Affected)
	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "same-note"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, conn, updateTable))

	_, err = conn.Exec(ctx, "recreate table "+deleteChildTable+" (id integer not null primary key, parent_id integer not null)")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "recreate table "+deleteParentTable+" (id integer not null primary key)")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "recreate table "+deleteChildTable+" (id integer not null primary key, parent_id integer not null references "+deleteParentTable+"(id))")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "insert into "+deleteParentTable+" (id) values (?)", 1)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "insert into "+deleteParentTable+" (id) values (?)", 2)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "insert into "+deleteChildTable+" (id, parent_id) values (?, ?)", 1, 2)
	require.NoError(t, err)

	pbDelete, err := conn.PrepareBatch(ctx, "delete from "+deleteParentTable+" where id = ?")
	require.NoError(t, err)
	defer pbDelete.Close(ctx)

	require.NoError(t, pbDelete.Add(1))
	require.NoError(t, pbDelete.Add(2))

	deleteResult, err := pbDelete.Exec(ctx)
	require.ErrorContains(t, err, "batch failed at row 1")
	require.EqualValues(t, 1, deleteResult.Affected)

	var remainingIDs []int32
	rows, err := conn.Query(ctx, "select id from "+deleteParentTable+" order by id")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id int32
		require.NoError(t, rows.Scan(&id))
		remainingIDs = append(remainingIDs, id)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []int32{2}, remainingIDs)
}

func TestTxNestedPrepareBatchCommitAndRollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_nested_pb")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)
	seedBatchDMLRows(t, ctx, conn, table)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	nestedUpdate, err := tx.Begin(ctx)
	require.NoError(t, err)

	pbUpdate, err := nestedUpdate.PrepareBatch(ctx, "update "+table+" set note = ? where id = ?")
	require.NoError(t, err)
	defer pbUpdate.Close(ctx)

	require.NoError(t, pbUpdate.Add("one-nested", 1))
	updateResult, err := pbUpdate.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, updateResult.Affected)
	require.NoError(t, nestedUpdate.Commit(ctx))

	nestedDelete, err := tx.Begin(ctx)
	require.NoError(t, err)

	pbDelete, err := nestedDelete.PrepareBatch(ctx, "delete from "+table+" where id = ?")
	require.NoError(t, err)
	defer pbDelete.Close(ctx)

	require.NoError(t, pbDelete.Add(4))
	deleteResult, err := pbDelete.Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleteResult.Affected)
	require.NoError(t, nestedDelete.Rollback(ctx))

	require.NoError(t, tx.Commit(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "one-nested"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}

func TestTxPrepareBatchReturningRejectDoesNotAbortExplicitTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, defaultConnTestRunner.CreateConfig(ctx, t).ConnString())
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	table := batchTestTableName(t, "fbx_txpb_returning")

	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
	require.NoError(t, err)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	_, err = tx.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?) returning id")
	require.ErrorContains(t, err, "batch supports only insert, update, and delete statements without result sets")

	_, err = tx.Exec(ctx, "insert into "+table+" (id, note) values (?, ?)", 1, "ok")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	require.Equal(t, []batchDMLRow{
		{ID: 1, Note: "ok"},
	}, loadBatchDMLRows(t, ctx, conn, table))
}
