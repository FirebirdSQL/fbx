package fbx_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func skipUnlessSchemasSupported(ctx context.Context, t testing.TB, conn *fbx.Conn) {
	t.Helper()
	if getServerMajorVersionForConn(ctx, t, conn) < 6 {
		t.Skip("schemas / search_path are supported only in Firebird/RedDatabase 6+")
	}
}

func quotedSearchPath(names ...string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = `"` + strings.ReplaceAll(n, `"`, `""`) + `"`
	}
	return strings.Join(quoted, ", ")
}

func currentSearchPath(ctx context.Context, t testing.TB, conn *fbx.Conn) string {
	t.Helper()
	var sp string
	err := conn.QueryRow(ctx, "select rdb$get_context('SYSTEM', 'SEARCH_PATH') from rdb$database").Scan(&sp)
	require.NoError(t, err)
	return sp
}

func testTableValue(ctx context.Context, t testing.TB, conn *fbx.Conn) int {
	t.Helper()
	var val int
	err := conn.QueryRow(ctx, "select val from test").Scan(&val)
	require.NoError(t, err)
	return val
}

func mustExecCtx(ctx context.Context, t testing.TB, conn *fbx.Conn, sql string, args ...any) {
	t.Helper()
	_, err := conn.Exec(ctx, sql, args...)
	require.NoErrorf(t, err, "sql: %s", sql)
}

func connectWithSearchPath(ctx context.Context, t testing.TB, searchPath string) *fbx.Conn {
	t.Helper()
	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.RuntimeParams["search_path"] = searchPath
	conn, err := fbx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	return conn
}

func dropSchema(ctx context.Context, conn *fbx.Conn, schema string) {

	rows, err := conn.Query(ctx, "select rdb$relation_name from rdb$relations where rdb$schema_name = ? and rdb$view_blr is null and coalesce(rdb$system_flag, 0) = 0", schema)
	if err != nil {
		return
	}
	var names []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			names = append(names, strings.TrimRight(name, " "))
		}
	}
	rows.Close()
	for _, name := range names {
		_, _ = conn.Exec(ctx, fmt.Sprintf("drop table %s", fbx.Identifier{schema, name}.Sanitize()))
	}

	_, _ = conn.Exec(ctx, "drop schema if exists "+fbx.Identifier{schema}.Sanitize())
}

func setupSearchPathSchemas(ctx context.Context, t testing.TB, conn *fbx.Conn, schemas ...string) {
	t.Helper()

	for _, schema := range schemas {
		dropSchema(ctx, conn, schema)
	}

	for i, schema := range schemas {
		quoted := fbx.Identifier{schema}.Sanitize()
		mustExecCtx(ctx, t, conn, "create schema "+quoted)
		mustExecCtx(ctx, t, conn, fmt.Sprintf("create table %s.test(val integer)", quoted))
		mustExecCtx(ctx, t, conn, fmt.Sprintf("insert into %s.test values(?)", quoted), i+1)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cleanupConn, err := fbx.Connect(cleanupCtx, os.Getenv("FBX_TEST_DATABASE"))
		if err != nil {
			t.Logf("search_path cleanup: connect failed: %v", err)
			return
		}
		defer func() { _ = cleanupConn.Close(cleanupCtx) }()

		for _, schema := range schemas {
			dropSchema(cleanupCtx, cleanupConn, schema)
		}
	})
}

func TestParseConfigWithoutSearchPathHasNoRuntimeParam(t *testing.T) {
	t.Setenv("FBSEARCHPATH", "")
	config, err := fbx.ParseConfig("host=localhost application_name=fbxtest")
	require.NoError(t, err)
	_, ok := config.RuntimeParams["search_path"]
	assert.False(t, ok)
}

func TestParseConfigSearchPathFromEnv(t *testing.T) {
	t.Setenv("FBSEARCHPATH", "env_schema")

	t.Run("env only", func(t *testing.T) {
		config, err := fbx.ParseConfig("host=localhost")
		require.NoError(t, err)
		assert.Equal(t, "env_schema", config.RuntimeParams["search_path"])
	})

	t.Run("connection string overrides env", func(t *testing.T) {
		config, err := fbx.ParseConfig("host=localhost search_path=override")
		require.NoError(t, err)
		assert.Equal(t, "override", config.RuntimeParams["search_path"])
	})
}

