package fbconn

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestStartServiceActionRejectsActiveAction(t *testing.T) {
	conn := &ServiceConn{
		conn:         &FbConn{},
		actionActive: true,
	}

	err := conn.startServiceAction(context.Background(), []byte{fbproto.Isc_action_svc_backup})

	require.ErrorIs(t, err, errServiceActionActive)
}

func TestRequestServiceClosesConnectionOnUnexpectedResponse(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		serverDone <- runUnexpectedServiceResponseProbeServer(serverConn)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := ConnectServiceConfig(ctx, testServiceConfig(config))
	require.NoError(t, err)

	err = conn.startServiceAction(context.Background(), []byte{fbproto.Isc_action_svc_backup})

	require.NoError(t, <-serverDone)
	require.ErrorContains(t, err, "received unexpected service response")
	require.True(t, conn.conn.IsClosed())
	select {
	case <-conn.conn.CleanupDone():
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func runUnexpectedServiceResponseProbeServer(conn net.Conn) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
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
	if _, err := readProbeServiceStart(r, 42); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.Op_sql_response); err != nil {
		return err
	}
	return w.WriteInt32(0)
}
