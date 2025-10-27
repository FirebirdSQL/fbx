package fbproto

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOpContAuth_WriteRead_Roundtrip(t *testing.T) {
	enc, _ := NewEncoding("UTF8")
	orig := &OpContAuth{
		Encoding:   enc,
		AuthData:   []byte{1, 2, 3, 4, 5},
		PluginName: "Gss",
		PluginList: "Gss,Srp",
		Keys:       []byte{9, 8, 7},
	}

	var buf bytes.Buffer
	w := NewXDRWriter(&buf)
	// Write full message (includes opcode)
	require.NoError(t, orig.Write(w))

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	// Simulate dispatcher reading opcode first
	op, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, Op_cont_auth, op)

	got := &OpContAuth{Encoding: enc}
	require.NoError(t, got.Read(r))

	require.Equal(t, orig.AuthData, got.AuthData)
	require.Equal(t, orig.PluginName, got.PluginName)
	require.Equal(t, orig.PluginList, got.PluginList)
	require.Equal(t, orig.Keys, got.Keys)
}
