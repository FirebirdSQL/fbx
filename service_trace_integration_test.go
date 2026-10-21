package fbx_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

type traceIntegrationOutput struct {
	mu     sync.Mutex
	buffer bytes.Buffer
	err    error
}

func (w *traceIntegrationOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	return w.buffer.Write(p)
}

func (w *traceIntegrationOutput) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

func TestServiceTraceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("service trace integration test is disabled in short mode")
	}
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("service trace integration test requires FBX_TEST_DATABASE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	conn, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		require.NoError(t, conn.Close(cleanupCtx))
	}()
	t.Logf("trace server version: %s", conn.ServerVersion())
	var attachmentID int64
	require.NoError(t, conn.QueryRow(ctx, "select current_connection from rdb$database").Scan(&attachmentID))
	settings := fmt.Sprintf("enabled = true\nconnection_id = %d\nlog_statement_finish = true\ntime_threshold = 0\n", attachmentID)
	configuration := "database {\n" + settings + "}\n"
	if conn.ServerVersion().Major < 3 {
		configuration = "<database>\n" + settings + "</database>\n"
	}
	manager, err := fbservice.NewManager(connString)
	require.NoError(t, err)
	writerErr := errors.New("trace output failed")
	for _, test := range []struct {
		termination string
		session     bool
	}{
		{"stop", false}, {"cancel", false}, {"writer error", false},
		{"stop", true}, {"cancel", true}, {"writer error", true},
	} {
		t.Run(fmt.Sprintf("%s/session=%t", test.termination, test.session), func(t *testing.T) {
			termination := test.termination
			traceCtx, traceCancel := context.WithCancel(ctx)
			output := new(traceIntegrationOutput)
			done := make(chan error, 1)
			finished := false
			t.Cleanup(func() {
				traceCancel()
				if !finished {
					select {
					case err := <-done:
						t.Logf("trace cleanup: %v; output: %s", err, output.String())
					case <-time.After(10 * time.Second):
						t.Error("trace did not finish during cleanup")
					}
				}
			})
			name := fmt.Sprintf("fbx trace %s %d", termination, time.Now().UnixNano())
			traceConfiguration := configuration
			if termination == "stop" {
				// Exercise the largest start SPB accepted by the server. Padding
				// stays inside a configuration comment and does not change tracing.
				traceConfiguration += "#" + strings.Repeat(" ", 65535-7-len(name)-len(configuration)-2) + "\n"
			}
			var sessionID int
			var session *fbservice.TraceSession
			if test.session {
				session, err = manager.TraceStartSession(traceCtx, name, traceConfiguration, fbservice.TraceOptions{Output: output})
				if err != nil {
					finished = true
				}
				require.NoError(t, err)
				sessionID = session.ID()
				require.Positive(t, sessionID)
				go func() { done <- session.Wait() }()
			} else {
				go func() {
					done <- manager.TraceStart(traceCtx, name, traceConfiguration, fbservice.TraceOptions{Output: output})
				}()
				started := false
				require.Eventually(t, func() bool {
					select {
					case err := <-done:
						finished = true
						t.Errorf("trace returned before reporting its ID: %v; output: %s", err, output.String())
						return true
					default:
					}
					line, _, _ := strings.Cut(output.String(), "\n")
					_, err := fmt.Sscanf(line, "Trace session ID %d started", &sessionID)
					started = err == nil && sessionID > 0
					return err == nil && sessionID > 0
				}, 10*time.Second, 50*time.Millisecond)
				require.True(t, started, "trace start output: %s", output.String())
			}
			list := func() string {
				t.Helper()
				var report bytes.Buffer
				require.NoError(t, manager.TraceList(ctx, fbservice.TraceOptions{Output: &report}))
				for _, block := range strings.Split(report.String(), "Session ID: ") {
					if strings.HasPrefix(block, strconv.Itoa(sessionID)+"\n") || strings.HasPrefix(block, strconv.Itoa(sessionID)+"\r\n") {
						return block
					}
				}
				return ""
			}
			report := list()
			require.Contains(t, report, name)
			require.Contains(t, report, "active")
			query := func(marker string) {
				t.Helper()
				var value int
				require.NoError(t, conn.QueryRow(ctx, "select 1 /* "+marker+" */ from rdb$database").Scan(&value))
				require.Equal(t, 1, value)
			}
			query("fbx_trace_before_pause")
			require.Eventually(t, func() bool { return strings.Contains(output.String(), "fbx_trace_before_pause") }, 10*time.Second, 50*time.Millisecond)
			var control bytes.Buffer
			options := fbservice.TraceOptions{Output: &control}
			require.NoError(t, manager.TraceSuspend(ctx, sessionID, options))
			require.Contains(t, control.String(), "paused")
			require.Contains(t, list(), "suspend")
			query("fbx_trace_during_pause")
			control.Reset()
			require.NoError(t, manager.TraceResume(ctx, sessionID, options))
			require.Contains(t, control.String(), "resumed")
			require.Contains(t, list(), "active")
			query("fbx_trace_after_resume")
			require.Eventually(t, func() bool { return strings.Contains(output.String(), "fbx_trace_after_resume") }, 10*time.Second, 50*time.Millisecond)
			require.NotContains(t, output.String(), "fbx_trace_during_pause")
			switch termination {
			case "stop":
				control.Reset()
				require.NoError(t, manager.TraceStop(ctx, sessionID, options))
				require.Contains(t, control.String(), "stopped")
			case "cancel":
				if session != nil {
					session.Cancel()
				} else {
					traceCancel()
				}
			case "writer error":
				output.mu.Lock()
				output.err = writerErr
				output.mu.Unlock()
				query("fbx_trace_writer_failure")
			}
			select {
			case err := <-done:
				finished = true
				switch termination {
				case "stop":
					require.NoError(t, err)
				case "cancel":
					require.ErrorIs(t, err, context.Canceled)
				case "writer error":
					require.ErrorIs(t, err, writerErr)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("trace did not stop")
			}
			require.Eventually(t, func() bool { return list() == "" }, 10*time.Second, 50*time.Millisecond)
		})
	}
}
