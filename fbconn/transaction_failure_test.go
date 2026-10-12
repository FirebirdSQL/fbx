package fbconn

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestFailedTransactionEndRemainsActive(t *testing.T) {
	for _, operation := range []struct {
		name   string
		opcode int
		end    func(*FBTransaction, context.Context) error
	}{
		{"commit", fbproto.Op_commit, (*FBTransaction).Commit},
		{"rollback", fbproto.Op_rollback, (*FBTransaction).Rollback},
	} {
		for _, retry := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/retry=%t", operation.name, retry), func(t *testing.T) {
				client, server := net.Pipe()
				t.Cleanup(func() { _ = client.Close() })
				t.Cleanup(func() { _ = server.Close() })
				require.NoError(t, server.SetDeadline(time.Now().Add(2*time.Second)))
				conn := newDetachTestConn(client)
				tx := &FBTransaction{Connection: conn, Handle: 42}
				conn.activeTransactions = map[*FBTransaction]struct{}{tx: {}}
				conn.internalTransaction = tx

				serverDone := make(chan error, 1)
				go func() {
					defer func() { _ = server.Close() }()
					serverDone <- serveFailedTransactionEnd(server, operation.opcode)
				}()
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				var fbErr *FbError
				require.ErrorAs(t, operation.end(tx, ctx), &fbErr)
				require.Contains(t, fbErr.GdsCodes, fbproto.Isc_deadlock)
				require.False(t, tx.Closed, "the server rejected the transaction end")
				require.True(t, conn.ActiveTx(), "Close must still be able to roll back the transaction")
				if retry {
					require.NoError(t, tx.Rollback(ctx))
					require.True(t, tx.Closed)
					require.False(t, conn.ActiveTx())
				}
				require.NoError(t, conn.Close(ctx))
				require.NoError(t, <-serverDone)
				require.True(t, tx.Closed)
				require.False(t, conn.ActiveTx())
			})
		}
	}
}

func TestResultReaderRollsBackFailedAutocommit(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = server.Close() })
	require.NoError(t, server.SetDeadline(time.Now().Add(2*time.Second)))
	conn := newDetachTestConn(client)
	conn.config.Autocommit = true
	tx := &FBTransaction{Connection: conn, Handle: 42}
	conn.activeTransactions = map[*FBTransaction]struct{}{tx: {}}
	conn.internalTransaction = tx
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, conn.lock())
	reader := &ResultReader{fbConn: conn, ctx: ctx, boundTransaction: tx}
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = server.Close() }()
		serverDone <- serveFailedTransactionEnd(server, fbproto.Op_commit)
	}()

	_, err := reader.Close()
	var fbErr *FbError
	require.ErrorAs(t, err, &fbErr)
	require.Contains(t, fbErr.GdsCodes, fbproto.Isc_deadlock)
	require.False(t, conn.ActiveTx(), "a later query must not commit the failed autocommit transaction")
	require.True(t, tx.Closed)
	require.NoError(t, conn.Close(ctx))
	require.NoError(t, <-serverDone)
}

func serveFailedTransactionEnd(conn net.Conn, opcode int) error {
	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
	for _, step := range []struct {
		opcode int
		handle int
		code   int
	}{
		{opcode, 42, fbproto.Isc_deadlock},
		{fbproto.Op_rollback, 42, 0},
		{fbproto.Op_detach, 0, 0},
	} {
		gotOpcode, err := r.ReadInt32()
		if err != nil {
			return err
		}
		handle, err := r.ReadInt32()
		if err != nil {
			return err
		}
		if gotOpcode != step.opcode || handle != step.handle {
			return fmt.Errorf("received opcode=%d handle=%d, want opcode=%d handle=%d", gotOpcode, handle, step.opcode, step.handle)
		}
		// op_response: object, blob ID, empty data, then the status vector.
		for _, value := range []int{fbproto.Op_response, 0, 0, 0, 0, fbproto.Isc_arg_gds, step.code, fbproto.Isc_arg_end} {
			if err := w.WriteInt32(value); err != nil {
				return err
			}
		}
	}
	got, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if got != fbproto.Op_disconnect {
		return fmt.Errorf("expected disconnect, got %d", got)
	}
	return nil
}
