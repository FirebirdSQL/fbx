package fbconn

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestServiceContextConnectFailure(t *testing.T) {
	for _, stage := range []string{"lookup", "dial"} {
		for _, retry := range []bool{false, true} {
			for _, outcome := range []string{"active", "canceled", "deadline"} {
				t.Run(stage+"/"+outcome+"/retry="+strconv.FormatBool(retry), func(t *testing.T) {
					ctx, interrupt, want := serviceContextTestContext(t, outcome)
					original := &fbconnError{msg: "connection setup failed", safeToRetry: retry}
					config := serviceContextConfig(t, nil)
					if stage == "lookup" {
						config.ConnectionConfig.LookupFunc = func(context.Context, string) ([]string, error) {
							interrupt()
							return nil, original
						}
					} else {
						config.ConnectionConfig.DialFunc = func(context.Context, string, string) (net.Conn, error) {
							interrupt()
							return nil, original
						}
					}
					conn, err := ConnectServiceConfig(ctx, config)
					require.Nil(t, conn)
					require.ErrorIs(t, err, original)
					var connectErr *ConnectError
					require.ErrorAs(t, err, &connectErr)
					requireServiceContextCause(t, err, want)
					require.Equal(t, retry, SafeToRetry(err))
				})
			}
		}
	}
}

func TestServiceContextConnectTimeout(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run("retry="+strconv.FormatBool(retry), func(t *testing.T) {
			config := serviceContextConfig(t, nil)
			config.ConnectionConfig.ConnectTimeout = 10 * time.Millisecond
			original := &fbconnError{msg: "dial failed after timeout", safeToRetry: retry}
			config.ConnectionConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
				<-ctx.Done()
				return nil, original
			}
			conn, err := ConnectServiceConfig(context.Background(), config)
			require.Nil(t, conn)
			require.ErrorIs(t, err, original)
			requireServiceContextCause(t, err, context.DeadlineExceeded)
			require.Equal(t, retry, SafeToRetry(err))
		})
	}
}

func TestServiceContextAttachFailure(t *testing.T) {
	for _, outcome := range []string{"active", "canceled", "deadline", "connect timeout"} {
		t.Run(outcome, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = client.Close() })
			t.Cleanup(func() { _ = server.Close() })
			require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))
			require.NoError(t, server.SetDeadline(time.Now().Add(5*time.Second)))
			config := serviceContextConfig(t, client)
			ctx, interrupt, want := serviceContextTestContext(t, outcome)
			if outcome == "connect timeout" {
				ctx = context.Background()
				want = context.DeadlineExceeded
				config.ConnectionConfig.ConnectTimeout = 100 * time.Millisecond
				dialContext := make(chan context.Context, 1)
				config.ConnectionConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
					dialContext <- ctx
					return client, nil
				}
				interrupt = func() { <-((<-dialContext).Done()) }
			}
			done := make(chan error, 1)
			go func() {
				defer func() { _ = server.Close() }()
				done <- runServiceContextAttachProbe(server, interrupt)
			}()
			conn, err := ConnectServiceConfig(ctx, config)
			require.Nil(t, conn)
			require.NoError(t, <-done)
			var serverErr *FbError
			require.ErrorAs(t, err, &serverErr)
			require.Contains(t, serverErr.GdsCodes, fbproto.Isc_att_shutdown)
			requireServiceContextCause(t, err, want)
			require.False(t, SafeToRetry(err))
		})
	}
}

func runServiceContextAttachProbe(conn net.Conn, interrupt func()) error {
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
	interrupt()
	return writeServiceContextShutdown(w)
}

func TestServiceContextCloseFailure(t *testing.T) {
	for _, outcome := range []string{"active", "canceled", "deadline"} {
		t.Run(outcome, func(t *testing.T) {
			conn, done := newServiceContextProbe(t, "before start", nil)
			ctx, interrupt, want := serviceContextTestContext(t, outcome)
			original := errors.New("close failed")
			conn.conn.conn = &serviceContextCloseConn{Conn: conn.conn.conn, interrupt: interrupt, err: original}
			err := conn.Close(ctx)
			require.NoError(t, <-done)
			require.ErrorIs(t, err, original)
			requireServiceContextCause(t, err, want)
			require.False(t, SafeToRetry(err))
		})
	}
}

type serviceContextCloseConn struct {
	net.Conn
	interrupt func()
	err       error
}

func (c *serviceContextCloseConn) Close() error {
	err := c.Conn.Close()
	c.interrupt()
	return errors.Join(err, c.err)
}
