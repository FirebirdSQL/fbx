package fbx_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestDefaultTransactionIsolation(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	const query = "select mon$isolation_mode, mon$read_only, mon$lock_timeout from mon$transactions where mon$transaction_id = current_transaction"
	for _, isolation := range []fbx.TxIsoLevel{fbx.ReadCommitted, fbx.RepeatableRead, fbx.Serializable} {
		for _, autocommit := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/autocommit=%t", isolation, autocommit), func(t *testing.T) {
				config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
				require.NoError(t, err)
				config.DefaultTransactionIsolation = isolation
				config.DefaultTransactionReadOnly = true
				config.Autocommit = autocommit
				conn := mustConnect(t, config)
				defer closeConn(t, conn)
				want := map[fbx.TxIsoLevel]int{fbx.ReadCommitted: 4, fbx.RepeatableRead: 1, fbx.Serializable: 0}[isolation]
				if isolation == fbx.ReadCommitted && conn.ServerVersion().Major <= 3 {
					want = 2
				}

				var iso, readOnly, lockTimeout int
				for _, mode := range []fbx.QueryExecMode{fbx.QueryExecModeCacheStatement, fbx.QueryExecModeExec} {
					require.NoError(t, conn.QueryRow(ctx, query, mode).Scan(&iso, &readOnly, &lockTimeout))
					require.Equal(t, want, iso)
					require.Equal(t, 1, readOnly)
				}
				if !autocommit {
					require.NoError(t, conn.Commit(ctx))
				}

				for _, opts := range []fbx.TxOptions{
					{},
					{AccessMode: fbx.ReadOnly, LockTimeout: 7 * time.Second},
					{IsoLevel: fbx.RepeatableRead},
				} {
					tx, err := conn.BeginTx(ctx, opts)
					require.NoError(t, err)
					func() {
						defer func() { require.NoError(t, tx.Rollback(ctx)) }()
						require.NoError(t, tx.QueryRow(ctx, query).Scan(&iso, &readOnly, &lockTimeout))
						if opts.IsoLevel != "" {
							require.Equal(t, 1, iso)
						} else {
							require.Equal(t, want, iso)
						}
						if opts.AccessMode == fbx.ReadOnly {
							require.Equal(t, 1, readOnly)
							require.Equal(t, 7, lockTimeout)
						} else {
							require.Zero(t, readOnly, "implicit read-only setting must not change explicit transactions")
						}
					}()
				}

				for _, raw := range [][]byte{nil, {}, {fbproto.Isc_tpb_version3, fbproto.Isc_tpb_concurrency}} {
					tx, err := conn.BeginTxTPB(ctx, raw)
					require.NoError(t, err)
					func() {
						defer func() { require.NoError(t, tx.Rollback(ctx)) }()
						require.NoError(t, tx.QueryRow(ctx, query).Scan(&iso, &readOnly, &lockTimeout))
						require.Equal(t, 1, iso, "raw TPB must preserve server defaults")
						require.Zero(t, readOnly)
					}()
				}
			})
		}
	}
}
