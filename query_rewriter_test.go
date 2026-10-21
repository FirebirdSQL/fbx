package fbx_test

import (
	"context"
	"os"
	"testing"

	"github.com/firebirdsql/fbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamedArgsQueryRewriter(t *testing.T) {
	t.Parallel()

	connConfig := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	connConfig.DefaultQueryExecMode = fbx.QueryExecModeExec

	conn := mustConnect(t, connConfig)
	defer closeConn(t, conn)

	var a, c int
	var b string
	err := conn.QueryRow(context.Background(), "select cast(@arg1 as int), cast(@arg2 as varchar(255)), cast(@arg1 as int) from rdb$database", fbx.NamedArgs{"arg1": 42, "arg2": "test"}).Scan(&a, &b, &c)
	require.NoError(t, err)
	assert.Equal(t, 42, a)
	assert.Equal(t, "test", b)
	assert.Equal(t, 42, c)

	ensureConnValid(t, conn)
}
