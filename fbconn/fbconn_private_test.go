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
