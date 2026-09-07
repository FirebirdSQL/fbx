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
				conn:         conn,
				transaction:  closedTx,
				cursorOpened: true,
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
			require.True(t, st.cursorOpened, "closed transaction must be rejected before cursor cleanup")
			require.False(t, currentTx.Closed)
		})
	}
}
