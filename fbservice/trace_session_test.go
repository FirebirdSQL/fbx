package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestTraceSessionFragmentedStartup(t *testing.T) {
	const header = "Trace session ID 2147483647 started\r\n"
	// A service response can split the startup line anywhere, including in the
	// ID or CRLF, and can include events in the same response as the header.
	for split := 0; split <= len(header); split++ {
		t.Run(fmt.Sprint(split), func(t *testing.T) {
			manager, conn := newTestManager(t)
			var output bytes.Buffer
			events := strings.Repeat("event\n", 1000)
			conn.run = func(context.Context) error {
				if _, err := io.WriteString(conn.output, header[:split]); err != nil {
					return err
				}
				_, err := io.WriteString(conn.output, header[split:]+events)
				return err
			}
			session, err := manager.TraceStartSession(context.Background(), "", "config", fbservice.TraceOptions{Output: &output})
			require.NoError(t, err)
			require.Equal(t, 2147483647, session.ID())
			require.NoError(t, session.Wait())
			require.NoError(t, session.Wait())
			require.True(t, conn.closed)
			require.Equal(t, header+events, output.String())
		})
	}
}

func TestTraceSessionRejectsMissingOrInvalidStartup(t *testing.T) {
	for name, output := range map[string]string{
		"empty": "", "incomplete": "Trace session ID 42 started",
		"diagnostic": "Can not start trace session. There are no trace plugins loaded\n",
		"zero":       "Trace session ID 0 started\n", "negative": "Trace session ID -1 started\n",
		"overflow": "Trace session ID 2147483648 started\n", "signed": "Trace session ID +42 started\n",
		"wrong message": "Trace session ID 42 stopped\n", "long line": strings.Repeat("x", 4097),
		"later ID": "startup failed\nTrace session ID 42 started\n",
	} {
		t.Run(name, func(t *testing.T) {
			manager, conn := newTestManager(t)
			var received bytes.Buffer
			conn.run = func(context.Context) error {
				_, err := io.WriteString(conn.output, output)
				return err
			}
			session, err := manager.TraceStartSession(context.Background(), "name", "config", fbservice.TraceOptions{Output: &received})
			require.Error(t, err)
			require.Nil(t, session)
			require.True(t, conn.closed)
			require.Equal(t, output, received.String())
		})
	}
}

func TestTraceSessionValidationBeforeConnect(t *testing.T) {
	for _, test := range []struct {
		name, configuration string
		options             []fbservice.TraceOptions
	}{
		{"missing output", "config", nil},
		{"nil output", "config", []fbservice.TraceOptions{{}}},
		{"multiple options", "config", []fbservice.TraceOptions{{Output: io.Discard}, {}}},
		{"empty config", "", []fbservice.TraceOptions{{Output: io.Discard}}},
		{"oversized config", strings.Repeat("c", 65532), []fbservice.TraceOptions{{Output: io.Discard}}},
		{"NUL\x00name", "config", []fbservice.TraceOptions{{Output: io.Discard}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager, conn := newTestManager(t)
			session, err := manager.TraceStartSession(context.Background(), test.name, test.configuration, test.options...)
			require.Error(t, err)
			require.Nil(t, session)
			require.False(t, conn.opened)
		})
	}
	manager, err := fbservice.NewManager("host=primary,secondary user=sysdba")
	require.NoError(t, err)
	fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		t.Error("must not connect with fallback servers")
		return nil, errors.New("unexpected connection")
	})
	session, err := manager.TraceStartSession(context.Background(), "", "config", fbservice.TraceOptions{Output: io.Discard})
	require.ErrorContains(t, err, "single server")
	require.Nil(t, session)
}

type traceSessionTestWriter func([]byte) (int, error)

func (w traceSessionTestWriter) Write(p []byte) (int, error) { return w(p) }

func TestTraceSessionErrors(t *testing.T) {
	writeErr, readErr, closeErr := errors.New("write failed"), errors.New("read failed"), errors.New("close failed")
	for _, afterStart := range []bool{false, true} {
		for _, failure := range []string{"write", "short write", "read"} {
			t.Run(fmt.Sprintf("%s/afterStart=%t", failure, afterStart), func(t *testing.T) {
				manager, conn := newTestManager(t)
				conn.closeErr = closeErr
				writes := 0
				writer := traceSessionTestWriter(func(p []byte) (int, error) {
					writes++
					if afterStart && writes == 1 || failure == "read" {
						return len(p), nil
					}
					if failure == "short write" {
						return len(p) - 1, nil
					}
					return 0, writeErr
				})
				conn.run = func(context.Context) error {
					if afterStart {
						if _, err := io.WriteString(conn.output, "Trace session ID 42 started\n"); err != nil {
							return err
						}
					}
					if failure == "read" {
						return readErr
					}
					_, err := io.WriteString(conn.output, "Trace session ID 42 started\n")
					return err
				}
				session, err := manager.TraceStartSession(context.Background(), "name", "config", fbservice.TraceOptions{Output: writer})
				if afterStart {
					require.NoError(t, err)
					err = session.Wait()
				} else {
					require.Nil(t, session)
				}
				require.ErrorIs(t, err, map[string]error{"write": writeErr, "short write": io.ErrShortWrite, "read": readErr}[failure])
				require.ErrorIs(t, err, closeErr)
				require.True(t, conn.closed)
			})
		}
	}
	manager, _ := newTestManager(t)
	fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		return nil, readErr
	})
	session, err := manager.TraceStartSession(context.Background(), "name", "config", fbservice.TraceOptions{Output: io.Discard})
	require.ErrorIs(t, err, readErr)
	require.Nil(t, session)
}

func TestTraceSessionCancellationBeforeID(t *testing.T) {
	for _, beforeConnect := range []bool{false, true} {
		t.Run(fmt.Sprint(beforeConnect), func(t *testing.T) {
			manager, conn := newTestManager(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if beforeConnect {
				cancel()
			}
			conn.run = func(ctx context.Context) error {
				cancel()
				<-ctx.Done()
				return ctx.Err()
			}
			session, err := manager.TraceStartSession(ctx, "name", "config", fbservice.TraceOptions{Output: io.Discard})
			require.Nil(t, session)
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, !beforeConnect, conn.closed)
		})
	}
}

func TestTraceSessionsWithSameName(t *testing.T) {
	manager, _ := newTestManager(t)
	connections := make(chan fbservice.ServiceConnection, 2)
	for _, id := range []int{42, 43} {
		conn := &fakeServiceConnection{}
		conn.run = func(ctx context.Context) error {
			if _, err := fmt.Fprintf(conn.output, "Trace session ID %d started\n", id); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		}
		connections <- conn
	}
	fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		return <-connections, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a, err := manager.TraceStartSession(ctx, "same name", "config", fbservice.TraceOptions{Output: io.Discard})
	require.NoError(t, err)
	defer a.Cancel()
	b, err := manager.TraceStartSession(ctx, "same name", "config", fbservice.TraceOptions{Output: io.Discard})
	require.NoError(t, err)
	defer b.Cancel()
	require.Equal(t, 42, a.ID())
	require.Equal(t, 43, b.ID())
	waiters := make(chan error, 5)
	for range 5 {
		go func() {
			a.Cancel()
			waiters <- a.Wait()
		}()
	}
	for range 5 {
		require.ErrorIs(t, <-waiters, context.Canceled)
	}
	select {
	case <-b.Done():
		t.Fatal("canceling one session must not cancel another with the same name")
	default:
	}
	cancel()
	require.ErrorIs(t, b.Wait(), context.Canceled)
}