func TestConnConfigCopyPreservesSearchPath(t *testing.T) {
	config, err := fbx.ParseConfig("host=localhost search_path=s1,s2")
	require.NoError(t, err)

	copied := config.Copy()
	assert.Equal(t, "s1,s2", copied.RuntimeParams["search_path"])
	assert.Equal(t, config.ConnString(), copied.ConnString())

	copied.RuntimeParams["search_path"] = "changed"
	assert.Equal(t, "s1,s2", config.RuntimeParams["search_path"])
	assert.Equal(t, "changed", copied.RuntimeParams["search_path"])
}

func TestConnectWithSearchPathOnServersWithoutSchemas(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn := connectWithSearchPath(ctx, t, "PUBLIC")
	defer closeConn(t, conn)

	if getServerMajorVersionForConn(ctx, t, conn) >= 6 {
		t.Skip("check servers without schema support")
	}

	var n int
	err := conn.QueryRow(ctx, "select 1 from rdb$database").Scan(&n)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	rows, err := conn.Query(ctx, "select rdb$relation_id from rdb$database")
	require.NoError(t, err)
	defer rows.Close()

	fields := rows.FieldDescriptions()
	require.Len(t, fields, 1)
	assert.Empty(t, fields[0].Schema)
	assert.Equal(t, "RDB$DATABASE", fields[0].OriginalTableName)
}

func TestSearchPathDefault(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conf, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	delete(conf.RuntimeParams, "search_path")

	conn, err := fbx.ConnectConfig(ctx, conf)
	require.NoError(t, err)
	defer closeConn(t, conn)
	skipUnlessSchemasSupported(ctx, t, conn)

	assert.Equal(t, quotedSearchPath("PUBLIC", "SYSTEM"), currentSearchPath(ctx, t, conn))
}

func TestConnectWithSearchPathDPB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	setupConn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, setupConn)
	skipUnlessSchemasSupported(ctx, t, setupConn)

	const schema1 = "FBX_SP_DPB1"
	const schema2 = "FBX_SP_DPB2"
	setupSearchPathSchemas(ctx, t, setupConn, schema1, schema2)

	conn := connectWithSearchPath(ctx, t, schema2)
	defer closeConn(t, conn)

	assert.Equal(t, 2, testTableValue(ctx, t, conn))
	assert.Equal(t, quotedSearchPath(schema2, "SYSTEM"), currentSearchPath(ctx, t, conn))
}

func TestConnectWithSearchPathDPBVariants(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	setupConn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, setupConn)
	skipUnlessSchemasSupported(ctx, t, setupConn)

	const (
		s1         = "FBX_SP_VAR1"
		s2         = "FBX_SP_VAR2"
		lower_case = "fbx_sp_var lower"
	)
	setupSearchPathSchemas(ctx, t, setupConn, s1, s2, lower_case)

	tests := []struct {
		name          string
		searchPath    string
		expectedPath  []string
		expectedValue int // 0 => "select val from test" must fail
	}{
		{
			name:          "single schema",
			searchPath:    s1,
			expectedPath:  []string{s1, "SYSTEM"},
			expectedValue: 1,
		},
		{
			name:          "list keeps order",
			searchPath:    s2 + ", " + s1,
			expectedPath:  []string{s2, s1, "SYSTEM"},
			expectedValue: 2,
		},
		{
			name:          "unquoted names are upper-cased",
			searchPath:    strings.ToLower(s1),
			expectedPath:  []string{s1, "SYSTEM"},
			expectedValue: 1,
		},
		{
			name:          "quoted identifier with lower case",
			searchPath:    `"` + lower_case + `"`,
			expectedPath:  []string{lower_case, "SYSTEM"},
			expectedValue: 3,
		},
		{
			name:          "nonexistent schema is ignored",
			searchPath:    "FBX_SP_DOES_NOT_EXIST, " + s2,
			expectedPath:  []string{"FBX_SP_DOES_NOT_EXIST", s2, "SYSTEM"},
			expectedValue: 2,
		},
		{
			name:          "only nonexistent schema falls back",
			searchPath:    "FBX_SP_DOES_NOT_EXIST",
			expectedPath:  []string{"FBX_SP_DOES_NOT_EXIST", "SYSTEM"},
			expectedValue: 0,
		},
		{
			name:          "empty search path leaves only SYSTEM",
			searchPath:    "",
			expectedPath:  []string{"SYSTEM"},
			expectedValue: 0,
		},
		{
			name:          "SYSTEM is not appended twice",
			searchPath:    "SYSTEM, " + s1,
			expectedPath:  []string{"SYSTEM", s1},
			expectedValue: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := connectWithSearchPath(ctx, t, tt.searchPath)
			defer closeConn(t, conn)

			assert.Equal(t, quotedSearchPath(tt.expectedPath...), currentSearchPath(ctx, t, conn))

			var val int
			err := conn.QueryRow(ctx, "select val from test").Scan(&val)
			if tt.expectedValue == 0 {
				require.Error(t, err)
				var fbErr *fbconn.FbError
				assert.ErrorAs(t, err, &fbErr)
				assert.Contains(t, err.Error(), "Table unknown")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedValue, val)
			}
			ensureConnValid(t, conn)
		})
	}
}

