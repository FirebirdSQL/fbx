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
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/internal/stmtcache"
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

func TestCacheStatementModeRequiresCache(t *testing.T) {
	conn := &Conn{
		config: &ConnConfig{
			DefaultQueryExecMode: QueryExecModeCacheStatement,
		},
		preparedStatements: make(map[string]*fbconn.Statement),
		preparedBatches:    make(map[string]*PreparedBatch),
	}

	_, err := conn.Exec(context.Background(), "select 1 from rdb$database")
	require.ErrorIs(t, err, errDisabledStatementCache)

	rows, err := conn.Query(context.Background(), "select 1 from rdb$database")
	require.ErrorIs(t, err, errDisabledStatementCache)
	require.ErrorIs(t, rows.Err(), errDisabledStatementCache)
}

func TestExecAndQueryShareCachedStatement(t *testing.T) {
	ctx := context.Background()
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 2

	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	const query = "select cast(? as integer) from rdb$database where 'rs_328477_shared' = 'rs_328477_shared'"
	_, err := conn.Exec(ctx, query, int32(1))
	require.NoError(t, err)

	cachedStatement := conn.statementCache.Get(query)
	require.NotNil(t, cachedStatement)
	require.Same(t, cachedStatement, conn.preparedStatements[cachedStatement.Description.Name])
	require.NotContains(t, conn.preparedStatements, query)

	var result int32
	err = conn.QueryRow(ctx, query, int32(2)).Scan(&result)
	require.NoError(t, err)
	require.EqualValues(t, 2, result)
	require.Same(t, cachedStatement, conn.statementCache.Get(query))
	require.Same(t, cachedStatement, conn.preparedStatements[cachedStatement.Description.Name])
	require.NotContains(t, conn.preparedStatements, query)
	require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, query))
}

func TestExecPreparedNameInvalidatesCachedSQL(t *testing.T) {
	ctx := context.Background()
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 2

	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	const query = "select cast(? as integer) from rdb$database where 'rs_328477_alias' = 'rs_328477_alias'"
	_, err := conn.Exec(ctx, query, int32(1))
	require.NoError(t, err)
	cachedStatement := conn.statementCache.Get(query)
	require.NotNil(t, cachedStatement)

	const name = "rs_328477_alias"
	_, err = conn.Prepare(ctx, name, query)
	require.NoError(t, err)
	explicitStatement := conn.preparedStatements[name]
	require.NotSame(t, cachedStatement, explicitStatement)

	_, err = conn.Exec(ctx, name)
	require.Error(t, err)
	require.Nil(t, conn.statementCache.Get(query))
	require.Equal(t, []*fbconn.Statement{cachedStatement}, conn.statementCache.GetInvalidated())

	_, err = conn.Exec(ctx, "select 1 from rdb$database", QueryExecModeExec)
	require.NoError(t, err)
	require.Same(t, explicitStatement, conn.preparedStatements[name])
	require.NoError(t, conn.Deallocate(ctx, name))
}

func TestCacheLookupDoesNotTakeOwnershipOfNamedStatement(t *testing.T) {
	ctx := context.Background()
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 1

	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	const query = "select cast('rs_328477_named' as varchar(64)) from rdb$database"
	name := stmtcache.StatementName(query)
	description, err := conn.Prepare(ctx, name, query)
	require.NoError(t, err)
	explicitStatement := conn.preparedStatements[name]
	require.NotNil(t, explicitStatement)
	require.Same(t, description, explicitStatement.Description)

	var value string
	require.NoError(t, conn.QueryRow(ctx, query).Scan(&value))
	require.Equal(t, "rs_328477_named", value)
	require.Zero(t, conn.statementCache.Len())
	require.Same(t, explicitStatement, conn.preparedStatements[name])
}

func TestCachedStatementEvictionDeallocatesExactStatement(t *testing.T) {
	ctx := context.Background()
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 1

	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	const cachedSQL = "select cast('rs_328477_identity' as varchar(64)) from rdb$database"
	_, err := conn.Exec(ctx, cachedSQL)
	require.NoError(t, err)
	cachedStatement := conn.statementCache.Get(cachedSQL)
	require.NotNil(t, cachedStatement)

	explicitDescription, err := conn.Prepare(ctx, cachedSQL, cachedSQL)
	require.NoError(t, err)
	explicitStatement := conn.preparedStatements[cachedSQL]
	require.NotNil(t, explicitStatement)
	require.Same(t, explicitDescription, explicitStatement.Description)
	require.NotSame(t, cachedStatement, explicitStatement)
	require.EqualValues(t, 2, countStatementHandlesForSQL(t, conn, cachedSQL))

	const evictingSQL = "select cast('rs_328477_evict' as varchar(64)) from rdb$database"
	_, err = conn.Exec(ctx, evictingSQL)
	require.NoError(t, err)
	require.Equal(t, []*fbconn.Statement{cachedStatement}, conn.statementCache.GetInvalidated())

	_, err = conn.Exec(ctx, "select 1 from rdb$database", QueryExecModeExec)
	require.NoError(t, err)
	require.Empty(t, conn.statementCache.GetInvalidated())
	require.NotContains(t, conn.preparedStatements, cachedStatement.Description.Name)
	require.Same(t, explicitStatement, conn.preparedStatements[cachedSQL])
	require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, cachedSQL))

	var value string
	err = conn.QueryRow(ctx, cachedSQL).Scan(&value)
	require.NoError(t, err)
	require.Equal(t, "rs_328477_identity", value)

	require.NoError(t, conn.Deallocate(ctx, cachedSQL))
	require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, cachedSQL))
}

