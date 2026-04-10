package fbx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeTxLockTimeout(t *testing.T) {
	t.Parallel()

	t.Run("zero means default", func(t *testing.T) {
		seconds, err := normalizeTxLockTimeout(0)
		require.NoError(t, err)
		require.Equal(t, 0, seconds)
	})

	t.Run("whole seconds", func(t *testing.T) {
		seconds, err := normalizeTxLockTimeout(7 * time.Second)
		require.NoError(t, err)
		require.Equal(t, 7, seconds)
	})

	t.Run("negative fails", func(t *testing.T) {
		_, err := normalizeTxLockTimeout(-time.Second)
		require.Error(t, err)
	})

	t.Run("fractional seconds are truncated", func(t *testing.T) {
		seconds, err := normalizeTxLockTimeout(1500 * time.Millisecond)
		require.NoError(t, err)
		require.Equal(t, 1, seconds)
	})

	t.Run("range overflow fails", func(t *testing.T) {
		_, err := normalizeTxLockTimeout(32768 * time.Second)
		require.Error(t, err)
	})
}
