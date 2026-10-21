package fbconn

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func retryTestTransaction(t *testing.T) (*FbConn, *FBTransaction, net.Conn) {
	t.Helper()
	client, server := net.Pipe()
	conn := newDetachTestConn(client)
	conn.config.Autocommit = true
	tx := &FBTransaction{Connection: conn, Handle: 42}
	conn.activeTransactions = map[*FBTransaction]struct{}{tx: {}}
	conn.internalTransaction = tx
	require.NoError(t, server.SetDeadline(time.Now().Add(3*time.Second)))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = conn.Close(ctx)
		_ = server.Close()
		select {
		case <-conn.CleanupDone():
		case <-time.After(3 * time.Second):
			t.Error("transport cleanup did not finish")
		}
	})
	return conn, tx, server
}

func retryTestRequest(r *fbproto.XDRReader, opcode int) error {
	got, err := r.ReadInt32()
	if err != nil {
		return err
	}
	handle, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if got != opcode || handle != 42 {
		return fmt.Errorf("got opcode=%d handle=%d, want opcode=%d handle=42", got, handle, opcode)
	}
	return nil
}

func retryTestResponse(w *fbproto.XDRWriter, code int) error {
	for _, value := range []int{fbproto.Op_response, 0, 0, 0, 0, fbproto.Isc_arg_gds, code, fbproto.Isc_arg_end} {
		if err := w.WriteInt32(value); err != nil {
			return err
		}
	}
	return nil
}

func TestInternalCommitLostReplyDisallowsRetry(t *testing.T) {
	conn, tx, server := retryTestTransaction(t)
	done := make(chan error, 1)
	go func() {
		err := retryTestRequest(fbproto.NewXDRReader(server), fbproto.Op_commit)
		_ = server.Close()
		done <- err
	}()
	err := conn.commitInternalTx(context.Background(), tx)
	require.Error(t, err)
	require.NoError(t, <-done)
	assert.True(t, conn.IsClosed())
	assert.False(t, SafeToRetry(err), "cleanup must not authorize repeating possibly committed SQL")
	var fbErr *FbError
	require.ErrorAs(t, err, &fbErr)
	assert.Equal(t, "08006", fbErr.Code)
	t.Logf("retry=%t; error=%v", SafeToRetry(err), err)
}

func TestInternalCommitCanceledBeforeSendRollsBack(t *testing.T) {
	conn, tx, server := retryTestTransaction(t)
	done := make(chan error, 1)
	go func() {
		err := retryTestRequest(fbproto.NewXDRReader(server), fbproto.Op_rollback)
		if err == nil {
			err = retryTestResponse(fbproto.NewXDRWriter(server), 0)
		}
		done <- err
	}()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := conn.commitInternalTx(ctx, tx)
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, SafeToRetry(err), "the SQL preceding commit has already executed")
	require.NoError(t, <-done)
	assert.False(t, conn.IsClosed(), "confirmed rollback preserves the connection")
	assert.True(t, tx.Closed)
	assert.False(t, conn.ActiveTx(), "the next request must not inherit the transaction")
	t.Logf("closed=%t; active=%t; retry=%t", conn.IsClosed(), conn.ActiveTx(), SafeToRetry(err))
}

func TestInternalCommitRefusalRollsBack(t *testing.T) {
	for _, rollbackFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("rollback_fails=%t", rollbackFails), func(t *testing.T) {
			conn, tx, server := retryTestTransaction(t)
			done := make(chan error, 1)
			go func() {
				r, w := fbproto.NewXDRReader(server), fbproto.NewXDRWriter(server)
				if err := retryTestRequest(r, fbproto.Op_commit); err != nil {
					done <- err
					return
				}
				if err := retryTestResponse(w, fbproto.Isc_deadlock); err != nil {
					done <- err
					return
				}
				if err := retryTestRequest(r, fbproto.Op_rollback); err != nil {
					done <- err
					return
				}
				code := 0
				if rollbackFails {
					code = fbproto.Isc_lock_conflict
				}
				done <- retryTestResponse(w, code)
			}()
			err := conn.commitInternalTx(context.Background(), tx)
			var fbErr *FbError
			require.ErrorAs(t, err, &fbErr)
			assert.Contains(t, fbErr.GdsCodes, fbproto.Isc_deadlock)
			assert.False(t, SafeToRetry(err))
			require.NoError(t, <-done)
			assert.Equal(t, rollbackFails, conn.IsClosed())
			if !rollbackFails {
				assert.False(t, conn.ActiveTx())
			}
			if rollbackFails {
				assert.Contains(t, err.Error(), "rollback internal tx")
			}
		})
	}
}

func TestTransactionEndRejectsUnexpectedReply(t *testing.T) {
	for _, end := range []struct {
		name   string
		opcode int
		call   func(*FBTransaction, context.Context) error
	}{
		{"commit", fbproto.Op_commit, (*FBTransaction).Commit},
		{"rollback", fbproto.Op_rollback, (*FBTransaction).Rollback},
	} {
		t.Run(end.name, func(t *testing.T) {
			conn, tx, server := retryTestTransaction(t)
			done := make(chan error, 1)
			go func() {
				if err := retryTestRequest(fbproto.NewXDRReader(server), end.opcode); err != nil {
					done <- err
					return
				}
				w := fbproto.NewXDRWriter(server)
				for _, value := range []int{fbproto.Op_accept, 13, 1, 0} {
					if err := w.WriteInt32(value); err != nil {
						done <- err
						return
					}
				}
				done <- nil
			}()
			err := end.call(tx, context.Background())
			assert.Error(t, err, "an unrelated protocol message is not confirmation")
			assert.True(t, conn.IsClosed(), "unexpected response makes protocol state unusable")
			require.NoError(t, <-done)
		})
	}
}

