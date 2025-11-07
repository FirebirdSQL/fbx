package fbx_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/red-soft/fbx"
	"github.com/red-soft/fbx/fbconn"
	"github.com/red-soft/fbx/fbxtest"
	"github.com/stretchr/testify/require"
)

func TestTransactionSuccessfulCommit(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	//create temporary table foo(
	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	if _, err := conn.Exec(context.Background(), createSql); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("conn.Begin failed: %v", err)
	}

	_, err = tx.Exec(context.Background(), "insert into "+t.Name()+" (id) values (1)")
	if err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("tx.Commit failed: %v", err)
	}

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from "+t.Name()).Scan(&n)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("Did not receive correct number of rows: %v", n)
	}
}

func TestTxCommitWhenTxBroken(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)
	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	if _, err := conn.Exec(context.Background(), createSql); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("conn.Begin failed: %v", err)
	}

	if _, err := tx.Exec(context.Background(), "insert into "+t.Name()+"(id) values (1)"); err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	// Purposely break transaction
	if _, err := tx.Exec(context.Background(), "syntax error"); err == nil {
		t.Fatal("Unexpected success")
	}

	err = tx.Commit(context.Background())
	if err != fbx.ErrTxCommitRollback {
		t.Fatalf("Expected error %v, got %v", fbx.ErrTxCommitRollback, err)
	}

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from "+t.Name()).Scan(&n)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if n != 0 {
		t.Fatalf("Did not receive correct number of rows: %v", n)
	}
}

func TestTxCommitWhenDeferredConstraintFailure(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	fbxtest.SkipCockroachDB(t, conn, "Server does not support deferred constraint (https://github.com/cockroachdb/cockroach/issues/31632)")

	createSql := `
    recreate table  ` + t.Name() + ` (
      id integer,
      unique (id) initially deferred
    );
  `

	if _, err := conn.Exec(context.Background(), createSql); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("conn.Begin failed: %v", err)
	}

	if _, err := tx.Exec(context.Background(), "insert into  "+t.Name()+" (id) values (1)"); err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	if _, err := tx.Exec(context.Background(), "insert into "+t.Name()+" (id) values (1)"); err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	err = tx.Commit(context.Background())
	if pgErr, ok := err.(*fbconn.PgError); !ok || pgErr.Code != "23505" {
		t.Fatalf("Expected unique constraint violation 23505, got %#v", err)
	}

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from "+t.Name()).Scan(&n)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if n != 0 {
		t.Fatalf("Did not receive correct number of rows: %v", n)
	}
}

func TestTxCommitSerializationFailure(t *testing.T) {
	t.Parallel()

	c1 := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, c1)

	if c1.FbConn().ParameterStatus("crdb_version") != "" {
		t.Skip("Skipping due to known server issue: (https://github.com/cockroachdb/cockroach/issues/60754)")
	}

	c2 := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, c2)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := c1.Exec(ctx, `recreate table `+t.Name()+`(num integer)`)
	if err != nil {
		t.Fatalf("Unable to create temporary table: %v", err)
	}
	defer c1.Exec(ctx, `drop table `+t.Name())

	tx1, err := c1.BeginTx(ctx, fbx.TxOptions{IsoLevel: fbx.Serializable})
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx1.Rollback(ctx)

	tx2, err := c2.BeginTx(ctx, fbx.TxOptions{IsoLevel: fbx.Serializable})
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx2.Rollback(ctx)

	_, err = tx1.Exec(ctx, `insert into tx_serializable_sums(num) select cast(sum(num) as int) from `+t.Name())
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	_, err = tx2.Exec(ctx, `insert into tx_serializable_sums(num) select cast(sum(num) as int) from `+t.Name())
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	err = tx1.Commit(ctx)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	err = tx2.Commit(ctx)
	if fbErr, ok := err.(*fbconn.FbError); !ok || fbErr.Code != "40001" {
		t.Fatalf("Expected serialization error 40001, got %#v", err)
	}

	ensureConnValid(t, c1)
	ensureConnValid(t, c2)
}

