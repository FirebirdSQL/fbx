package fbx_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbxtest"
)

type testTracer struct {
	traceQueryStart   func(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryStartData) context.Context
	traceQueryEnd     func(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryEndData)
	tracePrepareStart func(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareStartData) context.Context
	tracePrepareEnd   func(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareEndData)
	traceConnectStart func(ctx context.Context, data fbx.TraceConnectStartData) context.Context
	traceConnectEnd   func(ctx context.Context, data fbx.TraceConnectEndData)
}

type ctxKey string

func (tt *testTracer) TraceQueryStart(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryStartData) context.Context {
	if tt.traceQueryStart != nil {
		return tt.traceQueryStart(ctx, conn, data)
	}
	return ctx
}

func (tt *testTracer) TraceQueryEnd(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryEndData) {
	if tt.traceQueryEnd != nil {
		tt.traceQueryEnd(ctx, conn, data)
	}
}

func (tt *testTracer) TracePrepareStart(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareStartData) context.Context {
	if tt.tracePrepareStart != nil {
		return tt.tracePrepareStart(ctx, conn, data)
	}
	return ctx
}

func (tt *testTracer) TracePrepareEnd(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareEndData) {
	if tt.tracePrepareEnd != nil {
		tt.tracePrepareEnd(ctx, conn, data)
	}
}

func (tt *testTracer) TraceConnectStart(ctx context.Context, data fbx.TraceConnectStartData) context.Context {
	if tt.traceConnectStart != nil {
		return tt.traceConnectStart(ctx, data)
	}
	return ctx
}

func (tt *testTracer) TraceConnectEnd(ctx context.Context, data fbx.TraceConnectEndData) {
	if tt.traceConnectEnd != nil {
		tt.traceConnectEnd(ctx, data)
	}
}

func TestTraceExec(t *testing.T) {
	t.Parallel()

	tracer := &testTracer{}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		traceQueryStartCalled := false
		tracer.traceQueryStart = func(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryStartData) context.Context {
			traceQueryStartCalled = true
			require.Equal(t, `select cast(? as varchar(255)) from rdb$database`, data.SQL)
			require.Len(t, data.Args, 1)
			require.Equal(t, `testing`, data.Args[0])
			return context.WithValue(ctx, ctxKey(ctxKey("fromTraceQueryStart")), "foo")
		}

		traceQueryEndCalled := false
		tracer.traceQueryEnd = func(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryEndData) {
			traceQueryEndCalled = true
			require.Equal(t, "foo", ctx.Value(ctxKey(ctxKey("fromTraceQueryStart"))))
			require.Equal(t, `SELECT`, data.QueryStatus.String())
			require.NoError(t, data.Err)
		}

		_, err := conn.Exec(ctx, `select cast(? as varchar(255)) from rdb$database`, "testing")
		require.NoError(t, err)
		require.True(t, traceQueryStartCalled)
		require.True(t, traceQueryEndCalled)
	})
}

func TestTraceQuery(t *testing.T) {
	t.Parallel()

	tracer := &testTracer{}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		traceQueryStartCalled := false
		tracer.traceQueryStart = func(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryStartData) context.Context {
			traceQueryStartCalled = true
			require.Equal(t, `select cast(? as varchar(255)) from rdb$database`, data.SQL)
			require.Len(t, data.Args, 1)
			require.Equal(t, `testing`, data.Args[0])
			return context.WithValue(ctx, ctxKey("fromTraceQueryStart"), "foo")
		}

		traceQueryEndCalled := false
		tracer.traceQueryEnd = func(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryEndData) {
			traceQueryEndCalled = true
			require.Equal(t, "foo", ctx.Value(ctxKey("fromTraceQueryStart")))
			require.Equal(t, `SELECT`, data.QueryStatus.String())
			require.NoError(t, data.Err)
		}

		var s string
		err := conn.QueryRow(ctx, `select cast(? as varchar(255)) from rdb$database`, "testing").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "testing", s)
		require.True(t, traceQueryStartCalled)
		require.True(t, traceQueryEndCalled)
	})
}

