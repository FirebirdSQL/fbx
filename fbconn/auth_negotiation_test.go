package fbconn

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestShouldRespondToCondAcceptStopsForKnownPluginsOnly(t *testing.T) {
	msg := &fbproto.OpCondAccept{
		PluginName: "Srp",
		ServerKeys: []byte{2, 3, 'S', 'r', 'p'},
	}

	require.False(t, shouldRespondToCondAccept(msg))
}

func TestShouldRespondToCondAcceptStopsForEmptyChallengeWithoutServerKeys(t *testing.T) {
	msg := &fbproto.OpCondAccept{PluginName: "Srp"}

	require.False(t, shouldRespondToCondAccept(msg))
}

func TestShouldRespondToCondAcceptContinuesForChallengeWithoutServerKeys(t *testing.T) {
	msg := &fbproto.OpCondAccept{
		PluginName: "Srp",
		Data:       []byte("server-challenge"),
	}

	require.True(t, shouldRespondToCondAccept(msg))
}

func TestShouldRespondToDataAcceptStopsForChallengeWithoutServerKeys(t *testing.T) {
	msg := &fbproto.OpDataAccept{
		OpCondAccept: fbproto.OpCondAccept{
			PluginName: "Srp",
			Data:       []byte("server-challenge"),
		},
	}

	require.False(t, shouldRespondToDataAccept(msg))
}

func TestShouldRespondToDataAcceptStopsWhenAuthComplete(t *testing.T) {
	msg := &fbproto.OpDataAccept{
		OpCondAccept: fbproto.OpCondAccept{AuthComplete: true},
	}

	require.False(t, shouldRespondToDataAccept(msg))
}

func TestShouldRespondToDataAcceptContinuesForWireCryptKeys(t *testing.T) {
	msg := &fbproto.OpDataAccept{
		OpCondAccept: fbproto.OpCondAccept{
			PluginName: "Srp",
			Data:       []byte("server-challenge"),
			ServerKeys: []byte{
				0, 9, 'S', 'y', 'm', 'm', 'e', 't', 'r', 'i', 'c',
				1, 5, 'A', 'r', 'c', '4', '8',
			},
		},
	}

	require.True(t, shouldRespondToDataAccept(msg))
}

func TestShouldRespondToCondAcceptContinuesForWireCryptKeys(t *testing.T) {
	msg := &fbproto.OpCondAccept{
		PluginName: "Srp",
		Data:       []byte("server-challenge"),
		ServerKeys: []byte{
			0, 9, 'S', 'y', 'm', 'm', 'e', 't', 'r', 'i', 'c',
			1, 5, 'A', 'r', 'c', '4', '8',
		},
	}

	require.True(t, shouldRespondToCondAccept(msg))
}

func TestDataAcceptChallengeWithoutKeysSendsAttachWithSpecificAuthData(t *testing.T) {
	result := runAuthNegotiationProbe(t, fbproto.Op_accept_data)

	require.Equal(t, fbproto.Op_attach, result.opcode)
	require.True(t, result.hasSpecificAuthData)
}

func TestCondAcceptChallengeWithoutKeysSendsContAuth(t *testing.T) {
	result := runAuthNegotiationProbe(t, fbproto.Op_cond_accept)

	require.Equal(t, fbproto.Op_cont_auth, result.opcode)
	require.False(t, result.hasSpecificAuthData)
}

type authNegotiationProbeResult struct {
	opcode              int
	hasSpecificAuthData bool
	err                 error
}

func runAuthNegotiationProbe(t *testing.T, acceptOpcode int) authNegotiationProbeResult {
	t.Helper()

	clientConn, serverConn := net.Pipe()
	resultCh := make(chan authNegotiationProbeResult, 1)
	go func() {
		defer serverConn.Close()
		resultCh <- runAuthNegotiationProbeServer(serverConn, acceptOpcode)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := ConnectConfig(ctx, config)
	if err == nil {
		require.NoError(t, conn.Close(context.Background()))
	}

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		return result
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		return authNegotiationProbeResult{}
	}
}

func runAuthNegotiationProbeServer(conn net.Conn, acceptOpcode int) authNegotiationProbeResult {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return authNegotiationProbeResult{err: err}
	}

	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)

	if err := readProbeConnect(r); err != nil {
		return authNegotiationProbeResult{err: err}
	}
	if err := writeProbeAccept(w, encoding, acceptOpcode); err != nil {
		return authNegotiationProbeResult{err: err}
	}

	opcode, err := r.ReadInt32()
	if err != nil {
		return authNegotiationProbeResult{err: err}
	}
	result := authNegotiationProbeResult{opcode: opcode}
	if opcode == fbproto.Op_attach {
		dpb, err := readProbeAttachDPB(r, encoding)
		if err != nil {
			return authNegotiationProbeResult{err: err}
		}
		result.hasSpecificAuthData = probeDPBHasTag(dpb, fbproto.Isc_dpb_specific_auth_data)
	}
	return result
}

func readProbeConnect(r *fbproto.XDRReader) error {
	opcode, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if opcode != fbproto.Op_connect {
		return fmt.Errorf("expected op_connect, got %d", opcode)
	}

	for range 3 {
		if _, err := r.ReadInt32(); err != nil {
			return err
		}
	}
	if _, err := r.ReadBuffer(); err != nil {
		return err
	}
	protocolCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if _, err := r.ReadBuffer(); err != nil {
		return err
	}
	for range protocolCount * 5 {
		if _, err := r.ReadInt32(); err != nil {
			return err
		}
	}
	return nil
}

func writeProbeAccept(w *fbproto.XDRWriter, encoding *fbproto.Encoding, opcode int) error {
	if err := w.WriteInt32(opcode); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.PROTOCOL_VERSION19); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.GenericAchitectureClient); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.Ptype_lazy_send); err != nil {
		return err
	}
	if err := w.WriteBuffer(probeSrpServerData()); err != nil {
		return err
	}
	if err := w.WriteString("Srp256", encoding); err != nil {
		return err
	}
	if err := w.WriteInt32(0); err != nil {
		return err
	}
	return w.WriteBuffer(nil)
}

func readProbeAttachDPB(r *fbproto.XDRReader, encoding *fbproto.Encoding) ([]byte, error) {
	if _, err := r.ReadInt32(); err != nil {
		return nil, err
	}
	if _, err := r.ReadString(encoding); err != nil {
		return nil, err
	}
	return r.ReadBuffer()
}

func probeSrpServerData() []byte {
	salt := []byte("C5145A3209566ECAF4C936E71F5BD44A18075EB88BAD7120B3C589939B8FAB77")
	serverPublic := []byte("2")

	data := make([]byte, 0, 4+len(salt)+len(serverPublic))
	data = binary.LittleEndian.AppendUint16(data, uint16(len(salt)))
	data = append(data, salt...)
	data = binary.LittleEndian.AppendUint16(data, uint16(len(serverPublic)))
	data = append(data, serverPublic...)
	return data
}

func probeDPBHasTag(dpb []byte, tag byte) bool {
	if len(dpb) == 0 || dpb[0] != fbproto.Isc_dpb_version2 {
		return false
	}

	for offset := 1; offset < len(dpb); {
		currentTag := dpb[offset]
		offset++
		if currentTag == tag {
			return true
		}
		if offset+4 > len(dpb) {
			return false
		}
		length := int(binary.LittleEndian.Uint32(dpb[offset : offset+4]))
		offset += 4
		if offset+length > len(dpb) {
			return false
		}
		offset += length
	}
	return false
}