func TestDeallocateCachedStatementRemovesCacheEntry(t *testing.T) {
	ctx := context.Background()
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 1

	conn := mustConnect(t, connConfig)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	const query = "select cast('rs_328477_deallocate' as varchar(64)) from rdb$database"
	_, err := conn.Exec(ctx, query)
	require.NoError(t, err)
	st := conn.statementCache.Get(query)
	require.NotNil(t, st)

	require.NoError(t, conn.Deallocate(ctx, query))
	require.Nil(t, conn.statementCache.Get(query))
	require.NotContains(t, conn.preparedStatements, st.Description.Name)
	require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, query))

	var value string
	require.NoError(t, conn.QueryRow(ctx, query).Scan(&value))
	require.Equal(t, "rs_328477_deallocate", value)
	require.NotSame(t, st, conn.statementCache.Get(query))
}

func TestDeallocateInvalidatedCachedStatementBySQL(t *testing.T) {
	ctx := context.Background()
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 1

	conn := mustConnect(t, connConfig)
	deallocated := false
	defer func() {
		if !deallocated {
			require.NoError(t, conn.Close(context.Background()))
		}
	}()

	const query = "select cast('rs_328477_pending' as varchar(64)) from rdb$database"
	_, err := conn.Exec(ctx, query)
	require.NoError(t, err)
	st := conn.statementCache.Get(query)
	require.NotNil(t, st)
	explicitDescription, err := conn.Prepare(ctx, query, query)
	require.NoError(t, err)
	explicitStatement := conn.preparedStatements[query]
	require.NotNil(t, explicitStatement)
	require.Same(t, explicitDescription, explicitStatement.Description)
	require.NotSame(t, st, explicitStatement)
	require.EqualValues(t, 2, countStatementHandlesForSQL(t, conn, query))

	conn.statementCache.Invalidate(query)
	require.Same(t, st, conn.statementCache.GetInvalidatedBySQL(query))
	require.NoError(t, conn.Deallocate(ctx, query))
	require.Nil(t, conn.statementCache.GetInvalidatedBySQL(query))
	require.Empty(t, conn.statementCache.GetInvalidated())
	require.NotContains(t, conn.preparedStatements, st.Description.Name)
	require.NotContains(t, conn.preparedStatements, query)
	require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, query))

	require.NoError(t, conn.Close(ctx))
	deallocated = true
}

func TestDeallocateOneOfMultipleInvalidatedCachedStatements(t *testing.T) {
	ctx := context.Background()
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 2

	conn := mustConnect(t, connConfig)
	deallocated := false
	defer func() {
		if !deallocated {
			require.NoError(t, conn.Close(context.Background()))
		}
	}()

	const firstSQL = "select cast('rs_328477_pending_first' as varchar(64)) from rdb$database"
	const secondSQL = "select cast('rs_328477_pending_second' as varchar(64)) from rdb$database"
	_, err := conn.Exec(ctx, firstSQL)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, secondSQL)
	require.NoError(t, err)
	firstStatement := conn.statementCache.Get(firstSQL)
	secondStatement := conn.statementCache.Get(secondSQL)
	require.NotNil(t, firstStatement)
	require.NotNil(t, secondStatement)

	conn.statementCache.InvalidateAll()
	require.Len(t, conn.statementCache.GetInvalidated(), 2)
	require.NoError(t, conn.Deallocate(ctx, firstSQL))
	require.Nil(t, conn.statementCache.GetInvalidatedBySQL(firstSQL))
	require.Same(t, secondStatement, conn.statementCache.GetInvalidatedBySQL(secondSQL))
	require.Equal(t, []*fbconn.Statement{secondStatement}, conn.statementCache.GetInvalidated())
	require.NotContains(t, conn.preparedStatements, firstStatement.Description.Name)
	require.Contains(t, conn.preparedStatements, secondStatement.Description.Name)

	_, err = conn.Exec(ctx, "select 1 from rdb$database", QueryExecModeExec)
	require.NoError(t, err)
	require.Empty(t, conn.statementCache.GetInvalidated())
	require.NotContains(t, conn.preparedStatements, secondStatement.Description.Name)
	require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, firstSQL))
	require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, secondSQL))

	require.NoError(t, conn.Close(ctx))
	deallocated = true
}

