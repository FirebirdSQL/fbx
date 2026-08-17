package fbconn

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/auth"
)

func TestBuildServiceAttachSPB(t *testing.T) {
	config, err := ParseConfig("firebird://sysdba:masterkey@localhost/employee")
	require.NoError(t, err)
	config.Role = "RDB$ADMIN"
	config.ClientVersion = "fbx-test"

	authManager, err := auth.NewAuthManager(auth.PluginsConfig{
		User:     config.User,
		Password: config.Password,
	}, []string{"Legacy_Auth"})
	require.NoError(t, err)
	require.NotNil(t, authManager.BeginHandshake())

	spb := buildServiceAttachSPB(config, authManager)
	items := readServiceAttachSPB(t, spb)

	require.Equal(t, "sysdba", string(items[fbproto.Isc_spb_user_name]))
	require.Equal(t, "RDB$ADMIN", string(items[fbproto.Isc_spb_sql_role_name]))
	require.Equal(t, "fbx-test", string(items[fbproto.Isc_spb_client_version]))
	require.Equal(t, "employee", string(items[fbproto.Isc_spb_expected_db]))
	require.Equal(t, "Legacy_Auth", string(items[fbproto.Isc_spb_auth_plugin_list]))
	require.Equal(t, "Legacy_Auth", string(items[fbproto.Isc_spb_auth_plugin_name]))
	require.Contains(t, items, byte(fbproto.Isc_spb_utf8_filename))
}

func TestConnectServiceSendsServiceAttach(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	resultCh := make(chan serviceAttachProbeResult, 1)
	go func() {
		defer serverConn.Close()
		resultCh <- runServiceAttachProbeServer(serverConn)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = ConnectServiceConfig(ctx, testServiceConfig(config))
	require.Error(t, err)

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		require.Equal(t, defaultServiceName, result.serviceName)
		require.NotEmpty(t, result.spb)
		require.Equal(t, byte(fbproto.Isc_spb_version3), result.spb[0])
		require.Contains(t, readServiceAttachSPB(t, result.spb), byte(fbproto.Isc_spb_specific_auth_data))
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestConnectServiceAttachAndClose(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	resultCh := make(chan serviceLifecycleProbeResult, 1)
	go func() {
		defer serverConn.Close()
		resultCh <- runServiceLifecycleProbeServer(serverConn)
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
	require.NoError(t, conn.Close(context.Background()))

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		require.Equal(t, 42, result.detachedHandle)
		require.Equal(t, fbproto.Op_disconnect, result.finalOpcode)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestServiceConnCloseReturnsDetachError(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		serverDone <- runServiceAttachSuccessProbeServer(serverConn)
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
	require.NoError(t, <-serverDone)

	err = conn.Close(context.Background())

	require.Error(t, err)
	require.ErrorContains(t, err, "detach from service manager")
}

func TestConnectServiceConfigRejectsDatabaseOnlyOptions(t *testing.T) {
	tests := map[string]func(*Config){
		"ValidateConnect": func(config *Config) {
			config.ValidateConnect = func(context.Context, *FbConn) error { return nil }
		},
		"AfterConnect": func(config *Config) {
			config.AfterConnect = func(context.Context, *FbConn) error { return nil }
		},
		"CreateDatabase": func(config *Config) {
			config.CreateDatabase = true
		},
		"DPB": func(config *Config) {
			config.DPB = []byte{fbproto.Isc_dpb_version2}
		},
	}

	for name, configure := range tests {
		t.Run(name, func(t *testing.T) {
			config, err := ParseConfig("firebird://sysdba:masterkey@localhost/employee")
			require.NoError(t, err)
			config.ValidateConnect = nil
			config.AfterConnect = nil
			config.CreateDatabase = false
			config.DPB = nil
			configure(config)

			_, err = ConnectServiceConfig(context.Background(), testServiceConfig(config))

			require.Error(t, err)
		})
	}
}

func TestConnectServiceConfigClosesConnectionOnEncodingError(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	trackedConn := &closeTrackingConn{Conn: clientConn}

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.Encoding = "invalid-encoding"
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return trackedConn, nil
	}

	_, err = ConnectServiceConfig(context.Background(), testServiceConfig(config))

	require.ErrorContains(t, err, "encoding setup failed")
	require.True(t, trackedConn.closed)
}

func testServiceConfig(config *Config) *ServiceConfig {
	return &ServiceConfig{
		ConnectionConfig: config,
		ExpectedDatabase: config.Database,
	}
}

type closeTrackingConn struct {
	net.Conn
	closed bool
}

func (c *closeTrackingConn) Close() error {
	c.closed = true
	return c.Conn.Close()
}

type serviceAttachProbeResult struct {
	serviceName string
	spb         []byte
	err         error
}

type serviceLifecycleProbeResult struct {
	detachedHandle int
	finalOpcode    int
	err            error
}

func runServiceAttachProbeServer(conn net.Conn) serviceAttachProbeResult {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return serviceAttachProbeResult{err: err}
	}

	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return serviceAttachProbeResult{err: err}
	}
	if err := writeProbeAccept(w, encoding, fbproto.Op_accept_data); err != nil {
		return serviceAttachProbeResult{err: err}
	}

	opcode, err := r.ReadInt32()
	if err != nil {
		return serviceAttachProbeResult{err: err}
	}
	if opcode != fbproto.Op_service_attach {
		return serviceAttachProbeResult{err: fmt.Errorf("expected op_service_attach, got %d", opcode)}
	}
	if _, err := r.ReadInt32(); err != nil {
		return serviceAttachProbeResult{err: err}
	}
	serviceName, err := r.ReadString(encoding)
	if err != nil {
		return serviceAttachProbeResult{err: err}
	}
	spb, err := r.ReadBuffer()
	if err != nil {
		return serviceAttachProbeResult{err: err}
	}
	return serviceAttachProbeResult{serviceName: serviceName, spb: spb}
}

func runServiceLifecycleProbeServer(conn net.Conn) serviceLifecycleProbeResult {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return serviceLifecycleProbeResult{err: err}
	}

	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return serviceLifecycleProbeResult{err: err}
	}
	if err := writeAuthenticatedProbeAccept(w, encoding); err != nil {
		return serviceLifecycleProbeResult{err: err}
	}
	if err := readProbeServiceAttach(r, encoding); err != nil {
		return serviceLifecycleProbeResult{err: err}
	}
	if err := writeProbeResponse(w, 42); err != nil {
		return serviceLifecycleProbeResult{err: err}
	}

	opcode, err := r.ReadInt32()
	if err != nil {
		return serviceLifecycleProbeResult{err: err}
	}
	if opcode != fbproto.Op_service_detach {
		return serviceLifecycleProbeResult{err: fmt.Errorf("expected op_service_detach, got %d", opcode)}
	}
	handle, err := r.ReadInt32()
	if err != nil {
		return serviceLifecycleProbeResult{err: err}
	}
	finalOpcode, err := r.ReadInt32()
	if err != nil {
		return serviceLifecycleProbeResult{err: err}
	}
	return serviceLifecycleProbeResult{
		detachedHandle: handle,
		finalOpcode:    finalOpcode,
	}
}

