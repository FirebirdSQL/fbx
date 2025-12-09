package fbx_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestCrateDBConnect(t *testing.T) {
	t.Parallel()

	connString := os.Getenv("PGX_TEST_CRATEDB_CONN_STRING")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_CRATEDB_CONN_STRING")
	}

	conn, err := fbx.Connect(context.Background(), connString)
	require.Nil(t, err)
	defer closeConn(t, conn)

	assert.Equal(t, connString, conn.Config().ConnString())

	var result int
	err = conn.QueryRow(context.Background(), "select 1 +1").Scan(&result)
	if err != nil {
		t.Fatalf("QueryRow Scan unexpectedly failed: %v", err)
	}
	if result != 2 {
		t.Errorf("bad result: %d", result)
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()

	connString := os.Getenv("PGX_TEST_DATABASE")
	config := mustParseConfig(t, connString)

	conn, err := fbx.ConnectConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Unable to establish connection: %v", err)
	}

	assertConfigsEqual(t, config, conn.Config(), "Conn.Config() returns original config")

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

func TestConnectWithPreferSimpleProtocol(t *testing.T) {
	t.Parallel()

	connConfig := mustParseConfig(t, os.Getenv("PGX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = fbx.QueryExecModeExec

	conn := mustConnect(t, connConfig)
	defer closeConn(t, conn)

	// If simple protocol is used we should be able to correctly scan the result
	// into a pgtype.Text as the integer will have been encoded in text.

	var s fbtype.Text
	err := conn.QueryRow(context.Background(), "select cast(? as int) from rdb$database", 42).Scan(&s)
	require.NoError(t, err)
	require.Equal(t, fbtype.Text{String: "42", Valid: true}, s)

	ensureConnValid(t, conn)
}

func TestConnectConfigRequiresConnConfigFromParseConfig(t *testing.T) {
	config := &fbx.ConnConfig{}
	require.PanicsWithValue(t, "config must be created by ParseConfig", func() {
		fbx.ConnectConfig(context.Background(), config)
	})
}

func TestConfigContainsConnStr(t *testing.T) {
	connStr := os.Getenv("PGX_TEST_DATABASE")
	config, err := fbx.ParseConfig(connStr)
	require.NoError(t, err)
	assert.Equal(t, connStr, config.ConnString())
}

func TestConfigCopyReturnsEqualConfig(t *testing.T) {
	connString := "postgres://jack:secret@localhost:5432/mydb?application_name=pgxtest&search_path=myschema&connect_timeout=5"
	original, err := fbx.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assertConfigsEqual(t, original, copied, t.Name())
}

func TestConfigCopyCanBeUsedToConnect(t *testing.T) {
	connString := os.Getenv("PGX_TEST_DATABASE")
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

	config, err = fbx.ParseConfig("description_cache_capacity=0")
	require.NoError(t, err)
	require.EqualValues(t, 0, config.DescriptionCacheCapacity)

	config, err = fbx.ParseConfig("description_cache_capacity=42")
	require.NoError(t, err)
	require.EqualValues(t, 42, config.DescriptionCacheCapacity)

	//	default_query_exec_mode
	//		Possible values: "cache_statement", "cache_describe", "describe_exec", "exec", and "simple_protocol". See

	config, err = fbx.ParseConfig("default_query_exec_mode=cache_statement")
	require.NoError(t, err)
	require.Equal(t, fbx.QueryExecModeCacheStatement, config.DefaultQueryExecMode)

	config, err = fbx.ParseConfig("default_query_exec_mode=exec")
	require.NoError(t, err)
	require.Equal(t, fbx.QueryExecModeExec, config.DefaultQueryExecMode)
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
		table := strings.ReplaceAll(t.Name(), "/", "_") + "_conn_test"
		results := mustExec(t, conn, "recreate table "+table+" (id integer primary key)")
		assert.Equal(t, "DDL", results.String())

		// Accept parameters
		results = mustExec(t, conn, "insert into "+table+"(id) values(?)", 1)
		assert.Equal(t, "INSERT", results.String())

		err := conn.DeallocateAll(ctx)
		assert.NoError(t, err)

		results = mustExec(t, conn, "drop table "+table)
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
		table := strings.ReplaceAll(t.Name(), "/", "_")
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		commandTag, err := conn.Exec(ctx, "recreate table "+table+"(id integer primary key);")
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

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	closeConn(t, conn)

	_, err := conn.Exec(context.Background(), "select 1 from rdb$database")
	require.Error(t, err)
	assert.True(t, fbconn.SafeToRetry(err))
}

func TestPrepare(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	//select $1::varchar
	_, err := conn.Prepare(context.Background(), "test", "select cast(? as varchar) from rdb$database")
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

func TestPrepareBadSQLFailure(t *testing.T) {
	t.Parallel()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
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

		_, err := conn.Prepare(context.Background(), "test", "select cast('fail' as varchar) from rdb$database")
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
		_, err := conn.Prepare(context.Background(), "test", "select cast(? as varchar) from rdb$database")
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
		sql := "select cast(? as varchar) from rdb$database"
		sd, err := conn.Prepare(ctx, sql, sql)
		require.NoError(t, err)
		require.Equal(t, "stmt_da3e336d81d8925213324f4b1730faf9486903ab1095eb76", sd.Name)

		var s string
		err = conn.QueryRow(ctx, sql, "hello").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "hello", s)

		err = conn.Deallocate(ctx, sql)
		require.NoError(t, err)
	})
}

