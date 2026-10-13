package fbxpool_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbxpool"
)

func TestConnect(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	connString := os.Getenv("FBX_TEST_DATABASE")
	pool, err := fbxpool.New(ctx, connString)
	require.NoError(t, err)
	assert.Equal(t, connString, pool.Config().ConnString())
	pool.Close()
}

func TestConnectConfig(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	connString := os.Getenv("FBX_TEST_DATABASE")
	config, err := fbxpool.ParseConfig(connString)
	require.NoError(t, err)
	pool, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	assertConfigsEqual(t, config, pool.Config(), "Pool.Config() returns original config")
	pool.Close()
}

func TestParseConfigExtractsPoolArguments(t *testing.T) {
	t.Parallel()

	config, err := fbxpool.ParseConfig("pool_max_conns=42 pool_min_conns=1 pool_min_idle_conns=2")
	assert.NoError(t, err)
	assert.EqualValues(t, 42, config.MaxConns)
	assert.EqualValues(t, 1, config.MinConns)
	assert.EqualValues(t, 2, config.MinIdleConns)
	assert.NotContains(t, config.ConnConfig.RuntimeParams, "pool_max_conns")
	assert.NotContains(t, config.ConnConfig.RuntimeParams, "pool_min_conns")
}

func TestConstructorIgnoresContext(t *testing.T) {
	t.Parallel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	assert.NoError(t, err)
	var cancel func()
	config.BeforeConnect = func(context.Context, *fbx.ConnConfig) error {
		// cancel the query's context before we actually Dial to ensure the Dial's
		// context isn't cancelled
		cancel()
		return nil
	}

	pool, err := fbxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)

	assert.EqualValues(t, 0, pool.Stat().TotalConns())

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	_, err = pool.Exec(ctx, "SELECT 1")
	assert.ErrorIs(t, err, context.Canceled)
	assert.EqualValues(t, 1, pool.Stat().TotalConns())
}

func TestConnectConfigRequiresConnConfigFromParseConfig(t *testing.T) {
	t.Parallel()

	config := &fbxpool.Config{}

	require.PanicsWithValue(t, "config must be created by ParseConfig", func() {
		_, _ = fbxpool.NewWithConfig(context.Background(), config)
	})
}

func TestConfigCopyReturnsEqualConfig(t *testing.T) {
	connString := "firebird://ivan:secret@localhost:3050/mydb?application_name=fbxtest&connect_timeout=5"
	original, err := fbxpool.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()

	assertConfigsEqual(t, original, copied, t.Name())
}

func TestConfigCopyCanBeUsedToConnect(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	original, err := fbxpool.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assert.NotPanics(t, func() {
		_, err = fbxpool.NewWithConfig(context.Background(), copied)
	})
	assert.NoError(t, err)
}

func TestPoolAcquireAndConnRelease(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	c.Release()
}

func TestPoolAcquireAndConnHijack(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)

	connsBeforeHijack := pool.Stat().TotalConns()

	conn := c.Hijack()
	defer func() { require.NoError(t, conn.Close(ctx)) }()

	connsAfterHijack := pool.Stat().TotalConns()
	require.Equal(t, connsBeforeHijack-1, connsAfterHijack)

	var n int32
	err = conn.QueryRow(ctx, `select 1 from rdb$database`).Scan(&n)
	require.NoError(t, err)
	require.Equal(t, int32(1), n)
}

func TestPoolAcquireChecksIdleConns(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	controllerConn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { require.NoError(t, controllerConn.Close(ctx)) }()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	var conns []*fbxpool.Conn
	for i := 0; i < 3; i++ {
		c, err := pool.Acquire(ctx)
		require.NoError(t, err)
		conns = append(conns, c)
	}

	require.EqualValues(t, 3, pool.Stat().TotalConns())

	var connIds []int32
	for _, c := range conns {
		var id int32
		err = c.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id)
		require.NoError(t, err)
		connIds = append(connIds, id)
		err = c.Conn().FbConn().Disconnect(ctx)
		require.NoError(t, err)
		c.Release()
	}

	// All conns are dead they don't know it and neither does the pool.
	require.EqualValues(t, 3, pool.Stat().TotalConns())

	// Wait long enough so the pool will realize it needs to check the connections.
	time.Sleep(time.Second)

	// Pool should try all existing connections and find them dead, then create a new connection which should successfully ping.
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// The original 3 conns should have been terminated and the a new conn established for the ping.
	require.EqualValues(t, 1, pool.Stat().TotalConns())
	c, err := pool.Acquire(ctx)
	require.NoError(t, err)

	var id int32
	err = c.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id)
	require.NoError(t, err)
	c.Release()

	require.NotContains(t, connIds, id)
}

