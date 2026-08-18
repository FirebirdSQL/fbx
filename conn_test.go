package fbx_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestConnect(t *testing.T) {
	t.Parallel()

	connString := os.Getenv("FBX_TEST_DATABASE")
	config := mustParseConfig(t, connString)

	conn, err := fbx.ConnectConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Unable to establish connection: %v", err)
	}

	assertConfigsEqual(t, config, conn.Config(), "Conn.Config() returns original config")
	require.Equal(t, conn.FbConn().ServerVersion(), conn.ServerVersion())
	require.Positive(t, conn.ServerVersion().Major)

	var currentDB string
	err = conn.QueryRow(context.Background(), "select mon$attachment_name from mon$attachments where mon$attachment_id = current_connection").Scan(&currentDB)
	if err != nil {
		t.Fatalf("QueryRow Scan unexpectedly failed: %v", err)
	}
	if currentDB != config.Config.Database {
		t.Errorf("Did not connect to specified database (%v)", config.Config.Database)
	}

	var user string
	err = conn.QueryRow(context.Background(), "select trim(mon$user) from mon$attachments where mon$attachment_id = current_connection").Scan(&user)
	if err != nil {
		t.Fatalf("QueryRow Scan unexpectedly failed: %v", err)
	}
	if user != strings.ToUpper(config.Config.User) {
		t.Errorf("Did not connect as specified user (%v)", config.Config.User)
	}

	err = conn.Close(context.Background())
	if err != nil {
		t.Fatal("Unable to close connection")
	}
}

func TestCreateDatabase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := newCreateDatabaseTestConfig(t, "created.fdb")
	conn, err := fbx.CreateDatabaseConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	var result int32
	err = conn.QueryRow(ctx, "select 1 from rdb$database").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, int32(1), result)
}

func TestDropDatabase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := newCreateDatabaseTestConfig(t, "dropped.fdb")
	conn, err := fbx.CreateDatabaseConfig(ctx, config)
	require.NoError(t, err)

	err = conn.DropDatabase(ctx)
	require.NoError(t, err)
	assert.True(t, conn.IsClosed())

	_, err = os.Stat(config.Database)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDropDatabaseConfig(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := newCreateDatabaseTestConfig(t, "drop-config.fdb")
	conn, err := fbx.CreateDatabaseConfig(ctx, config)
	require.NoError(t, err)
	closeConn(t, conn)

	err = fbx.DropDatabaseConfig(ctx, config)
	require.NoError(t, err)

	_, err = os.Stat(config.Database)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDropDatabaseMissingDatabase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := newCreateDatabaseTestConfig(t, "missing.fdb")
	_, err := os.Stat(config.Database)
	require.ErrorIs(t, err, os.ErrNotExist)

	err = fbx.DropDatabaseConfig(ctx, config)
	require.Error(t, err)

	_, err = os.Stat(config.Database)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCreateDatabaseExistingDatabase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := newCreateDatabaseTestConfig(t, "existing.fdb")
	conn, err := fbx.CreateDatabaseConfig(ctx, config)
	require.NoError(t, err)
	closeConn(t, conn)

	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err = fbx.CreateDatabaseConfig(ctx, config)
		if conn != nil {
			closeConn(t, conn)
		}

		var retryErr *fbconn.FbError
		if !errors.As(err, &retryErr) ||
			!strings.Contains(strings.ToLower(retryErr.Message), "object database is in use") ||
			time.Now().After(deadline) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
	require.Error(t, err)

	var fbErr *fbconn.FbError
	require.ErrorAs(t, err, &fbErr)
	assert.Truef(t,
		fbErr.Code == "HY000" || fbErr.Code == "08001",
		"unexpected SQLSTATE: %s (%s)", fbErr.Code, fbErr.Message,
	)
	assert.Truef(t,
		strings.Contains(fbErr.Message, "database or file exists") ||
			strings.Contains(fbErr.Message, "Error while trying to create file"),
		"unexpected error message: %s", fbErr.Message,
	)
}

func TestConnectWithPreferSimpleProtocol(t *testing.T) {
	t.Parallel()

	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = fbx.QueryExecModeExec

	conn := mustConnect(t, connConfig)
	defer closeConn(t, conn)

	// If simple protocol is used we should be able to correctly scan the result
	// into a fbtype.Text as the integer will have been encoded in text.

	var s fbtype.Text
	err := conn.QueryRow(context.Background(), "select cast(? as int) from rdb$database", 42).Scan(&s)
	require.NoError(t, err)
	require.Equal(t, fbtype.Text{String: "42", Valid: true}, s)

	ensureConnValid(t, conn)
}

