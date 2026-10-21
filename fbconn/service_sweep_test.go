package fbconn

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestServiceConnSweepParameters(t *testing.T) {
	for name, tt := range map[string]struct {
		version string
		workers int
		timeout int
		want    []byte
	}{
		"defaults need no version query": {},
		"legacy RDB":                     {"LI-V3.0.8.0 RedDatabase", 2, 17, []byte{100, 2, 0, 0, 0}},
		"version 4 uses legacy tag":      {"LI-V4.0.0.0", 2, 0, []byte{100, 2, 0, 0, 0}},
		"RDB 5":                          {"LI-V5.0.0.0 RedDatabase", 2, 17, []byte{52, 2, 0, 0, 0}},
		"Firebird 5":                     {"WI-V5.0.3.1683 Firebird 5.0", 2, 0, []byte{52, 2, 0, 0, 0}},
		"RDB 6 both parameters":          {"LI-V6.0.0.0 RedDatabase", 2, 17, []byte{52, 2, 0, 0, 0, 101, 17, 0, 0, 0}},
		"timeout only":                   {"LI-V6.0.0.0 RedDatabase", 0, 258, []byte{101, 2, 1, 0, 0}},
		"old server omits timeout":       {"LI-V5.3.0.0 RedDatabase", 0, 258, nil},
		"maximum timeout":                {"LI-V6.0.0.0 RedDatabase", 0, 2147483, []byte{101, 155, 196, 32, 0}},
	} {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			done := make(chan error, 1)
			wantSPB := append([]byte{3, 106, 2, 0, 'd', 'b', 108, 2, 0, 0, 0}, tt.want...)
			var versionReply []byte
			if tt.version != "" {
				versionReply = serviceVersionReply(tt.version)
			}
			go func() {
				defer func() { _ = server.Close() }()
				done <- runSweepProbeServer(server, versionReply, wantSPB)
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			require.NoError(t, conn.Repair(ctx, "db", RepairOptions{
				Operation: RepairSweep, ParallelWorkers: tt.workers, SweepTimeout: tt.timeout,
			}))
			require.False(t, conn.actionActive)
			require.NoError(t, conn.Close(ctx))
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}

func TestServiceConnSweepRejectsBadVersionBeforeStarting(t *testing.T) {
	for name, reply := range map[string][]byte{
		"empty": {}, "truncated item": {55}, "truncated length": {55, 1},
		"truncated value": {55, 2, 0, 'a'}, "unknown item": {255},
		"server truncation": {2}, "invalid version": serviceVersionReply("unknown"),
		"missing end": {55, 0, 0}, "invalid end": {55, 0, 0, 2},
	} {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			done := make(chan error, 1)
			go func() {
				defer func() { _ = server.Close() }()
				done <- runSweepProbeServer(server, reply, nil)
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			require.ErrorContains(t, conn.Repair(ctx, "db", RepairOptions{
				Operation: RepairSweep, ParallelWorkers: 1, SweepTimeout: 1,
			}), "parse service server version")
			require.False(t, conn.actionActive)
			require.NoError(t, conn.Close(ctx))
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}

func TestServiceConnSweepValidationBeforeVersionQuery(t *testing.T) {
	var conn *ServiceConn
	for _, options := range []RepairOptions{
		{Operation: RepairSweep, ParallelWorkers: -1},
		{Operation: RepairSweep, ParallelWorkers: 1, SweepTimeout: -1},
		{Operation: RepairSweep, ParallelWorkers: 1, SweepTimeout: 2147484},
	} {
		err := conn.Repair(context.Background(), "db", options)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "connection is nil")
	}
	options := RepairOptions{Operation: RepairSweep, ParallelWorkers: 1}
	require.ErrorContains(t, conn.Repair(context.Background(), "db", options), "service connection is nil")
	conn = &ServiceConn{conn: &FbConn{}, actionActive: true}
	require.ErrorIs(t, conn.Repair(context.Background(), "db", options), errServiceActionActive)
}

func serviceVersionReply(version string) []byte {
	reply := []byte{55, byte(len(version)), byte(len(version) >> 8)}
	reply = append(reply, version...)
	return append(reply, 1)
}

// A nil versionReply expects an action without querying the version. A nil
// wantSPB expects a clean detach after a failed version query, without an action.
func runSweepProbeServer(conn net.Conn, versionReply, wantSPB []byte) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r, w := fbproto.NewXDRReader(conn), fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return err
	}
	if err := writeAuthenticatedProbeAccept(w, encoding); err != nil {
		return err
	}
	if err := readProbeServiceAttach(r, encoding); err != nil {
		return err
	}
	if err := writeProbeResponse(w, 42); err != nil {
		return err
	}
	if versionReply != nil {
		sendItems, receiveItems, _, err := readProbeServiceInfo(r, 42)
		if err != nil {
			return err
		}
		if len(sendItems) != 0 || !bytes.Equal(receiveItems, []byte{55}) {
			return fmt.Errorf("unexpected server version query: send %v, receive %v", sendItems, receiveItems)
		}
		if err := writeProbeResponseData(w, 0, versionReply); err != nil {
			return err
		}
	}
	if wantSPB != nil {
		spb, err := readProbeServiceStart(r, 42)
		if err != nil {
			return err
		}
		if !bytes.Equal(spb, wantSPB) {
			return fmt.Errorf("unexpected sweep SPB: got %v, want %v", spb, wantSPB)
		}
		if err := writeProbeResponse(w, 0); err != nil {
			return err
		}
		_, receiveItems, _, err := readProbeServiceInfo(r, 42)
		if err != nil {
			return err
		}
		if !bytes.Equal(receiveItems, []byte{63}) {
			return fmt.Errorf("unexpected sweep output query: %v", receiveItems)
		}
		if err := writeProbeResponseData(w, 0, []byte{63, 0, 0, 1}); err != nil {
			return err
		}
	}
	_, _, err = readProbeServiceDetach(r)
	return err
}