func TestPoolAcquireChecksIdleConnsWithShouldPing(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	controllerConn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { require.NoError(t, controllerConn.Close(ctx)) }()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	// Replace the default ShouldPing func
	var shouldPingLastCalledWith *fbxpool.ShouldPingParams
	config.ShouldPing = func(ctx context.Context, params fbxpool.ShouldPingParams) bool {
		shouldPingLastCalledWith = &params
		return false
	}

	pool, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	c.Release()

	time.Sleep(time.Millisecond * 200)

	c, err = pool.Acquire(ctx)
	require.NoError(t, err)
	conn := c.Conn()

	require.NotNil(t, shouldPingLastCalledWith)
	assert.Equal(t, conn, shouldPingLastCalledWith.Conn)
	assert.InDelta(t, time.Millisecond*200, shouldPingLastCalledWith.IdleDuration, float64(time.Millisecond*100))

	c.Release()
}

func TestPoolAcquireWithMaxConnsEqualsMaxInt32(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.MaxConns = math.MaxInt32

	pool, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	c.Release()
}

func TestPoolAcquireFunc(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	var n int32
	err = pool.AcquireFunc(ctx, func(c *fbxpool.Conn) error {
		return c.QueryRow(ctx, "select 1 from rdb$database").Scan(&n)
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}

func TestPoolAcquireFuncReturnsFnError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	err = pool.AcquireFunc(ctx, func(c *fbxpool.Conn) error {
		return fmt.Errorf("some error")
	})
	require.EqualError(t, err, "some error")
}

func TestPoolBeforeConnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.BeforeConnect = func(ctx context.Context, cfg *fbx.ConnConfig) error {
		cfg.RuntimeParams["application_name"] = "app_before_connect"
		return nil
	}

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	var str string
	// Limit the request to the current connection to avoid interference from
	// other concurrent test attachments running in parallel.
	err = db.QueryRow(ctx, `select mon$remote_process from mon$attachments where mon$attachment_id = current_connection`).Scan(&str)

	require.NoError(t, err)
	assert.EqualValues(t, "app_before_connect", str)
}

func TestPoolAfterConnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.AfterConnect = func(ctx context.Context, c *fbx.Conn) error {
		_, err := c.Prepare(ctx, "ps1", "select 1 from rdb$database")
		return err
	}

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	var n int32
	err = db.QueryRow(ctx, "ps1").Scan(&n)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
}

func TestPoolBeforeAcquire(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	acquireAttempts := 0

	config.BeforeAcquire = func(ctx context.Context, c *fbx.Conn) bool {
		acquireAttempts++
		return acquireAttempts%2 == 0
	}

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	conns := make([]*fbxpool.Conn, 4)
	for i := range conns {
		conns[i], err = db.Acquire(ctx)
		assert.NoError(t, err)
	}

	for _, c := range conns {
		c.Release()
	}
	waitForReleaseToComplete()

	assert.EqualValues(t, 8, acquireAttempts)

	conns = db.AcquireAllIdle(ctx)
	assert.Len(t, conns, 2)

	for _, c := range conns {
		c.Release()
	}
	waitForReleaseToComplete()

	assert.EqualValues(t, 12, acquireAttempts)
}

func TestPoolPrepareConn(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	acquireAttempts := 0

	config.PrepareConn = func(context.Context, *fbx.Conn) (bool, error) {
		acquireAttempts++
		var err error
		if acquireAttempts%3 == 0 {
			err = errors.New("PrepareConn error")
		}
		return acquireAttempts%2 == 0, err
	}

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	var errorCount int
	conns := make([]*fbxpool.Conn, 0, 4)
	for {
		conn, err := db.Acquire(ctx)
		if err != nil {
			errorCount++
			continue
		}
		conns = append(conns, conn)
		if len(conns) == 4 {
			break
		}
	}
	const wantErrorCount = 3
	assert.Equal(t, wantErrorCount, errorCount, "Acquire() should have failed %d times", wantErrorCount)

	for _, c := range conns {
		c.Release()
	}
	waitForReleaseToComplete()

	assert.EqualValues(t, len(conns)*2+wantErrorCount-1, acquireAttempts)

	conns = db.AcquireAllIdle(ctx)
	assert.Len(t, conns, 1)

	for _, c := range conns {
		c.Release()
	}
	waitForReleaseToComplete()

	assert.EqualValues(t, 14, acquireAttempts)
}

