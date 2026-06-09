package fbproto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpConnectRequestWrite(t *testing.T) {
	var buf bytes.Buffer
	w := NewXDRWriter(&buf)

	require.NoError(t, (&OpConnectRequest{}).Write(w))
	require.Equal(t, []byte{
		0, 0, 0, byte(Op_connect_request),
		0, 0, 0, byte(P_req_async),
		0, 0, 0, 0,
		0, 0, 0, 0,
	}, buf.Bytes())

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	opcode, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, Op_connect_request, opcode)

	reqType, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, P_req_async, reqType)

	reqObjAndPartner, err := r.ReadInt64()
	require.NoError(t, err)
	require.EqualValues(t, 0, reqObjAndPartner)

	_, err = r.ReadInt32()
	require.ErrorIs(t, err, io.EOF)
}

func TestOpQueEventsWrite(t *testing.T) {
	msg := &OpQueEvents{
		DatabaseHandle: 0,
		EventItems:     []byte{EPB_version1, 3, 'f', 'o', 'o', 1, 0, 0, 0},
		EventID:        42,
	}

	var buf bytes.Buffer
	w := NewXDRWriter(&buf)
	require.NoError(t, msg.Write(w))
	require.Equal(t, []byte{
		0, 0, 0, byte(Op_que_events),
		0, 0, 0, 0,
		0, 0, 0, 9,
		EPB_version1, 3, 'f', 'o', 'o', 1, 0, 0, 0,
		0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 42,
	}, buf.Bytes())

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	opcode, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, Op_que_events, opcode)

	dbHandle, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, msg.DatabaseHandle, dbHandle)

	items, err := r.ReadBuffer()
	require.NoError(t, err)
	require.Equal(t, msg.EventItems, items)

	astAndArg, err := r.ReadInt64()
	require.NoError(t, err)
	require.EqualValues(t, 0, astAndArg)

	eventID, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, msg.EventID, eventID)

	_, err = r.ReadInt32()
	require.ErrorIs(t, err, io.EOF)
}

func TestOpCancelEventsWrite(t *testing.T) {
	msg := &OpCancelEvents{
		DatabaseHandle: 0,
		EventID:        123,
	}

	var buf bytes.Buffer
	w := NewXDRWriter(&buf)
	require.NoError(t, msg.Write(w))
	require.Equal(t, []byte{
		0, 0, 0, byte(Op_cancel_events),
		0, 0, 0, 0,
		0, 0, 0, 123,
	}, buf.Bytes())

	r := NewXDRReader(bytes.NewReader(buf.Bytes()))
	opcode, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, Op_cancel_events, opcode)

	dbHandle, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, msg.DatabaseHandle, dbHandle)

	eventID, err := r.ReadInt32()
	require.NoError(t, err)
	require.Equal(t, msg.EventID, eventID)

	_, err = r.ReadInt32()
	require.ErrorIs(t, err, io.EOF)
}
