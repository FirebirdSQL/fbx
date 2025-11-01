package fbconn_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/red-soft/fbx/fbconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var generateSeriesTemplate = `WITH RECURSIVE generate_series(%[3]s)
AS (SELECT %[1]d FROM RDB$DATABASE UNION ALL SELECT (%[3]s + 1)
FROM generate_series WHERE %[3]s < %[2]d)
SELECT %[3]s FROM generate_series`

func generateSeriesQuery(begin, end int, fieldName string) string {
	if len(fieldName) == 0 {
		fieldName = "generate_series"
	}
	return fmt.Sprintf(generateSeriesTemplate, begin, end, fieldName)
}

func closeConn(t testing.TB, conn *fbconn.PgConn) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, conn.Close(ctx))
	select {
	case <-conn.CleanupDone():
	case <-time.After(30 * time.Second):
		t.Fatal("Connection cleanup exceeded maximum time")
	}
}

// Do a simple query to ensure the connection is still usable
func ensureConnValid(t *testing.T, pgConn *fbconn.PgConn) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//select generate_series(1,$1)
	result := pgConn.ExecParams(ctx, "WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < 3) SELECT generate_series FROM generate_series", [][]byte{[]byte("3")}).Read()
	cancel()

	require.Nil(t, result.Err)
	assert.Equal(t, 3, len(result.Rows))
	// original expected 3 strings "1", "2", "3"
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x1}, result.Rows[0][0])
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x2}, result.Rows[1][0])
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x3}, result.Rows[2][0])
}
