package fbx

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseConfig(t testing.TB, connString string) *ConnConfig {
	config, err := ParseConfig(connString)
	require.Nil(t, err)
	return config
}

func mustConnect(t testing.TB, config *ConnConfig) *Conn {
	conn, err := ConnectConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Unable to establish connection: %v", err)
	}
	return conn
}

// Ensures the connection limits the size of its cached objects.
// This test examines the internals of *Conn so must be in the same package.
func TestStmtCacheSizeLimit(t *testing.T) {
	const cacheLimit = 16

	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.StatementCacheCapacity = cacheLimit
	conn := mustConnect(t, connConfig)
	defer func() {
		err := conn.Close(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}()

	// run a set of unique queries that should overflow the cache
	ctx := context.Background()
	for i := range cacheLimit * 2 {
		uniqueString := fmt.Sprintf("unique %d", i)
		uniqueSQL := fmt.Sprintf("select '%s' from rdb$database", uniqueString)
		var output string
		err := conn.QueryRow(ctx, uniqueSQL).Scan(&output)
		require.NoError(t, err)
		require.Equal(t, uniqueString, output)
	}
	// preparedStatements contains cacheLimit+1 because deallocation happens before the query
	assert.Len(t, conn.preparedStatements, cacheLimit+1)
	assert.Equal(t, cacheLimit, conn.statementCache.Len())
}

func TestQueryExecModeExecCloseWithCanceledContextDeallocatesStatement(t *testing.T) {
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeExec
	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	sql := "select cast('query_exec_cancel_close' as varchar(64)) from rdb$database"
	ctx, cancel := context.WithCancel(context.Background())
	rows, err := conn.Query(ctx, sql)
	require.NoError(t, err)

	cancel()
	rows.Close()

	assert.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, sql))
}

type cancelingErrorValuer struct {
	cancel context.CancelFunc
}

func (v cancelingErrorValuer) Value() (driver.Value, error) {
	v.cancel()
	return nil, errors.New("canceling valuer encode error")
}

func TestQueryExecModeExecEncodeFailureWithCanceledContextDeallocatesStatement(t *testing.T) {
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeExec
	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	sql := "select cast(? as integer) from rdb$database where 'query_exec_encode_cancel' = 'query_exec_encode_cancel'"
	ctx, cancel := context.WithCancel(context.Background())
	_, err := conn.Query(ctx, sql, cancelingErrorValuer{cancel: cancel})
	require.ErrorContains(t, err, "canceling valuer encode error")

	assert.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, sql))
}

func countStatementHandlesForSQL(t testing.TB, conn *Conn, sql string) int64 {
	t.Helper()

	var count int64
	err := conn.QueryRow(
		context.Background(),
		"select count(*) from mon$statements where mon$attachment_id = current_connection and cast(mon$sql_text as varchar(255)) = ?",
		QueryExecModeExec,
		sql,
	).Scan(&count)
	require.NoError(t, err)
	return count
}

func TestConnDeallocateInvalidatedCachedStatementsWithActiveTransactions(t *testing.T) {
	tests := []struct {
		name    string
		trigger func(context.Context, Tx) error
	}{
		{
			name: "Exec",
			trigger: func(ctx context.Context, tx Tx) error {
				_, err := tx.Exec(
					ctx,
					"select cast(317502 as integer) from rdb$database",
					QueryExecModeExec,
				)
				return err
			},
		},
		{
			name: "Query",
			trigger: func(ctx context.Context, tx Tx) error {
				var n int32
				err := tx.QueryRow(
					ctx,
					"select cast(317502 as integer) from rdb$database",
					QueryExecModeExec,
				).Scan(&n)
				if err != nil {
					return err
				}
				if n != 317502 {
					return fmt.Errorf("unexpected query result: %d", n)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
			connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
			connConfig.StatementCacheCapacity = 1

			conn := mustConnect(t, connConfig)
			defer func() {
				require.NoError(t, conn.Close(context.Background()))
			}()

			tx1, err := conn.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx1.Rollback(context.Background()) }()

			// Starting tx2 makes it the connection's current internal transaction. A
			// statement subsequently prepared through tx1 is therefore explicitly
			// bound to tx1 in fbconn.Statement.
			tx2, err := conn.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx2.Rollback(context.Background()) }()

			const boundSQL = "select cast(? as integer) + 1 from rdb$database"
			var n int32
			err = tx1.QueryRow(ctx, boundSQL, 1).Scan(&n)
			require.NoError(t, err)
			require.EqualValues(t, 2, n)
			require.Contains(t, conn.preparedStatements, boundSQL)

			// The cache only has room for one statement, so this puts the statement
			// bound to tx1 on the invalidated list.
			const evictingSQL = "select cast(? as integer) + 2 from rdb$database"
			err = tx2.QueryRow(ctx, evictingSQL, 1).Scan(&n)
			require.NoError(t, err)
			require.EqualValues(t, 3, n)
			invalidated := conn.statementCache.GetInvalidated()
			require.Len(t, invalidated, 1)
			require.Same(t, conn.preparedStatements[boundSQL], invalidated[0])

			// Exec and Query both deallocate invalidated statements before starting
			// their own operation. Deallocating a tx1-bound statement while called
			// through tx2 must not affect either transaction or leak its server handle.
			require.NoError(t, tt.trigger(ctx, tx2))
			require.Empty(t, conn.statementCache.GetInvalidated())
			require.NotContains(t, conn.preparedStatements, boundSQL)
			require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, boundSQL))

			err = tx1.QueryRow(ctx, "select cast(11 as integer) from rdb$database", QueryExecModeExec).Scan(&n)
			require.NoError(t, err)
			require.EqualValues(t, 11, n)

			err = tx2.QueryRow(ctx, "select cast(22 as integer) from rdb$database", QueryExecModeExec).Scan(&n)
			require.NoError(t, err)
			require.EqualValues(t, 22, n)

			require.NoError(t, tx1.Rollback(ctx))
			require.NoError(t, tx2.Rollback(ctx))

			err = conn.QueryRow(ctx, "select cast(33 as integer) from rdb$database", QueryExecModeExec).Scan(&n)
			require.NoError(t, err)
			require.EqualValues(t, 33, n)
		})
	}
}

func TestScanErrorRollsBackInternalTxAutocommit(t *testing.T) {
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	require.True(t, connConfig.Autocommit, "default Autocommit must be true for this test")

	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	ctx := context.Background()
	const table = "fbx_scan_err_rollback_test"
	_, err := conn.Exec(ctx, "recreate table "+table+" (id integer)")
	require.NoError(t, err)
	defer func() { _, _ = conn.Exec(context.Background(), "drop table "+table) }()

	conn.config.Autocommit = false
	_, err = conn.Exec(ctx, "insert into "+table+" (id) values (1)")
	require.NoError(t, err)
	require.True(t, conn.fbConn.ActiveTx(), "setup insert must remain in the internal tx")

	conn.config.Autocommit = true
	var dst time.Time
	err = conn.QueryRow(ctx, "select cast(42 as integer) from rdb$database").Scan(&dst)
	require.Error(t, err)
	var scanErr ScanArgError
	require.ErrorAs(t, err, &scanErr, "expected ScanArgError from type-mismatched Scan")

	var count int64
	err = conn.QueryRow(ctx, "select count(*) from "+table).Scan(&count)
	require.NoError(t, err)
	assert.EqualValues(t, 0, count,
		"scan-time error must roll back the implicit internal tx, dropping the setup INSERT")
}
