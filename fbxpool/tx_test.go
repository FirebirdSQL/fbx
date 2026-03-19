package fbxpool_test

import (
	"context"
	"os"
	"testing"
	"time"

	"rdb.red-soft.ru/fbx/fbxpool"
	"github.com/stretchr/testify/require"
)

func TestTxExec(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	testExec(t, ctx, tx)
}

func TestTxQuery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	testQuery(t, ctx, tx)
}

func TestTxQueryRow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	testQueryRow(t, ctx, tx)
}
