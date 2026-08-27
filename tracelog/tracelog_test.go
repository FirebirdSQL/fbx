package tracelog_test

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbxpool"
	"rdb.red-soft.ru/fbx/fbxtest"
	"rdb.red-soft.ru/fbx/tracelog"
)

var defaultConnTestRunner fbxtest.ConnTestRunner

func init() {
	defaultConnTestRunner = fbxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		return config
	}
}

type testLog struct {
	lvl  tracelog.LogLevel
	msg  string
	data map[string]any
}

type testLogger struct {
	logs []testLog

	mux sync.Mutex
}

func (l *testLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	l.mux.Lock()
	defer l.mux.Unlock()

	data["ctxdata"] = ctx.Value("ctxdata")
	l.logs = append(l.logs, testLog{lvl: level, msg: msg, data: data})
}

func (l *testLogger) Clear() {
	l.mux.Lock()
	defer l.mux.Unlock()

	l.logs = l.logs[0:0]
}

func (l *testLogger) FilterByMsg(msg string) (res []testLog) {
	l.mux.Lock()
	defer l.mux.Unlock()

	for _, log := range l.logs {
		if log.msg == msg {
			res = append(res, log)
		}
	}

	return res
}

func TestContextGetsPassedToLogMethod(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		logger.Clear() // Clear any logs written when establishing connection

		ctx = context.WithValue(ctx, "ctxdata", "foo")
		_, err := conn.Exec(ctx, `select 1 from rdb$database`)
		require.NoError(t, err)
		require.NotEmpty(t, logger.logs)
		for _, l := range logger.logs {
			require.Equal(t, "foo", l.data["ctxdata"])
		}
	})
}

func TestLoggerFunc(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	const testMsg = "foo"

	buf := bytes.Buffer{}
	logger := log.New(&buf, "", 0)

	createAdapterFn := func(logger *log.Logger) tracelog.LoggerFunc {
		return func(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
			logger.Printf("%s", testMsg)
		}
	}

	config := defaultConnTestRunner.CreateConfig(ctx, t)
	config.Tracer = &tracelog.TraceLog{
		Logger:   createAdapterFn(logger),
		LogLevel: tracelog.LogLevelTrace,
	}

	conn, err := fbx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer conn.Close(ctx)

	buf.Reset() // Clear logs written when establishing connection

	if _, err := conn.Exec(ctx, "select 1 from rdb$database"); err != nil {
		t.Fatal(err)
	}

	msg := strings.TrimSpace(buf.String())
	if !strings.Contains(msg, testMsg) {
		t.Errorf("Expected logger function output to contain '%s', but it was '%s'", testMsg, buf.String())
	}
}

func TestLogQuery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		logger.Clear() // Clear any logs written when establishing connection

		_, err := conn.Exec(ctx, `select cast(? as varchar(100)) from rdb$database`, "testing")
		require.NoError(t, err)

		logs := logger.FilterByMsg("Query")
		require.Len(t, logs, 1)
		require.Equal(t, tracelog.LogLevelInfo, logs[0].lvl)
		require.Equal(t, conn.FbConn().AttachmentID(), logs[0].data["attachment_id"])

		logger.Clear()

		_, err = conn.Exec(ctx, `select cast(? as varchar(100)) from non_existing_table`, "testing")
		require.Error(t, err)

		logs = logger.FilterByMsg("Query")
		require.Len(t, logs, 1)
		require.Equal(t, tracelog.LogLevelError, logs[0].lvl)
		require.Equal(t, err, logs[0].data["err"])
		require.Equal(t, conn.FbConn().AttachmentID(), logs[0].data["attachment_id"])
	})
}

func TestLogQueryArgsHandlesUTF8(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		logger.Clear() // Clear any logs written when establishing connection

		var s string
		for i := 0; i < 63; i++ {
			s += "0"
		}
		s += "😊"

		_, err := conn.Exec(ctx, `select cast(? as varchar(256)) from rdb$database`, s)
		require.NoError(t, err)

		logs := logger.FilterByMsg("Query")
		require.Len(t, logs, 1)
		require.Equal(t, tracelog.LogLevelInfo, logs[0].lvl)
		require.Equal(t, s, logs[0].data["args"].([]any)[0])

		logger.Clear()

		_, err = conn.Exec(ctx, `select cast(? as varchar(256)) from rdb$database`, s+"000")
		require.NoError(t, err)

		logs = logger.FilterByMsg("Query")
		require.Len(t, logs, 1)
		require.Equal(t, tracelog.LogLevelInfo, logs[0].lvl)
		require.Equal(t, s+" (truncated 3 bytes)", logs[0].data["args"].([]any)[0])
	})
}

func TestLogConnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	config := defaultConnTestRunner.CreateConfig(ctx, t)
	config.Tracer = tracer

	conn1, err := fbx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer conn1.Close(ctx)
	require.Len(t, logger.logs, 1)
	require.Equal(t, "Connect", logger.logs[0].msg)
	require.Equal(t, tracelog.LogLevelInfo, logger.logs[0].lvl)

	logger.Clear()

	config, err = fbx.ParseConfig("host=/invalid")
	require.NoError(t, err)
	config.Tracer = tracer

	conn2, err := fbx.ConnectConfig(ctx, config)
	require.Nil(t, conn2)
	require.Error(t, err)
	require.Len(t, logger.logs, 1)
	require.Equal(t, "Connect", logger.logs[0].msg)
	require.Equal(t, tracelog.LogLevelError, logger.logs[0].lvl)
}

func TestLogAcquire(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	config := defaultConnTestRunner.CreateConfig(ctx, t)
	config.Tracer = tracer

	poolConfig, err := fbxpool.ParseConfig(config.ConnString())
	require.NoError(t, err)

	poolConfig.ConnConfig = config
	pool1, err := fbxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)
	defer pool1.Close()

	conn1, err := pool1.Acquire(ctx)
	require.NoError(t, err)
	defer conn1.Release()
	require.Len(t, logger.logs, 2) // Has both the Connect and Acquire logs
	require.Equal(t, "Acquire", logger.logs[1].msg)
	require.Equal(t, tracelog.LogLevelDebug, logger.logs[1].lvl)

	logger.Clear()

	// create a 2nd pool with a bad host to verify the error handling
	poolConfig, err = fbxpool.ParseConfig("host=/invalid")
	require.NoError(t, err)
	poolConfig.ConnConfig.Tracer = tracer

	pool2, err := fbxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)
	defer pool2.Close()

	conn2, err := pool2.Acquire(ctx)
	require.Error(t, err)
	require.Nil(t, conn2)
	require.Len(t, logger.logs, 2)
	require.Equal(t, "Acquire", logger.logs[1].msg)
	require.Equal(t, tracelog.LogLevelError, logger.logs[1].lvl)
}

func TestLogRelease(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	config := defaultConnTestRunner.CreateConfig(ctx, t)
	config.Tracer = tracer

	poolConfig, err := fbxpool.ParseConfig(config.ConnString())
	require.NoError(t, err)

	poolConfig.ConnConfig = config
	pool1, err := fbxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	defer pool1.Close()

	conn1, err := pool1.Acquire(ctx)
	require.NoError(t, err)

	logger.Clear()
	conn1.Release()
	require.Len(t, logger.logs, 1)
	require.Equal(t, "Release", logger.logs[0].msg)
	require.Equal(t, tracelog.LogLevelDebug, logger.logs[0].lvl)
}

func TestLogPrepare(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, []fbx.QueryExecMode{
		fbx.QueryExecModeCacheStatement,
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		logger.Clear() // Clear any logs written when establishing connection

		_, err := conn.Exec(ctx, `select cast(? as varchar(100)) from rdb$database`, "testing")
		require.NoError(t, err)

		logs := logger.FilterByMsg("Prepare")
		require.Len(t, logs, 1)
		require.Equal(t, tracelog.LogLevelInfo, logs[0].lvl)

		logger.Clear()

		_, err = conn.Exec(ctx, `select cast(? as varchar(100)) from`, "testing")
		require.Error(t, err)

		logs = logger.FilterByMsg("Prepare")
		require.Len(t, logs, 1)
		require.Equal(t, err, logs[0].data["err"])
	})

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		logger.Clear() // Clear any logs written when establishing connection

		_, err := conn.Prepare(ctx, "test_query_1", `select cast(? as integer) from rdb$database`)
		require.NoError(t, err)

		require.Len(t, logger.logs, 1)
		require.Equal(t, "Prepare", logger.logs[0].msg)
		require.Equal(t, tracelog.LogLevelInfo, logger.logs[0].lvl)

		logger.Clear()

		_, err = conn.Prepare(ctx, `test_query_2`, `select cast(? as integer) from`)
		require.Error(t, err)

		require.Len(t, logger.logs, 1)
		require.Equal(t, "Prepare", logger.logs[0].msg)
		require.Equal(t, err, logger.logs[0].data["err"])
	})
}

// https://github.com/jackc/fbx/pull/2120
func TestConcurrentUsage(t *testing.T) {
	t.Parallel()

	if isRedDatabase50(t) {
		t.Skip("Rdb 5.0 server crash")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	logger := &testLogger{}
	tracer := &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	config, err := fbxpool.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.ConnConfig.Tracer = tracer

	for i := 0; i < 50; i++ {
		func() {
			pool, err := fbxpool.NewWithConfig(ctx, config)
			require.NoError(t, err)

			defer pool.Close()

			eg := errgroup.Group{}

			for i := 0; i < 5; i++ {
				eg.Go(func() error {
					_, err := pool.Exec(ctx, `select 1 from rdb$database`)
					return err
				})
			}

			err = eg.Wait()
			require.NoError(t, err)
		}()
	}
}
