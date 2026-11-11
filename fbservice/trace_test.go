package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

type fakeTraceConnection struct {
	action, name, configuration string
	sessionID                   int
	output                      io.Writer
	run                         func(context.Context) error
}

func (c *fakeTraceConnection) recordTrace(ctx context.Context, action string, options fbconn.TraceOptions) error {
	c.action, c.output = action, options.Output
	if c.run != nil {
		return c.run(ctx)
	}
	return nil
}

func (c *fakeTraceConnection) TraceStart(ctx context.Context, name, configuration string, options fbconn.TraceOptions) error {
	c.name, c.configuration = name, configuration
	return c.recordTrace(ctx, "start", options)
}

func (c *fakeTraceConnection) TraceStop(ctx context.Context, sessionID int, options fbconn.TraceOptions) error {
	c.sessionID = sessionID
	return c.recordTrace(ctx, "stop", options)
}

func (c *fakeTraceConnection) TraceSuspend(ctx context.Context, sessionID int, options fbconn.TraceOptions) error {
	c.sessionID = sessionID
	return c.recordTrace(ctx, "suspend", options)
}

func (c *fakeTraceConnection) TraceResume(ctx context.Context, sessionID int, options fbconn.TraceOptions) error {
	c.sessionID = sessionID
	return c.recordTrace(ctx, "resume", options)
}

func (c *fakeTraceConnection) TraceList(ctx context.Context, options fbconn.TraceOptions) error {
	return c.recordTrace(ctx, "list", options)
}

func traceOperations(m *fbservice.Manager) map[string]func(context.Context, ...fbservice.TraceOptions) error {
	return map[string]func(context.Context, ...fbservice.TraceOptions) error{
		"start": func(ctx context.Context, options ...fbservice.TraceOptions) error {
			return m.TraceStart(ctx, "my trace", "database { enabled = true }", options...)
		},
		"stop": func(ctx context.Context, options ...fbservice.TraceOptions) error {
			return m.TraceStop(ctx, 42, options...)
		},
		"suspend": func(ctx context.Context, options ...fbservice.TraceOptions) error {
			return m.TraceSuspend(ctx, 42, options...)
		},
		"resume": func(ctx context.Context, options ...fbservice.TraceOptions) error {
			return m.TraceResume(ctx, 42, options...)
		},
		"list": m.TraceList,
	}
}

func TestManagerTraceOperations(t *testing.T) {
	for action := range traceOperations(nil) {
		t.Run(action, func(t *testing.T) {
			manager, conn := newTestManager(t)
			var output bytes.Buffer
			require.NoError(t, traceOperations(manager)[action](context.Background(), fbservice.TraceOptions{Output: &output}))
			require.Equal(t, action, conn.action)
			require.Same(t, &output, conn.output)
			require.Empty(t, conn.expectedDatabase)
			require.True(t, conn.closed)
			if action == "start" {
				require.Equal(t, "my trace", conn.name)
				require.Equal(t, "database { enabled = true }", conn.configuration)
			} else if action != "list" {
				require.Equal(t, 42, conn.sessionID)
			}
		})
	}
}

func TestManagerTraceOptions(t *testing.T) {
	for action := range traceOperations(nil) {
		for name, options := range map[string][]fbservice.TraceOptions{
			"omitted": nil, "empty": {{}}, "discard": {{Output: io.Discard}}, "multiple": {{Output: io.Discard}, {}},
		} {
			t.Run(action+"/"+name, func(t *testing.T) {
				manager, conn := newTestManager(t)
				err := traceOperations(manager)[action](context.Background(), options...)
				if name == "multiple" || (action == "start" || action == "list") && name != "discard" {
					require.Error(t, err)
					require.False(t, conn.opened)
				} else {
					require.NoError(t, err)
					require.True(t, conn.closed)
				}
			})
		}
	}
}

func TestManagerTraceValidationBeforeConnect(t *testing.T) {
	for _, test := range []struct{ name, configuration string }{
		{"trace", ""}, {"bad\x00name", "config"}, {"trace", "bad\x00config"},
		{strings.Repeat("n", 65536), "config"}, {"trace", strings.Repeat("c", 65536)},
	} {
		manager, conn := newTestManager(t)
		require.Error(t, manager.TraceStart(context.Background(), test.name, test.configuration, fbservice.TraceOptions{Output: io.Discard}))
		require.False(t, conn.opened)
	}
	ids := []int{-1, 0}
	if strconv.IntSize > 32 {
		overflow := int64(1) << 31
		ids = append(ids, int(overflow))
	}
	for _, id := range ids {
		manager, conn := newTestManager(t)
		for _, run := range []func(context.Context, int, ...fbservice.TraceOptions) error{manager.TraceStop, manager.TraceSuspend, manager.TraceResume} {
			require.ErrorContains(t, run(context.Background(), id), "session ID")
			require.False(t, conn.opened)
		}
	}
	manager, conn := newTestManager(t)
	require.NoError(t, manager.TraceStart(context.Background(), "", "config", fbservice.TraceOptions{Output: io.Discard}))
	require.True(t, conn.closed)
}

func TestManagerTraceCancellationAndErrors(t *testing.T) {
	operationErr, closeErr, connectErr := errors.New("operation failed"), errors.New("close failed"), errors.New("connect failed")
	for action := range traceOperations(nil) {
		t.Run(action, func(t *testing.T) {
			manager, conn := newTestManager(t)
			run := traceOperations(manager)[action]
			options := fbservice.TraceOptions{Output: io.Discard}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.ErrorIs(t, run(ctx, options), context.Canceled)
			require.False(t, conn.opened)
			conn.run = func(context.Context) error { return operationErr }
			conn.closeErr = closeErr
			err := run(context.Background(), options)
			require.ErrorIs(t, err, operationErr)
			require.ErrorIs(t, err, closeErr)
			require.True(t, conn.closed)
			fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
				return nil, connectErr
			})
			require.ErrorIs(t, run(context.Background(), options), connectErr)
		})
	}
}