func TestPoolAfterRelease(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	func() {
		pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		defer pool.Close()
	}()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	afterReleaseCount := 0

	config.AfterRelease = func(c *fbx.Conn) bool {
		afterReleaseCount++
		return afterReleaseCount%2 == 1
	}

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	connIds := map[int32]struct{}{}

	for i := 0; i < 10; i++ {
		conn, err := db.Acquire(ctx)
		assert.NoError(t, err)
		rows, err := conn.Query(ctx, "select mon$attachment_id from mon$attachments where mon$attachment_id = current_connection")
		assert.NoError(t, err)
		var id int32
		assert.True(t, rows.Next())
		err = rows.Scan(&id)
		assert.NoError(t, err)
		rows.Close()
		connIds[id] = struct{}{}
		conn.Release()
		waitForReleaseToComplete()
	}

	assert.EqualValues(t, 5, len(connIds))
}

func TestPoolBeforeClose(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	func() {
		pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		defer pool.Close()
	}()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	connIDs := make(chan int32, 5)
	config.BeforeClose = func(c *fbx.Conn) {
		var id int32
		err = c.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id)
		connIDs <- id
		require.NoError(t, err)
	}

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	acquiredIDs := make([]int32, 0, 5)
	closedIDs := make([]int32, 0, 5)
	for i := 0; i < 5; i++ {
		conn, err := db.Acquire(ctx)
		assert.NoError(t, err)
		var id int32
		err = conn.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id)
		require.NoError(t, err)
		acquiredIDs = append(acquiredIDs, id)

		conn.Release()
		db.Reset()
		closedIDs = append(closedIDs, <-connIDs)
	}

	assert.ElementsMatch(t, acquiredIDs, closedIDs)
}

func TestPoolAcquireAllIdle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	db, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer db.Close()

	conns := make([]*fbxpool.Conn, 3)
	for i := range conns {
		conns[i], err = db.Acquire(ctx)
		assert.NoError(t, err)
	}

	for _, c := range conns {
		if c != nil {
			c.Release()
		}
	}
	waitForReleaseToComplete()

	conns = db.AcquireAllIdle(ctx)
	assert.Len(t, conns, 3)

	for _, c := range conns {
		c.Release()
	}
}

func TestPoolReset(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	db, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer db.Close()

	conns := make([]*fbxpool.Conn, 3)
	for i := range conns {
		conns[i], err = db.Acquire(ctx)
		assert.NoError(t, err)
	}

	db.Reset()

	for _, c := range conns {
		if c != nil {
			c.Release()
		}
	}
	waitForReleaseToComplete()

	require.EqualValues(t, 0, db.Stat().TotalConns())
}

func TestConnReleaseChecksMaxConnLifetime(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.MaxConnLifetime = 250 * time.Millisecond

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	c, err := db.Acquire(ctx)
	require.NoError(t, err)

	time.Sleep(config.MaxConnLifetime)

	c.Release()
	waitForReleaseToComplete()

	stats := db.Stat()
	assert.EqualValues(t, 0, stats.TotalConns())
}

