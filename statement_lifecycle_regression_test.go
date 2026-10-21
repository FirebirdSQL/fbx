package fbx

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/stretchr/testify/require"
)

func TestPreparedNameWorksInEveryQueryExecMode(t *testing.T) {
	for _, capacity := range []int{0, 2} {
		for _, mode := range []QueryExecMode{QueryExecModeCacheStatement, QueryExecModeExec} {
			for _, inTx := range []bool{false, true} {
				t.Run(fmt.Sprintf("cache_capacity_%d/%s/tx_%t", capacity, mode, inTx), func(t *testing.T) {
					ctx := context.Background()
					config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
					config.StatementCacheCapacity = capacity
					conn := mustConnect(t, config)
					t.Cleanup(func() { require.NoError(t, conn.Close(context.Background())) })

					const name = "statement_mode_regression"
					const sql = "select cast(? as integer) from rdb$database where 'statement_mode_regression' = 'statement_mode_regression'"
					_, err := conn.Prepare(ctx, name, sql)
					require.NoError(t, err)
					st := conn.preparedStatements[name]
					require.NotNil(t, st)

					var executor interface {
						Exec(context.Context, string, ...any) (fbconn.QueryStatus, error)
						QueryRow(context.Context, string, ...any) Row
					} = conn
					if inTx {
						tx, err := conn.Begin(ctx)
						require.NoError(t, err)
						t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
						executor = tx
					}

					var value int32
					require.NoError(t, executor.QueryRow(ctx, name, mode, int32(11)).Scan(&value))
					require.EqualValues(t, 11, value)
					_, err = executor.Exec(ctx, name, mode, int32(22))
					require.NoError(t, err)
					// Neither executing nor closing the returned rows owns this statement.
					require.NoError(t, executor.QueryRow(ctx, name, mode, int32(33)).Scan(&value))
					require.EqualValues(t, 33, value)
					require.Same(t, st, conn.preparedStatements[name])
					if conn.statementCache != nil {
						require.Zero(t, conn.statementCache.Len())
						require.Empty(t, conn.statementCache.GetInvalidated())
					}
					require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, sql))
				})
			}
		}
	}
}

func TestTxPreparedNameSurvivesPreparingTransaction(t *testing.T) {
	endings := []struct {
		name string
		end  func(context.Context, Tx) error
	}{
		{name: "commit", end: func(ctx context.Context, tx Tx) error { return tx.Commit(ctx) }},
		{name: "rollback", end: func(ctx context.Context, tx Tx) error { return tx.Rollback(ctx) }},
	}
	for _, ending := range endings {
		for _, secondTxBeforePrepare := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/second_tx_before_prepare_%t", ending.name, secondTxBeforePrepare), func(t *testing.T) {
				ctx := context.Background()
				config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
				config.StatementCacheCapacity = 1
				conn := mustConnect(t, config)
				t.Cleanup(func() { require.NoError(t, conn.Close(context.Background())) })

				preparingTx, err := conn.Begin(ctx)
				require.NoError(t, err)
				t.Cleanup(func() { _ = preparingTx.Rollback(context.Background()) })

				var otherTx Tx
				if secondTxBeforePrepare {
					// This makes preparingTx different from fbconn's current transaction.
					otherTx, err = conn.Begin(ctx)
					require.NoError(t, err)
					t.Cleanup(func() { _ = otherTx.Rollback(context.Background()) })
				}

				const name = "tx_prepared_lifetime_regression"
				const sql = "select cast(? as integer) from rdb$database where 'tx_prepared_lifetime' = 'tx_prepared_lifetime'"
				description, err := preparingTx.Prepare(ctx, name, sql)
				require.NoError(t, err)
				st := conn.preparedStatements[name]
				require.NotNil(t, st)
				require.Same(t, description, st.Description)
				require.Zero(t, conn.statementCache.Len())

				var value int32
				require.NoError(t, preparingTx.QueryRow(ctx, name, int32(11)).Scan(&value))
				require.EqualValues(t, 11, value)
				if otherTx != nil {
					require.NoError(t, otherTx.QueryRow(ctx, name, int32(22)).Scan(&value))
					require.EqualValues(t, 22, value)
				}
				require.NoError(t, ending.end(ctx, preparingTx))

				for _, mode := range []QueryExecMode{QueryExecModeCacheStatement, QueryExecModeExec} {
					require.NoError(t, conn.QueryRow(ctx, name, mode, int32(33)).Scan(&value))
					require.EqualValues(t, 33, value)
					_, err = conn.Exec(ctx, name, mode, int32(44))
					require.NoError(t, err)
					require.Same(t, st, conn.preparedStatements[name])
				}
				if otherTx != nil {
					require.NoError(t, otherTx.QueryRow(ctx, name, int32(55)).Scan(&value))
					require.EqualValues(t, 55, value)
					require.NoError(t, otherTx.Rollback(ctx))
				}

				laterTx, err := conn.Begin(ctx)
				require.NoError(t, err)
				t.Cleanup(func() { _ = laterTx.Rollback(context.Background()) })
				reusedDescription, err := laterTx.Prepare(ctx, name, sql)
				require.NoError(t, err)
				require.Same(t, description, reusedDescription)
				require.NoError(t, laterTx.QueryRow(ctx, name, QueryExecModeExec, int32(66)).Scan(&value))
				require.EqualValues(t, 66, value)
				require.NoError(t, laterTx.Rollback(ctx))
				require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, sql))

				require.NoError(t, conn.Deallocate(ctx, name))
				require.NotContains(t, conn.preparedStatements, name)
				require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, sql))
			})
		}
	}
}

