package fbconn

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

type prepareCleanupTestConn struct {
	noopConn
	written  bytes.Buffer
	writeErr error
}

func (c *prepareCleanupTestConn) Write(p []byte) (int, error) {
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	return c.written.Write(p)
}

func TestPrepareErrorCleanup(t *testing.T) {
	for _, mode := range []string{"success", "write error", "allocation error", "closed by error handler"} {
		t.Run(mode, func(t *testing.T) {
			var responses bytes.Buffer
			w := fbproto.NewXDRWriter(&responses)
			if mode != "allocation error" {
				require.NoError(t, writeProbeResponse(w, 42))
			}
			require.NoError(t, w.WriteInt32(fbproto.Op_response))
			require.NoError(t, w.WriteInt32(0))
			require.NoError(t, w.WriteInt64(0))
			require.NoError(t, w.WriteBuffer(nil))
			for _, n := range []int{fbproto.Isc_arg_gds, fbproto.Isc_dsql_error, fbproto.Isc_arg_end} {
				require.NoError(t, w.WriteInt32(n))
			}

			transport := &prepareCleanupTestConn{}
			conn := newDetachTestConn(transport)
			encoding, err := fbproto.NewEncoding("UTF8")
			require.NoError(t, err)
			conn.Encoding = encoding
			conn.client = fbproto.NewClient(&responses, conn.writer, conn.Encoding)
			defer func() {
				conn.asyncClose()
				select {
				case <-conn.CleanupDone():
				case <-time.After(time.Second):
					t.Error("connection cleanup did not finish")
				}
			}()

			cleanupErr := &fbconnError{msg: "cleanup write failed", safeToRetry: true}
			var prepareBytes int
			conn.config.OnFbError = func(_ *FbConn, _ *FbError) bool {
				prepareBytes = transport.written.Len()
				if mode == "write error" {
					transport.writeErr = cleanupErr
				}
				return mode != "closed by error handler"
			}

			tx := &FBTransaction{Connection: conn, Handle: 7}
			st, err := conn.PrepareTx(context.Background(), tx, "invalid SQL")
			require.Nil(t, st)
			var fbErr *FbError
			require.ErrorAs(t, err, &fbErr)
			require.Contains(t, fbErr.GdsCodes, fbproto.Isc_dsql_error)
			require.False(t, SafeToRetry(err))
			require.Empty(t, conn.activeStatements)
			cleanupBytes := transport.written.Bytes()[prepareBytes:]
			if mode == "success" {
				require.False(t, conn.IsClosed())
				require.Len(t, cleanupBytes, 12)
				r := fbproto.NewXDRReader(bytes.NewReader(cleanupBytes))
				for _, want := range []int{fbproto.Op_free_statement, 42, fbproto.DSQL_drop} {
					got, err := r.ReadInt32()
					require.NoError(t, err)
					require.Equal(t, want, got)
				}
				require.Equal(t, 1, conn.client.LazyCounter())
			} else {
				require.True(t, conn.IsClosed())
				require.Empty(t, cleanupBytes)
				if mode == "write error" {
					require.ErrorIs(t, err, cleanupErr)
				} else {
					require.Same(t, fbErr, err)
				}
			}
		})
	}
}

func TestPrepareUnexpectedInfoResponseClosesConnection(t *testing.T) {
	var responses bytes.Buffer
	w := fbproto.NewXDRWriter(&responses)
	require.NoError(t, writeProbeResponse(w, 42))
	require.NoError(t, writeProbeResponse(w, 0))
	require.NoError(t, w.WriteInt32(fbproto.Op_sql_response))
	require.NoError(t, w.WriteInt32(0))

	conn := newDetachTestConn(&prepareCleanupTestConn{})
	encoding, err := fbproto.NewEncoding("UTF8")
	require.NoError(t, err)
	conn.Encoding = encoding
	conn.client = fbproto.NewClient(&responses, conn.writer, encoding)
	defer func() {
		conn.asyncClose()
		select {
		case <-conn.CleanupDone():
		case <-time.After(time.Second):
			t.Error("connection cleanup did not finish")
		}
	}()

	tx := &FBTransaction{Connection: conn, Handle: 7}
	st, err := conn.PrepareTx(context.Background(), tx, "select 1 from rdb$database")
	require.Nil(t, st)
	require.ErrorContains(t, err, "received unexpected message")
	require.True(t, conn.IsClosed())
	require.Zero(t, conn.client.LazyCounter(), "must not send FreeStatement after an unexpected response")
	require.Empty(t, conn.activeStatements)
}

func TestStatementRejectsClosedTransactionWithReusedHandle(t *testing.T) {
	for _, method := range []string{"Exec", "ExecTx"} {
		t.Run(method, func(t *testing.T) {
			// No wire client is installed: rejecting the closed transaction
			// must not need any connection or cursor interaction.
			conn := &FbConn{}
			closedTx := &FBTransaction{Connection: conn, Handle: 7, Closed: true}
			currentTx := &FBTransaction{Connection: conn, Handle: closedTx.Handle}
			conn.internalTransaction = currentTx
			st := &Statement{
				conn:              conn,
				transaction:       closedTx,
				cursorTransaction: currentTx,
				Description: &StatementDescription{
					StatementInfo: &StatementInfo{},
				},
			}

			var result *Result
			if method == "Exec" {
				result = st.Exec(context.Background(), nil).Read()
			} else {
				result = st.ExecTx(context.Background(), closedTx, nil).Read()
			}
			require.ErrorIs(t, result.Err, ErrTransactionClosed)
			require.Same(t, currentTx, st.cursorTransaction, "closed transaction must be rejected before cursor cleanup")
			require.False(t, currentTx.Closed)
		})
	}
}

func TestStatementCloseCursorAfterExecutionTransactionEnds(t *testing.T) {
	// No wire client is installed: ending the cursor's transaction already
	// closes the server cursor, even if its handle has since been reused.
	conn := &FbConn{}
	closedTx := &FBTransaction{Connection: conn, Handle: 7, Closed: true}
	currentTx := &FBTransaction{Connection: conn, Handle: closedTx.Handle}
	conn.internalTransaction = currentTx
	st := &Statement{
		conn:              conn,
		transaction:       currentTx,
		cursorTransaction: closedTx,
	}

	require.NoError(t, st.CloseCursor(context.Background()))
	require.NoError(t, st.CloseCursor(context.Background()))
	require.Nil(t, st.cursorTransaction)
	require.Same(t, currentTx, st.transaction)
	require.False(t, currentTx.Closed)
}
