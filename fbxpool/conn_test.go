package fbxpool_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbxpool"
	"github.com/stretchr/testify/require"
)

func TestConnExec(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer c.Release()

	testExec(t, ctx, c)
}

func TestConnQuery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer c.Release()

	testQuery(t, ctx, c)
}

func TestConnQueryRow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer c.Release()

	testQueryRow(t, ctx, c)
}
