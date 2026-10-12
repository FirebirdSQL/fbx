package fbx_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

// The coordinator prepares two participants, completes the first, then crashes.
// The service must recover the second using the first participant's decision.
func prepareMaintenanceDistributedTransaction(t *testing.T, ctx context.Context, config fbconn.Config, commit bool) int64 {
	t.Helper()
	participantConfig := config.Copy()
	participantConfig.Database = filepath.Join(filepath.Dir(config.Database), fmt.Sprintf("participant-%t.fdb", commit))
	participant, err := fbconn.CreateDatabaseConfig(ctx, participantConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, participant.Close(cleanup))
		require.NoError(t, fbconn.DropDatabaseConfig(cleanup, participantConfig))
	})
	require.NoError(t, participant.Exec(ctx, "create table MAINT_DATA (ID integer)").Read().Err)
	require.NoError(t, participant.Close(ctx))
	return crashMaintenanceClient(t, ctx, config, maintenanceClientScenario{
		Database: config.Database, Prepare: true, ParticipantDatabase: participantConfig.Database, CommitParticipant: commit,
	})
}

func prepareMaintenanceClientDistributedTransaction(t *testing.T, ctx context.Context, c *fbconn.FbConn, config fbconn.Config, scenario maintenanceClientScenario) int64 {
	t.Helper()
	var transactions []*fbconn.FBTransaction
	var ids []int64
	// TDR strings use one-byte lengths. The server can attach both participants
	// locally using the database path, independent of the client's server host.
	databasePath, host := maintenanceDatabaseIdentity(t, ctx, c)
	require.LessOrEqual(t, len(host), 255)
	description := []byte{1, 1, byte(len(host))} // TDR version and host site.
	description = append(description, host...)
	config.Database = scenario.ParticipantDatabase
	participant, err := fbconn.ConnectConfig(ctx, &config)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, participant.Close(cleanup))
	})
	connections := []*fbconn.FbConn{participant, c}
	for i, c := range connections {
		path := databasePath
		if i == 0 {
			path = scenario.ParticipantDatabase
		}
		tx, id := maintenanceClientTransaction(t, ctx, c)
		transactions = append(transactions, tx)
		ids = append(ids, id)
		require.LessOrEqual(t, len(path), 255)
		description = append(description, 2, byte(len(path)))
		description = append(description, path...)
		description = append(description, 3, 8)
		description = binary.LittleEndian.AppendUint64(description, uint64(id))
	}
	for i, tx := range transactions {
		prepareMaintenanceClientTransaction(t, ctx, connections[i], tx, description)
	}
	if scenario.CommitParticipant {
		require.NoError(t, transactions[0].Commit(ctx))
	} else {
		require.NoError(t, transactions[0].Rollback(ctx))
	}
	require.NoError(t, participant.Close(ctx))
	return ids[1]
}

func maintenanceDatabaseIdentity(t *testing.T, ctx context.Context, c *fbconn.FbConn) (string, string) {
	t.Helper()
	require.NoError(t, c.Client().WriteMessage(&fbproto.OpInfoDatabase{Parameters: []byte{fbproto.Isc_info_db_id}, BufferLength: 1024}))
	require.NoError(t, c.Client().Flush())
	message, err := c.ReceiveMessage(ctx)
	require.NoError(t, err)
	require.IsType(t, &fbproto.OpResponse{}, message)
	data := message.(*fbproto.OpResponse).Data
	require.GreaterOrEqual(t, len(data), 6)
	require.Equal(t, byte(fbproto.Isc_info_db_id), data[0])
	require.GreaterOrEqual(t, data[3], byte(2)) // Path and host; remote layers may append more strings.
	pathEnd := 5 + int(data[4])
	require.Less(t, pathEnd, len(data))
	hostEnd := pathEnd + 1 + int(data[pathEnd])
	require.LessOrEqual(t, hostEnd, len(data))
	return string(data[5:pathEnd]), string(data[pathEnd+1 : hostEnd])
}
