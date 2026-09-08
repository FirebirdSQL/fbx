package fbxpool_test

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
	"rdb.red-soft.ru/fbx/fbxpool"
)

func setupSchemas(ctx context.Context, t testing.TB, schemas ...string) {
	t.Helper()

	drop := func(ctx context.Context, c *fbx.Conn) {
		for _, s := range schemas {
			_ = c.DeallocateAll(ctx)
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

func TestParseConfigKeepsSearchPathInConnConfig(t *testing.T) {
	config, err := fbxpool.ParseConfig("host=localhost pool_max_conns=3 search_path=tenant_1,public")
	require.NoError(t, err)
	assert.Equal(t, "tenant_1,public", config.ConnConfig.RuntimeParams["search_path"])
	assert.EqualValues(t, 3, config.MaxConns)

	copied := config.Copy()
	assert.Equal(t, "tenant_1,public", copied.ConnConfig.RuntimeParams["search_path"])
}

func TestPoolSearchPathAppliesToEveryConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()
	if conn.ServerVersion().Major < 6 {
		t.Skip("schemas / search_path are supported only in Firebird/RedDatabase 6+")
	}

	const schema = "FBX_POOL_SP_ALL"
	setupSchemas(ctx, t, schema)

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.MaxConns = 3
	config.MinConns = 3

	pool, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	conns := make([]*fbxpool.Conn, 0, config.MaxConns)
	defer func() {
		for _, c := range conns {
			c.Release()
		}
	}()
	for i := int32(0); i < config.MaxConns; i++ {
		c, err := pool.Acquire(ctx)
		conns = append(conns, c)
		require.NoError(t, err)
	}

	attachments := map[string]bool{}
	for _, c := range conns {
		var attachment string
		require.NoError(t, c.QueryRow(ctx, "select cast(current_connection as varchar(20)) from rdb$database").Scan(&attachment))
		attachments[strings.TrimSpace(attachment)] = true

		var sp string
		require.NoError(t, c.QueryRow(ctx, "select rdb$get_context('SYSTEM', 'SEARCH_PATH') from rdb$database").Scan(&sp))
		assert.Equal(t, fmt.Sprintf(`"%s", "SYSTEM"`, schema), sp)

		var val int
		require.NoError(t, c.QueryRow(ctx, "select val from test").Scan(&val))
		assert.Equal(t, 1, val)
	}
	assert.Len(t, attachments, int(config.MaxConns))
}
