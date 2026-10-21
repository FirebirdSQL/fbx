package fbx

import (
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
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

func TestGetTPB(t *testing.T) {
	t.Parallel()

	t.Run("no options means nil tpb", func(t *testing.T) {
		tpb, err := TxOptions{}.GetTPB()
		require.NoError(t, err)
		require.Nil(t, tpb)
	})

	t.Run("lock timeout only produces tpb", func(t *testing.T) {
		tpb, err := TxOptions{LockTimeout: 7 * time.Second}.GetTPB()
		require.NoError(t, err)
		require.NotNil(t, tpb)
		require.NotEmpty(t, tpb)
	})

	t.Run("access mode only preserves default isolation", func(t *testing.T) {
		tpb, err := TxOptions{AccessMode: ReadOnly}.GetTPB()
		require.NoError(t, err)
		require.Equal(t, []byte{
			fbproto.Isc_tpb_version3,
			fbproto.Isc_tpb_read,
			fbproto.Isc_tpb_wait,
		}, tpb)
	})

	t.Run("isolation only defaults to read write", func(t *testing.T) {
		tpb, err := TxOptions{IsoLevel: RepeatableRead}.GetTPB()
		require.NoError(t, err)
		require.Equal(t, []byte{
			fbproto.Isc_tpb_version3,
			fbproto.Isc_tpb_write,
			fbproto.Isc_tpb_concurrency,
			fbproto.Isc_tpb_wait,
		}, tpb)
	})

	t.Run("unknown access mode fails", func(t *testing.T) {
		_, err := TxOptions{AccessMode: TxAccessMode("bad-mode")}.GetTPB()
		require.ErrorContains(t, err, "unknown transaction access mode")
	})

	t.Run("unknown iso level fails", func(t *testing.T) {
		_, err := TxOptions{IsoLevel: TxIsoLevel("bad-level")}.GetTPB()
		require.ErrorContains(t, err, "unknown transaction iso level")
	})
}