func TestCachedStatementPreparedInTransactionIsReusableAfterTransactionEnds(t *testing.T) {
	tests := []struct {
		name  string
		close func(context.Context, Tx) error
	}{
		{name: "commit", close: func(ctx context.Context, tx Tx) error { return tx.Commit(ctx) }},
		{name: "rollback", close: func(ctx context.Context, tx Tx) error { return tx.Rollback(ctx) }},
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

			tx, err := conn.Begin(ctx)
			require.NoError(t, err)
			currentTx, err := conn.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = currentTx.Rollback(context.Background()) }()
			const query = "select cast(? as integer) from rdb$database where 'rs_328477_tx' = 'rs_328477_tx'"
			_, err = tx.Exec(ctx, query, int32(1))
			require.NoError(t, err)
			cachedStatement := conn.statementCache.Get(query)
			require.NotNil(t, cachedStatement)

			require.NoError(t, tt.close(ctx, tx))

			var value int32
			require.NoError(t, conn.QueryRow(ctx, query, int32(2)).Scan(&value))
			require.EqualValues(t, 2, value)
			require.Same(t, cachedStatement, conn.statementCache.Get(query))
			require.NoError(t, currentTx.Rollback(ctx))
		})
	}
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

			// Starting tx2 makes it the connection's current internal transaction.
			// The cached statement below is prepared and executed through tx1,
			// while its lifetime belongs to the connection.
			tx2, err := conn.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx2.Rollback(context.Background()) }()

			const boundSQL = "select cast(? as integer) + 1 from rdb$database"
			var n int32
			err = tx1.QueryRow(ctx, boundSQL, 1).Scan(&n)
			require.NoError(t, err)
			require.EqualValues(t, 2, n)
			boundStatement := conn.statementCache.Get(boundSQL)
			require.NotNil(t, boundStatement)
			require.Same(t, boundStatement, conn.preparedStatements[boundStatement.Description.Name])

			// The cache only has room for one statement, so this puts the statement
			// prepared through tx1 on the invalidated list.
			const evictingSQL = "select cast(? as integer) + 2 from rdb$database"
			err = tx2.QueryRow(ctx, evictingSQL, 1).Scan(&n)
			require.NoError(t, err)
			require.EqualValues(t, 3, n)
			invalidated := conn.statementCache.GetInvalidated()
			require.Len(t, invalidated, 1)
			require.Same(t, boundStatement, invalidated[0])

			// Exec and Query both deallocate invalidated statements before starting
			// their own operation. Deallocating a statement prepared through tx1
			// when called through tx2 must not affect either transaction or leak its server handle.
			require.NoError(t, tt.trigger(ctx, tx2))
			require.Empty(t, conn.statementCache.GetInvalidated())
			require.NotContains(t, conn.preparedStatements, boundStatement.Description.Name)
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

type earlyExecErrorTracer struct {
	starts  int
	ends    int
	endData TraceQueryEndData
}

func (t *earlyExecErrorTracer) TraceQueryStart(ctx context.Context, conn *Conn, data TraceQueryStartData) context.Context {
	t.starts++
	return ctx
}

func (t *earlyExecErrorTracer) TraceQueryEnd(ctx context.Context, conn *Conn, data TraceQueryEndData) {
	t.ends++
	t.endData = data
}

func TestExecTraceEndsWhenInvalidatedStatementDeallocateFails(t *testing.T) {
	ctx := context.Background()
	tracer := &earlyExecErrorTracer{}
	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = QueryExecModeCacheStatement
	connConfig.StatementCacheCapacity = 1
	connConfig.Tracer = tracer

	conn := mustConnect(t, connConfig)
	defer conn.Close(ctx)

	const cachedSQL = "select 'early_trace_deallocate' from rdb$database"
	_, err := conn.Exec(ctx, cachedSQL)
	require.NoError(t, err)
	st := conn.statementCache.Get(cachedSQL)
	require.NotNil(t, st)
	conn.statementCache.Invalidate(cachedSQL)
	require.Same(t, st, conn.statementCache.GetInvalidatedBySQL(cachedSQL))

	tracer.starts = 0
	tracer.ends = 0

	const notCachedSQL = "select 'early_trace_not_executed' from rdb$database"
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = conn.Exec(canceledCtx, notCachedSQL)
	require.ErrorIs(t, err, context.Canceled)

	require.Equal(t, 1, tracer.starts)
	require.Equal(t, 1, tracer.ends)
	require.Equal(t, err, tracer.endData.Err)
	require.Zero(t, tracer.endData.QueryStatus)
	// Deallocation failed before RemoveInvalidated, so the new statement was not prepared.
	require.Same(t, st, conn.statementCache.GetInvalidatedBySQL(cachedSQL))
	require.Nil(t, conn.statementCache.Get(notCachedSQL))
}