func TestSetSearchPathDeallocatesNamedPreparedStatement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, []fbx.QueryExecMode{fbx.QueryExecModeCacheStatement}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)

		const (
			schema1 = "FBX_SP_PREP1"
			schema2 = "FBX_SP_PREP2"
			name    = "prepared"
			sql     = "select val from test"
		)
		setupSearchPathSchemas(ctx, t, conn, schema1, schema2)

		mustExecCtx(ctx, t, conn, "set search_path to "+schema1)

		_, err := conn.Prepare(ctx, name, sql)
		require.NoError(t, err)

		var val int
		require.NoError(t, conn.QueryRow(ctx, name).Scan(&val))
		assert.Equal(t, 1, val)

		mustExecCtx(ctx, t, conn, "set search_path to "+schema2)

		require.Error(t, conn.QueryRow(ctx, name).Scan(&val))

		_, err = conn.Prepare(ctx, name, sql)
		require.NoError(t, err)

		require.NoError(t, conn.QueryRow(ctx, name).Scan(&val))
		assert.Equal(t, 2, val)
	})
}

func TestSetSearchPathStatementSyntaxVariants(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, []fbx.QueryExecMode{fbx.QueryExecModeCacheStatement}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)

		const schema1 = "FBX_SP_SYN1"
		const schema2 = "FBX_SP_SYN2"
		setupSearchPathSchemas(ctx, t, conn, schema1, schema2)

		variants := []struct {
			name string
			sql  string
		}{
			{"upper case", "SET SEARCH_PATH TO %s"},
			{"mixed case", "Set Search_Path To %s"},
			{"newlines and tabs", "set\n\tsearch_path\n\tto\n\t%s"},
			{"multiple spaces", "set    search_path     to     %s"},
			{"leading whitespace", "   set search_path to %s"},
			{"leading block comment", "/* switch tenant */ set search_path to %s"},
			{"trailing line comment", "set search_path to %s -- tenant"},
			{"schema list", "set search_path to %s, SYSTEM"},
			{"quoted schema", `set search_path to "%s"`},
		}

		for _, v := range variants {
			mustExecCtx(ctx, t, conn, "set search_path to "+schema1)
			require.Equal(t, 1, testTableValue(ctx, t, conn), v.name)

			mustExecCtx(ctx, t, conn, fmt.Sprintf(v.sql, schema2))
			assert.Equal(t, 2, testTableValue(ctx, t, conn), v.name)
		}
	})
}

func TestSetSearchPathListResolutionOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)

		const schema1 = "FBX_SP_ORD1"
		const schema2 = "FBX_SP_ORD2"
		const schema3 = "FBX_SP_ORD3"
		setupSearchPathSchemas(ctx, t, conn, schema1, schema2, schema3)
		mustExecCtx(ctx, t, conn, fmt.Sprintf("create table %s.only_here(val integer)", schema3))
		mustExecCtx(ctx, t, conn, fmt.Sprintf("insert into %s.only_here values (33)", schema3))
		defer func() {
			_ = conn.DeallocateAll(ctx)
			_, _ = conn.Exec(ctx, fmt.Sprintf("drop table %s.only_here", schema3))
		}()

		mustExecCtx(ctx, t, conn, fmt.Sprintf("set search_path to %s, %s, %s", schema1, schema2, schema3))
		assert.Equal(t, 1, testTableValue(ctx, t, conn))

		mustExecCtx(ctx, t, conn, fmt.Sprintf("set search_path to %s, %s, %s", schema2, schema1, schema3))
		assert.Equal(t, 2, testTableValue(ctx, t, conn))

		var only int
		require.NoError(t, conn.QueryRow(ctx, "select val from only_here").Scan(&only))
		assert.Equal(t, 33, only)

		ensureConnValid(t, conn)
	})
}

