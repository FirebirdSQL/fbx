package fbproto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadSliceConvertsFailedResponseToErrorResponse(t *testing.T) {
	t.Parallel()

	var input bytes.Buffer
	w := NewXDRWriter(&input)
	require.NoError(t, w.WriteInt32(Op_response))
	require.NoError(t, w.WriteInt32(7))
	require.NoError(t, w.WriteInt64(11))
	require.NoError(t, w.WriteBuffer(nil))
	require.NoError(t, w.WriteInt32(Isc_arg_gds))
	require.NoError(t, w.WriteInt32(Isc_att_shutdown))
	require.NoError(t, w.WriteInt32(Isc_arg_end))

	encoding, err := NewEncoding("UTF8")
	require.NoError(t, err)
	client := NewClient(&input, io.Discard, encoding)

	msg, err := client.ReadSlice(0, 0)
	require.NoError(t, err)

	response, ok := msg.(*ErrorResponse)
	require.True(t, ok)
	require.Equal(t, 7, response.Object)
	require.EqualValues(t, 11, response.BlobId)
	require.Equal(t, []int{Isc_att_shutdown}, response.GdsCodes)
}