func newCreateDatabaseTestConfig(t testing.TB, filename string) *fbx.ConnConfig {
	t.Helper()

	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "FBX_TEST_DATABASE")
	}

	config := mustParseConfig(t, connString)

	dir, err := os.MkdirTemp("", "fbx-create-db-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	require.NoError(t, os.MkdirAll(dir, 0o777))
	require.NoError(t, os.Chmod(dir, 0o777))

	config.Database = filepath.Join(dir, fmt.Sprintf("fbx-create-%d-%s", time.Now().UnixNano(), filename))
	t.Cleanup(func() {
		_ = os.Remove(config.Database)
	})
	return config
}

func TestConnectConfigRequiresConnConfigFromParseConfig(t *testing.T) {
	config := &fbx.ConnConfig{}
	require.PanicsWithValue(t, "config must be created by ParseConfig", func() {
		fbx.ConnectConfig(context.Background(), config)
	})
}

func TestConfigContainsConnStr(t *testing.T) {
	connStr := os.Getenv("FBX_TEST_DATABASE")
	config, err := fbx.ParseConfig(connStr)
	require.NoError(t, err)
	assert.Equal(t, connStr, config.ConnString())
}

func TestParseConfigCreateDatabase(t *testing.T) {
	config, err := fbx.ParseConfig("user=ivan password=secret host=localhost port=3050 database=new.fdb create_database=true")
	require.NoError(t, err)
	assert.True(t, config.CreateDatabase)
	assert.Empty(t, config.RuntimeParams["create_database"])
}

func TestConfigCopyReturnsEqualConfig(t *testing.T) {
	connString := "firebird://ivan:secret@localhost:3050/mydb?application_name=fbxtest&connect_timeout=5"
	original, err := fbx.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assertConfigsEqual(t, original, copied, t.Name())
}

func TestConfigCopyCanBeUsedToConnect(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	original, err := fbx.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assert.NotPanics(t, func() {
		_, err = fbx.ConnectConfig(context.Background(), copied)
	})
	assert.NoError(t, err)
}

func TestParseConfigExtractsStatementCacheOptions(t *testing.T) {
	t.Parallel()

	config, err := fbx.ParseConfig("statement_cache_capacity=0")
	require.NoError(t, err)
	require.EqualValues(t, 0, config.StatementCacheCapacity)

	config, err = fbx.ParseConfig("statement_cache_capacity=42")
	require.NoError(t, err)
	require.EqualValues(t, 42, config.StatementCacheCapacity)

	config, err = fbx.ParseConfig("default_query_exec_mode=cache_statement")
	require.NoError(t, err)
	require.Equal(t, fbx.QueryExecModeCacheStatement, config.DefaultQueryExecMode)

	config, err = fbx.ParseConfig("default_query_exec_mode=exec")
	require.NoError(t, err)
	require.Equal(t, fbx.QueryExecModeExec, config.DefaultQueryExecMode)

	config, err = fbx.ParseConfig("fetch_batch_size=42")
	require.NoError(t, err)
	require.EqualValues(t, 42, config.FetchBatchSize)
	require.Empty(t, config.RuntimeParams["fetch_batch_size"])
}

func TestParseConfigExtractsDefaultQueryExecMode(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		connString           string
		defaultQueryExecMode fbx.QueryExecMode
	}{
		{"", fbx.QueryExecModeCacheStatement},
		{"default_query_exec_mode=cache_statement", fbx.QueryExecModeCacheStatement},
		{"default_query_exec_mode=exec", fbx.QueryExecModeExec},
	} {
		config, err := fbx.ParseConfig(tt.connString)
		require.NoError(t, err)
		require.Equalf(t, tt.defaultQueryExecMode, config.DefaultQueryExecMode, "connString: `%s`", tt.connString)
		require.Empty(t, config.RuntimeParams["default_query_exec_mode"])
	}
}

func TestParseConfigErrors(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		connString           string
		expectedErrSubstring string
	}{
		{"default_query_exec_mode=does_not_exist", "does_not_exist"},
	} {
		config, err := fbx.ParseConfig(tt.connString)
		require.Nil(t, config)
		require.ErrorContains(t, err, tt.expectedErrSubstring)
	}
}

