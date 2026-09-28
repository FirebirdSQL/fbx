package fbconn_test

import (
	"bytes"
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

type writeCountConn struct {
	net.Conn
	writes int
}

func (c *writeCountConn) Write(p []byte) (int, error) {
	c.writes++
	return c.Conn.Write(p)
}

func TestDeferredExecuteUsesFirstFetch(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}

	for _, compression := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "compressed"}[compression], func(t *testing.T) {
			ctx := context.Background()
			config, err := fbconn.ParseConfig(connString)
			require.NoError(t, err)
			config.PipelineExecuteFetch = true
			config.WireCompression = compression
			config.FetchBatchSize = 4
			var wire *writeCountConn
			config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
				raw, err := (&net.Dialer{}).DialContext(ctx, network, address)
				if err != nil {
					return nil, err
				}
				wire = &writeCountConn{Conn: raw}
				return wire, nil
			}
			conn, err := fbconn.ConnectConfig(ctx, config)
			require.NoError(t, err)
			defer func() { require.NoError(t, conn.Close(ctx)) }()
			tx, err := conn.StartTransaction(ctx)
			require.NoError(t, err)
			defer func() { require.NoError(t, tx.Rollback(ctx)) }()
			st, err := conn.PrepareTx(ctx, tx,
				"WITH RECURSIVE s(n) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT n+1 FROM s WHERE n < 9) SELECT n FROM s")
			require.NoError(t, err)
			defer func() { require.NoError(t, st.Deallocate(ctx)) }()

			var trace bytes.Buffer
			conn.Client().Trace(&trace, fbproto.TracerOptions{SuppressTimestamps: true})
			wire.writes = 0
			rr := st.Exec(ctx, nil)
			require.Zero(t, wire.writes, "deferred execute must remain buffered until the first fetch")
			require.Contains(t, trace.String(), "C\tExecute ")
			require.NotContains(t, trace.String(), "C\tFetch ")
			require.NotContains(t, trace.String(), "S\tResponse ")

			require.True(t, rr.NextRow())
			require.Equal(t, 1, wire.writes, "execute and first fetch must share a socket write")
			count := 1
			for rr.NextRow() {
				count++
			}
			_, err = rr.Close()
			require.NoError(t, err)
			require.Equal(t, 9, count)
			out := trace.String()
			execute := strings.Index(out, "C\tExecute ")
			fetch := strings.Index(out, "C\tFetch ")
			response := strings.Index(out, "S\tResponse ")
			require.True(t, execute >= 0 && fetch > execute && response > fetch, "trace: %s", out)
			require.Greater(t, strings.Count(out, "C\tFetch "), 1)

			// Closing before reading sends Execute alone and consumes its response.
			trace.Reset()
			wire.writes = 0
			rr = st.Exec(ctx, nil)
			require.Zero(t, wire.writes)
			_, err = rr.Close()
			require.NoError(t, err)
			require.Greater(t, wire.writes, 0)
			require.NotContains(t, trace.String(), "C\tFetch ")
			require.Contains(t, trace.String(), "S\tResponse ")

			result := st.Exec(ctx, nil).Read()
			require.NoError(t, result.Err)
			require.Len(t, result.Rows, 9)
			empty, err := conn.PrepareTx(ctx, tx, "SELECT 1 FROM RDB$DATABASE WHERE 1=0")
			require.NoError(t, err)
			result = empty.Exec(ctx, nil).Read()
			require.NoError(t, result.Err)
			require.Empty(t, result.Rows)
			require.NoError(t, empty.Deallocate(ctx))
		})
	}
}

func TestDeferredExecuteErrorKeepsConnectionInSync(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}
	ctx := context.Background()
	config, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)
	config.PipelineExecuteFetch = true
	var reportedErrors int
	config.OnFbError = func(*fbconn.FbConn, *fbconn.FbError) bool {
		reportedErrors++
		return true
	}
	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close(ctx)) }()
	tx, err := conn.StartTransaction(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, tx.Rollback(ctx)) }()
	st, err := conn.PrepareTx(ctx, tx, "SELECT 1 FROM RDB$DATABASE")
	require.NoError(t, err)
	defer func() { require.NoError(t, st.Deallocate(ctx)) }()

	invalidTx := &fbconn.FBTransaction{Connection: conn, Handle: 999999}
	rr := st.ExecTx(ctx, invalidTx, nil)
	require.False(t, rr.NextRow())
	_, err = rr.Close()
	require.Error(t, err)
	require.Equal(t, 1, reportedErrors, "the discarded fetch error must not reach OnFbError")
	result := st.ExecTx(ctx, tx, nil).Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, 1)
}

func TestDeferredExecuteErrorOnClose(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}
	ctx := context.Background()
	config, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)
	config.PipelineExecuteFetch = true
	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close(ctx)) }()
	tx, err := conn.StartTransaction(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, tx.Rollback(ctx)) }()
	st, err := conn.PrepareTx(ctx, tx, "SELECT 1 FROM RDB$DATABASE")
	require.NoError(t, err)
	defer func() { require.NoError(t, st.Deallocate(ctx)) }()

	invalidTx := &fbconn.FBTransaction{Connection: conn, Handle: 999999}
	_, err = st.ExecTx(ctx, invalidTx, nil).Close()
	require.Error(t, err)
	result := st.ExecTx(ctx, tx, nil).Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, 1)
}

func TestDeferredExecuteConfig(t *testing.T) {
	t.Setenv("FBPIPELINE_EXECUTE_FETCH", "")
	config, err := fbconn.ParseConfig("")
	require.NoError(t, err)
	require.True(t, config.PipelineExecuteFetch)

	config, err = fbconn.ParseConfig("pipeline_execute_fetch=true")
	require.NoError(t, err)
	require.True(t, config.PipelineExecuteFetch)
	require.NotContains(t, config.RuntimeParams, "pipeline_execute_fetch")

	config, err = fbconn.ParseConfig("pipeline_execute_fetch=false")
	require.NoError(t, err)
	require.False(t, config.PipelineExecuteFetch)

	t.Setenv("FBPIPELINE_EXECUTE_FETCH", "true")
	config, err = fbconn.ParseConfig("")
	require.NoError(t, err)
	require.True(t, config.PipelineExecuteFetch)
}

func TestDeferredExecuteDisabled(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}
	ctx := context.Background()
	config, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)
	config.PipelineExecuteFetch = false
	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close(ctx)) }()
	tx, err := conn.StartTransaction(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, tx.Rollback(ctx)) }()
	st, err := conn.PrepareTx(ctx, tx, "SELECT 1 FROM RDB$DATABASE")
	require.NoError(t, err)
	defer func() { require.NoError(t, st.Deallocate(ctx)) }()

	var trace bytes.Buffer
	conn.Client().Trace(&trace, fbproto.TracerOptions{SuppressTimestamps: true})
	rr := st.Exec(ctx, nil)
	out := trace.String()
	require.Contains(t, out, "C\tExecute ")
	require.Contains(t, out, "S\tResponse ")
	require.NotContains(t, out, "C\tFetch ")
	result := rr.Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, 1)
	out = trace.String()
	require.Contains(t, out, "C\tFetch ")
	require.Less(t, strings.Index(out, "S\tResponse "), strings.Index(out, "C\tFetch "))
}