func TestSetSearchPathByQueryDeallocatesStatements(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)

		const schema1 = "FBX_SP_QRY1"
		const schema2 = "FBX_SP_QRY2"
		setupSearchPathSchemas(ctx, t, conn, schema1, schema2)

		mustExecCtx(ctx, t, conn, "set search_path to "+schema1)
		require.Equal(t, 1, testTableValue(ctx, t, conn))

		rows, err := conn.Query(ctx, "set search_path to "+schema2)
		require.NoError(t, err)
		rows.Close()

		assert.Equal(t, quotedSearchPath(schema2, "SYSTEM"), currentSearchPath(ctx, t, conn))
		assert.Equal(t, 2, testTableValue(ctx, t, conn))
	})
}

func TestAlterSessionResetRestoresSearchPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	setupConn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, setupConn)
	skipUnlessSchemasSupported(ctx, t, setupConn)

	const schema1 = "FBX_SP_RESET1"
	const schema2 = "FBX_SP_RESET2"
	setupSearchPathSchemas(ctx, t, setupConn, schema1, schema2)

	for _, mode := range fbxtest.AllQueryExecModes {
		t.Run(mode.String(), func(t *testing.T) {
			config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
			require.NoError(t, err)
			config.RuntimeParams["search_path"] = schema1
			config.DefaultQueryExecMode = mode

			conn, err := fbx.ConnectConfig(ctx, config)
			require.NoError(t, err)
			defer closeConn(t, conn)

			require.Equal(t, 1, testTableValue(ctx, t, conn))

			mustExecCtx(ctx, t, conn, "set search_path to "+schema2)
			require.Equal(t, 2, testTableValue(ctx, t, conn))

			mustExecCtx(ctx, t, conn, "alter session reset")
			assert.Equal(t, quotedSearchPath(schema1, "SYSTEM"), currentSearchPath(ctx, t, conn))
			assert.Equal(t, 1, testTableValue(ctx, t, conn))
		})
	}
}

func TestSchemaErrorMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)

		const schema = "FBX_SP_ERR"
		setupSearchPathSchemas(ctx, t, conn, schema)

		_, err := conn.Exec(ctx, fmt.Sprintf("create table %s.test(val integer)", schema))
		require.Error(t, err)
		var fbErr *fbconn.FbError
		require.ErrorAs(t, err, &fbErr)
		assert.Contains(t, err.Error(), fmt.Sprintf(`"%s"."TEST"`, schema))
		assert.Contains(t, err.Error(), "already exists")

		mustExecCtx(ctx, t, conn, "set search_path to "+schema)
		_, err = conn.Exec(ctx, "drop schema "+schema)
		require.Error(t, err, "non-empty schema cannot be dropped")

		ensureConnValid(t, conn)
	})
}

func TestFieldDescriptionsSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)

		const schema1 = "FBX_SP_META1"
		const schema2 = "FBX_SP_META2"
		setupSearchPathSchemas(ctx, t, conn, schema1, schema2)

		mustExecCtx(ctx, t, conn, "recreate table PUBLIC.fbx_sp_public_t(id integer)")
		defer func() {
			_ = conn.DeallocateAll(ctx)
			_, _ = conn.Exec(ctx, "drop table PUBLIC.fbx_sp_public_t")
		}()

		mustExecCtx(ctx, t, conn, "set search_path to "+schema2)

		rows, err := conn.Query(ctx, fmt.Sprintf(`
			select a.val as v1,
			       b.val as v2,
			       test.val as v_unqualified,
			       1 as literal,
			       d.rdb$relation_id as sys,
			       p.id as pub
			from %s.test a
			cross join %s.test b
			cross join test
			cross join rdb$database d
			left join PUBLIC.fbx_sp_public_t p on 1 = 0`, schema1, schema2))
		require.NoError(t, err)
		defer rows.Close()

		fields := rows.FieldDescriptions()
		require.Len(t, fields, 6)

		assert.Equal(t, "V1", fields[0].Name)
		assert.Equal(t, schema1, fields[0].Schema)
		assert.Equal(t, "TEST", fields[0].OriginalTableName)
		assert.Equal(t, "VAL", fields[0].OriginalName)

		assert.Equal(t, schema2, fields[1].Schema)
		assert.Equal(t, "TEST", fields[1].OriginalTableName)

		assert.Equal(t, schema2, fields[2].Schema)
		assert.Equal(t, "TEST", fields[2].OriginalTableName)

		assert.Empty(t, fields[3].Schema)
		assert.Empty(t, fields[3].OriginalTableName)

		assert.Equal(t, "SYSTEM", fields[4].Schema)
		assert.Equal(t, "RDB$DATABASE", fields[4].OriginalTableName)

		assert.Equal(t, "PUBLIC", fields[5].Schema)
		assert.Equal(t, "FBX_SP_PUBLIC_T", fields[5].OriginalTableName)

		require.NoError(t, rows.Err())
	})
}

