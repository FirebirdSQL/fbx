package fbproto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpAcceptReadsCompressionFlag(t *testing.T) {
	var buf bytes.Buffer
	w := NewXDRWriter(&buf)
	require.NoError(t, w.WriteInt32(-32755)) // 0xffff800d: protocol 13 with the protocol flag
	require.NoError(t, w.WriteInt32(GenericArchitectureClient))
	require.NoError(t, w.WriteInt32(Ptype_lazy_send|Pflag_compress))

	var accept OpAccept
	require.NoError(t, accept.Read(NewXDRReader(&buf)))
	require.Equal(t, PROTOCOL_VERSION13, accept.Version)
	require.Equal(t, GenericArchitectureClient, accept.Architecture)
	require.Equal(t, Ptype_lazy_send, accept.Type)
	require.True(t, accept.Compress)
}
