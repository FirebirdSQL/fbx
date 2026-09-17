package fbconn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

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