func TestTransactionSuccessfulRollback(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	createSql := `
    recreate ` + t.Name() + ` foo(
      id integer,
      unique (id)
    );
  `

	if _, err := conn.Exec(context.Background(), createSql); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("conn.Begin failed: %v", err)
	}

	_, err = tx.Exec(context.Background(), "insert into "+t.Name()+"(id) values (1)")
	if err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	err = tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("tx.Rollback failed: %v", err)
	}

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from "+t.Name()).Scan(&n)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if n != 0 {
		t.Fatalf("Did not receive correct number of rows: %v", n)
	}
}

func TestTransactionRollbackFailsClosesConnection(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	ctx, cancel := context.WithCancel(context.Background())

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	cancel()

	err = tx.Rollback(ctx)
	require.Error(t, err)

	require.True(t, conn.IsClosed())
}

func TestBeginIsoLevels(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	fbxtest.SkipCockroachDB(t, conn, "Server always uses SERIALIZABLE isolation (https://www.cockroachlabs.com/docs/stable/demo-serializable.html)")

	isoLevels := []fbx.TxIsoLevel{fbx.Serializable, fbx.RepeatableRead, fbx.ReadCommitted, fbx.ReadUncommitted}
	for _, iso := range isoLevels {
		tx, err := conn.BeginTx(context.Background(), fbx.TxOptions{IsoLevel: iso})
		if err != nil {
			t.Fatalf("conn.Begin failed: %v", err)
		}

		var level fbx.TxIsoLevel
		conn.QueryRow(context.Background(), "select current_setting('transaction_isolation')").Scan(&level)
		if level != iso {
			t.Errorf("Expected to be in isolation level %v but was %v", iso, level)
		}

		err = tx.Rollback(context.Background())
		if err != nil {
			t.Fatalf("tx.Rollback failed: %v", err)
		}
	}
}

