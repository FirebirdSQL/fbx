package fbconn

import (
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/auth"
)

func TestShouldRespondToDataAcceptStopsForChallengeWithoutServerKeys(t *testing.T) {
	msg := &fbproto.OpDataAccept{
		OpCondAccept: fbproto.OpCondAccept{
			PluginName: "Srp",
			Data:       []byte("server-challenge"),
		},
	}

	require.False(t, shouldRespondToDataAccept(msg))
}

func TestShouldRespondToDataAcceptStopsWhenAuthComplete(t *testing.T) {
	msg := &fbproto.OpDataAccept{
		OpCondAccept: fbproto.OpCondAccept{AuthComplete: true},
	}

	require.False(t, shouldRespondToDataAccept(msg))
}

func TestShouldRespondToDataAcceptContinuesForWireCryptKeys(t *testing.T) {
	msg := &fbproto.OpDataAccept{
		OpCondAccept: fbproto.OpCondAccept{
			PluginName: "Srp",
			Data:       []byte("server-challenge"),
			ServerKeys: []byte{
				0, 9, 'S', 'y', 'm', 'm', 'e', 't', 'r', 'i', 'c',
				1, 5, 'A', 'r', 'c', '4', '8',
			},
		},
	}

	require.True(t, shouldRespondToDataAccept(msg))
}

func TestDataAcceptChallengeWithoutKeysSendsAttachWithSpecificAuthData(t *testing.T) {
	result := runAuthNegotiationProbe(t, fbproto.Op_accept_data)

	require.Equal(t, fbproto.Op_attach, result.opcode)
	require.True(t, result.hasSpecificAuthData)
}

func TestCondAcceptChallengeWithoutKeysSendsContAuth(t *testing.T) {
	result := runAuthNegotiationProbe(t, fbproto.Op_cond_accept)

	require.Equal(t, fbproto.Op_cont_auth, result.opcode)
	require.False(t, result.hasSpecificAuthData)
}

func TestAcceptSendsAttachWithInitialSpecificAuthData(t *testing.T) {
	result := runAuthNegotiationProbe(t, fbproto.Op_accept)

	require.Equal(t, fbproto.Op_attach, result.opcode)
	require.True(t, result.hasSpecificAuthData)
}

