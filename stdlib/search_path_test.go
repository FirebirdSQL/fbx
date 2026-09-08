package stdlib_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
)

func skipUnlessSchemasSupported(t testing.TB) {
	t.Helper()
	db := openDB(t)
	defer closeDB(t, db)
	if getServerMajorVersion(t, db) < 6 {
		t.Skip("schemas / search_path are supported only in Firebird/RedDatabase 6+")
	}
}

func setupSchemas(t testing.TB, schemas ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	drop := func(ctx context.Context, c *fbx.Conn) {
		for _, s := range schemas {
			_, _ = c.Exec(ctx, fmt.Sprintf("drop table if exists %s.test", s))
			_, _ = c.Exec(ctx, fmt.Sprintf("drop schema if exists %s", s))
		}
	}

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()

	drop(ctx, conn)
	for i, s := range schemas {
		_, err := conn.Exec(ctx, "create schema "+s)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, fmt.Sprintf("create table %s.test(val integer)", s))
		require.NoError(t, err)
		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s.test values (%d)", s, i+1))
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c, err := fbx.Connect(cleanupCtx, os.Getenv("FBX_TEST_DATABASE"))
		if err != nil {
			return
		}
		drop(cleanupCtx, c)
		_ = c.Close(cleanupCtx)
	})
}

func TestSearchPathViaSQLOpenConnString(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	skipUnlessSchemasSupported(t)

	const schema = "FBX_STDLIB_SP_DSN"
	setupSchemas(t, schema)

	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.RuntimeParams["search_path"] = schema

	dsn := config.ConnString()
	switch {
	case strings.HasPrefix(dsn, "firebird://") || strings.HasPrefix(dsn, "firebirdsql://"):
		if strings.Contains(dsn, "?") {
			dsn += "&search_path=" + schema
		} else {
			dsn += "?search_path=" + schema
		}
	default:
		dsn = strings.TrimSpace(dsn + " search_path=" + schema)
	}
	db, err := sql.Open("fbx", dsn)
	require.NoError(t, err)
	defer closeDB(t, db)

	var sp string
	require.NoError(t, db.QueryRowContext(ctx, "select rdb$get_context('SYSTEM', 'SEARCH_PATH') from rdb$database").Scan(&sp))
	assert.Equal(t, fmt.Sprintf(`"%s", "SYSTEM"`, schema), sp)

	var val int
	require.NoError(t, db.QueryRowContext(ctx, "select val from test").Scan(&val))
	assert.Equal(t, 1, val)
}

func TestPreparedStmtAfterSetSearchPath(t *testing.T) {
	skipUnlessSchemasSupported(t)

	const schema1 = "FBX_STDLIB_SP_STMT1"
	const schema2 = "FBX_STDLIB_SP_STMT2"
	setupSchemas(t, schema1, schema2)

	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer closeResource(t, conn)

		_, err = conn.ExecContext(ctx, "set search_path to "+schema1)
		require.NoError(t, err)

		stmt, err := conn.PrepareContext(ctx, "select val from test where val > ?")
		require.NoError(t, err)
		defer closeStmt(t, stmt)

		var val int
		require.NoError(t, stmt.QueryRowContext(ctx, 0).Scan(&val))
		assert.Equal(t, 1, val)

		_, err = conn.ExecContext(ctx, "set search_path to "+schema2)
		require.NoError(t, err)

		require.NoError(t, stmt.QueryRowContext(ctx, 0).Scan(&val))
		assert.Equal(t, 2, val)

		res, err := stmt.ExecContext(ctx, 0)
		require.NoError(t, err)
		_, err = res.RowsAffected()
		require.NoError(t, err)
	})
}

func TestSchemaQualifiedIdentifiersThroughStdlib(t *testing.T) {
	skipUnlessSchemasSupported(t)

	const schema1 = "FBX_STDLIB_SP_Q1"
	const schema2 = "FBX_STDLIB_SP_Q2"
	setupSchemas(t, schema1, schema2)

	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var v1, v2 int
		err := db.QueryRowContext(ctx, fmt.Sprintf("select a.val, b.val from %s.test a cross join %s.test b", schema1, schema2)).Scan(&v1, &v2)
		require.NoError(t, err)
		assert.Equal(t, 1, v1)
		assert.Equal(t, 2, v2)

		_, err = db.ExecContext(ctx, fmt.Sprintf("update %s.test set val = val + 10", schema2))
		require.NoError(t, err)

		require.NoError(t, db.QueryRowContext(ctx, fmt.Sprintf("select val from %s.test", schema2)).Scan(&v2))
		assert.Equal(t, 12, v2)

		_, err = db.ExecContext(ctx, fmt.Sprintf("update %s.test set val = val - 10", schema2))
		require.NoError(t, err)
	})
}
