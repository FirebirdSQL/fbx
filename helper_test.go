package fbx_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbxtest"

	"github.com/stretchr/testify/require"
)

var defaultConnTestRunner fbxtest.ConnTestRunner

var (
	serverMajorVersionCheckOnce sync.Once
	serverMajorVersionValue     int
	engineMajorRegexp           = regexp.MustCompile(`\d+`)
)

func init() {
	defaultConnTestRunner = fbxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		return config
	}
}

func mustConnectString(t testing.TB, connString string) *fbx.Conn {
	conn, err := fbx.Connect(context.Background(), connString)
	if err != nil {
		t.Fatalf("Unable to establish connection: %v", err)
	}
	return conn
}

func getServerMajorVersionForConn(ctx context.Context, t testing.TB, conn interface {
	QueryRow(ctx context.Context, sql string, args ...any) fbx.Row
}) int {
	t.Helper()
	serverMajorVersionCheckOnce.Do(func() {
		var version string
		err := conn.QueryRow(ctx, "select rdb$get_context('SYSTEM','ENGINE_VERSION') from rdb$database").Scan(&version)
		require.NoError(t, err)

		major, ok := parseEngineMajorVersion(version)
		require.Truef(t, ok, "failed to parse engine version: %q", version)
		serverMajorVersionValue = major
	})
	return serverMajorVersionValue
}

func skipIfBatchUnsupported(ctx context.Context, t testing.TB, conn *fbx.Conn) {
	t.Helper()
	serverMajorVersion := getServerMajorVersionForConn(ctx, t, conn)
	if serverMajorVersion < 4 {
		t.Skipf("batch is not supported by server engine version %d", serverMajorVersion)
	}
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

func mustParseConfig(t testing.TB, connString string) *fbx.ConnConfig {
	config, err := fbx.ParseConfig(connString)
	require.Nil(t, err)
	return config
}

func mustConnect(t testing.TB, config *fbx.ConnConfig) *fbx.Conn {
	conn, err := fbx.ConnectConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Unable to establish connection: %v", err)
	}
	return conn
}

func closeConn(t testing.TB, conn *fbx.Conn) {
	err := conn.Close(context.Background())
	if err != nil {
		t.Fatalf("conn.Close unexpectedly failed: %v", err)
	}
}

func mustExec(t testing.TB, conn *fbx.Conn, sql string, arguments ...any) (commandTag fbconn.QueryStatus) {
	var err error
	if commandTag, err = conn.Exec(context.Background(), sql, arguments...); err != nil {
		t.Fatalf("Exec unexpectedly failed with %v: %v", sql, err)
	}
	return
}

// Do a simple query to ensure the connection is still usable
func ensureConnValid(t testing.TB, conn *fbx.Conn) {
	var sum, rowCount int32

	rows, err := conn.Query(context.Background(), generateSeriesQuery("1", "?", ""), 10)
	if err != nil {
		t.Fatalf("conn.Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n int32
		rows.Scan(&n)
		sum += n
		rowCount++
	}

	if rows.Err() != nil {
		t.Fatalf("conn.Query failed: %v", rows.Err())
	}

	if rowCount != 10 {
		t.Error("Select called onDataRow wrong number of times")
	}
	if sum != 55 {
		t.Error("Wrong values returned")
	}
}

func assertConfigsEqual(t *testing.T, expected, actual *fbx.ConnConfig, testName string) {
	if !assert.NotNil(t, expected) {
		return
	}
	if !assert.NotNil(t, actual) {
		return
	}

	assert.Equalf(t, expected.Tracer, actual.Tracer, "%s - Tracer", testName)
	assert.Equalf(t, expected.ConnString(), actual.ConnString(), "%s - ConnString", testName)
	assert.Equalf(t, expected.StatementCacheCapacity, actual.StatementCacheCapacity, "%s - StatementCacheCapacity", testName)
	assert.Equalf(t, expected.DescriptionCacheCapacity, actual.DescriptionCacheCapacity, "%s - DescriptionCacheCapacity", testName)
	assert.Equalf(t, expected.DefaultQueryExecMode, actual.DefaultQueryExecMode, "%s - DefaultQueryExecMode", testName)
	assert.Equalf(t, expected.Host, actual.Host, "%s - Host", testName)
	assert.Equalf(t, expected.Database, actual.Database, "%s - Database", testName)
	assert.Equalf(t, expected.Port, actual.Port, "%s - Port", testName)
	assert.Equalf(t, expected.User, actual.User, "%s - User", testName)
	assert.Equalf(t, expected.Password, actual.Password, "%s - Password", testName)
	assert.Equalf(t, expected.ConnectTimeout, actual.ConnectTimeout, "%s - ConnectTimeout", testName)
	assert.Equalf(t, expected.FetchBatchSize, actual.FetchBatchSize, "%s - FetchBatchSize", testName)
	assert.Equalf(t, expected.RuntimeParams, actual.RuntimeParams, "%s - RuntimeParams", testName)

	// Can't test function equality, so just test that they are set or not.
	assert.Equalf(t, expected.ValidateConnect == nil, actual.ValidateConnect == nil, "%s - ValidateConnect", testName)
	assert.Equalf(t, expected.AfterConnect == nil, actual.AfterConnect == nil, "%s - AfterConnect", testName)

	if assert.Equalf(t, len(expected.Fallbacks), len(actual.Fallbacks), "%s - Fallbacks", testName) {
		for i := range expected.Fallbacks {
			assert.Equalf(t, expected.Fallbacks[i].Host, actual.Fallbacks[i].Host, "%s - Fallback %d - Host", testName, i)
			assert.Equalf(t, expected.Fallbacks[i].Port, actual.Fallbacks[i].Port, "%s - Fallback %d - Port", testName, i)
		}
	}
}

var generateSeriesTemplate = `WITH RECURSIVE generate_series(%[3]s)
AS (SELECT %[1]s FROM RDB$DATABASE UNION ALL SELECT (%[3]s + 1)
FROM generate_series WHERE %[3]s < %[2]s)
SELECT %[3]s FROM generate_series`

func generateSeriesQuery(begin, end, fieldName string) string {
	if len(fieldName) == 0 {
		fieldName = "generate_series"
	}
	return fmt.Sprintf(generateSeriesTemplate, begin, end, fieldName)
}