func TestStatementCursorClosesAfterOtherTransactionEnds(t *testing.T) {
	for _, finish := range []string{"commit", "rollback"} {
		for _, source := range []string{"cached", "conn_prepared", "tx_prepared"} {
			t.Run(finish+"/"+source, func(t *testing.T) {
				ctx := context.Background()
				config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
				config.DefaultQueryExecMode = QueryExecModeCacheStatement
				config.StatementCacheCapacity = 1
				conn := mustConnect(t, config)
				t.Cleanup(func() { require.NoError(t, conn.Close(context.Background())) })
				t.Logf("server: %s", conn.ServerVersion())

				txA, err := conn.Begin(ctx)
				require.NoError(t, err)
				t.Cleanup(func() { _ = txA.Rollback(context.Background()) })
				// txB becomes the connection's current internal transaction,
				// but all executions below use txA.
				txB, err := conn.Begin(ctx)
				require.NoError(t, err)
				t.Cleanup(func() { _ = txB.Rollback(context.Background()) })

				const sql = "select 41 from rdb$database"
				query := sql
				if source != "cached" {
					query = "cursor_transaction_regression"
					if source == "conn_prepared" {
						_, err = conn.Prepare(ctx, query, sql)
					} else {
						_, err = txA.Prepare(ctx, query, sql)
					}
					require.NoError(t, err)
				}

				checkQuery := func(step string) {
					t.Helper()
					var value int
					// Scan closes Rows, including the cursor owned by txA.
					require.NoError(t, txA.QueryRow(ctx, query).Scan(&value), step)
					require.Equal(t, 41, value, step)
				}
				checkQuery("before ending txB")
				if finish == "commit" {
					require.NoError(t, txB.Commit(ctx))
				} else {
					require.NoError(t, txB.Rollback(ctx))
				}
				checkQuery("first execution after ending txB")
				checkQuery("second execution after ending txB")
			})
		}
	}
}

func TestDeallocateAllFailurePreservesStatementCache(t *testing.T) {
	for _, invalidated := range []bool{false, true} {
		t.Run(fmt.Sprintf("invalidated_%t", invalidated), func(t *testing.T) {
			ctx := context.Background()
			config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
			config.StatementCacheCapacity = 1
			conn := mustConnect(t, config)
			t.Cleanup(func() { require.NoError(t, conn.Close(context.Background())) })

			const sql = "select cast('deallocate_all_retry' as varchar(64)) from rdb$database"
			_, err := conn.Exec(ctx, sql, QueryExecModeCacheStatement)
			require.NoError(t, err)
			st := conn.statementCache.Get(sql)
			require.NotNil(t, st)
			require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, sql))
			if invalidated {
				conn.statementCache.Invalidate(sql)
			}
			cache := conn.statementCache

			// Cancellation before DeallocateAll deterministically fails deallocation
			// before sending a drop request, while leaving the connection usable.
			canceledCtx, cancel := context.WithCancel(ctx)
			cancel()
			require.ErrorIs(t, conn.DeallocateAll(canceledCtx), context.Canceled)
			require.False(t, conn.IsClosed())
			require.Same(t, cache, conn.statementCache)
			require.Same(t, st, conn.preparedStatements[st.Description.Name])
			if invalidated {
				require.Same(t, st, conn.statementCache.GetInvalidatedBySQL(sql))
				require.Equal(t, []*fbconn.Statement{st}, conn.statementCache.GetInvalidated())
			} else {
				require.Same(t, st, conn.statementCache.Get(sql))
				require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, sql))
			}

			require.NoError(t, conn.DeallocateAll(ctx))
			require.Empty(t, conn.preparedStatements)
			require.Zero(t, conn.statementCache.Len())
			require.Empty(t, conn.statementCache.GetInvalidated())
			require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, sql))

			// A retry leaves the cache ready to own newly prepared statements.
			var value string
			require.NoError(t, conn.QueryRow(ctx, sql, QueryExecModeCacheStatement).Scan(&value))
			require.Equal(t, "deallocate_all_retry", value)
			require.NotSame(t, st, conn.statementCache.Get(sql))
			require.NoError(t, conn.DeallocateAll(ctx))
			require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, sql))
		})
	}
}