func TestExec(t *testing.T) {
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		results := mustExec(t, conn, "recreate table conn_exec_test (id integer primary key)")
		assert.Equal(t, "DDL", results.String())

		// Accept parameters
		results = mustExec(t, conn, "insert into conn_exec_test (id) values(?)", 1)
		assert.Equal(t, "INSERT", results.String())

		err := conn.DeallocateAll(ctx)
		assert.NoError(t, err)

		results = mustExec(t, conn, "drop table conn_exec_test")
		assert.Equal(t, "DDL", results.String())
	})
}

type testQueryRewriter struct {
	sql  string
	args []any
}

func (qr *testQueryRewriter) RewriteQuery(ctx context.Context, conn *fbx.Conn, sql string, args []any) (newSQL string, newArgs []any, err error) {
	return qr.sql, qr.args, nil
}

func TestExecWithQueryRewriter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		qr := testQueryRewriter{sql: "select cast(? as int) from rdb$database", args: []any{42}}
		_, err := conn.Exec(ctx, "should be replaced", &qr)
		require.NoError(t, err)
	})
}

func TestExecFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		if _, err := conn.Exec(context.Background(), "selct;"); err == nil {
			t.Fatal("Expected SQL syntax error")
		}

		rows, _ := conn.Query(context.Background(), "select 1 from rdb$database")
		rows.Close()
		if rows.Err() != nil {
			t.Fatalf("Exec failure appears to have broken connection: %v", rows.Err())
		}
	})
}

func TestExecFailureWithArguments(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Exec(context.Background(), "selct $1;", 1)
		if err == nil {
			t.Fatal("Expected SQL syntax error")
		}
		assert.False(t, fbconn.SafeToRetry(err))

		_, err = conn.Exec(context.Background(), "select $1::varchar(1);", "1", "2")
		require.Error(t, err)
	})
}

func TestExecContextWithoutCancelation(t *testing.T) {
	//t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		commandTag, err := conn.Exec(ctx, "recreate table conn_exec_context_test (id integer primary key);")
		if err != nil {
			t.Fatal(err)
		}
		if commandTag.String() != "DDL" {
			t.Fatalf("Unexpected results from Exec: %v", commandTag)
		}
		assert.False(t, fbconn.SafeToRetry(err))
	})
}

func TestExecContextFailureWithoutCancelation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		_, err := conn.Exec(ctx, "selct;")
		if err == nil {
			t.Fatal("Expected SQL syntax error")
		}
		assert.False(t, fbconn.SafeToRetry(err))

		rows, _ := conn.Query(context.Background(), "select 1 from rdb$database")
		rows.Close()
		if rows.Err() != nil {
			t.Fatalf("ExecEx failure appears to have broken connection: %v", rows.Err())
		}
		assert.False(t, fbconn.SafeToRetry(err))
	})
}

func TestExecContextFailureWithoutCancelationWithArguments(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		_, err := conn.Exec(ctx, "selct $1;", 1)
		if err == nil {
			t.Fatal("Expected SQL syntax error")
		}
		assert.False(t, fbconn.SafeToRetry(err))
	})
}

func TestExecFailureCloseBefore(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	closeConn(t, conn)

	_, err := conn.Exec(context.Background(), "select 1 from rdb$database")
	require.Error(t, err)
	assert.True(t, fbconn.SafeToRetry(err))
}

func TestPrepare(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)

	//select $1::varchar
	_, err := conn.Prepare(context.Background(), "test", "select cast(? as varchar(255)) from rdb$database")
	if err != nil {
		t.Errorf("Unable to prepare statement: %v", err)
		return
	}

	var s string
	err = conn.QueryRow(context.Background(), "test", "hello").Scan(&s)
	if err != nil {
		t.Errorf("Executing prepared statement failed: %v", err)
	}

	if s != "hello" {
		t.Errorf("Prepared statement did not return expected value: %v", s)
	}

	err = conn.Deallocate(context.Background(), "test")
	if err != nil {
		t.Errorf("conn.Deallocate failed: %v", err)
	}

	// Create another prepared statement to ensure Deallocate left the connection
	// in a working state and that we can reuse the prepared statement name.

	_, err = conn.Prepare(context.Background(), "test", "select cast(? as integer) from rdb$database")
	if err != nil {
		t.Errorf("Unable to prepare statement: %v", err)
		return
	}

	var n int32
	err = conn.QueryRow(context.Background(), "test", int32(1)).Scan(&n)
	if err != nil {
		t.Errorf("Executing prepared statement failed: %v", err)
	}

	if n != 1 {
		t.Errorf("Prepared statement did not return expected value: %v", s)
	}

	err = conn.DeallocateAll(context.Background())
	if err != nil {
		t.Errorf("conn.Deallocate failed: %v", err)
	}
}

