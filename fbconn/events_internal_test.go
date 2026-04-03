package fbconn

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/vax"
)

type testEventHandler struct {
	calls int
}

func (h *testEventHandler) EventOccurred(*EventHandle) {
	h.calls++
}

func TestEventHandleToEventBlockAndCount(t *testing.T) {
	t.Parallel()

	enc, err := fbproto.NewEncoding("UTF8")
	require.NoError(t, err)

	handle, err := newEventHandle("TEST_EVENT", &testEventHandler{}, enc)
	require.NoError(t, err)

	handle.setInternalCount(1234)
	block := handle.toEventBlock()
	require.GreaterOrEqual(t, len(block), 7)
	require.Equal(t, byte(fbproto.EPB_version1), block[0])
	require.Equal(t, byte(len([]byte("TEST_EVENT"))), block[1])
	require.Equal(t, "TEST_EVENT", string(block[2:2+len("TEST_EVENT")]))
	require.Equal(t, 1234, vax.Integer(block, len(block)-4, 4))

	handle.calculateCount()
	require.Equal(t, 1234, handle.EventCount())

	handle.setInternalCount(1240)
	handle.calculateCount()
	require.Equal(t, 6, handle.EventCount())
}

func TestReadSingleEventFromXDR(t *testing.T) {
	t.Parallel()

	eventItems := []byte{fbproto.EPB_version1, 4, 'T', 'E', 'S', 'T', 11, 0, 0, 0}
	var packet bytes.Buffer
	xw := fbproto.NewXDRWriter(&packet)
	require.NoError(t, xw.WriteInt32(0))
	require.NoError(t, xw.WriteBuffer(eventItems))
	require.NoError(t, xw.WriteInt64(0))
	require.NoError(t, xw.WriteInt32(77))

	xr := fbproto.NewXDRReader(bytes.NewReader(packet.Bytes()))
	var scratch []byte
	eventID, eventCount, err := readSingleEventFromXDR(xr, &scratch)
	require.NoError(t, err)
	require.Equal(t, 77, eventID)
	require.Equal(t, 11, eventCount)
}

func TestReadSingleEventFromXDRInvalidEPBVersion(t *testing.T) {
	t.Parallel()

	eventItems := []byte{2, 4, 'T', 'E', 'S', 'T', 11, 0, 0, 0}
	var packet bytes.Buffer
	xw := fbproto.NewXDRWriter(&packet)
	require.NoError(t, xw.WriteInt32(0))
	require.NoError(t, xw.WriteBuffer(eventItems))
	require.NoError(t, xw.WriteInt64(0))
	require.NoError(t, xw.WriteInt32(77))

	xr := fbproto.NewXDRReader(bytes.NewReader(packet.Bytes()))
	var scratch []byte
	_, _, err := readSingleEventFromXDR(xr, &scratch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid event buffer version")
}

func TestReadSingleEventFromXDRShortPacket(t *testing.T) {
	t.Parallel()

	var packet bytes.Buffer
	xw := fbproto.NewXDRWriter(&packet)
	require.NoError(t, xw.WriteInt32(0))

	xr := fbproto.NewXDRReader(bytes.NewReader(packet.Bytes()))
	var scratch []byte
	_, _, err := readSingleEventFromXDR(xr, &scratch)
	require.Error(t, err)
}

func TestReadSingleEventFromXDRIncompleteEventBuffer(t *testing.T) {
	t.Parallel()

	eventItems := []byte{fbproto.EPB_version1, 4, 'T', 'E', 'S', 'T', 11, 0, 0, 0}
	var packet bytes.Buffer
	xw := fbproto.NewXDRWriter(&packet)
	require.NoError(t, xw.WriteInt32(0))
	require.NoError(t, xw.WriteBuffer(eventItems))
	require.NoError(t, xw.WriteInt64(0))
	require.NoError(t, xw.WriteInt32(77))

	raw := packet.Bytes()
	truncated := raw[:9] // db_handle + event buffer length + partial event buffer
	xr := fbproto.NewXDRReader(bytes.NewReader(truncated))
	var scratch []byte
	_, _, err := readSingleEventFromXDR(xr, &scratch)
	require.Error(t, err)
}