// https://github.com/jackc/pgx/pull/1795
func TestDeallocateInAbortedTransaction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		sql := "select cast(? as varchar) from rdb$database"
		sd, err := tx.Prepare(ctx, sql, sql)
		require.NoError(t, err)
		require.Equal(t, "stmt_da3e336d81d8925213324f4b1730faf9486903ab1095eb76", sd.Name)

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
		require.Equal(t, "stmt_da3e336d81d8925213324f4b1730faf9486903ab1095eb76", sd.Name)
	})
}

func TestDeallocateMissingPreparedStatementStillClearsFromPreparedStatementMap(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Prepare(ctx, "ps", "select cast(? as varchar) from rdb$database")
		require.NoError(t, err)

		err = conn.Deallocate(ctx, "ps")
		require.NoError(t, err)

		_, err = conn.Prepare(ctx, "ps", "select  cast(? as varchar), cast(? as varchar) from rdb$database")
		require.NoError(t, err)

		var s1, s2 string
		err = conn.QueryRow(ctx, "ps", "hello", "world").Scan(&s1, &s2)
		require.NoError(t, err)
		require.Equal(t, "hello", s1)
		require.Equal(t, "world", s2)
	})
}

func TestFatalRxError(t *testing.T) {
	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	_, err := conn.Exec(context.Background(), "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;")
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var n int32
		var s string
		err := conn.QueryRow(context.Background(), "select 1, fbsleep(10) from rdb$database").Scan(&n, &s)
		if fbErr, ok := err.(*fbconn.FbError); ok { //&& fbErr.Severity == "FATAL" {
			fmt.Println(fbErr)
		} else {
			t.Errorf("Expected QueryRow Scan to return fatal FbError, but instead received %v", err)
			return
		}
	}()

	if err := conn.FbConn().Disconnect(context.Background()); err != nil {
		t.Fatalf("Unable to disconnect: %v", err)
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
			conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
			defer closeConn(t, conn)

			otherConn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
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
	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	// spot check that the standard postgres type names aren't qualified
	nameOIDs := map[string]uint32{
		"_int8": fbtype.Int8ArrayOID,
		"int8":  fbtype.Int8OID,
		"json":  fbtype.JSONOID,
		"text":  fbtype.TextOID,
	}
	for name, oid := range nameOIDs {
		dtByName, ok := conn.TypeMap().TypeForName(name)
		if !ok {
			t.Fatalf("Expected type named %v to be present", name)
		}
		dtByOID, ok := conn.TypeMap().TypeForOID(oid)
		if !ok {
			t.Fatalf("Expected type OID %v to be present", oid)
		}
		if dtByName != dtByOID {
			t.Fatalf("Expected type named %v to be the same as type OID %v", name, oid)
		}
	}

	ensureConnValid(t, conn)
}