func TestTransactionCommitCanceledBeforeSendIsRetryable(t *testing.T) {
	conn, tx, _ := retryTestTransaction(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := tx.Commit(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.True(t, SafeToRetry(err), "the commit command itself has not been sent")
	assert.False(t, conn.IsClosed())
	assert.False(t, tx.Closed)
}

var errLostCommitReply = errors.New("retryTest: discard the commit response")

type retryTestDropReadConn struct {
	net.Conn
	drop atomic.Bool
}

func (c *retryTestDropReadConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 && c.drop.Swap(false) {
		_ = c.Close()
		return 0, errLostCommitReply
	}
	return n, err
}

func retryTestConnect(t *testing.T, ctx context.Context, config *Config) *FbConn {
	t.Helper()
	conn, err := ConnectConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = conn.Close(cleanupCtx)
	})
	return conn
}

func retryTestDatabase(t *testing.T) (context.Context, *Config, *FbConn, string) {
	t.Helper()
	url := os.Getenv("FBX_TEST_DATABASE")
	if url == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	config, err := ParseConfig(url)
	require.NoError(t, err)
	config.Autocommit = true
	admin := retryTestConnect(t, ctx, config)
	t.Logf("server=%v", admin.ServerVersion())
	table := fmt.Sprintf("fbx_rev_%d", time.Now().UnixNano()%1000000000000)
	require.NoError(t, admin.ExecImmediate(ctx, "create table "+table+" (id integer)"))
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := admin.ExecImmediate(cleanupCtx, "drop table "+table); err != nil {
			t.Errorf("remove regression table: %v", err)
		}
	})
	return ctx, config, admin, table
}

func retryTestCount(t *testing.T, ctx context.Context, conn *FbConn, table string, id int) uint32 {
	t.Helper()
	result := conn.ExecParams(ctx, fmt.Sprintf("select cast(count(*) as integer) from %s where id=%d", table, id), nil).Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, 1)
	require.Len(t, result.Rows[0][0], 4)
	return binary.BigEndian.Uint32(result.Rows[0][0])
}

func TestAutocommitCanceledBeforeCommitDoesNotLeakRows(t *testing.T) {
	ctx, config, admin, table := retryTestDatabase(t)
	config.Autocommit = false
	conn := retryTestConnect(t, ctx, config)
	require.NoError(t, conn.ExecImmediate(ctx, "insert into "+table+" (id) values (1)"))
	conn.config.Autocommit = true
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	err := conn.commitInternalTx(canceled, conn.internalTransaction)
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, SafeToRetry(err))
	assert.True(t, conn.IsClosed() || !conn.ActiveTx())
	if conn.IsClosed() {
		config.Autocommit = true
		conn = retryTestConnect(t, ctx, config)
	}
	require.NoError(t, conn.ExecImmediate(ctx, "insert into "+table+" (id) values (2)"))
	previousRows := retryTestCount(t, ctx, admin, table, 1)
	nextRows := retryTestCount(t, ctx, admin, table, 2)
	t.Logf("rows from failed call=%d; rows from next call=%d", previousRows, nextRows)
	assert.Equal(t, uint32(0), previousRows, "the next query committed changes from the failed call")
	assert.Equal(t, uint32(1), nextRows)
}

func TestAutocommitLostCommitReplyDoesNotDuplicateRows(t *testing.T) {
	ctx, config, admin, table := retryTestDatabase(t)
	config.Autocommit = false
	dial := config.DialFunc
	var transport *retryTestDropReadConn
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		transport = &retryTestDropReadConn{Conn: conn}
		return transport, nil
	}
	conn := retryTestConnect(t, ctx, config)
	require.NoError(t, conn.ExecImmediate(ctx, "insert into "+table+" (id) values (1)"))
	conn.config.Autocommit = true
	transport.drop.Store(true)
	err := conn.commitInternalTx(ctx, conn.internalTransaction)
	require.ErrorIs(t, err, errLostCommitReply)
	require.True(t, conn.IsClosed())
	assert.False(t, SafeToRetry(err))
	if SafeToRetry(err) {
		// Model an application honoring the public retry indicator. This is
		// the same decision that stdlib turns into driver.ErrBadConn.
		retryConfig := config.Copy()
		retryConfig.DialFunc = dial
		retryConfig.Autocommit = true
		retryConn := retryTestConnect(t, ctx, retryConfig)
		require.NoError(t, retryConn.ExecImmediate(ctx, "insert into "+table+" (id) values (1)"))
	}
	rows := retryTestCount(t, ctx, admin, table, 1)
	t.Logf("rows after honoring SafeToRetry=%d", rows)
	assert.Equal(t, uint32(1), rows, "SQL was repeated after its commit had already succeeded")
}
