package fbx_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

// Two prepared participants in separate test databases form a distributed
// transaction. Complete the first participant, then recover the second through
// the service API using the persisted decision of the first participant.
func prepareMaintenanceDistributedTransaction(t *testing.T, ctx context.Context, config fbconn.Config, commit bool) int64 {
	t.Helper()
	c, err := fbconn.ConnectConfig(ctx, &config)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, c.Close(cleanup))
	})
	var transactions []*fbconn.FBTransaction
	var ids []int64
	// TDR strings use one-byte lengths. The server can attach both participants
	// locally using the database path, independent of the client's server host.
	databasePath, host := maintenanceDatabaseIdentity(t, ctx, c)
	require.LessOrEqual(t, len(host), 255)
	description := []byte{1, 1, byte(len(host))} // TDR version and host site.
	description = append(description, host...)
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
	connections := []*fbconn.FbConn{participant, c}
	for i, c := range connections {
		path := databasePath
		if i == 0 {
			path = participantConfig.Database
		}
		tx, err := c.StartTransaction(ctx)
		require.NoError(t, err)
		result := c.ExecTx(ctx, tx, "select cast(current_transaction as varchar(30)) from RDB$DATABASE").Read()
		require.NoError(t, result.Err)
		id, err := strconv.ParseInt(strings.TrimSpace(string(result.Rows[0][0])), 10, 64)
		require.NoError(t, err)
		result = c.ExecTx(ctx, tx, fmt.Sprintf("insert into MAINT_DATA values (%d)", id)).Read()
		require.NoError(t, result.Err)
		transactions = append(transactions, tx)
		ids = append(ids, id)
		require.LessOrEqual(t, len(path), 255)
		description = append(description, 2, byte(len(path)))
		description = append(description, path...)
		description = append(description, 3, 8)
		description = binary.LittleEndian.AppendUint64(description, uint64(id))
	}
	for i, tx := range transactions {
		c := connections[i]
		require.NoError(t, c.Client().WriteMessage(maintenancePrepareTransaction{handle: tx.Handle, description: description}))
		require.NoError(t, c.Client().Flush())
		message, err := c.ReceiveMessage(ctx)
		require.NoError(t, err)
		require.IsType(t, &fbproto.OpResponse{}, message)
	}
	if commit {
		require.NoError(t, transactions[0].Commit(ctx))
	} else {
		require.NoError(t, transactions[0].Rollback(ctx))
	}
	require.NoError(t, participant.Close(ctx))
	require.NoError(t, c.Close(ctx))
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