func TestUnregisteredTypeUsableAsStringArgumentAndBaseResult(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		var n uint64
		err := conn.QueryRow(context.Background(), "select $1::uint64", "42").Scan(&n)
		if err != nil {
			t.Fatal(err)
		}

		if n != 42 {
			t.Fatalf("Expected n to be 42, but was %v", n)
		}
	})
}

func TestDomainType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		// Domain type uint64 is a PostgreSQL domain of underlying type numeric.

		// In the extended protocol preparing "select $1::uint64" appears to create a statement that expects a param OID of
		// uint64 but a result OID of the underlying numeric.

		var s string
		err := conn.QueryRow(ctx, "select $1::uint64", "24").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "24", s)

		// Register type
		uint64Type, err := conn.LoadType(ctx, "uint64")
		require.NoError(t, err)
		conn.TypeMap().RegisterType(uint64Type)

		var n uint64
		err = conn.QueryRow(ctx, "select $1::uint64", uint64(24)).Scan(&n)
		require.NoError(t, err)

		// String is still an acceptable argument after registration
		err = conn.QueryRow(ctx, "select $1::uint64", "7").Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n != 7 {
			t.Fatalf("Expected n to be 7, but was %v", n)
		}
	})
}

func TestLoadTypeSameNameInDifferentSchemas(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, `create schema pgx_a;
create type pgx_a.point as (a text, b text);
create schema pgx_b;
create type pgx_b.point as (c text);
`)
		require.NoError(t, err)

		// Register types
		for _, typename := range []string{"pgx_a.point", "pgx_b.point"} {
			// Obviously using conn while a tx is in use and registering a type after the connection has been established are
			// really bad practices, but for the sake of convenience we do it in the test here.
			dt, err := conn.LoadType(ctx, typename)
			require.NoError(t, err)
			conn.TypeMap().RegisterType(dt)
		}

		type aPoint struct {
			A string
			B string
		}

		type bPoint struct {
			C string
		}

		var a aPoint
		var b bPoint
		err = tx.QueryRow(ctx, `select '(foo,bar)'::pgx_a.point, '(baz)'::pgx_b.point`).Scan(&a, &b)
		require.NoError(t, err)
		require.Equal(t, aPoint{"foo", "bar"}, a)
		require.Equal(t, bPoint{"baz"}, b)
	})
}

func TestLoadCompositeType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, "create type compositetype as (attr1 int, attr2 int)")
		require.NoError(t, err)

		_, err = tx.Exec(ctx, "alter type compositetype drop attribute attr1")
		require.NoError(t, err)

		_, err = conn.LoadType(ctx, "compositetype")
		require.NoError(t, err)
	})
}

func TestLoadRangeType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, "create type examplefloatrange as range (subtype=float8, subtype_diff=float8mi)")
		require.NoError(t, err)

		// Register types
		newRangeType, err := conn.LoadType(ctx, "examplefloatrange")
		require.NoError(t, err)
		conn.TypeMap().RegisterType(newRangeType)
		conn.TypeMap().RegisterDefaultFbType(fbtype.Range[float64]{}, "examplefloatrange")

		var inputRangeType = fbtype.Range[float64]{
			Lower:     1.0,
			Upper:     2.0,
			LowerType: fbtype.Inclusive,
			UpperType: fbtype.Inclusive,
			Valid:     true,
		}
		var outputRangeType fbtype.Range[float64]
		err = tx.QueryRow(ctx, "SELECT $1::examplefloatrange", inputRangeType).Scan(&outputRangeType)
		require.NoError(t, err)
		require.Equal(t, inputRangeType, outputRangeType)
	})
}