type batchDMLRow struct {
	ID   int32
	Note string
}

func batchTestTableName(t testing.TB, prefix string) string {
	t.Helper()

	var b strings.Builder
	b.Grow(len(prefix) + len(t.Name()) + 1)
	b.WriteString(prefix)
	b.WriteByte('_')
	for _, r := range t.Name() {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	name := b.String()
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

func seedBatchDMLRows(t testing.TB, ctx context.Context, conn *fbx.Conn, table string) {
	t.Helper()

	for _, row := range []batchDMLRow{
		{ID: 1, Note: "one"},
		{ID: 2, Note: "two"},
		{ID: 3, Note: "three"},
		{ID: 4, Note: "four"},
	} {
		_, err := conn.Exec(ctx, "insert into "+table+" (id, note) values (?, ?)", row.ID, row.Note)
		require.NoError(t, err)
	}
}

func loadBatchDMLRows(t testing.TB, ctx context.Context, conn *fbx.Conn, table string) []batchDMLRow {
	t.Helper()

	rows, err := conn.Query(ctx, fmt.Sprintf("select id, note from %s order by id", table))
	require.NoError(t, err)
	defer rows.Close()

	var result []batchDMLRow
	for rows.Next() {
		var row batchDMLRow
		require.NoError(t, rows.Scan(&row.ID, &row.Note))
		result = append(result, row)
	}
	require.NoError(t, rows.Err())
	return result
}

func TestPrepareBatchUpdateAndDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipIfBatchUnsupported(ctx, t, conn)
		table := batchTestTableName(t, "fbx_bud")

		_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
		require.NoError(t, err)
		seedBatchDMLRows(t, ctx, conn, table)

		pbUpdate, err := conn.PrepareBatch(ctx, "update "+table+" set note = ? where id = ?")
		require.NoError(t, err)
		defer pbUpdate.Close(ctx)

		require.NoError(t, pbUpdate.Add("one-updated", 1))
		require.NoError(t, pbUpdate.Add("two-updated", 2))

		updateResult, err := pbUpdate.Exec(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 2, updateResult.Affected)

		pbDelete, err := conn.PrepareBatch(ctx, "delete from "+table+" where id = ?")
		require.NoError(t, err)
		defer pbDelete.Close(ctx)

		require.NoError(t, pbDelete.Add(3))
		require.NoError(t, pbDelete.Add(4))

		deleteResult, err := pbDelete.Exec(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 2, deleteResult.Affected)

		require.Equal(t, []batchDMLRow{
			{ID: 1, Note: "one-updated"},
			{ID: 2, Note: "two-updated"},
		}, loadBatchDMLRows(t, ctx, conn, table))
	})
}

func TestPrepareNamedBatchUpdateAndDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipIfBatchUnsupported(ctx, t, conn)
		table := batchTestTableName(t, "fbx_nbv")
		updateName := batchTestTableName(t, "upd_batch")
		deleteName := batchTestTableName(t, "del_batch")

		_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
		require.NoError(t, err)
		seedBatchDMLRows(t, ctx, conn, table)

		_, err = conn.Prepare(ctx, updateName, "batch update "+table+" set note = ? where id = ?")
		require.NoError(t, err)
		defer func() { require.NoError(t, conn.Deallocate(ctx, updateName)) }()

		_, err = conn.Exec(ctx, updateName, "one-named", 1)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, updateName, "two-named", 2)
		require.NoError(t, err)

		updateResult, err := conn.ExecBatch(ctx, updateName)
		require.NoError(t, err)
		require.EqualValues(t, 2, updateResult.Affected)

		_, err = conn.Prepare(ctx, deleteName, "batch delete from "+table+" where id = ?")
		require.NoError(t, err)
		defer func() { require.NoError(t, conn.Deallocate(ctx, deleteName)) }()

		_, err = conn.Exec(ctx, deleteName, 3)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, deleteName, 4)
		require.NoError(t, err)

		deleteResult, err := conn.ExecBatch(ctx, deleteName)
		require.NoError(t, err)
		require.EqualValues(t, 2, deleteResult.Affected)

		require.Equal(t, []batchDMLRow{
			{ID: 1, Note: "one-named"},
			{ID: 2, Note: "two-named"},
		}, loadBatchDMLRows(t, ctx, conn, table))
	})
}

