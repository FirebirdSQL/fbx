package fbconn

import (
	"encoding/binary"
	"io"
	"math"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// Use the native service field order: host, ID, state, remote site, database path;
// participants may omit fields, and recovery advice follows all participants.
func limboDetailsProbeData() ([]byte, []LimboTransaction) {
	data := []byte{19, 255, 255, 255, 255, 20, 42, 0, 0, 0}
	text := func(tag byte, value string) {
		data = append(data, tag)
		data = binary.LittleEndian.AppendUint16(data, uint16(len(value)))
		data = append(data, value...)
	}
	text(26, "coordinator")
	data = append(data, 18, 7, 0, 0, 0, 21, 22)
	text(27, "")
	text(28, "db 1.fdb")
	data = append(data, 46)
	data = binary.LittleEndian.AppendUint64(data, math.MaxInt64)
	data = append(data, 21, 23)
	text(27, "узел")
	text(28, "/data/база.fdb")
	text(26, "coordinator")
	data = append(data, 18, 7, 0, 0, 0, 21, 24)
	text(28, "/data/rollback.fdb")
	text(26, "unknown")
	data = append(data, 18, 9, 0, 0, 0, 21, 25)
	text(28, "unknown.fdb")
	text(26, "omitted")
	data = append(data, 18, 10, 0, 0, 0)
	text(28, "omitted.fdb")
	text(26, "missing-id")
	data = append(data, 21, 25)
	text(27, "missing")
	text(28, "missing-id.fdb")
	data = append(data, 21, 22)
	text(28, "missing-fields.fdb")
	data = append(data, 29, 30, 47)
	data = binary.LittleEndian.AppendUint64(data, 1<<40)
	data = append(data, 48)
	data = binary.LittleEndian.AppendUint64(data, 2<<40)
	data = append(data, 46)
	data = binary.LittleEndian.AppendUint64(data, 1)
	data = append(data, 29, 31, 20, 99, 0, 0, 0, 18, 7, 0, 0, 0, 29, 33,
		20, 100, 0, 0, 0, 18, 11, 0, 0, 0)
	return data, []LimboTransaction{
		{ID: math.MaxUint32},
		{
			ID: 42, Distributed: true, Advice: LimboAdviceCommit,
			Participants: []LimboParticipant{
				{ID: 7, State: LimboStatePrepared, HostSite: "coordinator", DatabasePath: "db 1.fdb"},
				{ID: math.MaxInt64, State: LimboStateCommitted, RemoteSite: "узел", DatabasePath: "/data/база.fdb"},
				{ID: 7, State: LimboStateRolledBack, HostSite: "coordinator", DatabasePath: "/data/rollback.fdb"},
				{ID: 9, HostSite: "unknown", DatabasePath: "unknown.fdb"},
				{ID: 10, HostSite: "omitted", DatabasePath: "omitted.fdb"},
				{HostSite: "missing-id", RemoteSite: "missing", DatabasePath: "missing-id.fdb"},
				{State: LimboStatePrepared, DatabasePath: "missing-fields.fdb"},
			},
		},
		{ID: 1 << 40},
		{ID: 2 << 40, Distributed: true, Participants: []LimboParticipant{{ID: 1}}, Advice: LimboAdviceRollback},
		{ID: 99, Distributed: true, Participants: []LimboParticipant{{ID: 7}}},
		{ID: 100, Distributed: true, Participants: []LimboParticipant{{ID: 11}}},
	}
}

func TestServiceConnLimboTransactionDetails(t *testing.T) {
	data, details := limboDetailsProbeData()
	chunks := make([][]byte, len(data))
	for i := range data {
		chunks[i] = data[i : i+1]
	}
	for name, tt := range map[string]struct {
		chunks  [][]byte
		want    []LimboTransaction
		wantErr string
	}{
		"empty": {},
		"single maximum ID": {
			chunks: [][]byte{binary.LittleEndian.AppendUint64([]byte{47}, math.MaxInt64)},
			want:   []LimboTransaction{{ID: math.MaxInt64}},
		},
		"fragmented distributed": {chunks: chunks, want: details},
		"truncated ID after a complete transaction": {
			chunks: [][]byte{{19, 42, 0, 0, 0, 47, 1}}, wantErr: "truncated transaction information",
		},
		"truncated participant string": {
			chunks: [][]byte{{20, 42, 0, 0, 0, 26, 10, 0, 'a'}}, wantErr: "truncated transaction information",
		},
		"unknown tag": {chunks: [][]byte{{20, 42, 0, 0, 0, 255}}, wantErr: "unknown limbo information tag"},
		"unknown state": {
			chunks: [][]byte{{20, 42, 0, 0, 0, 18, 7, 0, 0, 0, 21, 255}}, wantErr: "unknown limbo participant state",
		},
		"unknown advice": {
			chunks: [][]byte{{20, 42, 0, 0, 0, 29, 255}}, wantErr: "unknown limbo recovery advice",
		},
		"participant without a transaction": {
			chunks: [][]byte{{18, 7, 0, 0, 0}}, wantErr: "without a distributed transaction",
		},
		"participant in a single transaction": {
			chunks: [][]byte{{19, 42, 0, 0, 0, 18, 7, 0, 0, 0}}, wantErr: "without a distributed transaction",
		},
		"participant after advice": {
			chunks: [][]byte{{20, 42, 0, 0, 0, 29, 30, 18, 7, 0, 0, 0}}, wantErr: "after recovery advice",
		},
		"top-level ID overflow": {
			chunks:  [][]byte{binary.LittleEndian.AppendUint64([]byte{48}, uint64(math.MaxInt64)+1)},
			wantErr: "ID exceeds int64",
		},
		"participant ID overflow": {
			chunks:  [][]byte{binary.LittleEndian.AppendUint64([]byte{20, 42, 0, 0, 0, 46}, uint64(math.MaxInt64)+1)},
			wantErr: "ID exceeds int64",
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			done := make(chan error, 1)
			go func() {
				defer func() { _ = server.Close() }()
				done <- runLimboProbeServer(server, tt.chunks)
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			got, err := conn.LimboTransactionDetails(ctx, "db")
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
				require.False(t, conn.actionActive)
				require.False(t, conn.conn.IsClosed())
			} else {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, got)
				require.True(t, conn.conn.IsClosed())
			}
			require.NoError(t, conn.Close(ctx))
			select {
			case err := <-done:
				if tt.wantErr == "" {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, io.EOF)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}
