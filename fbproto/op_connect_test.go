package fbproto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/firebirdsql/fbx/fbproto/auth"
	"github.com/stretchr/testify/require"
)

func TestOpConnectAdvertisesDisabledWireCrypt(t *testing.T) {
	_, uid := writeTestOpConnect(t, false, 0)

	requireWireCryptInt(t, uid, 0)
}

func TestOpConnectAdvertisesWireCryptWhenEnabled(t *testing.T) {
	_, uid := writeTestOpConnect(t, true, 0)

	requireWireCryptInt(t, uid, 1)
}

func TestOpConnectDefaultsToAttachOperation(t *testing.T) {
	operation, _ := writeTestOpConnect(t, true, 0)

	require.Equal(t, Op_attach, operation)
}

func TestOpConnectCanAdvertiseCreateOperation(t *testing.T) {
	operation, _ := writeTestOpConnect(t, true, Op_create)

	require.Equal(t, Op_create, operation)
}

func TestOpConnectAdvertisesWireCompression(t *testing.T) {
	for _, compress := range []bool{false, true} {
		t.Run(fmt.Sprintf("compress=%t", compress), func(t *testing.T) {
			encoding, err := NewEncoding("UTF8")
			require.NoError(t, err)

			var buf bytes.Buffer
			err = (&OpConnect{
				Database:        "employee",
				User:            "sysdba",
				AuthPluginName:  "Srp",
				AuthPluginsList: []string{"Srp"},
				Compress:        compress,
				Encoding:        encoding,
			}).Write(NewXDRWriter(&buf))
			require.NoError(t, err)

			r := NewXDRReader(&buf)
			for range 4 {
				_, err = r.ReadInt32()
				require.NoError(t, err)
			}
			_, err = r.ReadBuffer()
			require.NoError(t, err)
			protocolCount, err := r.ReadInt32()
			require.NoError(t, err)
			_, err = r.ReadBuffer()
			require.NoError(t, err)

			for range protocolCount {
				for range 3 {
					_, err = r.ReadInt32()
					require.NoError(t, err)
				}
				maxType, err := r.ReadInt32()
				require.NoError(t, err)
				require.Equal(t, compress, maxType&Pflag_compress != 0)
				_, err = r.ReadInt32()
				require.NoError(t, err)
			}
		})
	}
}

func writeTestOpConnect(t *testing.T, wireCrypt bool, operation int) (int, []byte) {
	t.Helper()

	encoding, err := NewEncoding("UTF8")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = (&OpConnect{
		Operation:       operation,
		Database:        "employee",
		User:            "sysdba",
		SystemUser:      "tester",
		SystemHost:      "localhost",
		AuthPluginName:  "Srp",
		AuthPluginsList: []string{"Srp"},
		AuthSpecificData: []auth.SpecificDataChunk{
			{Part: 0, Data: []byte("client-public-key")},
		},
		WireCrypt: wireCrypt,
		Encoding:  encoding,
	}).Write(NewXDRWriter(&buf))
	require.NoError(t, err)

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	requireInt32(t, r, Op_connect)
	gotOperation, err := r.ReadInt32()
	require.NoError(t, err)
	requireInt32(t, r, ConnectVersion3)
	requireInt32(t, r, GenericArchitectureClient)
	_, err = r.ReadBuffer()
	require.NoError(t, err)
	_, err = r.ReadInt32()
	require.NoError(t, err)
	uid, err := r.ReadBuffer()
	require.NoError(t, err)
	return gotOperation, uid
}

func requireInt32(t *testing.T, r *XDRReader, expected int) {
	t.Helper()

	got, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func requireWireCryptInt(t *testing.T, uid []byte, expected int32) {
	t.Helper()

	pos := bytes.IndexByte(uid, CNCT_client_crypt)
	require.NotEqual(t, -1, pos)
	require.LessOrEqual(t, pos+6, len(uid))
	require.Equal(t, byte(4), uid[pos+1])
	require.Equal(t, expected, int32(binary.LittleEndian.Uint32(uid[pos+2:pos+6])))
}