func TestTemporaryQueryArgumentCountFailureDeallocatesStatement(t *testing.T) {
	ctx := context.Background()
	conn := mustConnect(t, mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE")))
	t.Cleanup(func() { require.NoError(t, conn.Close(context.Background())) })

	const sql = "select cast(? as integer) from rdb$database where 'temporary_parameter_count' = 'temporary_parameter_count'"
	rows, err := conn.Query(ctx, sql, QueryExecModeExec)
	require.Error(t, err)
	require.Error(t, rows.Err())
	// Query failed before producing a cursor, so no Rows.Close call should be
	// needed to release the newly prepared server statement.
	require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, sql))

	var value int32
	require.NoError(t, conn.QueryRow(ctx, sql, QueryExecModeExec, int32(42)).Scan(&value))
	require.EqualValues(t, 42, value)
	require.EqualValues(t, 0, countStatementHandlesForSQL(t, conn, sql))
}

func TestPreparedNameArgumentCountFailureInvalidatesResolvedSQL(t *testing.T) {
	for _, mode := range []QueryExecMode{QueryExecModeCacheStatement, QueryExecModeExec} {
		for _, query := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/query_%t", mode, query), func(t *testing.T) {
				ctx := context.Background()
				config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
				config.StatementCacheCapacity = 2
				conn := mustConnect(t, config)
				t.Cleanup(func() { require.NoError(t, conn.Close(context.Background())) })

				const sql = "select cast(? as integer) from rdb$database where 'resolved_parameter_count' = 'resolved_parameter_count'"
				// A registered name can itself be the SQL of an unrelated cached statement.
				const name = "select cast(99 as integer) from rdb$database where 'unrelated_cached_name' = 'unrelated_cached_name'"
				_, err := conn.Exec(ctx, sql, QueryExecModeCacheStatement, int32(1))
				require.NoError(t, err)
				_, err = conn.Exec(ctx, name, QueryExecModeCacheStatement)
				require.NoError(t, err)
				cachedStatement := conn.statementCache.Get(sql)
				unrelatedStatement := conn.statementCache.Get(name)
				require.NotNil(t, cachedStatement)
				require.NotNil(t, unrelatedStatement)
				_, err = conn.Prepare(ctx, name, sql)
				require.NoError(t, err)
				explicitStatement := conn.preparedStatements[name]

				if query {
					rows, err := conn.Query(ctx, name, mode)
					require.Error(t, err)
					require.Error(t, rows.Err())
				} else {
					_, err := conn.Exec(ctx, name, mode)
					require.Error(t, err)
				}
				require.Nil(t, conn.statementCache.Get(sql))
				require.Same(t, unrelatedStatement, conn.statementCache.Get(name))
				require.Equal(t, []*fbconn.Statement{cachedStatement}, conn.statementCache.GetInvalidated())
				require.Same(t, explicitStatement, conn.preparedStatements[name])

				// The next call cleans up only the invalidated cached handle. The
				// explicit statement remains usable after the argument error.
				var value int32
				require.NoError(t, conn.QueryRow(ctx, name, mode, int32(42)).Scan(&value))
				require.EqualValues(t, 42, value)
				require.Empty(t, conn.statementCache.GetInvalidated())
				require.NotContains(t, conn.preparedStatements, cachedStatement.Description.Name)
				require.Same(t, explicitStatement, conn.preparedStatements[name])
				require.Same(t, unrelatedStatement, conn.statementCache.Get(name))
				require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, sql))
				require.EqualValues(t, 1, countStatementHandlesForSQL(t, conn, name))
			})
		}
	}
}
