package cleanupctx_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/internal/cleanupctx"
	"github.com/stretchr/testify/require"
)

func TestConfiguredCleanupDeadline(t *testing.T) {
	previous := cleanupctx.CleanupTimeout()
	t.Cleanup(func() { require.NoError(t, fbx.SetCleanupTimeout(previous)) })
	const timeout = 150 * time.Millisecond
	require.NoError(t, fbx.SetCleanupTimeout(timeout))

	synctest.Test(t, func(t *testing.T) {
		parent, cancelParent := context.WithCancel(context.Background())
		cancelParent()
		ctx, cancel := cleanupctx.CleanupContext(parent)
		defer cancel()

		require.NoError(t, ctx.Err())
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.Equal(t, timeout, time.Until(deadline))
	})
}

func TestInvalidCleanupTimeoutPreservesSetting(t *testing.T) {
	previous := cleanupctx.CleanupTimeout()
	t.Cleanup(func() { require.NoError(t, fbconn.SetCleanupTimeout(previous)) })
	const timeout = time.Second
	require.NoError(t, fbconn.SetCleanupTimeout(timeout))

	require.Error(t, fbconn.SetCleanupTimeout(149*time.Millisecond))
	require.Equal(t, timeout, cleanupctx.CleanupTimeout())
}