func TestPrepareBatchRejectsResultSetStatements(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipIfBatchUnsupported(ctx, t, conn)
		table := batchTestTableName(t, "fbx_brr")
		badBatchName := batchTestTableName(t, "bad_batch")

		_, err := conn.PrepareBatch(ctx, "select cast(? as integer) from rdb$database")
		require.ErrorContains(t, err, "batch supports only insert, update, and delete statements without result sets")

		_, err = conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, note varchar(50))")
		require.NoError(t, err)

		_, err = conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (?, ?) returning id")
		require.ErrorContains(t, err, "batch supports only insert, update, and delete statements without result sets")

		_, err = conn.PrepareBatch(ctx, "insert into "+table+" (id, note) values (1, 'no params')")
		require.ErrorContains(t, err, "statement used in batch must have parameters")

		_, err = conn.Prepare(ctx, badBatchName, "batch select cast(? as integer) from rdb$database")
		require.ErrorContains(t, err, "batch supports only insert, update, and delete statements without result sets")

		_, err = conn.Prepare(ctx, badBatchName+"_noparams", "batch update "+table+" set note = 'no params' where id = 1")
		require.ErrorContains(t, err, "statement used in batch must have parameters")

		ensureConnValid(t, conn)
	})
}

func TestPrepareBatchRejectsBlobParameters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipIfBatchUnsupported(ctx, t, conn)
		table := batchTestTableName(t, "fbx_bblob")
		badBatchName := batchTestTableName(t, "bad_blob_batch")

		_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, payload blob sub_type 0)")
		require.NoError(t, err)

		_, err = conn.PrepareBatch(ctx, "insert into "+table+" (id, payload) values (?, ?)")
		require.ErrorContains(t, err, "batch does not support blob parameters")

		_, err = conn.Prepare(ctx, badBatchName, "batch insert into "+table+" (id, payload) values (?, ?)")
		require.ErrorContains(t, err, "batch does not support blob parameters")

		ensureConnValid(t, conn)
	})
}

func TestPrepareBatchRejectsArrayParameters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipIfBatchUnsupported(ctx, t, conn)
		table := batchTestTableName(t, "fbx_barr")
		badBatchName := batchTestTableName(t, "bad_array_batch")

		_, err := conn.Exec(ctx, "recreate table "+table+" (id integer not null primary key, val smallint[1:3])")
		require.NoError(t, err)

		_, err = conn.PrepareBatch(ctx, "insert into "+table+" (id, val) values (?, ?)")
		require.ErrorContains(t, err, "batch does not support array parameters")

		_, err = conn.Prepare(ctx, badBatchName, "batch insert into "+table+" (id, val) values (?, ?)")
		require.ErrorContains(t, err, "batch does not support array parameters")

		ensureConnValid(t, conn)
	})
}

func TestPrepareBadSQLFailure(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)

	if _, err := conn.Prepare(context.Background(), "badSQL", "select foo"); err == nil {
		t.Fatal("Prepare should have failed with syntax error")
	}

	ensureConnValid(t, conn)
}

func TestPrepareIdempotency(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		for i := 0; i < 2; i++ {
			_, err := conn.Prepare(context.Background(), "test", "select cast(42 as integer) from rdb$database")
			if err != nil {
				t.Fatalf("%d. Unable to prepare statement: %v", i, err)
			}

			var n int32
			err = conn.QueryRow(context.Background(), "test").Scan(&n)
			if err != nil {
				t.Errorf("%d. Executing prepared statement failed: %v", i, err)
			}

			if n != int32(42) {
				t.Errorf("%d. Prepared statement did not return expected value: %v", i, n)
			}
		}

		_, err := conn.Prepare(context.Background(), "test", "select cast('fail' as varchar(255)) from rdb$database")
		if err == nil {
			t.Fatalf("Prepare statement with same name but different SQL should have failed but it didn't")
			return
		}
	})
}