func TestTracePrepare(t *testing.T) {
	t.Parallel()

	tracer := &testTracer{}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		tracePrepareStartCalled := false
		tracer.tracePrepareStart = func(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareStartData) context.Context {
			tracePrepareStartCalled = true
			require.Equal(t, `ps`, data.Name)
			require.Equal(t, `select cast(? as varchar(255)) from rdb$database`, data.SQL)
			return context.WithValue(ctx, ctxKey("fromTracePrepareStart"), "foo")
		}

		tracePrepareEndCalled := false
		tracer.tracePrepareEnd = func(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareEndData) {
			tracePrepareEndCalled = true
			require.False(t, data.AlreadyPrepared)
			require.NoError(t, data.Err)
		}

		_, err := conn.Prepare(ctx, "ps", `select cast(? as varchar(255)) from rdb$database`)
		require.NoError(t, err)
		require.True(t, tracePrepareStartCalled)
		require.True(t, tracePrepareEndCalled)

		tracePrepareStartCalled = false
		tracePrepareEndCalled = false
		tracer.tracePrepareEnd = func(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareEndData) {
			tracePrepareEndCalled = true
			require.True(t, data.AlreadyPrepared)
			require.NoError(t, data.Err)
		}

		_, err = conn.Prepare(ctx, "ps", `select cast(? as varchar(255)) from rdb$database`)
		require.NoError(t, err)
		require.True(t, tracePrepareStartCalled)
		require.True(t, tracePrepareEndCalled)
	})
}

func TestTraceConnect(t *testing.T) {
	t.Parallel()

	tracer := &testTracer{}

	config := defaultConnTestRunner.CreateConfig(context.Background(), t)
	config.Tracer = tracer

	traceConnectStartCalled := false
	tracer.traceConnectStart = func(ctx context.Context, data fbx.TraceConnectStartData) context.Context {
		traceConnectStartCalled = true
		require.NotNil(t, data.ConnConfig)
		return context.WithValue(ctx, ctxKey("fromTraceConnectStart"), "foo")
	}

	traceConnectEndCalled := false
	tracer.traceConnectEnd = func(ctx context.Context, data fbx.TraceConnectEndData) {
		traceConnectEndCalled = true
		require.NotNil(t, data.Conn)
		require.NoError(t, data.Err)
	}

	conn1, err := fbx.ConnectConfig(context.Background(), config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn1.Close(context.Background()))
	}()
	require.True(t, traceConnectStartCalled)
	require.True(t, traceConnectEndCalled)

	config, err = fbx.ParseConfig("host=/invalid")
	require.NoError(t, err)
	config.Tracer = tracer

	traceConnectStartCalled = false
	traceConnectEndCalled = false
	tracer.traceConnectEnd = func(ctx context.Context, data fbx.TraceConnectEndData) {
		traceConnectEndCalled = true
		require.Nil(t, data.Conn)
		require.Error(t, data.Err)
	}

	conn2, err := fbx.ConnectConfig(context.Background(), config)
	require.Nil(t, conn2)
	require.Error(t, err)
	require.True(t, traceConnectStartCalled)
	require.True(t, traceConnectEndCalled)
}

// Ensure tracer runs within a transaction.
func TestTraceWithinTx(t *testing.T) {
	t.Parallel()

	tracer := &testTracer{}

	ctr := defaultConnTestRunner
	ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := defaultConnTestRunner.CreateConfig(ctx, t)
		config.Tracer = tracer
		return config
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		var queries []string
		tracer.traceQueryStart = func(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryStartData) context.Context {
			queries = append(queries, data.SQL)
			return ctx
		}

		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(ctx)
		}()
		_, err = tx.Exec(ctx, `select cast(? as varchar(255)) from rdb$database`, "testing")
		require.NoError(t, err)
		err = tx.Commit(ctx)
		require.NoError(t, err)

		require.Len(t, queries, 1)
		require.Equal(t, `select cast(? as varchar(255)) from rdb$database`, queries[0])
	})
}