func TestCompressionNegotiationControlsFollowingRequestEncoding(t *testing.T) {
	for _, compressionAccepted := range []bool{false, true} {
		t.Run(fmt.Sprintf("accepted=%t", compressionAccepted), func(t *testing.T) {
			clientConn, serverConn := net.Pipe()
			resultCh := make(chan authNegotiationProbeResult, 1)
			go func() {
				defer func() { _ = serverConn.Close() }()
				resultCh <- runCompressionNegotiationProbeServer(serverConn, compressionAccepted)
			}()

			config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee?wire_compression=true")
			require.NoError(t, err)
			config.LookupFunc = func(context.Context, string) ([]string, error) {
				return []string{"127.0.0.1"}, nil
			}
			config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
				return clientConn, nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			conn, _ := ConnectConfig(ctx, config)
			if conn != nil {
				require.NoError(t, conn.Close(context.Background()))
			}

			select {
			case result := <-resultCh:
				require.NoError(t, result.err)
				require.True(t, result.compressionAdvertised)
				require.Equal(t, fbproto.Op_cont_auth, result.opcode)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}

func TestCondAcceptPluginSwitchWithoutChallengeSendsContAuth(t *testing.T) {
	result := runAuthNegotiationProbeWithOptions(t, fbproto.Op_cond_accept, []string{"Srp", "Srp256"}, nil)

	require.Equal(t, fbproto.Op_cont_auth, result.opcode)
	require.False(t, result.hasSpecificAuthData)
	require.Equal(t, "Srp256", result.authPluginName)
	require.Equal(t, "Srp,Srp256", result.authPluginList)
	require.NotEmpty(t, result.authData)
	_, err := hex.DecodeString(string(result.authData))
	require.NoError(t, err)
	require.Empty(t, result.authKeys)
}

type authNegotiationProbeResult struct {
	opcode                int
	hasSpecificAuthData   bool
	compressionAdvertised bool
	authPluginName        string
	authPluginList        string
	authData              []byte
	authKeys              []byte
	err                   error
}

func runAuthNegotiationProbe(t *testing.T, acceptOpcode int) authNegotiationProbeResult {
	return runAuthNegotiationProbeWithOptions(t, acceptOpcode, nil, probeSrpServerData())
}

func runAuthNegotiationProbeWithOptions(
	t *testing.T,
	acceptOpcode int,
	authPluginList []string,
	serverData []byte,
) authNegotiationProbeResult {
	t.Helper()

	clientConn, serverConn := net.Pipe()
	resultCh := make(chan authNegotiationProbeResult, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		resultCh <- runAuthNegotiationProbeServer(serverConn, acceptOpcode, serverData)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	if authPluginList != nil {
		config.AuthPluginList = authPluginList
	}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := ConnectConfig(ctx, config)
	if err == nil {
		require.NoError(t, conn.Close(context.Background()))
	}

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		return result
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		return authNegotiationProbeResult{}
	}
}

func runAuthNegotiationProbeServer(conn net.Conn, acceptOpcode int, serverData []byte) authNegotiationProbeResult {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return authNegotiationProbeResult{err: err}
	}

	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)

	if err := readProbeConnect(r); err != nil {
		return authNegotiationProbeResult{err: err}
	}
	if err := writeProbeAcceptWithData(w, encoding, acceptOpcode, serverData); err != nil {
		return authNegotiationProbeResult{err: err}
	}

	opcode, err := r.ReadInt32()
	if err != nil {
		return authNegotiationProbeResult{err: err}
	}
	result := authNegotiationProbeResult{opcode: opcode}
	switch opcode {
	case fbproto.Op_attach:
		dpb, err := readProbeAttachDPB(r, encoding)
		if err != nil {
			return authNegotiationProbeResult{err: err}
		}
		result.hasSpecificAuthData = probeDPBHasTag(dpb, fbproto.Isc_dpb_specific_auth_data)
	case fbproto.Op_cont_auth:
		msg := &fbproto.OpContAuth{Encoding: encoding}
		if err := msg.Read(r); err != nil {
			return authNegotiationProbeResult{err: err}
		}
		result.authPluginName = msg.PluginName
		result.authPluginList = msg.PluginList
		result.authData = msg.AuthData
		result.authKeys = msg.Keys
	}
	return result
}

func runCompressionNegotiationProbeServer(conn net.Conn, compressionAccepted bool) authNegotiationProbeResult {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return authNegotiationProbeResult{err: err}
	}

	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
	compressionAdvertised, err := readProbeConnectCompression(r)
	if err != nil {
		return authNegotiationProbeResult{err: err}
	}
	if err := writeProbeAcceptWithOptions(
		w,
		encoding,
		fbproto.Op_cond_accept,
		probeSrpServerData(),
		compressionAccepted,
	); err != nil {
		return authNegotiationProbeResult{err: err}
	}

	var opcode int
	if compressionAccepted {
		zr, zlibErr := zlib.NewReader(conn)
		if zlibErr != nil {
			return authNegotiationProbeResult{err: zlibErr}
		}
		opcode, err = fbproto.NewXDRReader(zr).ReadInt32()
		if closeErr := zr.Close(); err == nil {
			err = closeErr
		}
	} else {
		opcode, err = r.ReadInt32()
	}
	return authNegotiationProbeResult{
		opcode:                opcode,
		compressionAdvertised: compressionAdvertised,
		err:                   err,
	}
}

func readProbeConnect(r *fbproto.XDRReader) error {
	_, err := readProbeConnectCompression(r)
	return err
}

func readProbeConnectCompression(r *fbproto.XDRReader) (bool, error) {
	opcode, err := r.ReadInt32()
	if err != nil {
		return false, err
	}
	if opcode != fbproto.Op_connect {
		return false, fmt.Errorf("expected op_connect, got %d", opcode)
	}

	for range 3 {
		if _, err := r.ReadInt32(); err != nil {
			return false, err
		}
	}
	if _, err := r.ReadBuffer(); err != nil {
		return false, err
	}
	protocolCount, err := r.ReadInt32()
	if err != nil {
		return false, err
	}
	if _, err := r.ReadBuffer(); err != nil {
		return false, err
	}
	compressionAdvertised := false
	for range protocolCount {
		for range 3 {
			if _, err := r.ReadInt32(); err != nil {
				return false, err
			}
		}
		maxType, err := r.ReadInt32()
		if err != nil {
			return false, err
		}
		compressionAdvertised = compressionAdvertised || maxType&fbproto.Pflag_compress != 0
		if _, err := r.ReadInt32(); err != nil {
			return false, err
		}
	}
	return compressionAdvertised, nil
}

func writeProbeAccept(w *fbproto.XDRWriter, encoding *fbproto.Encoding, opcode int) error {
	return writeProbeAcceptWithOptions(w, encoding, opcode, probeSrpServerData(), false)
}

func writeProbeAcceptWithData(
	w *fbproto.XDRWriter,
	encoding *fbproto.Encoding,
	opcode int,
	data []byte,
) error {
	return writeProbeAcceptWithOptions(w, encoding, opcode, data, false)
}

func writeProbeAcceptWithOptions(
	w *fbproto.XDRWriter,
	encoding *fbproto.Encoding,
	opcode int,
	data []byte,
	compress bool,
) error {
	if err := w.WriteInt32(opcode); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.PROTOCOL_VERSION19); err != nil {
		return err
	}
	if err := w.WriteInt32(fbproto.GenericArchitectureClient); err != nil {
		return err
	}
	acceptType := fbproto.Ptype_lazy_send
	if compress {
		acceptType |= fbproto.Pflag_compress
	}
	if err := w.WriteInt32(acceptType); err != nil {
		return err
	}
	if opcode == fbproto.Op_accept {
		return nil
	}
	if err := w.WriteBuffer(data); err != nil {
		return err
	}
	if err := w.WriteString("Srp256", encoding); err != nil {
		return err
	}
	if err := w.WriteInt32(0); err != nil {
		return err
	}
	return w.WriteBuffer(nil)
}