func TestPrepareStatementCacheModes(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Prepare(context.Background(), "test", "select cast(? as varchar(255)) from rdb$database")
		require.NoError(t, err)

		var s string
		err = conn.QueryRow(context.Background(), "test", "hello").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "hello", s)
	})
}

func TestPrepareWithDigestedName(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		sql := "select cast(? as varchar(255)) from rdb$database"
		sd, err := conn.Prepare(ctx, sql, sql)
		require.NoError(t, err)
		require.Equal(t, "stmt_efcb7247499c7e1936769a619e6a36c23049b0171aa4d5cb", sd.Name)

		var s string
		err = conn.QueryRow(ctx, sql, "hello").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "hello", s)

		err = conn.Deallocate(ctx, sql)
		require.NoError(t, err)
	})
}

func TestDeallocateInAbortedTransaction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		sql := "select cast(? as varchar(255)) from rdb$database"
		sd, err := tx.Prepare(ctx, sql, sql)
		require.NoError(t, err)
		require.Equal(t, "stmt_efcb7247499c7e1936769a619e6a36c23049b0171aa4d5cb", sd.Name)

		var s string
		err = tx.QueryRow(ctx, sql, "hello").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "hello", s)

		_, err = tx.Exec(ctx, "select 1/0") // abort transaction with divide by zero error
		require.Error(t, err)

		err = conn.Deallocate(ctx, sql)
		require.NoError(t, err)

		err = tx.Rollback(ctx)
		require.NoError(t, err)

		sd, err = conn.Prepare(ctx, sql, sql)
		require.NoError(t, err)
		require.Equal(t, "stmt_efcb7247499c7e1936769a619e6a36c23049b0171aa4d5cb", sd.Name)
	})
}

func TestDeallocateMissingPreparedStatementStillClearsFromPreparedStatementMap(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Prepare(ctx, "ps", "select cast(? as varchar(255)) from rdb$database")
		require.NoError(t, err)

		err = conn.Deallocate(ctx, "ps")
		require.NoError(t, err)

		_, err = conn.Prepare(ctx, "ps", "select  cast(? as varchar(255)), cast(? as varchar(255)) from rdb$database")
		require.NoError(t, err)

		var s1, s2 string
		err = conn.QueryRow(ctx, "ps", "hello", "world").Scan(&s1, &s2)
		require.NoError(t, err)
		require.Equal(t, "hello", s1)
		require.Equal(t, "world", s2)
	})
}

func TestFatalRxError(t *testing.T) {
	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)

	_, err := conn.Exec(context.Background(), "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;")
	require.NoError(t, err)

	var wg sync.WaitGroup
	queryStarted := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		var n int32
		var s string
		close(queryStarted)
		err := conn.QueryRow(context.Background(), "select 1, fbsleep(10) from rdb$database").Scan(&n, &s)
		if err == nil {
			t.Errorf("Expected QueryRow Scan to return an error, but got nil")
			return
		}

		var fbErr *fbconn.FbError
		if errors.As(err, &fbErr) { // && fbErr.Severity == "FATAL" {
			fmt.Println(fbErr)
			return
		}
		t.Errorf("Expected QueryRow Scan to return fatal FbError, but instead received %v", err)
	}()

	<-queryStarted
	deadline := time.Now().Add(2 * time.Second)
	for !conn.FbConn().IsBusy() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if err := conn.FbConn().Conn().Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Unable to force-close socket: %v", err)
	}

	wg.Wait()

	if !conn.IsClosed() {
		t.Fatal("Connection should be closed")
	}
}

func TestFatalTxError(t *testing.T) {
	var err error
	t.Parallel()

	// Run timing sensitive test many times
	for i := 0; i < 50; i++ {
		func() {
			conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
			defer closeConn(t, conn)

			otherConn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
			defer otherConn.Close(context.Background())

			if err = conn.FbConn().Disconnect(context.Background()); err != nil {
				t.Fatalf("Unable to disconnect: %v", err)
			}

			err = conn.QueryRow(context.Background(), "select 1 from rdb$database").Scan(nil)
			if err == nil {
				t.Fatal("Expected error but none occurred")
			}

			if !conn.IsClosed() {
				t.Fatalf("Connection should be closed but isn't. Previous Query err: %v", err)
			}
		}()
	}
}