func TestManagerTraceStartSPBTotalLength(t *testing.T) {
	for _, test := range []struct {
		label, name, configuration string
		valid                      bool
	}{
		{"unnamed limit", "", strings.Repeat("c", 65531), true},
		{"unnamed overflow", "", strings.Repeat("c", 65532), false},
		{"maximum string", "", strings.Repeat("c", 65535), false},
		{"named limit", "n", strings.Repeat("c", 65527), true},
		{"named overflow", "n", strings.Repeat("c", 65528), false},
		{"long name limit", strings.Repeat("n", 65527), "c", true},
		{"long name overflow", strings.Repeat("n", 65528), "c", false},
		{"combined limit", strings.Repeat("n", 32764), strings.Repeat("c", 32764), true},
		{"combined overflow", strings.Repeat("n", 32765), strings.Repeat("c", 32764), false},
		{"UTF8 name limit", "т", strings.Repeat("c", 65526), true},
		{"UTF8 name overflow", "т", strings.Repeat("c", 65527), false},
		{"UTF8 configuration limit", "", strings.Repeat("т", 32765) + "c", true},
		{"UTF8 configuration overflow", "", strings.Repeat("т", 32766), false},
	} {
		t.Run(test.label, func(t *testing.T) {
			manager, conn := newTestManager(t)
			err := manager.TraceStart(context.Background(), test.name, test.configuration, fbservice.TraceOptions{Output: io.Discard})
			if test.valid {
				require.NoError(t, err)
				require.Equal(t, test.name, conn.name)
				require.Equal(t, test.configuration, conn.configuration)
				require.True(t, conn.closed)
			} else {
				require.ErrorContains(t, err, "trace start SPB")
				require.False(t, conn.opened)
				require.False(t, conn.closed)
			}
		})
	}
}

func TestManagerTraceRejectsMultipleServersAfterRecovery(t *testing.T) {
	for label, connString := range map[string]string{
		"keyword hosts":   "host=primary,secondary port=3050 user=sysdba password=masterkey",
		"URL hosts":       "firebird://sysdba:masterkey@primary:3050,secondary:3050/ignored",
		"different ports": "host=localhost,localhost port=3050,3051 user=sysdba password=masterkey",
	} {
		t.Run(label, func(t *testing.T) {
			manager, err := fbservice.NewManager(connString)
			require.NoError(t, err)
			primaryAvailable := false
			var selectedHosts []string
			fbservice.SetConnectForTest(manager, func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
				server := config.ConnectionConfig
				require.Len(t, server.Fallbacks, 1)
				host, port := server.Host, server.Port
				if !primaryAvailable {
					host, port = server.Fallbacks[0].Host, server.Fallbacks[0].Port
				}
				selectedHosts = append(selectedHosts, host+":"+strconv.Itoa(int(port)))
				return &fakeServiceConnection{}, nil
			})
			options := fbservice.TraceOptions{Output: io.Discard}
			ctx := context.Background()
			results := map[string]error{"start": manager.TraceStart(ctx, "trace", "config", options)}
			// If fallback were allowed, the start would choose the secondary,
			// then these operations would affect the recovered primary instead.
			primaryAvailable = true
			for _, action := range []string{"list", "stop", "suspend", "resume"} {
				results[action] = traceOperations(manager)[action](ctx, options)
			}
			for action, err := range results {
				require.ErrorContains(t, err, "trace operations require a single server", action)
			}
			require.Empty(t, selectedHosts, "trace must reject fallbacks before connecting")
			// Rejecting trace must not strip fallbacks from other operations.
			require.NoError(t, manager.Backup(ctx, "database", "backup"))
			require.Len(t, selectedHosts, 1)
		})
	}
}

type traceCleanupConnection struct {
	fakeServiceConnection
	closeContextErr error
	closeDeadline   bool
}

func (c *traceCleanupConnection) Close(ctx context.Context) error {
	c.closeContextErr = ctx.Err()
	_, c.closeDeadline = ctx.Deadline()
	return c.fakeServiceConnection.Close(ctx)
}

func TestManagerTraceControlWhileStreaming(t *testing.T) {
	manager, err := fbservice.NewManager("firebird://sysdba:masterkey@localhost/ignored")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := make(chan struct{})
	stream := &traceCleanupConnection{}
	stream.run = func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	// All connections are opened in a known order: start, then each control.
	connections := make(chan fbservice.ServiceConnection, 5)
	connections <- stream
	for range 4 {
		connections <- &fakeServiceConnection{}
	}
	close(connections)
	fbservice.SetConnectForTest(manager, func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		if config.ExpectedDatabase != "" {
			return nil, errors.New("trace unexpectedly targets a database")
		}
		return <-connections, nil
	})
	done := make(chan error, 1)
	go func() { done <- manager.TraceStart(ctx, "trace", "config", fbservice.TraceOptions{Output: io.Discard}) }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	for _, action := range []string{"list", "suspend", "resume", "stop"} {
		require.NoError(t, traceOperations(manager)[action](ctx, fbservice.TraceOptions{Output: io.Discard}))
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("trace stream did not stop after cancellation")
	}
	require.True(t, stream.closed)
	require.NoError(t, stream.closeContextErr)
	require.True(t, stream.closeDeadline)
}
