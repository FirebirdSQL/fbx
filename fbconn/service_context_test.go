package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn/ctxwatch"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func serviceContextCalls() map[string]func(context.Context, *ServiceConn, io.Writer) error {
	calls := maintenanceOutputCalls()
	calls["backup"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.Backup(ctx, "db", "backup", BackupOptions{Output: output})
	}
	calls["restore"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.Restore(ctx, "backup", "db", RestoreOptions{Output: output})
	}
	calls["statistics"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.Statistics(ctx, "db", StatisticsOptions{Output: output})
	}
	calls["limbo"] = func(ctx context.Context, c *ServiceConn, _ io.Writer) error {
		_, err := c.LimboTransactions(ctx, "db")
		return err
	}
	calls["limbo details"] = func(ctx context.Context, c *ServiceConn, _ io.Writer) error {
		_, err := c.LimboTransactionDetails(ctx, "db")
		return err
	}
	calls["trace start"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.TraceStart(ctx, "name", "config", TraceOptions{Output: output})
	}
	calls["trace stop"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.TraceStop(ctx, 42, TraceOptions{Output: output})
	}
	calls["trace suspend"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.TraceSuspend(ctx, 42, TraceOptions{Output: output})
	}
	calls["trace resume"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.TraceResume(ctx, 42, TraceOptions{Output: output})
	}
	calls["trace list"] = func(ctx context.Context, c *ServiceConn, output io.Writer) error {
		return c.TraceList(ctx, TraceOptions{Output: output})
	}
	return calls
}

func TestServiceContextServerError(t *testing.T) {
	for name, call := range serviceContextCalls() {
		for _, phase := range []string{"start", "output"} {
			for _, outcome := range []string{"canceled", "deadline", "active"} {
				t.Run(name+"/"+phase+"/"+outcome, func(t *testing.T) {
					ctx, interrupt, want := serviceContextTestContext(t, outcome)
					conn, done := newServiceContextProbe(t, phase, interrupt)
					err := call(ctx, conn, io.Discard)
					require.NoError(t, conn.Close(context.Background()))
					require.NoError(t, <-done)
					var serverErr *FbError
					require.ErrorAs(t, err, &serverErr)
					require.Contains(t, serverErr.GdsCodes, fbproto.Isc_att_shutdown)
					require.Equal(t, "08003", serverErr.Code)
					requireServiceContextCause(t, err, want)
					require.False(t, SafeToRetry(err))
				})
			}
		}
	}
}

func TestServiceContextAfterStartDisallowsRetry(t *testing.T) {
	for name, call := range serviceContextCalls() {
		for _, outcome := range []string{"canceled", "deadline"} {
			t.Run(name+"/"+outcome, func(t *testing.T) {
				ctx, interrupt, want := serviceContextTestContext(t, outcome)
				conn, done := newServiceContextProbe(t, "after start", nil)
				// Unwatch runs after the successful start response and before the
				// first output query. No request is in flight when cancellation occurs.
				conn.conn.contextWatcher = ctxwatch.NewContextWatcher(&serviceContextHandler{unwatch: interrupt})
				err := call(ctx, conn, io.Discard)
				require.NoError(t, conn.Close(context.Background()))
				require.NoError(t, <-done)
				requireServiceContextCause(t, err, want)
				require.False(t, SafeToRetry(err), "the server has already accepted the action")
			})
		}
	}
}