func TestInsertBoolArray(t *testing.T) {
	t.Skip("need implementation")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		if results := mustExec(t, conn, "create temporary table foo(spice bool[]);"); results.String() != "CREATE TABLE" {
			t.Error("Unexpected results from Exec")
		}

		// Accept parameters
		if results := mustExec(t, conn, "insert into foo(spice) values($1)", []bool{true, false, true}); results.String() != "INSERT 0 1" {
			t.Errorf("Unexpected results from Exec: %v", results)
		}
	})
}

func TestInsertTimestampArray(t *testing.T) {
	t.Skip("need implementation")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		if results := mustExec(t, conn, "create temporary table foo(spice timestamp[]);"); results.String() != "CREATE TABLE" {
			t.Error("Unexpected results from Exec")
		}

		// Accept parameters
		if results := mustExec(t, conn, "insert into foo(spice) values($1)", []time.Time{time.Unix(1419143667, 0), time.Unix(1419143672, 0)}); results.String() != "INSERT 0 1" {
			t.Errorf("Unexpected results from Exec: %v", results)
		}
	})
}

func TestIdentifierSanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ident    fbx.Identifier
		expected string
	}{
		{
			ident:    fbx.Identifier{`foo`},
			expected: `"foo"`,
		},
		{
			ident:    fbx.Identifier{`select`},
			expected: `"select"`,
		},
		{
			ident:    fbx.Identifier{`foo`, `bar`},
			expected: `"foo"."bar"`,
		},
		{
			ident:    fbx.Identifier{`you should " not do this`},
			expected: `"you should "" not do this"`,
		},
		{
			ident:    fbx.Identifier{`you should " not do this`, `please don't`},
			expected: `"you should "" not do this"."please don't"`,
		},
		{
			ident:    fbx.Identifier{`you should ` + string([]byte{0}) + `not do this`},
			expected: `"you should not do this"`,
		},
	}

	for i, tt := range tests {
		qval := tt.ident.Sanitize()
		if qval != tt.expected {
			t.Errorf("%d. Expected Sanitize %v to return %v but it was %v", i, tt.ident, tt.expected, qval)
		}
	}
}

func TestConnInitTypeMap(t *testing.T) {
	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)

	nameTypeIDs := map[string]uint32{
		"int":     fbproto.SQL_TYPE_LONG,
		"varchar": fbproto.SQL_TYPE_VARYING,
	}
	for name, typeid := range nameTypeIDs {
		dtByName, ok := conn.TypeMap().TypeForName(name)
		if !ok {
			t.Fatalf("Expected type named %v to be present", name)
		}
		dtByTypeID, ok := conn.TypeMap().TypeForTypeID(typeid, 0)
		if !ok {
			t.Fatalf("Expected type TypeID %v to be present", typeid)
		}
		if dtByName != dtByTypeID {
			t.Fatalf("Expected type named %v to be the same as type TypeID %v", name, typeid)
		}
	}

	ensureConnValid(t, conn)
}

func TestInsertDurationBigint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Exec(context.Background(), "recreate table t(duration_ms bigint not null)")
		require.NoError(t, err)

		const duration = time.Minute
		durationMs := duration.Milliseconds()

		result, err := conn.Exec(context.Background(), "insert into t(duration_ms) values(?)", durationMs)
		require.NoError(t, err)

		n := result.RowsAffected()
		require.EqualValues(t, 1, n)

		var storedMs int64
		err = conn.QueryRow(ctx, "select duration_ms from t").Scan(&storedMs)
		require.NoError(t, err)
		require.EqualValues(t, durationMs, storedMs)
	})
}

func TestConnDeallocateInvalidatedCachedStatementsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		var n int32
		err := conn.QueryRow(ctx, "select 1 / cast(? as integer) from rdb$database", 1).Scan(&n)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)

		// Divide by zero causes an error. baseRows.Close() calls Invalidate on the statement cache whenever an error was
		// encountered by the query. Use this to purposely invalidate the query. If we had access to private fields of conn
		// we could call conn.statementCache.InvalidateAll() instead.
		err = conn.QueryRow(ctx, "select 1 / cast(? as integer) from rdb$database", 0).Scan(&n)
		require.Error(t, err)

		ctx2, cancel2 := context.WithCancel(ctx)
		cancel2()
		err = conn.QueryRow(ctx2, "select 1 / cast(? as integer) from rdb$database", 1).Scan(&n)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)

		err = conn.QueryRow(ctx, "select 1 / cast(? as integer) from rdb$database", 1).Scan(&n)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
	})
}

