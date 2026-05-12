package fbproto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpCreateWrite(t *testing.T) {
	orig := &OpCreate{
		File: "new.fdb",
		DPB:  []byte{Isc_dpb_version2, Isc_dpb_sql_dialect, 4, 0, 0, 0, 3},
	}

	var buf bytes.Buffer
	w := NewXDRWriter(&buf)
	require.NoError(t, orig.Write(w))

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	op, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, Op_create, op)

	unusedHandle, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, 0, unusedHandle)

	enc, err := NewEncoding("UTF8")
	require.NoError(t, err)
	file, err := r.ReadString(enc)
	require.NoError(t, err)
	require.Equal(t, orig.File, file)

	dpb, err := r.ReadBuffer()
	require.NoError(t, err)
	require.Equal(t, orig.DPB, dpb)
}
