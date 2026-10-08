package stdlib

import (
	"context"
	"database/sql/driver"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
)

func TestValidatorRejectsAdditionalTransactions(t *testing.T) {
	url := os.Getenv("FBX_TEST_DATABASE")
	if url == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := fbx.Connect(ctx, url)
	require.NoError(t, err)
	defer func() { _ = conn.Close(context.Background()) }()
	first, err := conn.FbConn().StartTransaction(ctx)
	require.NoError(t, err)
	defer func() { _ = first.Rollback(context.Background()) }()
	second, err := conn.FbConn().StartTransaction(ctx)
	require.NoError(t, err)
	defer func() { _ = second.Rollback(context.Background()) }()
	// End the last bound transaction while an earlier handle stays active.
	require.NoError(t, second.Rollback(ctx))
	require.True(t, conn.FbConn().ActiveTx())
	require.False(t, conn.FbConn().ActiveExplicitTx())
	adapter := &Conn{
		conn:             conn,
		shouldPing:       func(context.Context, ShouldPingParams) bool { return false },
		resetSessionFunc: func(context.Context, *fbx.Conn) error { return nil },
	}
	assert.False(t, adapter.IsValid(), "database/sql must not pool a connection with unfinished transactions")
	assert.ErrorIs(t, adapter.ResetSession(ctx), driver.ErrBadConn)
}

func TestValidatorRejectsSQLManagedTransaction(t *testing.T) {
	url := os.Getenv("FBX_TEST_DATABASE")
	if url == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := fbx.Connect(ctx, url)
	require.NoError(t, err)
	defer func() { _ = conn.Close(context.Background()) }()
	observer, err := fbx.Connect(ctx, url)
	require.NoError(t, err)
	defer func() { _ = observer.Close(context.Background()) }()
	require.NoError(t, conn.FbConn().ExecImmediateTx(ctx, nil, "set transaction"))
	var serverTransactions int
	query := fmt.Sprintf("select cast(count(*) as integer) from mon$transactions where mon$attachment_id=%d", conn.FbConn().AttachmentID())
	require.NoError(t, observer.QueryRow(ctx, query).Scan(&serverTransactions))
	require.Positive(t, serverTransactions, "the SQL command created a server transaction")
	adapter := &Conn{
		conn:             conn,
		shouldPing:       func(context.Context, ShouldPingParams) bool { return false },
		resetSessionFunc: func(context.Context, *fbx.Conn) error { return nil },
	}
	t.Logf("server transactions=%d; registry active=%t; pool valid=%t", serverTransactions, conn.FbConn().ActiveTx(), adapter.IsValid())
	assert.False(t, adapter.IsValid())
	assert.ErrorIs(t, adapter.ResetSession(ctx), driver.ErrBadConn)
}

func TestValidatorRejectsUnfinishedOperationWithoutTransaction(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.Autocommit = true
	conn, err := fbx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { _ = conn.Close(context.Background()) }()
	adapter := &Conn{
		conn:             conn,
		shouldPing:       func(context.Context, ShouldPingParams) bool { return false },
		resetSessionFunc: func(context.Context, *fbx.Conn) error { return nil },
	}
	require.True(t, adapter.IsValid())
	reader := conn.FbConn().Exec(ctx, "commit")
	require.True(t, conn.FbConn().IsBusy())
	require.False(t, conn.FbConn().ActiveTx(), "SQL commit ended the transaction before ResultReader.Close")
	assert.False(t, adapter.IsValid())
	assert.ErrorIs(t, adapter.ResetSession(ctx), driver.ErrBadConn)
	_, err = reader.Close()
	require.NoError(t, err)
	require.True(t, adapter.IsValid())
	require.NoError(t, adapter.ResetSession(ctx))
}

func TestPrepareReleasesOnlyImplicitTransaction(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit=%t", explicit), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
			require.NoError(t, err)
			config.Autocommit = true
			driverConn, err := GetConnector(*config).Connect(ctx)
			require.NoError(t, err)
			adapter := driverConn.(*Conn)
			defer func() { _ = adapter.Close() }()
			var tx driver.Tx
			if explicit {
				tx, err = adapter.BeginTx(ctx, driver.TxOptions{})
				require.NoError(t, err)
				defer func() { _ = tx.Rollback() }()
			}
			stmt, err := adapter.PrepareContext(ctx, "select 42 from rdb$database")
			require.NoError(t, err)
			sharedStmt, err := adapter.PrepareContext(ctx, "select 42 from rdb$database")
			require.NoError(t, err)
			require.Equal(t, explicit, adapter.conn.FbConn().ActiveTx())
			require.Equal(t, !explicit, adapter.IsValid())
			require.NoError(t, stmt.Close())
			require.NoError(t, sharedStmt.Close())
			if explicit {
				require.NoError(t, tx.Rollback())
			}
			require.True(t, adapter.IsValid())
		})
	}
}