func TestStmtCacheInvalidationExec(t *testing.T) {
	ctx := context.Background()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)

	_, err := conn.Exec(ctx, `
		recreate table drop_cols (
			f1 integer not null,
			f2 integer not null
		)
	`)
	require.NoError(t, err)

	insertSQL := "insert into drop_cols (f1, f2) values (?, ?)"
	// Populate statement cache.
	_, err = conn.Exec(ctx, insertSQL, 1, 2)
	require.NoError(t, err)

	// Trigger an Exec error to invalidate the cached statement entry.
	type unsupported struct{}
	_, err = conn.Exec(ctx, insertSQL, unsupported{}, 2)
	require.Error(t, err)

	// The next Exec performs deallocation of invalidated cached statements.
	_, err = conn.Exec(ctx, "select 1 from rdb$database")
	require.NoError(t, err)

	// Replace table definition after deallocation so stale metadata is gone.
	_, err = conn.Exec(ctx, `
		recreate table drop_cols (
			f1 boolean not null,
			f2 integer not null
		)
	`)
	require.NoError(t, err)

	// Re-prepare and execute on the new schema.
	_, err = conn.Exec(ctx, insertSQL, true, 2)
	require.NoError(t, err)

	ensureConnValid(t, conn)
}

func TestErrNoRows(t *testing.T) {
	t.Parallel()

	// ensure we preserve old error message
	require.Equal(t, "no rows in result set", fbx.ErrNoRows.Error())

	require.ErrorIs(t, fbx.ErrNoRows, sql.ErrNoRows, "fbx.ErrNowRows must match fbx.ErrNoRows")
}

func TestConnDeallocateAllAfterSetSearch_path(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, t, conn) < 6 {
			t.Skip("schemas are supported only in Firebird/RedDatabase 6+")
		}

		_, err := conn.Exec(ctx, "drop table if exists test1.test")
		_, err = conn.Exec(ctx, "drop table if exists test2.test")
		_, err = conn.Exec(ctx, "drop schema if exists test1")
		_, err = conn.Exec(ctx, "drop schema if exists test2")

		_, err = conn.Exec(ctx, "create schema test1")
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "create schema test2")
		require.NoError(t, err)

		_, err = conn.Exec(ctx, "create table test1.test(val integer)")
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "create table test2.test(val integer)")
		require.NoError(t, err)

		test1Input := 1
		test2Input := 2

		_, err = conn.Exec(ctx, "insert into test1.test values(?)", test1Input)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "insert into test2.test values(?)", test2Input)
		require.NoError(t, err)

		_, err = conn.Exec(ctx, "set search_path to test1")
		require.NoError(t, err)

		var output1, output2 int
		statement := "select val from test"

		err = conn.QueryRow(ctx, statement).Scan(&output1)
		require.NoError(t, err)
		assert.Equal(t, test1Input, output1)

		_, err = conn.Exec(ctx, "set search_path to test2")
		require.NoError(t, err)

		err = conn.QueryRow(ctx, statement).Scan(&output2)
		require.NoError(t, err)
		assert.Equal(t, test2Input, output2)
	})
}

func TestConnectWithTimezone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	timezoneList := []string{
		"GMT",
		"Africa/Banjul",
		"Canada/Atlantic",
		"Etc/GMT+12",
		"+03:00",
		"-04:00",
	}

	for i, tzExpected := range timezoneList {
		config.RuntimeParams["timezone"] = tzExpected
		conn, err := fbx.ConnectConfig(ctx, config)
		require.NoError(t, err, i)
		require.NotNil(t, conn, i)
		defer conn.Close(ctx)
		if getServerMajorVersionForConn(ctx, t, conn) < 4 {
			t.Skip("timezone is supported only in Firebird 4+/RedDatabase 5+")
		}

		var tzActual string
		err = conn.QueryRow(ctx, "SELECT RDB$GET_CONTEXT('SYSTEM', 'SESSION_TIMEZONE') FROM RDB$DATABASE").Scan(&tzActual)
		require.NoError(t, err, i)
		require.Equal(t, tzExpected, tzActual, i)
	}

}
