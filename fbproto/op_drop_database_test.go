package fbproto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpDropDatabaseWrite(t *testing.T) {
	var buf bytes.Buffer
	w := NewXDRWriter(&buf)
	require.NoError(t, (&OpDropDatabase{}).Write(w))

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	op, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, Op_drop_database, op)

	unusedHandle, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, 0, unusedHandle)
}

func TestOpDropDatabaseTrace(t *testing.T) {
	enc, err := NewEncoding("UTF8")
	require.NoError(t, err)

	var in, out, trace bytes.Buffer
	client := NewClient(&in, &out, enc)
	client.Trace(&trace, TracerOptions{SuppressTimestamps: true})

	require.NoError(t, client.WriteMessage(&OpDropDatabase{}))
	require.Equal(t, "C\tDropDatabase\n", trace.String())
}
