package fbconn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandTag(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		queryStatus QueryStatus
		isInsert    bool
		isUpdate    bool
		isDelete    bool
		isSelect    bool
		isDDL       bool
	}{
		{queryStatus: QueryStatus{statementType: StatementInsert}, isInsert: true},
		{queryStatus: QueryStatus{statementType: StatementSelect}, isSelect: true},
		{queryStatus: QueryStatus{statementType: StatementUpdate}, isUpdate: true},
		{queryStatus: QueryStatus{statementType: StatementDelete}, isDelete: true},
		{queryStatus: QueryStatus{statementType: StatementDDL}, isDDL: true},
	}

	for i, tt := range tests {
		ct := tt.queryStatus
		assert.Equalf(t, tt.isInsert, ct.Insert(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isUpdate, ct.Update(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isDelete, ct.Delete(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isSelect, ct.Select(), "%d. %v", i, tt.queryStatus)
	}
}

func TestBatchCloseCanceledContextKeepsBatchOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn := &FbConn{status: connStatusIdle}
	st := &Statement{conn: conn}
	b := &Batch{st: st, state: batchStateCreated}
	st.activeBatch = b

	err := b.Close(ctx)

	require.Error(t, err)
	require.False(t, b.isClosed())
	require.Same(t, b, st.activeBatch)
	require.False(t, conn.IsClosed())
}

func TestStatementDeallocateCanceledContextKeepsStatementOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn := &FbConn{status: connStatusIdle}
	st := &Statement{
		conn: conn,
		Description: &StatementDescription{
			StatementHandle: 1,
		},
	}

	err := st.Deallocate(ctx)

	require.Error(t, err)
	require.False(t, st.closed)
	require.False(t, conn.IsClosed())
}

func TestTransactionEndCanceledContextKeepsTransactionOpen(t *testing.T) {
	for _, tt := range []struct {
		name string
		end  func(context.Context, *FBTransaction) error
	}{
		{name: "commit", end: func(ctx context.Context, tx *FBTransaction) error { return tx.Commit(ctx) }},
		{name: "rollback", end: func(ctx context.Context, tx *FBTransaction) error { return tx.Rollback(ctx) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			conn := &FbConn{
				status:             connStatusIdle,
				activeTransactions: make(map[*FBTransaction]struct{}),
			}
			tx := &FBTransaction{Connection: conn, Handle: 1}
			conn.activeTransactions[tx] = struct{}{}

			err := tt.end(ctx, tx)

			require.Error(t, err)
			require.False(t, tx.Closed)
			require.Contains(t, conn.activeTransactions, tx)
			require.False(t, conn.IsClosed())
		})
	}
}

func TestActiveExplicitTxIgnoresInternalTransaction(t *testing.T) {
	internalTx := &FBTransaction{}
	conn := &FbConn{
		internalTransaction: internalTx,
		explicitTransaction: false,
		activeTransactions:  map[*FBTransaction]struct{}{internalTx: {}},
	}

	require.True(t, conn.ActiveTx())
	require.False(t, conn.ActiveExplicitTx())

	conn.explicitTransaction = true
	require.True(t, conn.ActiveExplicitTx())

	internalTx.Closed = true
	require.False(t, conn.ActiveExplicitTx())
}