func TestConnReleaseClosesBusyConn(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	db, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer db.Close()

	c, err := db.Acquire(ctx)
	require.NoError(t, err)

	_, err = c.Query(ctx, generateSeriesQuery("1", "10", ""))
	require.NoError(t, err)

	c.Release()
	waitForReleaseToComplete()

	// wait for the connection to actually be destroyed
	for i := 0; i < 1000; i++ {
		if db.Stat().TotalConns() == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	stats := db.Stat()
	assert.EqualValues(t, 0, stats.TotalConns())
}

func TestPoolBackgroundChecksMaxConnLifetime(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.MaxConnLifetime = 100 * time.Millisecond
	config.HealthCheckPeriod = 100 * time.Millisecond

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	c, err := db.Acquire(ctx)
	require.NoError(t, err)
	c.Release()
	time.Sleep(config.MaxConnLifetime + 500*time.Millisecond)

	stats := db.Stat()
	assert.EqualValues(t, 0, stats.TotalConns())
	assert.EqualValues(t, 0, stats.MaxIdleDestroyCount())
	assert.EqualValues(t, 1, stats.MaxLifetimeDestroyCount())
	assert.EqualValues(t, 1, stats.NewConnsCount())
}

func TestPoolBackgroundChecksMaxConnIdleTime(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.MaxConnLifetime = 1 * time.Minute
	config.MaxConnIdleTime = 100 * time.Millisecond
	config.HealthCheckPeriod = 150 * time.Millisecond

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	c, err := db.Acquire(ctx)
	require.NoError(t, err)
	c.Release()
	time.Sleep(config.HealthCheckPeriod)

	for i := 0; i < 1000; i++ {
		if db.Stat().TotalConns() == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	stats := db.Stat()
	assert.EqualValues(t, 0, stats.TotalConns())
	assert.EqualValues(t, 1, stats.MaxIdleDestroyCount())
	assert.EqualValues(t, 0, stats.MaxLifetimeDestroyCount())
	assert.EqualValues(t, 1, stats.NewConnsCount())
}

func TestPoolBackgroundChecksMinConns(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.HealthCheckPeriod = 100 * time.Millisecond
	config.MinConns = 2

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	stats := db.Stat()
	for (stats.IdleConns() != 2 || stats.MaxLifetimeDestroyCount() != 0 || stats.NewConnsCount() != 2) && ctx.Err() == nil {
		time.Sleep(50 * time.Millisecond)
		stats = db.Stat()
	}
	require.EqualValues(t, 2, stats.IdleConns())
	require.EqualValues(t, 0, stats.MaxLifetimeDestroyCount())
	require.EqualValues(t, 2, stats.NewConnsCount())

	c, err := db.Acquire(ctx)
	require.NoError(t, err)

	stats = db.Stat()
	require.EqualValues(t, 1, stats.IdleConns())
	require.EqualValues(t, 0, stats.MaxLifetimeDestroyCount())
	require.EqualValues(t, 2, stats.NewConnsCount())

	err = c.Conn().Close(ctx)
	require.NoError(t, err)
	c.Release()

	stats = db.Stat()
	for (stats.IdleConns() != 2 || stats.MaxIdleDestroyCount() != 0 || stats.NewConnsCount() != 3) && ctx.Err() == nil {
		time.Sleep(50 * time.Millisecond)
		stats = db.Stat()
	}
	require.EqualValues(t, 2, stats.TotalConns())
	require.EqualValues(t, 0, stats.MaxIdleDestroyCount())
	require.EqualValues(t, 3, stats.NewConnsCount())
}

func TestPoolExec(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	testExec(t, ctx, pool)
}

func TestPoolQuery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	// Test common usage
	testQuery(t, ctx, pool)
	waitForReleaseToComplete()

	// Test expected pool behavior
	rows, err := pool.Query(ctx, generateSeriesQuery("1", "?", ""), 10)
	require.NoError(t, err)

	stats := pool.Stat()
	assert.EqualValues(t, 1, stats.AcquiredConns())
	assert.EqualValues(t, 1, stats.TotalConns())

	rows.Close()
	assert.NoError(t, rows.Err())
	waitForReleaseToComplete()

	stats = pool.Stat()
	assert.EqualValues(t, 0, stats.AcquiredConns())
	assert.EqualValues(t, 1, stats.TotalConns())

}

func TestPoolQueryRow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	testQueryRow(t, ctx, pool)
	waitForReleaseToComplete()

	stats := pool.Stat()
	assert.EqualValues(t, 0, stats.AcquiredConns())
	assert.EqualValues(t, 1, stats.TotalConns())
}

func TestPoolQueryRowErrNoRows(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	err = pool.QueryRow(ctx, "select * from rdb$database where false").Scan(nil)
	require.Equal(t, fbx.ErrNoRows, err)
}

func TestPoolQueryRowScanPanicReleasesConnection(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	require.Panics(t, func() {
		var greeting *string
		_ = pool.QueryRow(ctx, "select 'Hello, world!' from rdb$database").Scan(greeting) // Note lack of &. This means that a typed nil is passed to Scan.
	})

	// If the connection is not released this will block forever in the defer pool.Close().
}

