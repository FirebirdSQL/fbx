package pgconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		{queryStatus: QueryStatus{StatementInsert}, isInsert: true},
		{queryStatus: QueryStatus{StatementSelect}, isSelect: true},
		{queryStatus: QueryStatus{StatementUpdate}, isUpdate: true},
		{queryStatus: QueryStatus{StatementDelete}, isDelete: true},
		{queryStatus: QueryStatus{StatementDDL}, isDDL: true},
	}

	for i, tt := range tests {
		ct := tt.queryStatus
		assert.Equalf(t, tt.isInsert, ct.Insert(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isUpdate, ct.Update(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isDelete, ct.Delete(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isSelect, ct.Select(), "%d. %v", i, tt.queryStatus)
	}
}