func readProbeAttachDPB(r *fbproto.XDRReader, encoding *fbproto.Encoding) ([]byte, error) {
	if _, err := r.ReadInt32(); err != nil {
		return nil, err
	}
	if _, err := r.ReadString(encoding); err != nil {
		return nil, err
	}
	return r.ReadBuffer()
}

func probeSrpServerData() []byte {
	salt := []byte("C5145A3209566ECAF4C936E71F5BD44A18075EB88BAD7120B3C589939B8FAB77")
	serverPublic := []byte("2")

	data := make([]byte, 0, 4+len(salt)+len(serverPublic))
	data = binary.LittleEndian.AppendUint16(data, uint16(len(salt)))
	data = append(data, salt...)
	data = binary.LittleEndian.AppendUint16(data, uint16(len(serverPublic)))
	data = append(data, serverPublic...)
	return data
}

func probeDPBHasTag(dpb []byte, tag byte) bool {
	if len(dpb) == 0 || dpb[0] != fbproto.Isc_dpb_version2 {
		return false
	}

	for offset := 1; offset < len(dpb); {
		currentTag := dpb[offset]
		offset++
		if currentTag == tag {
			return true
		}
		if offset+4 > len(dpb) {
			return false
		}
		length := int(binary.LittleEndian.Uint32(dpb[offset : offset+4]))
		offset += 4
		if offset+length > len(dpb) {
			return false
		}
		offset += length
	}
	return false
}

const pluginName = "cert_connect_test"

type certificateCapturePlugin struct {
	certificate    []byte
	certificatePIN string
}

func (p *certificateCapturePlugin) Name() string        { return pluginName }
func (p *certificateCapturePlugin) Advertise() []string { return []string{p.Name()} }
func (p *certificateCapturePlugin) InitialClientData() []byte {
	return p.certificate
}
func (p *certificateCapturePlugin) HandleServerData(string, string, string, []byte) (*auth.AuthPluginStep, error) {
	return &auth.AuthPluginStep{Reply: p.certificate, AuthSpecific: p.certificate}, nil
}
func (p *certificateCapturePlugin) HandleAuthResponse([]byte) error { return nil }

func TestConnectConfigPassesCertificateToPluginConstructor(t *testing.T) {
	wantCert := []byte{0x01, 0x02, 0x03}
	wantPIN := "123"

	var constructed *certificateCapturePlugin
	require.NoError(t, auth.RegisterAuthPlugin(pluginName, func(conf auth.UserConfig) auth.AuthPlugin {
		plugin := &certificateCapturePlugin{certificate: conf.Certificate, certificatePIN: conf.CertificatePIN}
		constructed = plugin
		return plugin
	}))

	config, err := ParseConfig("")
	require.NoError(t, err)
	config.Certificate = wantCert
	config.CertificatePIN = wantPIN
	config.AuthPluginList = []string{pluginName}

	_, _ = ConnectConfig(context.Background(), config)

	require.NotEmpty(t, constructed)
	require.Equal(t, wantCert, constructed.certificate)
	require.Equal(t, wantPIN, constructed.certificatePIN)
}

func TestUserConfigPassedToPluginWhenSetCurrentPluginReused(t *testing.T) {
	srpPluginName := "Srp"
	pluginList := []string{pluginName, srpPluginName}
	wantCert := []byte{0x01, 0x02, 0x03}
	wantPIN := "123"
	registerErr := auth.RegisterAuthPlugin(pluginName, func(conf auth.UserConfig) auth.AuthPlugin {
		plugin := &certificateCapturePlugin{certificate: conf.Certificate, certificatePIN: conf.CertificatePIN}
		return plugin
	})
	if registerErr != nil {
		require.EqualError(t, registerErr, "plugin already exists")
	}
	userConfig := auth.UserConfig{Certificate: wantCert, CertificatePIN: wantPIN}
	m, err := auth.NewAuthManager(userConfig, pluginList)
	require.NoError(t, err)
	m.SetCurrentPlugin(pluginName)
	step, err := m.HandleServerChallenge(pluginName, nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.Equal(t, step.AuthSpecific, m.AuthSpecificData())
	require.Equal(t, step.AuthSpecific, wantCert)
	require.Equal(t, pluginName, m.GetCurrentPlugin().Name())

	_, ok := m.SetCurrentPlugin(srpPluginName)
	require.True(t, ok)
	step, err = m.HandleServerChallenge(srpPluginName, nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.Empty(t, step.AuthSpecific)

	_, ok = m.SetCurrentPlugin(pluginName)
	require.True(t, ok)
	step, err = m.HandleServerChallenge(pluginName, nil)
	require.NoError(t, err)
	require.NotNil(t, step)
	require.Equal(t, step.AuthSpecific, wantCert)
	require.Equal(t, pluginName, m.GetCurrentPlugin().Name())
}