func TestConnReleaseClosesConnInFailedTransaction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)

	var id int32
	err = c.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id)
	require.NoError(t, err)

	assert.False(t, c.Conn().FbConn().ActiveTx())

	_, err = c.Begin(ctx)
	assert.NoError(t, err)

	assert.True(t, c.Conn().FbConn().ActiveTx())

	_, err = c.Exec(ctx, "selct")
	assert.Error(t, err)

	assert.True(t, c.Conn().FbConn().ActiveTx())

	c.Release()
	waitForReleaseToComplete()

	c, err = pool.Acquire(ctx)
	require.NoError(t, err)

	var id2 int32
	err = c.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id2)
	require.NoError(t, err)

	assert.NotEqual(t, id, id2)
	assert.False(t, c.Conn().FbConn().ActiveTx())

	c.Release()
}

func TestConnReleaseClosesConnInTransaction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)

	var id int32
	err = c.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id)
	require.NoError(t, err)

	assert.False(t, c.Conn().FbConn().ActiveTx())

	_, err = c.Begin(ctx)
	assert.NoError(t, err)

	assert.True(t, c.Conn().FbConn().ActiveTx())

	c.Release()
	waitForReleaseToComplete()

	c, err = pool.Acquire(ctx)
	require.NoError(t, err)

	var id2 int32
	err = c.QueryRow(ctx, `select current_connection from rdb$database`).Scan(&id2)
	require.NoError(t, err)

	assert.NotEqual(t, id, id2)
	assert.False(t, c.Conn().FbConn().ActiveTx())

	c.Release()
}

func TestConnReleaseDestroysClosedConn(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)

	err = c.Conn().Close(ctx)
	require.NoError(t, err)

	assert.EqualValues(t, 1, pool.Stat().TotalConns())

	c.Release()
	waitForReleaseToComplete()

	// wait for the connection to actually be destroyed
	for i := 0; i < 1000; i++ {
		if pool.Stat().TotalConns() == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	assert.EqualValues(t, 0, pool.Stat().TotalConns())
}

func TestConnPoolQueryConcurrentLoad(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer pool.Close()

	n := 100
	done := make(chan bool)

	for i := 0; i < n; i++ {
		go func() {
			defer func() { done <- true }()
			testQuery(t, ctx, pool)
			testQueryRow(t, ctx, pool)
		}()
	}

	for i := 0; i < n; i++ {
		<-done
	}
}

func TestPoolConnCleanupOnExecFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.ConnConfig.OnFbError = func(c *fbconn.FbConn, fbErr *fbconn.FbError) bool {
		// close the connection on server-side error
		return false
	}

	db, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer db.Close()

	conn, err := db.Acquire(ctx)
	require.NoError(t, err)

	// Execute intentionally invalid query to force server-side error and trigger OnFbError-driven close.
	_, err = conn.Exec(ctx, "select * from non_existing_table")
	assert.Error(t, err)

	conn.Release()

	for i := 0; i < 1000; i++ {
		if db.Stat().TotalConns() == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	assert.EqualValues(t, 0, db.Stat().TotalConns())
}

func TestTxBeginFuncNestedTransactionCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	db, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer db.Close()

	createSql := `
		recreate table fbxpooltx(
      id integer,
      unique (id)
    );
  `

	_, err = db.Exec(ctx, createSql)
	require.NoError(t, err)

	defer func() {
		_, _ = db.Exec(ctx, "drop table fbxpooltx")
	}()

	err = fbx.BeginFunc(ctx, db, func(db fbx.Tx) error {
		_, err := db.Exec(ctx, "insert into fbxpooltx(id) values (1)")
		require.NoError(t, err)

		err = fbx.BeginFunc(ctx, db, func(db fbx.Tx) error {
			_, err := db.Exec(ctx, "insert into fbxpooltx(id) values (2)")
			require.NoError(t, err)

			err = fbx.BeginFunc(ctx, db, func(db fbx.Tx) error {
				_, err := db.Exec(ctx, "insert into fbxpooltx(id) values (3)")
				require.NoError(t, err)
				return nil
			})
			require.NoError(t, err)
			return nil
		})
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	var n int64
	err = db.QueryRow(ctx, "select count(*) from fbxpooltx").Scan(&n)
	require.NoError(t, err)
	require.EqualValues(t, 3, n)
}

func TestTxBeginFuncNestedTransactionRollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	db, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer db.Close()

	createSql := `
		recreate table fbxpooltx(
      id integer,
      unique (id)
    );
  `

	_, err = db.Exec(ctx, createSql)
	require.NoError(t, err)

	defer func() {
		_, _ = db.Exec(ctx, "drop table fbxpooltx")
	}()

	err = fbx.BeginFunc(ctx, db, func(db fbx.Tx) error {
		_, err := db.Exec(ctx, "insert into fbxpooltx(id) values (1)")
		require.NoError(t, err)

		err = fbx.BeginFunc(ctx, db, func(db fbx.Tx) error {
			_, err := db.Exec(ctx, "insert into fbxpooltx(id) values (2)")
			require.NoError(t, err)
			return errors.New("do a rollback")
		})
		require.EqualError(t, err, "do a rollback")

		_, err = db.Exec(ctx, "insert into fbxpooltx(id) values (3)")
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)

	var n int64
	err = db.QueryRow(ctx, "select count(*) from fbxpooltx").Scan(&n)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
}

func TestIdempotentPoolClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	// Close the open pool.
	require.NotPanics(t, func() { pool.Close() })

	// Close the already closed pool.
	require.NotPanics(t, func() { pool.Close() })
}

