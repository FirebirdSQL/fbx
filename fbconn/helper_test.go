package fbconn_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
)

var generateSeriesTemplate = `WITH RECURSIVE generate_series(%[3]s)
AS (SELECT %[1]d FROM RDB$DATABASE UNION ALL SELECT (%[3]s + 1)
FROM generate_series WHERE %[3]s < %[2]d)
SELECT %[3]s FROM generate_series`

var (
	serverMajorVersionCheckOnce sync.Once
	serverMajorVersionValue     int
	engineMajorRegexp           = regexp.MustCompile(`\d+`)
)

func generateSeriesQuery(begin, end int, fieldName string) string {
	if len(fieldName) == 0 {
		fieldName = "generate_series"
	}
	return fmt.Sprintf(generateSeriesTemplate, begin, end, fieldName)
}

func closeConn(t testing.TB, conn *fbconn.FbConn) {
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
func ensureConnValid(t *testing.T, fbConn *fbconn.FbConn) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//select generate_series(1,$1)
	result := fbConn.ExecParams(ctx, "WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series", [][]byte{[]byte{0, 0, 0, 3}}).Read()
	cancel()

	require.Nil(t, result.Err)
	assert.Equal(t, 3, len(result.Rows))
	// original expected 3 strings "1", "2", "3"
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x1}, result.Rows[0][0])
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x2}, result.Rows[1][0])
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x3}, result.Rows[2][0])
}

func getServerMajorVersion(ctx context.Context, t testing.TB, conn *fbconn.FbConn) int {
	t.Helper()
	serverMajorVersionCheckOnce.Do(func() {
		result := conn.Exec(ctx, "select rdb$get_context('SYSTEM','ENGINE_VERSION') from rdb$database").Read()
		require.NoError(t, result.Err)
		require.NotEmpty(t, result.Rows)
		require.NotEmpty(t, result.Rows[0])
		require.NotNil(t, result.Rows[0][0])

		version := string(result.Rows[0][0])
		major, ok := parseEngineMajorVersion(version)
		require.Truef(t, ok, "failed to parse engine version: %q", version)
		serverMajorVersionValue = major
	})
	return serverMajorVersionValue
}

func parseEngineMajorVersion(version string) (int, bool) {
	match := engineMajorRegexp.FindString(version)
	if match == "" {
		return 0, false
	}
	major, err := strconv.Atoi(match)
	if err != nil {
		return 0, false
	}
	return major, true
}