func TestLoadMultiRangeType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		fbxtest.SkipPostgreSQLVersionLessThan(t, conn, 14) // multirange data type was added in 14 postgresql

		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, "create type examplefloatrange as range (subtype=float8, subtype_diff=float8mi, multirange_type_name=examplefloatmultirange)")
		require.NoError(t, err)

		// Register types
		newRangeType, err := conn.LoadType(ctx, "examplefloatrange")
		require.NoError(t, err)
		conn.TypeMap().RegisterType(newRangeType)
		conn.TypeMap().RegisterDefaultFbType(fbtype.Range[float64]{}, "examplefloatrange")

		newMultiRangeType, err := conn.LoadType(ctx, "examplefloatmultirange")
		require.NoError(t, err)
		conn.TypeMap().RegisterType(newMultiRangeType)
		conn.TypeMap().RegisterDefaultFbType(fbtype.Multirange[fbtype.Range[float64]]{}, "examplefloatmultirange")

		var inputMultiRangeType = fbtype.Multirange[fbtype.Range[float64]]{
			{
				Lower:     1.0,
				Upper:     2.0,
				LowerType: fbtype.Inclusive,
				UpperType: fbtype.Inclusive,
				Valid:     true,
			},
			{
				Lower:     3.0,
				Upper:     4.0,
				LowerType: fbtype.Exclusive,
				UpperType: fbtype.Exclusive,
				Valid:     true,
			},
		}
		var outputMultiRangeType fbtype.Multirange[fbtype.Range[float64]]
		err = tx.QueryRow(ctx, "SELECT $1::examplefloatmultirange", inputMultiRangeType).Scan(&outputMultiRangeType)
		require.NoError(t, err)
		require.Equal(t, inputMultiRangeType, outputMultiRangeType)
	})
}

func TestStmtCacheInvalidationConn(t *testing.T) {
	ctx := context.Background()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	// create a table and fill it with some data
	_, err := conn.Exec(ctx, `
        RECREATE TABLE drop_cols (
            id int generated by default as identity PRIMARY KEY NOT NULL,
            f1 int NOT NULL,
            f2 int NOT NULL
        );
    `)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "INSERT INTO drop_cols (f1, f2) VALUES (1, 2)")
	require.NoError(t, err)

	getSQL := "SELECT * FROM drop_cols WHERE id = ?"

	// This query will populate the statement cache. We don't care about the result.
	rows, err := conn.Query(ctx, getSQL, 1)
	require.NoError(t, err)
	rows.Close()
	require.NoError(t, rows.Err())

	// Now, change the schema of the table out from under the statement, making it invalid.
	_, err = conn.Exec(ctx, "ALTER TABLE drop_cols DROP f1")
	require.NoError(t, err)

	// We must get an error the first time we try to re-execute a bad statement.
	// It is up to the application to determine if it wants to try again. We punt to
	// the application because there is no clear recovery path in the case of failed transactions
	// or batch operations and because automatic retry is tricky and we don't want to get
	// it wrong at such an importaint layer of the stack.
	rows, err = conn.Query(ctx, getSQL, 1)
	require.NoError(t, err)
	rows.Next()
	nextErr := rows.Err()
	rows.Close()
	for _, err := range []error{nextErr, rows.Err()} {
		if err == nil {
			t.Fatal(`expected "cached plan must not change result type": no error`)
		}
		if !strings.Contains(err.Error(), "cached plan must not change result type") {
			t.Fatalf(`expected "cached plan must not change result type", got: "%s"`, err.Error())
		}
	}

	// On retry, the statement should have been flushed from the cache.
	rows, err = conn.Query(ctx, getSQL, 1)
	require.NoError(t, err)
	rows.Next()
	err = rows.Err()
	require.NoError(t, err)
	rows.Close()
	require.NoError(t, rows.Err())

	ensureConnValid(t, conn)
}

