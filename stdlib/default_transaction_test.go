package stdlib_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/stdlib"
	"github.com/stretchr/testify/require"
)

func TestDefaultTransactionIsolation(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	const query = "select mon$isolation_mode, mon$read_only from mon$transactions where mon$transaction_id = current_transaction"
	for _, mode := range []fbx.QueryExecMode{fbx.QueryExecModeCacheStatement, fbx.QueryExecModeExec} {
		for _, isolation := range []string{"read committed", "repeatable read", "serializable"} {
			t.Run(mode.String()+"/"+isolation, func(t *testing.T) {
				config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
				require.NoError(t, err)
				config.DefaultTransactionIsolation = isolation
				config.DefaultQueryExecMode = mode
				db := stdlib.OpenDB(*config)
				defer closeDB(t, db)
				db.SetMaxOpenConns(1)
				want := map[string]int{"read committed": 4, "repeatable read": 1, "serializable": 0}[isolation]
				if isolation == "read committed" && getServerMajorVersion(t, db) <= 3 {
					want = 2
				}
				var iso, readOnly int
				require.NoError(t, db.QueryRowContext(ctx, query).Scan(&iso, &readOnly))
				require.Equal(t, want, iso)
				for _, opts := range []*sql.TxOptions{nil, {}, {ReadOnly: true}, {Isolation: sql.LevelRepeatableRead}} {
					tx, err := db.BeginTx(ctx, opts)
					require.NoError(t, err)
					func() {
						defer func() { require.NoError(t, tx.Rollback()) }()
						require.NoError(t, tx.QueryRowContext(ctx, query).Scan(&iso, &readOnly))
						if opts != nil && opts.Isolation == sql.LevelRepeatableRead {
							require.Equal(t, 1, iso)
						} else {
							require.Equal(t, want, iso)
						}
						if opts != nil && opts.ReadOnly {
							require.Equal(t, 1, readOnly)
						} else {
							require.Zero(t, readOnly)
						}
					}()
				}
				require.NoError(t, db.QueryRowContext(ctx, query).Scan(&iso, &readOnly))
				require.Equal(t, want, iso, "explicit override must not change the connection default")
			})
		}
	}
}
