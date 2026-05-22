package fbproto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpBatchReleaseWrite(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpBatchRelease{
		Kind:            Op_batch_rls,
		StatementHandle: 42,
	}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)
	require.Equal(t, []byte{
		0x00, 0x00, 0x00, byte(Op_batch_rls),
		0x00, 0x00, 0x00, 0x2a,
	}, buf.Bytes())
}

func TestOpBatchReleaseWriteInvalidKind(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpBatchRelease{
		Kind:            12345,
		StatementHandle: 42,
	}).Write(NewXDRWriter(&buf))

	require.Error(t, err)
	require.ErrorContains(t, err, "wrong batch release operation")
}

func TestOpBatchSyncWrite(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpBatchSync{}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)
	require.Equal(t, []byte{
		0x00, 0x00, 0x00, byte(Op_batch_sync),
	}, buf.Bytes())
}
