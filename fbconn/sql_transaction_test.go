package fbconn

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLTransactionLifecycle(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	for _, prepared := range []bool{false, true} {
		for _, sql := range []string{"commit", "rollback", "commit retain", "rollback retain", "commit retain snapshot", "rollback retain snapshot"} {
			t.Run(fmt.Sprintf("prepared=%t/%s", prepared, sql), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				config, err := ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
				require.NoError(t, err)
				conn := retryTestConnect(t, ctx, config)
				tx, err := conn.StartTransaction(ctx)
				require.NoError(t, err)
				handle := tx.Handle
				if prepared {
					st, err := conn.PrepareTx(ctx, tx, sql)
					require.NoError(t, err)
					result := st.ExecTx(ctx, tx, nil).Read()
					require.NoError(t, result.Err)
					require.NoError(t, st.Deallocate(ctx))
				} else {
					require.NoError(t, conn.ExecImmediateTx(ctx, tx, sql))
				}
				retaining := strings.Contains(sql, " retain")
				assert.Equal(t, !retaining, tx.Closed)
				assert.Equal(t, retaining, conn.ActiveTx())
				if retaining {
					assert.Equal(t, handle, tx.Handle)
					require.NoError(t, tx.Rollback(ctx))
				} else {
					assert.ErrorIs(t, conn.ExecImmediateTx(ctx, tx, "select 1 from rdb$database"), ErrTransactionClosed)
				}
				require.NoError(t, conn.Close(ctx), "SQL must not leave stale transaction handles for Close")
			})
		}
	}
}

func TestSQLCreatedTransactionCanBeEnded(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config, err := ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	conn := retryTestConnect(t, ctx, config)
	require.NoError(t, conn.ExecImmediateTx(ctx, nil, "set transaction"))
	require.Len(t, conn.activeTransactions, 1)
	for tx := range conn.activeTransactions {
		require.NoError(t, conn.ExecImmediateTx(ctx, tx, "rollback"))
		require.True(t, tx.Closed)
	}
	require.False(t, conn.ActiveTx())
	require.NoError(t, conn.Close(ctx))
}

func TestSQLTransactionEndInAutocommit(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	for _, prepared := range []bool{false, true} {
		t.Run(fmt.Sprintf("prepared=%t", prepared), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			config, err := ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
			require.NoError(t, err)
			config.Autocommit = true
			conn := retryTestConnect(t, ctx, config)
			if prepared {
				require.NoError(t, conn.Exec(ctx, "commit").Read().Err)
			} else {
				require.NoError(t, conn.ExecImmediate(ctx, "commit"))
			}
			require.False(t, conn.ActiveTx())
			require.NoError(t, conn.Exec(ctx, "select 1 from rdb$database").Read().Err)
			require.NoError(t, conn.Close(ctx))
		})
	}
}