func runServiceAttachSuccessProbeServer(conn net.Conn) error {
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
	return writeProbeResponse(w, 42)
}

func writeAuthenticatedProbeAccept(w *fbproto.XDRWriter, encoding *fbproto.Encoding) error {
	if err := w.WriteInt32(fbproto.Op_accept_data); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.PROTOCOL_VERSION19); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.GenericArchitectureClient); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.Ptype_lazy_send); err != nil {
		return err
	}
	if err := w.WriteBuffer(nil); err != nil {
		return err
	}
	if err := w.WriteString("Legacy_Auth", encoding); err != nil {
		return err
	}
	if err := w.WriteInt32(1); err != nil {
		return err
	}
	return w.WriteBuffer(nil)
}

func readProbeServiceAttach(r *fbproto.XDRReader, encoding *fbproto.Encoding) error {
	opcode, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if opcode != fbproto.Op_service_attach {
		return fmt.Errorf("expected op_service_attach, got %d", opcode)
	}
	if _, err := r.ReadInt32(); err != nil {
		return err
	}
	if _, err := r.ReadString(encoding); err != nil {
		return err
	}
	_, err = r.ReadBuffer()
	return err
}

func writeProbeResponse(w *fbproto.XDRWriter, object int) error {
	if err := w.WriteInt32(fbproto.Op_response); err != nil {
		return err
	}
	if err := w.WriteInt32(object); err != nil {
		return err
	}
	if err := w.WriteInt64(0); err != nil {
		return err
	}
	if err := w.WriteBuffer(nil); err != nil {
		return err
	}
	return w.WriteInt32(fbproto.Isc_arg_end)
}

func readServiceAttachSPB(t *testing.T, spb []byte) map[byte][]byte {
	t.Helper()
	require.NotEmpty(t, spb)
	require.Equal(t, byte(fbproto.Isc_spb_version3), spb[0])

	items := make(map[byte][]byte)
	for offset := 1; offset < len(spb); {
		tag := spb[offset]
		offset++
		require.LessOrEqual(t, offset+4, len(spb))
		length := int(binary.LittleEndian.Uint32(spb[offset : offset+4]))
		offset += 4
		require.LessOrEqual(t, offset+length, len(spb))
		items[tag] = append([]byte(nil), spb[offset:offset+length]...)
		offset += length
	}
	return items
}