func TestServiceContextServerErrorAfterUnwatch(t *testing.T) {
	for _, phase := range []string{"start", "output"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conn, done := newServiceContextProbe(t, phase, func() {})
			// The context is canceled while the request's watcher is being
			// unwatched, after the server error has already been read.
			conn.conn.contextWatcher = ctxwatch.NewContextWatcher(&serviceContextHandler{unwatch: cancel})
			if phase == "output" {
				// Leave the start request's watcher lifecycle unchanged.
				require.NoError(t, conn.startServiceAction(context.Background(), []byte{fbproto.Isc_action_svc_backup}))
			}
			var err error
			if phase == "start" {
				err = conn.Backup(ctx, "db", "backup", BackupOptions{})
			} else {
				err = conn.copyServiceOutput(ctx, io.Discard)
			}
			require.NoError(t, conn.Close(context.Background()))
			require.NoError(t, <-done)
			var serverErr *FbError
			require.ErrorAs(t, err, &serverErr)
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestServiceContextBeforeStartKeepsRetry(t *testing.T) {
	for name, call := range serviceContextCalls() {
		t.Run(name, func(t *testing.T) {
			conn, done := newServiceContextProbe(t, "before start", nil)
			ctx, interrupt, want := serviceContextTestContext(t, "canceled")
			interrupt()
			err := call(ctx, conn, io.Discard)
			require.NoError(t, conn.Close(context.Background()))
			require.NoError(t, <-done)
			require.ErrorIs(t, err, want)
			require.True(t, SafeToRetry(err), "no action has been sent to the server")
		})
	}
}

func TestServiceContextSuccessfulResponse(t *testing.T) {
	for name, call := range serviceContextCalls() {
		t.Run(name, func(t *testing.T) {
			ctx, interrupt, _ := serviceContextTestContext(t, "canceled")
			conn, done := newServiceContextProbe(t, "complete", interrupt)
			require.NoError(t, call(ctx, conn, io.Discard))
			require.ErrorIs(t, ctx.Err(), context.Canceled)
			require.NoError(t, conn.Close(context.Background()))
			require.NoError(t, <-done)
		})
	}
}

func TestServiceContextWriterFailureDisallowsRetry(t *testing.T) {
	calls := serviceContextCalls()
	delete(calls, "limbo")
	delete(calls, "limbo details")
	for name, call := range calls {
		for _, outcome := range []string{"active", "canceled"} {
			t.Run(name+"/"+outcome, func(t *testing.T) {
				ctx, interrupt, want := serviceContextTestContext(t, outcome)
				conn, done := newServiceContextProbe(t, "writer", nil)
				writerErr := &fbconnError{msg: "writer failed", safeToRetry: true}
				require.True(t, SafeToRetry(writerErr))
				err := call(ctx, conn, serviceContextWriter{interrupt: interrupt, err: writerErr})
				require.NoError(t, conn.Close(context.Background()))
				require.NoError(t, <-done)
				require.ErrorIs(t, err, writerErr)
				requireServiceContextCause(t, err, want)
				require.False(t, SafeToRetry(err))
			})
		}
	}
}

type serviceContextWriter struct {
	interrupt func()
	err       error
}

func (w serviceContextWriter) Write([]byte) (int, error) {
	w.interrupt()
	return 0, w.err
}

func serviceContextTestContext(t *testing.T, outcome string) (context.Context, func(), error) {
	t.Helper()
	if outcome == "deadline" {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		t.Cleanup(cancel)
		return ctx, func() { <-ctx.Done() }, context.DeadlineExceeded
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if outcome == "active" {
		return ctx, func() {}, nil
	}
	return ctx, cancel, context.Canceled
}

func requireServiceContextCause(t *testing.T, err, want error) {
	t.Helper()
	if want == nil {
		require.NotErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, context.DeadlineExceeded)
	} else {
		require.ErrorIs(t, err, want)
	}
	switch want {
	case context.DeadlineExceeded:
		require.True(t, Timeout(err))
	case nil:
		require.False(t, Timeout(err))
	}
}

// Keep cancellation from changing socket deadlines so a real shutdown response
// can arrive after cancellation, just as it can with a delayed cancel handler.
type serviceContextHandler struct {
	unwatch func()
}

func (*serviceContextHandler) HandleCancel(context.Context) {}
func (*serviceContextHandler) HandleUnwatchAfterCancel()    {}
func (*serviceContextHandler) HandleWatch(context.Context)  {}
func (h *serviceContextHandler) HandleUnwatch() {
	if h.unwatch != nil {
		h.unwatch()
	}
}

func newServiceContextProbe(t *testing.T, phase string, interrupt func()) (*ServiceConn, <-chan error) {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = server.Close() })
	require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, server.SetDeadline(time.Now().Add(5*time.Second)))
	done := make(chan error, 1)
	go func() {
		defer func() { _ = server.Close() }()
		done <- runServiceContextProbe(server, phase, interrupt)
	}()
	conn, err := ConnectServiceConfig(context.Background(), serviceContextConfig(t, client))
	require.NoError(t, err)
	return conn, done
}

func serviceContextConfig(t *testing.T, client net.Conn) *ServiceConfig {
	t.Helper()
	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil }
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) { return client, nil }
	config.BuildContextWatcherHandler = func(*FbConn) ctxwatch.Handler { return &serviceContextHandler{} }
	return testServiceConfig(config)
}

func runServiceContextProbe(conn net.Conn, phase string, interrupt func()) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r, w := fbproto.NewXDRReader(conn), fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return err
	}
	if err := writeAuthenticatedProbeAccept(w, encoding); err != nil {
		return err
	}
	if err := readProbeServiceAttach(r, encoding); err != nil {
		return err
	}
	if err := writeProbeResponse(w, 42); err != nil {
		return err
	}
	if phase != "before start" {
		if _, err := readProbeServiceStart(r, 42); err != nil {
			return err
		}
		if phase != "start" {
			if err := writeProbeResponse(w, 0); err != nil {
				return err
			}
			if phase != "after start" {
				if _, _, _, err := readProbeServiceInfo(r, 42); err != nil {
					return err
				}
			}
		}
	}
	if interrupt != nil {
		interrupt()
		var err error
		if phase == "complete" {
			err = writeProbeResponseData(w, 0, []byte{fbproto.Isc_info_end})
		} else {
			err = writeServiceContextShutdown(w)
		}
		if err != nil {
			return err
		}
	}
	if phase == "writer" {
		if err := writeProbeResponseData(w, 0, serviceOutputResponse([]byte("output"))); err != nil {
			return err
		}
	}
	// An output failure can close the transport directly (limbo) or leave it
	// for the caller to detach. A new output query must never arrive here.
	opcode, err := r.ReadInt32()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	if opcode != fbproto.Op_service_detach {
		return fmt.Errorf("expected service detach after failure, got %d", opcode)
	}
	if _, err := r.ReadInt32(); err != nil {
		return err
	}
	opcode, err = r.ReadInt32()
	if err == nil && opcode != fbproto.Op_disconnect {
		return fmt.Errorf("expected disconnect, got %d", opcode)
	}
	return err
}

func writeServiceContextShutdown(w *fbproto.XDRWriter) error {
	if err := w.WriteInt32(fbproto.Op_response); err != nil {
		return err
	}
	if err := w.WriteInt32(0); err != nil {
		return err
	}
	if err := w.WriteInt64(0); err != nil {
		return err
	}
	if err := w.WriteBuffer(nil); err != nil {
		return err
	}
	for _, value := range []int{fbproto.Isc_arg_gds, fbproto.Isc_att_shutdown, fbproto.Isc_arg_end} {
		if err := w.WriteInt32(value); err != nil {
			return err
		}
	}
	return nil
}
