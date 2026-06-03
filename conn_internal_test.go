package fbx

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"testing"

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
	for i := 0; i < cacheLimit*2; i++ {
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
