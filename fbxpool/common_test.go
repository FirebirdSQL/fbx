package fbxpool_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	pgx "github.com/red-soft/fbx"
	"github.com/red-soft/fbx/fbconn"
	"github.com/red-soft/fbx/fbxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Conn.Release is an asynchronous process that returns immediately. There is no signal when the actual work is
// completed. To test something that relies on the actual work for Conn.Release being completed we must simply wait.
// This function wraps the sleep so there is more meaning for the callers.
func waitForReleaseToComplete() {
	time.Sleep(500 * time.Millisecond)
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (fbconn.QueryStatus, error)
}

func testExec(t *testing.T, ctx context.Context, db execer) {
	results, err := db.Exec(ctx, "select RDB$SET_CONTEXT('USER_TRANSACTION', 'var', 'val') from rdb$database")
	require.NoError(t, err)
	assert.EqualValues(t, "SELECT", results.String())
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func testQuery(t *testing.T, ctx context.Context, db queryer) {
	var sum, rowCount int32

	rows, err := db.Query(ctx, generateSeriesQuery("1", "?", "generate_series"), 10)
	require.NoError(t, err)

	for rows.Next() {
		var n int32
		rows.Scan(&n)
		sum += n
		rowCount++
	}

	assert.NoError(t, rows.Err())
	assert.Equal(t, int32(10), rowCount)
	assert.Equal(t, int32(55), sum)
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func testQueryRow(t *testing.T, ctx context.Context, db queryRower) {
	var what, who string
	err := db.QueryRow(ctx, "select 'hello', cast(? as varchar) from rdb$database", "world").Scan(&what, &who)
	assert.NoError(t, err)
	assert.Equal(t, "hello", what)
	assert.Equal(t, "world", who)
}

type sendBatcher interface {
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

func testSendBatch(t *testing.T, ctx context.Context, db sendBatcher) {
	batch := &pgx.Batch{}
	batch.Queue("select 1")
	batch.Queue("select 2")

	br := db.SendBatch(ctx, batch)

	var err error
	var n int32
	err = br.QueryRow().Scan(&n)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n)

	err = br.QueryRow().Scan(&n)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)

	err = br.Close()
	assert.NoError(t, err)
}

type copyFromer interface {
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
}

func testCopyFrom(t *testing.T, ctx context.Context, db interface {
	execer
	queryer
	copyFromer
}) {
	_, err := db.Exec(ctx, `create temporary table foo(a int2, b int4, c int8, d varchar, e text, f date, g timestamptz)`)
	require.NoError(t, err)

	tzedTime := time.Date(2010, 2, 3, 4, 5, 6, 0, time.Local)

	inputRows := [][]any{
		{int16(0), int32(1), int64(2), "abc", "efg", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), tzedTime},
		{nil, nil, nil, nil, nil, nil, nil},
	}

	copyCount, err := db.CopyFrom(ctx, pgx.Identifier{"foo"}, []string{"a", "b", "c", "d", "e", "f", "g"}, pgx.CopyFromRows(inputRows))
	assert.NoError(t, err)
	assert.EqualValues(t, len(inputRows), copyCount)

	rows, err := db.Query(ctx, "select * from foo")
	assert.NoError(t, err)

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	assert.NoError(t, rows.Err())
	assert.Equal(t, inputRows, outputRows)
}

func assertConfigsEqual(t *testing.T, expected, actual *fbxpool.Config, testName string) {
	if !assert.NotNil(t, expected) {
		return
	}
	if !assert.NotNil(t, actual) {
		return
	}

	assert.Equalf(t, expected.ConnString(), actual.ConnString(), "%s - ConnString", testName)

	// Can't test function equality, so just test that they are set or not.
	assert.Equalf(t, expected.AfterConnect == nil, actual.AfterConnect == nil, "%s - AfterConnect", testName)
	assert.Equalf(t, expected.BeforeAcquire == nil, actual.BeforeAcquire == nil, "%s - BeforeAcquire", testName)
	assert.Equalf(t, expected.AfterRelease == nil, actual.AfterRelease == nil, "%s - AfterRelease", testName)

	assert.Equalf(t, expected.MaxConnLifetime, actual.MaxConnLifetime, "%s - MaxConnLifetime", testName)
	assert.Equalf(t, expected.MaxConnIdleTime, actual.MaxConnIdleTime, "%s - MaxConnIdleTime", testName)
	assert.Equalf(t, expected.MaxConns, actual.MaxConns, "%s - MaxConns", testName)
	assert.Equalf(t, expected.MinConns, actual.MinConns, "%s - MinConns", testName)
	assert.Equalf(t, expected.MinIdleConns, actual.MinIdleConns, "%s - MinIdleConns", testName)
	assert.Equalf(t, expected.HealthCheckPeriod, actual.HealthCheckPeriod, "%s - HealthCheckPeriod", testName)

	assertConnConfigsEqual(t, expected.ConnConfig, actual.ConnConfig, testName)
}

func assertConnConfigsEqual(t *testing.T, expected, actual *pgx.ConnConfig, testName string) {
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
	assert.Equalf(t, expected.DefaultQueryExecMode, actual.DefaultQueryExecMode, "%s - DefaultQueryExecMode", testName)
	assert.Equalf(t, expected.Host, actual.Host, "%s - Host", testName)
	assert.Equalf(t, expected.Database, actual.Database, "%s - Database", testName)
	assert.Equalf(t, expected.Port, actual.Port, "%s - Port", testName)
	assert.Equalf(t, expected.User, actual.User, "%s - User", testName)
	assert.Equalf(t, expected.Password, actual.Password, "%s - Password", testName)
	assert.Equalf(t, expected.ConnectTimeout, actual.ConnectTimeout, "%s - ConnectTimeout", testName)
	assert.Equalf(t, expected.RuntimeParams, actual.RuntimeParams, "%s - RuntimeParams", testName)

	// Can't test function equality, so just test that they are set or not.
	assert.Equalf(t, expected.ValidateConnect == nil, actual.ValidateConnect == nil, "%s - ValidateConnect", testName)
	assert.Equalf(t, expected.AfterConnect == nil, actual.AfterConnect == nil, "%s - AfterConnect", testName)
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
