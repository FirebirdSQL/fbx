package fbproto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpServiceAttachWrite(t *testing.T) {
	var buf bytes.Buffer
	enc, err := NewEncoding("UTF8")
	require.NoError(t, err)

	err = (&OpServiceAttach{
		Encoding:    enc,
		ServiceName: "service_mgr",
		SPB:         []byte{Isc_spb_version, Isc_spb_current_version},
	}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	requireInt32(t, r, Op_service_attach)
	requireInt32(t, r, 0)
	serviceName, err := r.ReadString(enc)
	require.NoError(t, err)
	require.Equal(t, "service_mgr", serviceName)
	spb, err := r.ReadBuffer()
	require.NoError(t, err)
	require.Equal(t, []byte{Isc_spb_version, Isc_spb_current_version}, spb)
}

func TestOpServiceDetachWrite(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpServiceDetach{
		ServiceHandle: 42,
	}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)
	require.Equal(t, []byte{
		0x00, 0x00, 0x00, byte(Op_service_detach),
		0x00, 0x00, 0x00, 0x2a,
	}, buf.Bytes())
}

func TestOpServiceStartWrite(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpServiceStart{
		ServiceHandle: 42,
		ActionSPB:     []byte{Isc_action_svc_backup, Isc_spb_verbose},
	}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	requireInt32(t, r, Op_service_start)
	requireInt32(t, r, 42)
	requireInt32(t, r, 0)
	actionSPB, err := r.ReadBuffer()
	require.NoError(t, err)
	require.Equal(t, []byte{Isc_action_svc_backup, Isc_spb_verbose}, actionSPB)
}

func TestOpServiceInfoWrite(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpServiceInfo{
		ServiceHandle: 42,
		SendItems:     []byte{Isc_info_svc_timeout, 1},
		ReceiveItems:  []byte{Isc_info_svc_to_eof},
		BufferLength:  4096,
	}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	requireInt32(t, r, Op_service_info)
	requireInt32(t, r, 42)
	requireInt32(t, r, 0)
	sendItems, err := r.ReadBuffer()
	require.NoError(t, err)
	require.Equal(t, []byte{Isc_info_svc_timeout, 1}, sendItems)
	receiveItems, err := r.ReadBuffer()
	require.NoError(t, err)
	require.Equal(t, []byte{Isc_info_svc_to_eof}, receiveItems)
	requireInt32(t, r, 4096)
}

func TestOpServiceInfoWriteDefaultsBufferLength(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpServiceInfo{
		ServiceHandle: 42,
		ReceiveItems:  []byte{Isc_info_svc_server_version},
	}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	requireInt32(t, r, Op_service_info)
	requireInt32(t, r, 42)
	requireInt32(t, r, 0)
	_, err = r.ReadBuffer()
	require.NoError(t, err)
	_, err = r.ReadBuffer()
	require.NoError(t, err)
	requireInt32(t, r, defaultServiceInfoBufferLength)
}

func TestOpServiceTrace(t *testing.T) {
	enc, err := NewEncoding("UTF8")
	require.NoError(t, err)

	var in, out, trace bytes.Buffer
	client := NewClient(&in, &out, enc)
	client.Trace(&trace, TracerOptions{SuppressTimestamps: true})

	require.NoError(t, client.WriteMessage(&OpServiceAttach{
		Encoding:    enc,
		ServiceName: "service_mgr",
	}))
	require.NoError(t, client.WriteMessage(&OpServiceStart{ServiceHandle: 42}))
	require.NoError(t, client.WriteMessage(&OpServiceInfo{ServiceHandle: 42}))
	require.NoError(t, client.WriteMessage(&OpServiceInfo{ServiceHandle: 42, BufferLength: 2048}))
	require.NoError(t, client.WriteMessage(&OpServiceDetach{ServiceHandle: 42}))

	require.Equal(t, ""+
		"C\tServiceAttach service=\"service_mgr\" spbLen=0\n"+
		"C\tServiceStart handle=42 actionLen=0\n"+
		"C\tServiceInfo handle=42 sendLen=0 recvLen=0 requestedBufSize=0 bufSize=1024\n"+
		"C\tServiceInfo handle=42 sendLen=0 recvLen=0 requestedBufSize=2048 bufSize=2048\n"+
		"C\tServiceDetach handle=42\n",
		trace.String())
}