func TestConnectEagerlyReachesMinPoolSize(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.MinConns = int32(12)
	config.MaxConns = int32(15)

	acquireAttempts := int64(0)
	connectAttempts := int64(0)

	config.PrepareConn = func(ctx context.Context, conn *fbx.Conn) (bool, error) {
		atomic.AddInt64(&acquireAttempts, 1)
		return true, nil
	}
	config.BeforeConnect = func(ctx context.Context, cfg *fbx.ConnConfig) error {
		atomic.AddInt64(&connectAttempts, 1)
		return nil
	}

	pool, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	for i := 0; i < 500; i++ {
		time.Sleep(10 * time.Millisecond)

		stat := pool.Stat()
		if stat.IdleConns() == 12 && stat.AcquireCount() == 0 && stat.TotalConns() == 12 && atomic.LoadInt64(&acquireAttempts) == 0 && atomic.LoadInt64(&connectAttempts) == 12 {
			return
		}
	}

	t.Fatal("did not reach min pool size")

}

func TestPoolAcquireUnlimitedLifetimeDoesNotExpire(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	// A MaxConnLifetime of zero means connections never expire due to age. The
	// acquire-time expiry check must not treat such connections as expired,
	// otherwise Acquire destroys and recreates them in a loop and ultimately fails.
	config.MaxConnLifetime = 0
	config.HealthCheckPeriod = 100 * time.Millisecond

	pool, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Establish a single connection and remember its identity.
	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	firstConn := c.Conn()
	c.Release()
	waitForReleaseToComplete()

	require.EqualValues(t, 1, pool.Stat().TotalConns())

	// Give the background health check several chances to (wrongly) reap the conn.
	time.Sleep(500 * time.Millisecond)

	// Re-acquiring must succeed and hand back the very same connection: it was
	// neither expired at acquire time nor destroyed by the health check.
	for range 5 {
		c, err = pool.Acquire(ctx)
		require.NoError(t, err)
		require.Same(t, firstConn, c.Conn())
		c.Release()
		waitForReleaseToComplete()
	}

	stats := pool.Stat()
	require.EqualValues(t, 0, stats.MaxLifetimeDestroyCount())
	require.EqualValues(t, 1, stats.TotalConns())
}

func TestPoolAcquirePingTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	// Set a very short ping timeout to force timeout during ping and destruction of the connection
	config.PingTimeout = 1 * time.Nanosecond

	var firstConn *fbx.Conn
	// Only ping the initially acquired connection to force its destruction and replacement.
	config.ShouldPing = func(_ context.Context, params fbxpool.ShouldPingParams) bool {
		return firstConn != nil && params.Conn == firstConn
	}
	// Limit to a single connection to ensure the same connection is reused
	config.MinConns = 1
	config.MaxConns = 1

	pool, err := fbxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	c, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, pool.Stat().TotalConns())
	firstConn = c.Conn()

	c.Release()
	require.EqualValues(t, 1, pool.Stat().TotalConns())

	c, err = pool.Acquire(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, pool.Stat().TotalConns())
	newConn := c.Conn()

	c.Release()
	require.NotEqual(t, firstConn, newConn)

	require.EqualValues(t, 1, pool.Stat().TotalConns())
	assert.Nil(t, ctx.Err())
	assert.NotEqualValues(t, firstConn, newConn,
		"Expected a new connection due to ping timeout, but got the same connection")
}