func TestIdentifierSanitizeSchemaQualified(t *testing.T) {

	tests := []struct {
		ident    fbx.Identifier
		expected string
	}{
		{fbx.Identifier{"T"}, `"T"`},
		{fbx.Identifier{"S", "T"}, `"S"."T"`},
		{fbx.Identifier{"S", "T", "C"}, `"S"."T"."C"`},
		{fbx.Identifier{"lower", "Mixed Case"}, `"lower"."Mixed Case"`},
		{fbx.Identifier{`Weird "Schema"`, `t"1`}, `"Weird ""Schema"""."t""1"`},
		{fbx.Identifier{"SYSTEM", "RDB$DATABASE"}, `"SYSTEM"."RDB$DATABASE"`},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.ident.Sanitize(), "%v", tt.ident)
	}
}

func TestPreparedBatchSearchPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)
		skipIfBatchUnsupported(ctx, t, conn)

		const schema1 = "FBX_SP_BATCH1"
		const schema2 = "FBX_SP_BATCH2"
		setupSearchPathSchemas(ctx, t, conn, schema1, schema2)

		countIn := func(schema string) int {
			var n int
			require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s.test", schema)).Scan(&n))
			return n
		}

		mustExecCtx(ctx, t, conn, "set search_path to "+schema1)
		pb, err := conn.PrepareBatch(ctx, "insert into test(val) values (?)")
		require.NoError(t, err)
		require.NoError(t, pb.Add(10))
		require.NoError(t, pb.Add(11))
		res, err := pb.Exec(ctx)
		require.NoError(t, err)
		assert.EqualValues(t, 2, res.Affected)
		require.NoError(t, pb.Close(ctx))

		assert.Equal(t, 3, countIn(schema1))
		assert.Equal(t, 1, countIn(schema2))

		mustExecCtx(ctx, t, conn, "set search_path to "+schema2)
		pb, err = conn.PrepareBatch(ctx, "insert into test(val) values (?)")
		require.NoError(t, err)
		require.NoError(t, pb.Add(20))
		_, err = pb.Exec(ctx)
		require.NoError(t, err)
		require.NoError(t, pb.Close(ctx))

		assert.Equal(t, 3, countIn(schema1))
		assert.Equal(t, 2, countIn(schema2))

		_, err = conn.Prepare(ctx, "sp_batch", fmt.Sprintf("batch insert into %s.test(val) values (?)", schema1))
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "sp_batch", 30)
		require.NoError(t, err)
		res, err = conn.ExecBatch(ctx, "sp_batch")
		require.NoError(t, err)
		assert.EqualValues(t, 1, res.Affected)
		require.NoError(t, conn.Deallocate(ctx, "sp_batch"))
		assert.Equal(t, 4, countIn(schema1))

		ensureConnValid(t, conn)
	})
}

func TestBatchInsertSearchPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipUnlessSchemasSupported(ctx, t, conn)
		skipIfBatchUnsupported(ctx, t, conn)

		const schema1 = "FBX_SP_BINS1"
		const schema2 = "FBX_SP_BINS2"
		setupSearchPathSchemas(ctx, t, conn, schema1, schema2)

		mustExecCtx(ctx, t, conn, "set search_path to "+schema2)

		n, err := conn.BatchInsert(ctx, fbx.Identifier{"TEST"}, []string{"VAL"}, fbx.BatchInsertRows([][]any{{100}, {101}, {102}}))
		require.NoError(t, err)
		assert.EqualValues(t, 3, n)

		n, err = conn.BatchInsert(ctx, fbx.Identifier{schema1, "TEST"}, []string{"VAL"}, fbx.BatchInsertRows([][]any{{200}}))
		require.NoError(t, err)
		assert.EqualValues(t, 1, n)

		var c1, c2 int
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s.test", schema1)).Scan(&c1))
		assert.Equal(t, 2, c1)
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s.test", schema2)).Scan(&c2))
		assert.Equal(t, 4, c2)
	})
}