func TestStmtCacheInvalidationTx(t *testing.T) {
	ctx := context.Background()

	conn := mustConnectString(t, os.Getenv("PGX_TEST_DATABASE"))
	defer closeConn(t, conn)

	if conn.FbConn().ParameterStatus("crdb_version") != "" {
		t.Skip("Server has non-standard prepare in errored transaction behavior (https://github.com/cockroachdb/cockroach/issues/84140)")
	}

	// create a table and fill it with some data
	_, err := conn.Exec(ctx, `
        RECREATE TABLE drop_cols (
            id int generated by default as identity PRIMARY KEY NOT NULL,
            f1 int NOT NULL,
            f2 int NOT NULL
        );
    `)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "INSERT INTO drop_cols (f1, f2) VALUES (1, 2)")
	require.NoError(t, err)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	getSQL := "SELECT * FROM drop_cols WHERE id = ?"

	// This query will populate the statement cache. We don't care about the result.
	rows, err := tx.Query(ctx, getSQL, 1)
	require.NoError(t, err)
	rows.Close()
	require.NoError(t, rows.Err())

	// Now, change the schema of the table out from under the statement, making it invalid.
	_, err = tx.Exec(ctx, "ALTER TABLE drop_cols DROP f1")
	require.NoError(t, err)

	// We must get an error the first time we try to re-execute a bad statement.
	// It is up to the application to determine if it wants to try again. We punt to
	// the application because there is no clear recovery path in the case of failed transactions
	// or batch operations and because automatic retry is tricky and we don't want to get
	// it wrong at such an importaint layer of the stack.
	rows, err = tx.Query(ctx, getSQL, 1)
	require.NoError(t, err)
	rows.Next()
	nextErr := rows.Err()
	rows.Close()
	for _, err := range []error{nextErr, rows.Err()} {
		if err == nil {
			t.Fatal(`expected "cached plan must not change result type": no error`)
		}
		if !strings.Contains(err.Error(), "cached plan must not change result type") {
			t.Fatalf(`expected "cached plan must not change result type", got: "%s"`, err.Error())
		}
	}

	rows, _ = tx.Query(ctx, getSQL, 1)
	rows.Close()
	err = rows.Err()
	// Retries within the same transaction are errors (really anything except a rollback
	// will be an error in this transaction).
	require.Error(t, err)
	rows.Close()

	err = tx.Rollback(ctx)
	require.NoError(t, err)

	// once we've rolled back, retries will work
	rows, err = conn.Query(ctx, getSQL, 1)
	require.NoError(t, err)
	rows.Next()
	err = rows.Err()
	require.NoError(t, err)
	rows.Close()

	ensureConnValid(t, conn)
}

func TestInsertDurationInterval(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Exec(context.Background(), "recreate table t(duration INTERVAL(0) NOT NULL)")
		require.NoError(t, err)

		result, err := conn.Exec(context.Background(), "insert into t(duration) values(?)", time.Minute)
		require.NoError(t, err)

		n := result.RowsAffected()
		require.EqualValues(t, 1, n)
	})
}

func TestRawValuesUnderlyingMemoryReused(t *testing.T) {
	//FIXME: check for possibility permanent memory allocation for rows
	t.Skip("too slow")
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		var buf []byte

		rows, err := conn.Query(ctx, `select cast(1 as int) from rdb$database`)
		require.NoError(t, err)

		for rows.Next() {
			buf = rows.RawValues()[0]
		}

		require.NoError(t, rows.Err())

		original := make([]byte, len(buf))
		copy(original, buf)

		for i := 0; i < 1_000_000; i++ {
			rows, err := conn.Query(ctx, `select cast(? as int) from rdb$database`, i)
			require.NoError(t, err)
			rows.Close()
			require.NoError(t, rows.Err())

			if !bytes.Equal(original, buf) {
				return
			}
		}

		t.Fatal("expected buffer from RawValues to be overwritten by subsequent queries but it was not")
	})
}

// https://github.com/jackc/pgx/issues/1847
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

func TestErrNoRows(t *testing.T) {
	t.Parallel()

	// ensure we preserve old error message
	require.Equal(t, "no rows in result set", fbx.ErrNoRows.Error())

	require.ErrorIs(t, fbx.ErrNoRows, sql.ErrNoRows, "fbx.ErrNowRows must match fbx.ErrNoRows")
}