func TestBeginFunc(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	_, err := conn.Exec(context.Background(), createSql)
	require.NoError(t, err)

	err = fbx.BeginFunc(context.Background(), conn, func(tx fbx.Tx) error {
		_, err := tx.Exec(context.Background(), "insert into "+t.Name()+"(id) values (1)")
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from "+t.Name()).Scan(&n)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}

func TestBeginFuncRollbackOnError(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	_, err := conn.Exec(context.Background(), createSql)
	require.NoError(t, err)

	err = fbx.BeginFunc(context.Background(), conn, func(tx fbx.Tx) error {
		_, err := tx.Exec(context.Background(), "insert into "+t.Name()+"(id) values (1)")
		require.NoError(t, err)
		return errors.New("some error")
	})
	require.EqualError(t, err, "some error")

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from "+t.Name()).Scan(&n)
	require.NoError(t, err)
	require.EqualValues(t, 0, n)
}

func TestBeginReadOnly(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	tx, err := conn.BeginTx(context.Background(), fbx.TxOptions{AccessMode: fbx.ReadOnly})
	if err != nil {
		t.Fatalf("conn.Begin failed: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), "recreate table "+t.Name()+"(id int primary key)")
	if err != nil {
		t.Fatalf("conn.Exec failed: %v", err)
	}
	err = tx.Commit(context.Background())
	if fbErr, ok := err.(*fbconn.FbError); !ok || fbErr.Code != "25006" {
		t.Errorf("Expected error SQLSTATE 25006, but got %#v", err)
	}
}

func TestTxNestedTransactionCommit(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	if _, err := conn.Exec(context.Background(), createSql); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec(context.Background(), "insert into foo(id) values (1)")
	if err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	nestedTx, err := tx.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = nestedTx.Exec(context.Background(), "insert into foo(id) values (2)")
	if err != nil {
		t.Fatalf("nestedTx.Exec failed: %v", err)
	}

	doubleNestedTx, err := nestedTx.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = doubleNestedTx.Exec(context.Background(), "insert into foo(id) values (3)")
	if err != nil {
		t.Fatalf("doubleNestedTx.Exec failed: %v", err)
	}

	err = doubleNestedTx.Commit(context.Background())
	if err != nil {
		t.Fatalf("doubleNestedTx.Commit failed: %v", err)
	}

	err = nestedTx.Commit(context.Background())
	if err != nil {
		t.Fatalf("nestedTx.Commit failed: %v", err)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("tx.Commit failed: %v", err)
	}

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from foo").Scan(&n)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if n != 3 {
		t.Fatalf("Did not receive correct number of rows: %v", n)
	}
}

func TestTxNestedTransactionRollback(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	if _, err := conn.Exec(context.Background(), createSql); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec(context.Background(), "insert into foo(id) values (1)")
	if err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	nestedTx, err := tx.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = nestedTx.Exec(context.Background(), "insert into foo(id) values (2)")
	if err != nil {
		t.Fatalf("nestedTx.Exec failed: %v", err)
	}

	err = nestedTx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("nestedTx.Rollback failed: %v", err)
	}

	_, err = tx.Exec(context.Background(), "insert into foo(id) values (3)")
	if err != nil {
		t.Fatalf("tx.Exec failed: %v", err)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("tx.Commit failed: %v", err)
	}

	var n int64
	err = conn.QueryRow(context.Background(), "select count(*) from foo").Scan(&n)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if n != 2 {
		t.Fatalf("Did not receive correct number of rows: %v", n)
	}
}

func TestTxBeginFuncNestedTransactionCommit(t *testing.T) {
	t.Parallel()

	db := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, db)

	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	_, err := db.Exec(context.Background(), createSql)
	require.NoError(t, err)

	err = fbx.BeginFunc(context.Background(), db, func(db fbx.Tx) error {
		_, err := db.Exec(context.Background(), "insert into foo(id) values (1)")
		require.NoError(t, err)

		err = fbx.BeginFunc(context.Background(), db, func(db fbx.Tx) error {
			_, err := db.Exec(context.Background(), "insert into foo(id) values (2)")
			require.NoError(t, err)

			err = fbx.BeginFunc(context.Background(), db, func(db fbx.Tx) error {
				_, err := db.Exec(context.Background(), "insert into foo(id) values (3)")
				require.NoError(t, err)
				return nil
			})
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	var n int64
	err = db.QueryRow(context.Background(), "select count(*) from foo").Scan(&n)
	require.NoError(t, err)
	require.EqualValues(t, 3, n)
}

func TestTxBeginFuncNestedTransactionRollback(t *testing.T) {
	t.Parallel()

	db := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, db)

	createSql := `
    recreate table ` + t.Name() + `(
      id integer,
      unique (id)
    );
  `

	_, err := db.Exec(context.Background(), createSql)
	require.NoError(t, err)

	err = fbx.BeginFunc(context.Background(), db, func(db fbx.Tx) error {
		_, err := db.Exec(context.Background(), "insert into foo(id) values (1)")
		require.NoError(t, err)

		err = fbx.BeginFunc(context.Background(), db, func(db fbx.Tx) error {
			_, err := db.Exec(context.Background(), "insert into foo(id) values (2)")
			require.NoError(t, err)
			return errors.New("do a rollback")
		})
		require.EqualError(t, err, "do a rollback")

		_, err = db.Exec(context.Background(), "insert into foo(id) values (3)")
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)

	var n int64
	err = db.QueryRow(context.Background(), "select count(*) from foo").Scan(&n)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
}

func TestTxSendBatchClosed(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	db := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, db)

	tx, err := db.Begin(context.Background())
	require.NoError(t, err)
	defer tx.Rollback(context.Background())

	err = tx.Commit(context.Background())
	require.NoError(t, err)

	batch := &fbx.Batch{}
	batch.Queue("select 1")
	batch.Queue("select 2")
	batch.Queue("select 3")

	br := tx.SendBatch(context.Background(), batch)
	defer br.Close()

	var n int

	_, err = br.Exec()
	require.Error(t, err)

	err = br.QueryRow().Scan(&n)
	require.Error(t, err)

	_, err = br.Query()
	require.Error(t, err)
}
