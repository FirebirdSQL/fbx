package fbconn

import (
	"bufio"
	"container/list"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/red-soft/fbx/fbconn/ctxwatch"
	"github.com/red-soft/fbx/fbconn/internal/bgreader"
	"github.com/red-soft/fbx/fbproto"
	"github.com/red-soft/fbx/fbproto/auth"
	"github.com/red-soft/fbx/fbproto/wirecrypt"
)

// Debug printing for authentication handshake. Enable by setting
// FBX_AUTH_DEBUG to a truthy value (e.g. 1, true).
func authPrintf(format string, args ...any) {
	if authDebugEnabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("FBX_AUTH_DEBUG"))); authDebugEnabled {
		fmt.Printf(format, args...)
	}
}

const (
	connStatusUninitialized = iota
	connStatusConnecting
	connStatusClosed
	connStatusIdle
	connStatusBusy
)

// Notice represents a notice response message reported by the PostgreSQL server. Be aware that this is distinct from
// LISTEN/NOTIFY notification.
type Notice FbError

// Notification is a message received from the PostgreSQL LISTEN/NOTIFY system
type Notification struct {
	PID     uint32 // backend pid that sent the notification
	Channel string // channel from which notification was received
	Payload string
}

// DialFunc is a function that can be used to connect to a PostgreSQL server.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// LookupFunc is a function that can be used to lookup IPs addrs from host. Optionally an ip:port combination can be
// returned in order to override the connection string's port.
type LookupFunc func(ctx context.Context, host string) (addrs []string, err error)

// BuildFrontendFunc is a function that can be used to create Client implementation for connection.
type BuildFrontendFunc func(r io.Reader, w io.Writer) *fbproto.Client

// FbErrorHandler is a function that handles errors returned from Firebird. This function must return true to keep
// the connection open. Returning false will cause the connection to be closed immediately. You should return
// false on any FATAL-severity errors. This will not receive network errors. The *FbConn is provided so the handler is
// aware of the origin of the error, but it must not invoke any query method.
type FbErrorHandler func(*FbConn, *FbError) bool

// FbWarningHandler is a function that handles warnings returned from Firebird.
// The *FbConn is provided so the handler is aware of the origin of the warning, but it must not invoke any query method.
type FbWarningHandler func(*FbConn, *FbWarning)

// NoticeHandler is a function that can handle notices received from the PostgreSQL server. Notices can be received at
// any time, usually during handling of a query response. The *FbConn is provided so the handler is aware of the origin
// of the notice, but it must not invoke any query method. Be aware that this is distinct from LISTEN/NOTIFY
// notification.
type NoticeHandler func(*FbConn, *Notice)

// NotificationHandler is a function that can handle notifications received from the PostgreSQL server. Notifications
// can be received at any time, usually during handling of a query response. The *FbConn is provided so the handler is
// aware of the origin of the notice, but it must not invoke any query method. Be aware that this is distinct from a
// notice event.
type NotificationHandler func(*FbConn, *Notification)

// FbConn is a low-level PostgreSQL connection handle. It is not safe for concurrent usage.
type FbConn struct {
	conn              net.Conn
	secretKey         uint32            // key to use to send a cancel query message to the server
	parameterStatuses map[string]string // parameters that have been reported by the server
	txStatus          byte
	client            *fbproto.Client
	wireCrypt         *wirecrypt.Manager
	bgReader          *bgreader.BGReader
	writer            *bufio.Writer
	slowWriteTimer    *time.Timer
	bgReaderStarted   chan struct{}
	dbHandle          int

	customData map[string]any

	config *Config

	status byte // One of connStatus* constants

	bufferingReceive    bool
	bufferingReceiveMux sync.Mutex
	bufferingReceiveMsg fbproto.ServerMessage
	bufferingReceiveErr error

	peekedMsg fbproto.ServerMessage

	// Reusable / preallocated resources
	pipeline          Pipeline
	contextWatcher    *ctxwatch.ContextWatcher
	fieldDescriptions [16]FieldDescription

	cleanupDone         chan struct{}
	Encoding            *fbproto.Encoding
	internalTransaction *FBTransaction
	explicitTransaction bool
	activeTransactions  map[*FBTransaction]struct{}

	preparedMap map[string]*StatementDescription

	// Tracks whether we've already sent the first op_cont_auth (where p_list
	// must include the client's plugin list). Subsequent op_cont_auth must use
	// an empty p_list.
	authContFirstSent bool
}

// Connect establishes a connection to a PostgreSQL server using the environment and connString (in URL or keyword/value
// format) to provide configuration. See documentation for [ParseConfig] for details. ctx can be used to cancel a
// connect attempt.
func Connect(ctx context.Context, connString string) (*FbConn, error) {
	config, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	return ConnectConfig(ctx, config)
}

// Connect establishes a connection to a PostgreSQL server using the environment and connString (in URL or keyword/value
// format) and ParseConfigOptions to provide additional configuration. See documentation for [ParseConfig] for details.
// ctx can be used to cancel a connect attempt.
func ConnectWithOptions(ctx context.Context, connString string, parseConfigOptions ParseConfigOptions) (*FbConn, error) {
	config, err := ParseConfigWithOptions(connString, parseConfigOptions)
	if err != nil {
		return nil, err
	}

	return ConnectConfig(ctx, config)
}

// Connect establishes a connection to a PostgreSQL server using config. config must have been constructed with
// [ParseConfig]. ctx can be used to cancel a connect attempt.
//
// If config.Fallbacks are present they will sequentially be tried in case of error establishing network connection. An
// authentication error will terminate the chain of attempts (like libpq:
// https://www.postgresql.org/docs/11/libpq-connect.html#LIBPQ-MULTIPLE-HOSTS) and be returned as the error.
func ConnectConfig(ctx context.Context, config *Config) (*FbConn, error) {
	// Default values are set in ParseConfig. Enforce initial creation by ParseConfig rather than setting defaults from
	// zero values.
	if !config.createdByParseConfig {
		panic("config must be created by ParseConfig")
	}

	var allErrors []error

	connectConfigs, errs := buildConnectOneConfigs(ctx, config)
	if len(errs) > 0 {
		allErrors = append(allErrors, errs...)
	}

	if len(connectConfigs) == 0 {
		return nil, &ConnectError{Config: config, err: fmt.Errorf("hostname resolving error: %w", errors.Join(allErrors...))}
	}

	fbConn, errs := connectPreferred(ctx, config, connectConfigs)
	if len(errs) > 0 {
		allErrors = append(allErrors, errs...)
		return nil, &ConnectError{Config: config, err: errors.Join(allErrors...)}
	}

	if config.AfterConnect != nil {
		err := config.AfterConnect(ctx, fbConn)
		if err != nil {
			fbConn.conn.Close()
			return nil, &ConnectError{Config: config, err: fmt.Errorf("AfterConnect error: %w", err)}
		}
	}

	return fbConn, nil
}

func (fbConn *FbConn) handleCondAccept(msg *fbproto.OpCondAccept, authManager *auth.AuthManager, respond bool) error {
	authPrintf("[AUTHDBG] cond_accept: plugin=%s data_len=%d\n", msg.PluginName, len(msg.Data))
	authPrintf("%#v\n", msg)

	if fbConn.wireCrypt != nil {
		if err := fbConn.wireCrypt.AddServerKeys(msg.ServerKeys); err != nil {
			return err
		}
	}

	step, err := authManager.HandleServerChallenge(msg.PluginName, msg.Data)
	if err != nil {
		return err
	}
	if respond && step != nil {
		pluginList := ""
		if !fbConn.authContFirstSent {
			pluginList = authManager.BuildList()
			fbConn.authContFirstSent = true
		}
		return fbConn.sendContAuth(step.Reply, msg.PluginName, pluginList, nil)
	}
	return nil
}

func (fbConn *FbConn) handleAuthResponse(msg *fbproto.OpResponse, config *Config, authManager *auth.AuthManager) (bool, error) {
	if msg.Failure {
		return false, fmt.Errorf("authentication failed: %s", msg.Message)
	}

	if err := authManager.HandleAuthResponse(msg.Data); err != nil {
		return false, err
	}

	if fbConn.wireCrypt != nil {
		if err := fbConn.wireCrypt.AddServerKeys(msg.Data); err != nil {
			return false, err
		}
	}

	sessionKey := authManager.SessionKey()
	authPrintf("[AUTHDBG] auth response: wire_crypt=%v sessionKey=%v\n",
		config.WireCrypt, len(sessionKey) > 0)
	if !config.WireCrypt || len(sessionKey) == 0 {
		return false, nil
	}

	initInfo, err := fbConn.enableWireCrypt(sessionKey)
	if err != nil {
		if errors.Is(err, wirecrypt.ErrNoCommonPlugin) {
			authPrintf("[AUTHDBG] wire encryption negotiation failed: %v\n", err)
			return false, nil
		}
		return false, err
	}
	if initInfo != nil {
		authPrintf("[AUTHDBG] wire encryption enabled with %s (nonce_len=%d)\n",
			initInfo.Identifier.Plugin, len(initInfo.Nonce))
		return true, nil
	}

	return false, nil
}

func (fbConn *FbConn) enableWireCrypt(sessionKey []byte) (*wirecrypt.InitResult, error) {
	if fbConn.wireCrypt == nil || len(sessionKey) == 0 {
		return nil, nil
	}

	initInfo, err := fbConn.wireCrypt.TryInitialize(sessionKey)
	if err != nil {
		return nil, err
	}

	if err := fbConn.client.WriteMessage(&fbproto.OpCrypt{Plugin: initInfo.Identifier.Plugin, Encoding: fbConn.Encoding}); err != nil {
		return nil, err
	}
	codec, err := wirecrypt.NewSymmetricCodec(initInfo.Identifier.Plugin, sessionKey, initInfo.Nonce)
	if err != nil {
		return nil, err
	}
	fbConn.client.SetCodec(initInfo.Identifier.Plugin, codec)

	fbConn.wireCrypt.ClearServerKeys()

	return initInfo, nil
}

func (fbConn *FbConn) sendContAuth(authData []byte, pluginName string, pluginList string, keys []byte) error {
	dataCopy := make([]byte, len(authData))
	copy(dataCopy, authData)
	// Debug: show what we are about to send in op_cont_auth
	// Print only prefix of proof to avoid clutter
	prefixLen := 16
	if len(dataCopy) < prefixLen {
		prefixLen = len(dataCopy)
	}
	authPrintf("[AUTHDBG] send op_cont_auth: plugin=%s list=%s proof_len=%d proof_prefix=%x\n",
		pluginName, pluginList, len(dataCopy), dataCopy[:prefixLen])
	msg := &fbproto.OpContAuth{
		Encoding:   fbConn.Encoding,
		AuthData:   dataCopy,
		PluginName: pluginName,
		PluginList: pluginList,
		Keys:       keys,
	}
	return fbConn.client.WriteMessage(msg)
}

// buildConnectOneConfigs resolves hostnames and builds a list of connectOneConfigs to try connecting to. It returns a
// slice of successfully resolved connectOneConfigs and a slice of errors. It is possible for both slices to contain
// values if some hosts were successfully resolved and others were not.
func buildConnectOneConfigs(ctx context.Context, config *Config) ([]*connectOneConfig, []error) {
	// Simplify usage by treating primary config and fallbacks the same.
	fallbackConfigs := []*FallbackConfig{
		{
			Host: config.Host,
			Port: config.Port,
		},
	}
	fallbackConfigs = append(fallbackConfigs, config.Fallbacks...)

	var configs []*connectOneConfig

	var allErrors []error

	for _, fb := range fallbackConfigs {
		// skip resolve for unix sockets
		if isAbsolutePath(fb.Host) {
			network, address := NetworkAddress(fb.Host, fb.Port)
			configs = append(configs, &connectOneConfig{
				network:          network,
				address:          address,
				originalHostname: fb.Host,
			})

			continue
		}

		ips, err := config.LookupFunc(ctx, fb.Host)
		if err != nil {
			allErrors = append(allErrors, err)
			continue
		}

		for _, ip := range ips {
			splitIP, splitPort, err := net.SplitHostPort(ip)
			if err == nil {
				port, err := strconv.ParseUint(splitPort, 10, 16)
				if err != nil {
					return nil, []error{fmt.Errorf("error parsing port (%s) from lookup: %w", splitPort, err)}
				}
				network, address := NetworkAddress(splitIP, uint16(port))
				configs = append(configs, &connectOneConfig{
					network:          network,
					address:          address,
					originalHostname: fb.Host,
				})
			} else {
				network, address := NetworkAddress(ip, fb.Port)
				configs = append(configs, &connectOneConfig{
					network:          network,
					address:          address,
					originalHostname: fb.Host,
				})
			}
		}
	}

	return configs, allErrors
}

// connectPreferred attempts to connect to the preferred host from connectOneConfigs. The connections are attempted in
// order. If a connection is successful it is returned. If no connection is successful then all errors are returned. If
// a connection attempt returns a [NotPreferredError], then that host will be used if no other hosts are successful.
func connectPreferred(ctx context.Context, config *Config, connectOneConfigs []*connectOneConfig) (*FbConn, []error) {
	octx := ctx
	var allErrors []error

	var fallbackConnectOneConfig *connectOneConfig
	for i, c := range connectOneConfigs {
		// ConnectTimeout restricts the whole connection process.
		if config.ConnectTimeout != 0 {
			// create new context first time or when previous host was different
			if i == 0 || (connectOneConfigs[i].address != connectOneConfigs[i-1].address) {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(octx, config.ConnectTimeout)
				defer cancel()
			}
		} else {
			ctx = octx
		}

		fbConn, err := connectOne(ctx, config, c, false)
		if fbConn != nil {
			return fbConn, nil
		}

		allErrors = append(allErrors, err)

		var fbErr *FbError
		if errors.As(err, &fbErr) {
			// pgx will try next host even if libpq does not in certain cases (see #2246)
			// consider change for the next major version

			const ERRCODE_INVALID_PASSWORD = "28P01"
			const ERRCODE_INVALID_CATALOG_NAME = "3D000"   // db does not exist
			const ERRCODE_INSUFFICIENT_PRIVILEGE = "42501" // missing connect privilege

			// auth failed due to invalid password, db does not exist or user has no permission
			if fbErr.Code == ERRCODE_INVALID_PASSWORD ||
				fbErr.Code == ERRCODE_INVALID_CATALOG_NAME ||
				fbErr.Code == ERRCODE_INSUFFICIENT_PRIVILEGE {
				return nil, allErrors
			}
		}

		var npErr *NotPreferredError
		if errors.As(err, &npErr) {
			fallbackConnectOneConfig = c
		}
	}

	if fallbackConnectOneConfig != nil {
		fbConn, err := connectOne(ctx, config, fallbackConnectOneConfig, true)
		if err == nil {
			return fbConn, nil
		}
		allErrors = append(allErrors, err)
	}

	return nil, allErrors
}

// connectOne makes one connection attempt to a single host.
func connectOne(ctx context.Context, config *Config, connectConfig *connectOneConfig,
	ignoreNotPreferredErr bool,
) (*FbConn, error) {
	fbConn := new(FbConn)
	fbConn.config = config
	fbConn.cleanupDone = make(chan struct{})
	fbConn.customData = make(map[string]any)
	fbConn.activeTransactions = make(map[*FBTransaction]struct{})

	var err error

	newPerDialConnectError := func(msg string, err error) *perDialConnectError {
		err = normalizeTimeoutError(ctx, err)
		e := &perDialConnectError{address: connectConfig.address, originalHostname: connectConfig.originalHostname, err: fmt.Errorf("%s: %w", msg, err)}
		return e
	}

	fbConn.conn, err = config.DialFunc(ctx, connectConfig.network, connectConfig.address)
	if err != nil {
		return nil, newPerDialConnectError("dial error", err)
	}

	fbConn.contextWatcher = ctxwatch.NewContextWatcher(config.BuildContextWatcherHandler(fbConn))
	fbConn.contextWatcher.Watch(ctx)
	defer fbConn.contextWatcher.Unwatch()

	fbConn.parameterStatuses = make(map[string]string)
	fbConn.status = connStatusConnecting

	fbConn.Encoding, err = fbproto.NewEncoding("UTF8")
	if err != nil {
		return nil, err
	}
	fbConn.bgReader = bgreader.New(fbConn.conn)
	fbConn.slowWriteTimer = time.AfterFunc(time.Duration(math.MaxInt64),
		func() {
			fbConn.bgReader.Start()
			fbConn.bgReaderStarted <- struct{}{}
		},
	)
	fbConn.slowWriteTimer.Stop()
	fbConn.bgReaderStarted = make(chan struct{})
	fbConn.writer = bufio.NewWriter(fbConn.conn)
	fbConn.client = fbproto.NewClient(fbConn.bgReader, fbConn.writer, fbConn.Encoding)
	fbConn.wireCrypt = wirecrypt.NewDefaultManager()

	authManager, err := auth.NewAuthManager(config.User, config.Password, config.AuthPluginList)
	if err != nil {
		fbConn.conn.Close()
		return nil, newPerDialConnectError("authentication setup failed", err)
	}
	initialHandshake := authManager.BeginHandshake()
	if initialHandshake == nil {
		fbConn.conn.Close()
		return nil, newPerDialConnectError("authentication setup failed", fmt.Errorf("no authentication plugins available"))
	}

	opConnect := fbproto.OpConnect{
		Database:         config.Database,
		User:             config.User,
		SystemUser:       config.SystemUser,
		Hostname:         config.Host,
		Password:         config.Password,
		AuthPluginName:   initialHandshake.PluginName,
		AuthPluginsList:  initialHandshake.PluginList,
		AuthSpecificData: initialHandshake.SpecificData,
		Options:          nil,
		WireCrypt:        config.WireCrypt,
		Encoding:         fbConn.Encoding,
	}

	if err = fbConn.client.WriteMessage(&opConnect); err != nil {
		fbConn.conn.Close()
		return nil, newPerDialConnectError("failed to send opConnect", err)
	}

	// Track if we've just enabled wire encryption and need to consume
	// the server's acknowledgment response for op_crypt.
	pendingCryptAck := false

connectLoop:
	for {
		if err = fbConn.writer.Flush(); err != nil {
			fbConn.conn.Close()
			return nil, newPerDialConnectError("failed to flush opConnect", err)
		}

		msg, err := fbConn.receiveMessage()
		if err != nil {
			fbConn.conn.Close()
			return nil, newPerDialConnectError("authentication failed", err)
		}

		authPrintf("[AUTHDBG] received %T\n", msg)
		switch msg := msg.(type) {
		case *fbproto.OpContAuth:
			fake := &fbproto.OpCondAccept{Encoding: fbConn.Encoding}
			fake.PluginName = msg.PluginName
			fake.Data = msg.AuthData
			if err := fbConn.handleCondAccept(fake, authManager, true); err != nil {
				fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			continue connectLoop
		case *fbproto.OpCondAccept:
			if err := fbConn.handleCondAccept(msg, authManager, true); err != nil {
				fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			continue connectLoop
		case *fbproto.OpResponse:
			if pendingCryptAck {
				// This is the ack for op_crypt; consume it and finish handshake
				pendingCryptAck = false
				break connectLoop
			}
			continueLoop, err := fbConn.handleAuthResponse(msg, config, authManager)
			if err != nil {
				fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			if continueLoop {
				// We enabled encryption; expect a crypt ack next
				pendingCryptAck = true
				continue connectLoop
			}
			break connectLoop
		case *fbproto.OpDataAccept:
			if err := fbConn.handleCondAccept(&msg.OpCondAccept, authManager, !msg.AuthComplete); err != nil {
				fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			if msg.AuthComplete {
				break connectLoop
			}
			continue connectLoop
		case *fbproto.ErrorResponse:
			fbConn.conn.Close()
			return nil, newPerDialConnectError("authentication failed", ErrorResponseToFbError(msg))
		default:
			fbConn.conn.Close()
			return nil, newPerDialConnectError("authentication failed", fmt.Errorf("unexpected message type %T", msg))
		}
	}

	if fbConn.wireCrypt != nil {
		fbConn.wireCrypt.ClearServerKeys()
	}

	var dpb *fbproto.DPBWriterV2
	if config.DPB == nil {
		dpb = fbproto.NewDPBWriterV2()
	} else {
		dpb = fbproto.NewDPBWriterV2FromBytes(config.DPB)
	}
	dpb.PutInt32(fbproto.Isc_dpb_sql_dialect, 3)
	dpb.PutString(fbproto.Isc_dpb_lc_ctype, config.Encoding)
	if config.User != "" {
		dpb.PutString(fbproto.Isc_dpb_user_name, config.User)
	}
	if config.Role != "" {
		dpb.PutString(fbproto.Isc_dpb_sql_role_name, config.Role)
	}
	dpb.PutInt32(fbproto.Isc_dpb_process_id, int32(os.Getpid()))
	app, ok := config.RuntimeParams["application_name"]
	if !ok {
		app = "fbx"
	}
	dpb.PutString(fbproto.Isc_dpb_process_name, app)
	//FIXME: fbx is timezone need? where get timezone?
	//dpb.AppendTag(fbproto.Isc_dpb_session_time_zone)
	dpb.PutString(fbproto.Isc_dpb_auth_plugin_list, authManager.BuildList())
	if plugin := authManager.GetCurrentPlugin(); plugin != nil {
		dpb.PutString(fbproto.Isc_dpb_auth_plugin_name, plugin.Name())
	}
	dpb.PutInt32(fbproto.Isc_dpb_utf8_filename, 1)
	if authSpecific := authManager.AuthSpecificData(); len(authSpecific) > 0 {
		// For SRP, clientData historically was ASCII-hex, and servers accept
		// that in specific_auth_data. For GSS, sends raw token bytes.
		// The server handles both when provided as a Buffer, so send raw bytes
		// to avoid double-hexing.
		dpb.PutTag(fbproto.Isc_dpb_specific_auth_data)
		dpb.PutBuffer(authSpecific)
	}

	opAttach := fbproto.OpAttach{
		File: config.Database,
		DPB:  dpb.Bytes(),
	}

	if err = fbConn.client.WriteMessage(&opAttach); err != nil {
		fbConn.conn.Close()
		return nil, newPerDialConnectError("failed to send opAttach", err)
	}
	if err = fbConn.writer.Flush(); err != nil {
		fbConn.conn.Close()
		return nil, newPerDialConnectError("failed to flush opAttach", err)
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.conn.Close()
		return nil, newPerDialConnectError("attachment failed", err)
	}
	switch m := msg.(type) {
	case *fbproto.OpResponse:
		fbConn.dbHandle = m.Object
	case *fbproto.ErrorResponse:
		fbConn.conn.Close()
		return nil, ErrorResponseToFbError(m)
	}

	fbConn.status = connStatusIdle
	return fbConn, nil
}

func (fbConn *FbConn) getStatementInfo(statementHandle int) (*fbproto.StatementInfo, error) {
	var err error
	var continueIndex int
	infoSize := 10240
	info := fbproto.StatementInfo{}
	for {
		if err = fbConn.client.WriteMessage(&fbproto.OpInfoSQL{
			StatementHandle:           statementHandle,
			StatementInformationItems: fbproto.GetSelectDescribeInfoItemsContinue(continueIndex),
			MaxBufferLength:           infoSize,
		}); err != nil {
			return nil, err
		}
		if err = fbConn.writer.Flush(); err != nil {
			return nil, err
		}

		msg, err := fbConn.receiveMessage()
		if err != nil {
			fbConn.asyncClose()
			return nil, err
		}
		var respData []byte
		switch m := msg.(type) {
		case *fbproto.OpResponse:
			respData = m.Data
		case *fbproto.ErrorResponse:
			return nil, ErrorResponseToFbError(m)
		default:
			return nil, fmt.Errorf("received unexpected message: %v", m)
		}

		err = info.Parse(respData)
		if err != nil {
			truncError := fbproto.StatementInfoTruncationError{}
			if errors.As(err, &truncError) {
				fmt.Printf("truncation error: %v\n", truncError)
				continueIndex = truncError.Index
				infoSize *= 2
			} else {
				return nil, fmt.Errorf("failed to parse statement info: %v", err)
			}
		} else {
			break
		}
	}
	return &info, nil
}

func startTLS(conn net.Conn, tlsConfig *tls.Config) (net.Conn, error) {
	err := binary.Write(conn, binary.BigEndian, []int32{8, 80877103})
	if err != nil {
		return nil, err
	}

	response := make([]byte, 1)
	if _, err = io.ReadFull(conn, response); err != nil {
		return nil, err
	}

	if response[0] != 'S' {
		return nil, errors.New("server refused TLS connection")
	}

	return tls.Client(conn, tlsConfig), nil
}

func (fbConn *FbConn) txPasswordMessage(password string) (err error) {
	//fbConn.frontend.WriteMessage(&pgproto3.PasswordMessage{Password: password})
	//return fbConn.flushWithPotentialWriteReadDeadlock()
	panic("implement me")
}

func hexMD5(s string) string {
	hash := md5.New()
	io.WriteString(hash, s)
	return hex.EncodeToString(hash.Sum(nil))
}

func (fbConn *FbConn) signalMessage() chan struct{} {
	//if fbConn.bufferingReceive {
	//	panic("BUG: signalMessage when already in progress")
	//}
	//
	//fbConn.bufferingReceive = true
	//fbConn.bufferingReceiveMux.Lock()
	//
	//ch := make(chan struct{})
	//go func() {
	//	fbConn.bufferingReceiveMsg, fbConn.bufferingReceiveErr = fbConn.frontend.Receive()
	//	fbConn.bufferingReceiveMux.Unlock()
	//	close(ch)
	//}()
	//
	//return ch
	panic("implement me")
}

// ReceiveMessage receives one wire protocol message from the PostgreSQL server. It must only be used when the
// connection is not busy. e.g. It is an error to call ReceiveMessage while reading the result of a query. The messages
// are still handled by the core fbconn message handling system so receiving a NotificationResponse will still trigger
// the OnNotification callback.
//
// This is a very low level method that requires deep understanding of the PostgreSQL wire protocol to use correctly.
// See https://www.postgresql.org/docs/current/protocol.html.
func (fbConn *FbConn) ReceiveMessage(ctx context.Context) (fbproto.ServerMessage, error) {
	if err := fbConn.lock(); err != nil {
		return nil, err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return nil, newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		err = &fbconnError{
			msg:         "receive message failed",
			err:         normalizeTimeoutError(ctx, err),
			safeToRetry: true,
		}
	}
	return msg, err
}

// peekMessage peeks at the next message without setting up context cancellation.
func (fbConn *FbConn) peekMessage() (fbproto.ServerMessage, error) {
	if fbConn.peekedMsg != nil {
		return fbConn.peekedMsg, nil
	}

	var msg fbproto.ServerMessage
	var err error

	if fbConn.bufferingReceive {
		fbConn.bufferingReceiveMux.Lock()
		msg = fbConn.bufferingReceiveMsg
		err = fbConn.bufferingReceiveErr
		fbConn.bufferingReceiveMux.Unlock()
		fbConn.bufferingReceive = false

		// If a timeout error happened in the background try the read again.
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			msg, err = fbConn.client.ReceiveMessage()
		}
	} else {
		msg, err = fbConn.client.ReceiveMessage()
	}

	if err != nil {
		// Close on anything other than timeout error - everything else is fatal
		var netErr net.Error
		isNetErr := errors.As(err, &netErr)
		if !(isNetErr && netErr.Timeout()) {
			fbConn.asyncClose()
		}

		return nil, err
	}

	fbConn.peekedMsg = msg
	return msg, nil
}

// receiveMessage receives a message without setting up context cancellation
func (fbConn *FbConn) receiveMessage() (fbproto.ServerMessage, error) {
	msg, err := fbConn.peekMessage()
	if err != nil {
		return nil, err
	}
	fbConn.peekedMsg = nil

	switch msg := msg.(type) {
	case *fbproto.ErrorResponse:
		err := ErrorResponseToFbError(msg)
		if fbConn.config.OnFbError != nil && !fbConn.config.OnFbError(fbConn, err) {
			fbConn.status = connStatusClosed
			fbConn.conn.Close() // Ignore error as the connection is already broken and there is already an error to return.
			close(fbConn.cleanupDone)
			return nil, err
		}
	case *fbproto.OpResponse:
		if msg.Warnings && fbConn.config.OnFbWarning != nil {
			fbConn.config.OnFbWarning(fbConn, OpResponseToFbWarning(msg))
		}
	}

	return msg, nil
}

// Conn returns the underlying net.Conn. This rarely necessary. If the connection will be directly used for reading or
// writing then SyncConn should usually be called before Conn.
func (fbConn *FbConn) Conn() net.Conn {
	return fbConn.conn
}

// ActiveTx returns true if connection have active transactions
func (fbConn *FbConn) ActiveTx() bool {
	return len(fbConn.activeTransactions) > 0
}

// SecretKey returns the backend secret key used to send a cancel query message to the server.
func (fbConn *FbConn) SecretKey() uint32 {
	//return fbConn.secretKey
	panic("implement me")
}

// Client returns the underlying *fbproto.Client
func (fbConn *FbConn) Client() *fbproto.Client {
	return fbConn.client
}

// Close closes a connection. It is safe to call Close on an already closed connection. Close attempts a clean close by
// sending the exit message to PostgreSQL. However, this could block so ctx is available to limit the time to wait. The
// underlying net.Conn.Close() will always be called regardless of any other errors.
func (fbConn *FbConn) Close(ctx context.Context) error {
	if fbConn.status == connStatusClosed {
		return nil
	}
	fbConn.status = connStatusClosed

	defer close(fbConn.cleanupDone)
	defer fbConn.conn.Close()

	if ctx != context.Background() {
		// Close may be called while a cancellable query is in progress. This will most often be triggered by panic when
		// a defer closes the connection (possibly indirectly via a transaction or a connection pool). Unwatch to end any
		// previous watch. It is safe to Unwatch regardless of whether a watch is already is progress.
		//
		// See https://github.com/jackc/fbconn/issues/29
		fbConn.contextWatcher.Unwatch()

		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	// Ignore any errors sending Terminate message and waiting for server to close connection.
	fbConn.client.WriteMessage(&fbproto.OpDetach{})
	fbConn.client.WriteMessage(&fbproto.OpDisconnect{})
	fbConn.flushWithPotentialWriteReadDeadlock()
	//FIXME: should we wait or not response from server?
	//fbConn.receiveMessage()
	return fbConn.conn.Close()
}

// asyncClose marks the connection as closed and asynchronously sends a cancel query message and closes the underlying
// connection.
func (fbConn *FbConn) asyncClose() {
	if fbConn.status == connStatusClosed {
		return
	}
	fbConn.status = connStatusClosed

	go func() {
		defer close(fbConn.cleanupDone)
		defer fbConn.conn.Close()

		//FIXME: fbx
		//deadline := time.Now().Add(time.Second * 15)

		//ctx, cancel := context.WithDeadline(context.Background(), deadline)
		//defer cancel()
		//
		//fbConn.CancelRequest(ctx)

		//fbConn.conn.SetDeadline(deadline)
		//
		//fbConn.frontend.WriteMessage(&pgproto3.Terminate{})
		//fbConn.flushWithPotentialWriteReadDeadlock()
	}()
}

// CleanupDone returns a channel that will be closed after all underlying resources have been cleaned up. A closed
// connection is no longer usable, but underlying resources, in particular the net.Conn, may not have finished closing
// yet. This is because certain errors such as a context cancellation require that the interrupted function call return
// immediately, but the error may also cause the connection to be closed. In these cases the underlying resources are
// closed asynchronously.
//
// This is only likely to be useful to connection pools. It gives them a way avoid establishing a new connection while
// an old connection is still being cleaned up and thereby exceeding the maximum pool size.
func (fbConn *FbConn) CleanupDone() chan (struct{}) {
	return fbConn.cleanupDone
}

// IsClosed reports if the connection has been closed.
//
// CleanupDone() can be used to determine if all cleanup has been completed.
func (fbConn *FbConn) IsClosed() bool {
	return fbConn.status < connStatusIdle
}

// IsBusy reports if the connection is busy.
func (fbConn *FbConn) IsBusy() bool {
	return fbConn.status == connStatusBusy
}

// lock locks the connection.
func (fbConn *FbConn) lock() error {
	switch fbConn.status {
	case connStatusBusy:
		return &connLockError{status: "conn busy"} // This only should be possible in case of an application bug.
	case connStatusClosed:
		return &connLockError{status: "conn closed"}
	case connStatusUninitialized:
		return &connLockError{status: "conn uninitialized"}
	}
	fbConn.status = connStatusBusy
	return nil
}

func (fbConn *FbConn) unlock() {
	switch fbConn.status {
	case connStatusBusy:
		fbConn.status = connStatusIdle
	case connStatusClosed:
	default:
		panic("BUG: cannot unlock unlocked connection") // This should only be possible if there is a bug in this package.
	}
}

// ParameterStatus returns the value of a parameter reported by the server (e.g.
// server_version). Returns an empty string for unknown parameters.
func (fbConn *FbConn) ParameterStatus(key string) string {
	return fbConn.parameterStatuses[key]
}

// QueryStatus simple query info
type QueryStatus struct {
	statementType int
}

// RowsAffected returns the number of rows affected
func (ct QueryStatus) RowsAffected() int64 {
	return 0
}

func (ct QueryStatus) String() string {
	switch ct.statementType {
	case StatementSelect:
		return "SELECT"
	case StatementInsert:
		return "INSERT"
	case StatementUpdate:
		return "UPDATE"
	case StatementDelete:
		return "DELETE"
	case StatementDDL:
		return "DDL"
	case StatementGetSegment:
		return "GET SEGMENT"
	case StatementPutSegment:
		return "PUT SEGMENT"
	case StatementProcedure:
		return "PROCEDURE"
	case StatementStartTransaction:
		return "START TRANSACTION"
	case StatementCommit:
		return "COMMIT"
	case StatementRollback:
		return "ROLLBACK"
	case StatementSelectForUpdate:
		return "SELECT FOR UPDATE"
	case StatementSetGenerator:
		return "SET GENERATOR"
	case StatementSavepoint:
		return "SAVEPOINT"
	}
	return ""
}

func (ct QueryStatus) Insert() bool {
	return ct.statementType == StatementInsert
}

func (ct QueryStatus) Update() bool {
	return ct.statementType == StatementUpdate
}

func (ct QueryStatus) Delete() bool {
	return ct.statementType == StatementDelete
}

func (ct QueryStatus) Select() bool {
	return ct.statementType == StatementSelect
}
func (ct QueryStatus) DDL() bool {
	return ct.statementType == StatementDDL
}

type FieldDescription struct {
	Index             int
	Type              uint32
	Format            int16 // 0 - text, 1 - binary
	Scale             int
	Subtype           int
	Length            int
	NullOk            bool
	Name              string
	TableAliasName    string
	RelName           string
	OwnerName         string
	OriginalName      string
	OriginalTableName string
	Nullable          bool
}

func (fbConn *FbConn) convertRowDescription(src []fbproto.FieldDescriptor) []FieldDescription {
	res := make([]FieldDescription, len(src))

	for i := range src {
		res[i].Index = src[i].Index
		res[i].Type = uint32(src[i].Type)
		res[i].Format = src[i].Format
		res[i].Scale = src[i].Scale
		res[i].Subtype = src[i].Subtype
		res[i].Length = src[i].Length
		res[i].NullOk = src[i].NullOk
		res[i].Name = src[i].Name
		res[i].TableAliasName = src[i].TableAliasName
		res[i].RelName = src[i].RelName
		res[i].OwnerName = src[i].OwnerName
		res[i].OriginalName = src[i].OriginalName
		res[i].OriginalTableName = src[i].OriginalTableName
		res[i].Nullable = src[i].Nullable
	}

	return res
}

// Prepare creates a prepared statement inside current internal transaction
func (fbConn *FbConn) Prepare(ctx context.Context, sql string) (*Statement, error) {
	tx, err := fbConn.ensureInternalTransaction(ctx, nil)
	if err != nil {
		return nil, err
	}
	st, err := fbConn.PrepareTx(ctx, tx, sql)
	if err != nil {
		return nil, err
	}
	return st, nil
}

// PrepareTx creates a prepared statement
func (fbConn *FbConn) PrepareTx(ctx context.Context, tx *FBTransaction, sql string) (*Statement, error) {
	var err error
	if err := fbConn.lock(); err != nil {
		return nil, err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return nil, newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	if err = fbConn.client.WriteMessage(&fbproto.OpAllocateStatement{}); err != nil {
		fbConn.asyncClose()
		return nil, err
	}
	if err = fbConn.client.WriteMessage(&fbproto.OpPrepareStatement{
		Encoding:          fbConn.Encoding,
		TransactionHandle: tx.Handle,
		StatementHandle:   0xFFFF,
		Dialect:           3,
		Query:             sql,
	}); err != nil {
		fbConn.asyncClose()
		return nil, err
	}
	if err = fbConn.writer.Flush(); err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	var statementHandle int
	switch m := msg.(type) {
	case *fbproto.OpResponse:
		statementHandle = m.Object
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	}

	msg, err = fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return nil, err
	}
	switch m := msg.(type) {
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	}

	statementInfo, err := fbConn.getStatementInfo(statementHandle)
	if err != nil {
		var fbError *FbError
		switch {
		case errors.As(err, &fbError):
			return nil, err
		default:
			fbConn.asyncClose()
			return nil, err
		}
	}

	desc := &StatementDescription{
		SQL:             sql,
		Fields:          fbConn.convertRowDescription(statementInfo.FieldDescriptors),
		Parameters:      fbConn.convertRowDescription(statementInfo.ParametersDescriptors),
		StatementHandle: statementHandle,
		StatementInfo:   statementInfo,
	}

	if tx == fbConn.internalTransaction {
		return &Statement{conn: fbConn, Description: desc}, nil
	} else {
		return &Statement{conn: fbConn, transaction: tx, Description: desc}, nil
	}
}

//func noticeResponseToNotice(msg *pgproto3.NoticeResponse) *Notice {
//pgerr := ErrorResponseToPgError((*pgproto3.ErrorResponse)(msg))
//return (*Notice)(pgerr)
//panic("implement me")
//}

// CancelRequest sends a cancel request to the PostgreSQL server. It returns an error if unable to deliver the cancel
// request, but lack of an error does not ensure that the query was canceled. As specified in the documentation, there
// is no way to be sure a query was canceled. See https://www.postgresql.org/docs/11/protocol-flow.html#id-1.10.5.7.9
func (fbConn *FbConn) CancelRequest(ctx context.Context) error {
	//// Open a cancellation request to the same server. The address is taken from the net.Conn directly instead of reusing
	//// the connection config. This is important in high availability configurations where fallback connections may be
	//// specified or DNS may be used to load balance.
	//serverAddr := fbConn.conn.RemoteAddr()
	//var serverNetwork string
	//var serverAddress string
	//if serverAddr.Network() == "unix" {
	//	// for unix sockets, RemoteAddr() calls getpeername() which returns the name the
	//	// server passed to bind(). For Postgres, this is always a relative path "./.s.PGSQL.5432"
	//	// so connecting to it will fail. Fall back to the config's value
	//	serverNetwork, serverAddress = NetworkAddress(fbConn.config.Host, fbConn.config.Port)
	//} else {
	//	serverNetwork, serverAddress = serverAddr.Network(), serverAddr.String()
	//}
	//cancelConn, err := fbConn.config.DialFunc(ctx, serverNetwork, serverAddress)
	//if err != nil {
	//	// In case of unix sockets, RemoteAddr() returns only the file part of the path. If the
	//	// first connect failed, try the config.
	//	if serverAddr.Network() != "unix" {
	//		return err
	//	}
	//	serverNetwork, serverAddr := NetworkAddress(fbConn.config.Host, fbConn.config.Port)
	//	cancelConn, err = fbConn.config.DialFunc(ctx, serverNetwork, serverAddr)
	//	if err != nil {
	//		return err
	//	}
	//}
	//defer cancelConn.Close()
	//
	//if ctx != context.Background() {
	//	contextWatcher := ctxwatch.NewContextWatcher(&DeadlineContextWatcherHandler{Conn: cancelConn})
	//	contextWatcher.Watch(ctx)
	//	defer contextWatcher.Unwatch()
	//}
	//
	//buf := make([]byte, 16)
	//binary.BigEndian.PutUint32(buf[0:4], 16)
	//binary.BigEndian.PutUint32(buf[4:8], 80877102)
	//binary.BigEndian.PutUint32(buf[8:12], fbConn.pid)
	//binary.BigEndian.PutUint32(buf[12:16], fbConn.secretKey)
	//
	//if _, err := cancelConn.Write(buf); err != nil {
	//	return fmt.Errorf("write to connection for cancellation: %w", err)
	//}
	//
	//// Wait for the cancel request to be acknowledged by the server.
	//// It copies the behavior of the libpq: https://github.com/postgres/postgres/blob/REL_16_0/src/interfaces/libpq/fe-connect.c#L4946-L4960
	//_, _ = cancelConn.Read(buf)
	//
	//return nil
	panic("implement me")
}

// WaitForNotification waits for a LISTEN/NOTIFY message to be received. It returns an error if a notification was not
// received.
func (fbConn *FbConn) WaitForNotification(ctx context.Context) error {
	//if err := fbConn.lock(); err != nil {
	//	return err
	//}
	//defer fbConn.unlock()
	//
	//if ctx != context.Background() {
	//	select {
	//	case <-ctx.Done():
	//		return newContextAlreadyDoneError(ctx)
	//	default:
	//	}
	//
	//	fbConn.contextWatcher.Watch(ctx)
	//	defer fbConn.contextWatcher.Unwatch()
	//}
	//
	//for {
	//	msg, err := fbConn.receiveMessage()
	//	if err != nil {
	//		return normalizeTimeoutError(ctx, err)
	//	}
	//
	//	switch msg.(type) {
	//	case *pgproto3.NotificationResponse:
	//		return nil
	//	}
	//}
	panic("implement me")
}

// ExecImmediate executes statement immediate inside internal transaction
func (fbConn *FbConn) ExecImmediate(ctx context.Context, sql string) error {
	tx, err := fbConn.ensureInternalTransaction(ctx, nil)
	if err != nil {
		fbConn.asyncClose()
		return err
	}
	return fbConn.ExecImmediateTx(ctx, tx, sql)
}

// ExecImmediateTx executes statement immediate inside specified transaction
func (fbConn *FbConn) ExecImmediateTx(ctx context.Context, tx *FBTransaction, sql string) error {
	var err error

	if err = fbConn.lock(); err != nil {
		return err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	txHandle := 0
	if tx != nil {
		txHandle = tx.Handle
	}
	if err = fbConn.client.WriteMessage(&fbproto.OpExecuteImmediate{
		Encoding:          fbConn.Encoding,
		TransactionHandle: txHandle,
		Dialect:           3,
		Query:             sql,
	}); err != nil {
		fbConn.asyncClose()
		return err
	}

	if err = fbConn.writer.Flush(); err != nil {
		fbConn.asyncClose()
		return err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return err
	}

	switch m := msg.(type) {
	case *fbproto.OpResponse:
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		panic(fmt.Sprintf("wrong response message type %T instead expected OpResponse or ErrorResponse", m))
	}
	return nil
}

// Exec executes SQL via the PostgreSQL simple query protocol. SQL may contain multiple queries. Execution is
// implicitly wrapped in a transaction unless a transaction is already in progress or SQL contains transaction control
// statements.
//
// Prefer ExecParams unless executing arbitrary SQL that may contain multiple queries.
func (fbConn *FbConn) Exec(ctx context.Context, sql string) *ResultReader {
	return fbConn.ExecParams(ctx, sql, nil)
}

func (fbConn *FbConn) ExecTx(ctx context.Context, tx *FBTransaction, sql string) *ResultReader {
	return fbConn.ExecParamsTx(ctx, tx, sql, nil)
}

// ExecParams executes a command via the PostgreSQL extended query protocol.
//
// sql is a SQL command string. It may only contain one query. Parameter substitution is positional using $1, $2, $3,
// etc.
//
// paramValues are the parameter values. It must be encoded in the format given by paramFormats.
//
// paramOIDs is a slice of data type OIDs for paramValues. If paramOIDs is nil, the server will infer the data type for
// all parameters. Any paramOID element that is 0 that will cause the server to infer the data type for that parameter.
// ExecParams will panic if len(paramOIDs) is not 0, 1, or len(paramValues).
//
// paramFormats is a slice of format codes determining for each paramValue column whether it is encoded in text or
// binary format. If paramFormats is nil all params are text format. ExecParams will panic if
// len(paramFormats) is not 0, 1, or len(paramValues).
//
// resultFormats is a slice of format codes determining for each result column whether it is encoded in text or
// binary format. If resultFormats is nil all results will be in text format.
//
// ResultReader must be closed before FbConn can be used again.
func (fbConn *FbConn) ExecParams(ctx context.Context, sql string, paramValues [][]byte) *ResultReader {
	tx, err := fbConn.ensureInternalTransaction(ctx, nil)
	if err != nil {
		// early check lock state and do not close connection
		closeConn := true
		var fbErr *FbError
		if SafeToRetry(err) || errors.As(err, &fbErr) {
			closeConn = false
		}
		res := ResultReader{
			fbConn: fbConn,
			ctx:    ctx,
			closed: true,
		}
		if closeConn {
			fbConn.asyncClose()
		}
		res.concludeCommand(QueryStatus{}, err)
		return &res
	}
	return fbConn.ExecParamsTx(ctx, tx, sql, paramValues)
}

func (fbConn *FbConn) ExecParamsTx(ctx context.Context, tx *FBTransaction, sql string, paramValues [][]byte) *ResultReader {
	st, err := fbConn.PrepareTx(ctx, tx, sql)
	if err != nil {
		// early check lock state and do not close connection
		closeConn := true
		var fbErr *FbError
		if SafeToRetry(err) || errors.As(err, &fbErr) {
			closeConn = false
		}
		if closeConn {
			_ = tx.Rollback(ctx)
		}
		res := ResultReader{
			fbConn: fbConn,
			ctx:    ctx,
			closed: true,
		}
		if closeConn {
			fbConn.asyncClose()
		}
		res.concludeCommand(QueryStatus{}, err)
		return &res
	}

	rr := st.Exec(ctx, paramValues)
	if rr.err != nil {
		_ = tx.Rollback(ctx)
		res := ResultReader{
			fbConn: fbConn,
			ctx:    ctx,
			closed: true,
		}
		fbConn.asyncClose()
		res.concludeCommand(QueryStatus{}, rr.err)
		return rr
	}
	return rr
}

//func (fbConn *FbConn) execExtendedPrefix(ctx context.Context, paramValues [][]byte) *ResultReader {
//
//	//if len(paramValues) > math.MaxUint16 {
//	//	result.concludeCommand(QueryStatus{}, fmt.Errorf("extended protocol limited to %v parameters", math.MaxUint16))
//	//	result.closed = true
//	//	fbConn.unlock()
//	//	return result
//	//}
//
//	return result
//}

//func (fbConn *FbConn) execExtendedSuffix(statement *StatementDescription, result *ResultReader) {
//
//	//result.readUntilRowDescription()
//}

// CopyTo executes the copy command sql and copies the results to w.
func (fbConn *FbConn) CopyTo(ctx context.Context, w io.Writer, sql string) (QueryStatus, error) {
	//if err := fbConn.lock(); err != nil {
	//	return QueryStatus{}, err
	//}
	//
	//if ctx != context.Background() {
	//	select {
	//	case <-ctx.Done():
	//		fbConn.unlock()
	//		return QueryStatus{}, newContextAlreadyDoneError(ctx)
	//	default:
	//	}
	//	fbConn.contextWatcher.Watch(ctx)
	//	defer fbConn.contextWatcher.Unwatch()
	//}
	//
	//// WriteMessage copy to command
	//fbConn.frontend.SendQuery(&pgproto3.Query{String: sql})
	//
	//err := fbConn.flushWithPotentialWriteReadDeadlock()
	//if err != nil {
	//	fbConn.asyncClose()
	//	fbConn.unlock()
	//	return QueryStatus{}, err
	//}
	//
	//// Read results
	//var commandTag QueryStatus
	//var fbErr error
	//for {
	//	msg, err := fbConn.receiveMessage()
	//	if err != nil {
	//		fbConn.asyncClose()
	//		return QueryStatus{}, normalizeTimeoutError(ctx, err)
	//	}
	//
	//	switch msg := msg.(type) {
	//	case *pgproto3.CopyDone:
	//	case *pgproto3.CopyData:
	//		_, err := w.Write(msg.Data)
	//		if err != nil {
	//			fbConn.asyncClose()
	//			return QueryStatus{}, err
	//		}
	//	case *pgproto3.ReadyForQuery:
	//		fbConn.unlock()
	//		return commandTag, fbErr
	//	case *pgproto3.CommandComplete:
	//		commandTag = fbConn.makeCommandTag(msg.QueryStatus)
	//	case *pgproto3.ErrorResponse:
	//		fbErr = ErrorResponseToPgError(msg)
	//	}
	//}
	panic("implement me")
}

// CopyFrom executes the copy command sql and copies all of r to the PostgreSQL server.
//
// Note: context cancellation will only interrupt operations on the underlying PostgreSQL network connection. Reads on r
// could still block.
func (fbConn *FbConn) CopyFrom(ctx context.Context, r io.Reader, sql string) (QueryStatus, error) {
	//if err := fbConn.lock(); err != nil {
	//	return QueryStatus{}, err
	//}
	//defer fbConn.unlock()
	//
	//if ctx != context.Background() {
	//	select {
	//	case <-ctx.Done():
	//		return QueryStatus{}, newContextAlreadyDoneError(ctx)
	//	default:
	//	}
	//	fbConn.contextWatcher.Watch(ctx)
	//	defer fbConn.contextWatcher.Unwatch()
	//}
	//
	//// WriteMessage copy from query
	//fbConn.frontend.SendQuery(&pgproto3.Query{String: sql})
	//err := fbConn.flushWithPotentialWriteReadDeadlock()
	//if err != nil {
	//	fbConn.asyncClose()
	//	return QueryStatus{}, err
	//}
	//
	//// WriteMessage copy data
	//abortCopyChan := make(chan struct{})
	//copyErrChan := make(chan error, 1)
	//signalMessageChan := fbConn.signalMessage()
	//var wg sync.WaitGroup
	//wg.Add(1)
	//
	//go func() {
	//	defer wg.Done()
	//	buf := iobufpool.Get(65536)
	//	defer iobufpool.Put(buf)
	//	(*buf)[0] = 'd'
	//
	//	for {
	//		n, readErr := r.Read((*buf)[5:cap(*buf)])
	//		if n > 0 {
	//			*buf = (*buf)[0 : n+5]
	//			pgio.SetInt32((*buf)[1:], int32(n+4))
	//
	//			writeErr := fbConn.frontend.SendUnbufferedEncodedCopyData(*buf)
	//			if writeErr != nil {
	//				// Write errors are always fatal, but we can't use asyncClose because we are in a different goroutine. Not
	//				// setting fbConn.status or closing fbConn.cleanupDone for the same reason.
	//				fbConn.conn.Close()
	//
	//				copyErrChan <- writeErr
	//				return
	//			}
	//		}
	//		if readErr != nil {
	//			copyErrChan <- readErr
	//			return
	//		}
	//
	//		select {
	//		case <-abortCopyChan:
	//			return
	//		default:
	//		}
	//	}
	//}()
	//
	//var fbErr error
	//var copyErr error
	//for copyErr == nil && fbErr == nil {
	//	select {
	//	case copyErr = <-copyErrChan:
	//	case <-signalMessageChan:
	//		// If fbConn.receiveMessage encounters an error it will call fbConn.asyncClose. But that is a race condition with
	//		// the goroutine. So instead check fbConn.bufferingReceiveErr which will have been set by the signalMessage. If an
	//		// error is found then forcibly close the connection without sending the Terminate message.
	//		if err := fbConn.bufferingReceiveErr; err != nil {
	//			fbConn.status = connStatusClosed
	//			fbConn.conn.Close()
	//			close(fbConn.cleanupDone)
	//			return QueryStatus{}, normalizeTimeoutError(ctx, err)
	//		}
	//		msg, _ := fbConn.receiveMessage()
	//
	//		switch msg := msg.(type) {
	//		case *pgproto3.ErrorResponse:
	//			fbErr = ErrorResponseToPgError(msg)
	//		default:
	//			signalMessageChan = fbConn.signalMessage()
	//		}
	//	}
	//}
	//close(abortCopyChan)
	//// Make sure io goroutine finishes before writing.
	//wg.Wait()
	//
	//if copyErr == io.EOF || fbErr != nil {
	//	fbConn.frontend.WriteMessage(&pgproto3.CopyDone{})
	//} else {
	//	fbConn.frontend.WriteMessage(&pgproto3.CopyFail{Response: copyErr.Error()})
	//}
	//err = fbConn.flushWithPotentialWriteReadDeadlock()
	//if err != nil {
	//	fbConn.asyncClose()
	//	return QueryStatus{}, err
	//}
	//
	//// Read results
	//var commandTag QueryStatus
	//for {
	//	msg, err := fbConn.receiveMessage()
	//	if err != nil {
	//		fbConn.asyncClose()
	//		return QueryStatus{}, normalizeTimeoutError(ctx, err)
	//	}
	//
	//	switch msg := msg.(type) {
	//	case *pgproto3.ReadyForQuery:
	//		return commandTag, fbErr
	//	case *pgproto3.CommandComplete:
	//		commandTag = fbConn.makeCommandTag(msg.QueryStatus)
	//	case *pgproto3.ErrorResponse:
	//		fbErr = ErrorResponseToPgError(msg)
	//	}
	//}
	panic("implement me")
}

// Commit commits internal implicit transaction
func (fbConn *FbConn) Commit(ctx context.Context) error {
	if fbConn.explicitTransaction {
		return nil
	}

	if fbConn.internalTransaction == nil || fbConn.internalTransaction.Closed {
		return ErrTransactionClosed
	}
	return fbConn.internalTransaction.Commit(ctx)
}

// Rollback rollbacks internal implicit transaction
func (fbConn *FbConn) Rollback(ctx context.Context) error {
	if fbConn.explicitTransaction {
		return nil
	}

	if fbConn.internalTransaction == nil || fbConn.internalTransaction.Closed {
		return ErrTransactionClosed
	}
	return fbConn.internalTransaction.Rollback(ctx)
}

// Disconnect ask server to close remote port
func (fbConn *FbConn) Disconnect(ctx context.Context) error {
	var err error
	if err = fbConn.lock(); err != nil {
		return err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	if err = fbConn.client.WriteMessage(&fbproto.OpDisconnect{}); err != nil {
		fbConn.asyncClose()
		return err
	}

	if err = fbConn.writer.Flush(); err != nil {
		fbConn.conn.Close()
		return err
	}

	return nil
}

// MultiResultReader is a reader for a command that could return multiple results such as Exec or ExecBatch.
type MultiResultReader struct {
	fbConn *FbConn
	ctx    context.Context

	rr *ResultReader

	closed bool
	err    error
}

// ReadAll reads all available results. Calling ReadAll is mutually exclusive with all other MultiResultReader methods.
func (mrr *MultiResultReader) ReadAll() ([]*Result, error) {
	var results []*Result

	for mrr.NextResult() {
		results = append(results, mrr.ResultReader().Read())
	}
	err := mrr.Close()

	return results, err
}

func (mrr *MultiResultReader) receiveMessage() (fbproto.ServerMessage, error) {
	//msg, err := mrr.fbConn.receiveMessage()
	//if err != nil {
	//	mrr.fbConn.contextWatcher.Unwatch()
	//	mrr.err = normalizeTimeoutError(mrr.ctx, err)
	//	mrr.closed = true
	//	mrr.fbConn.asyncClose()
	//	return nil, mrr.err
	//}
	//
	//switch msg := msg.(type) {
	//case *pgproto3.ReadyForQuery:
	//	mrr.closed = true
	//	mrr.fbConn.contextWatcher.Unwatch()
	//	mrr.fbConn.unlock()
	//case *pgproto3.ErrorResponse:
	//	mrr.err = ErrorResponseToPgError(msg)
	//}
	//
	//return msg, nil
	panic("implement me")
}

// NextResult returns advances the MultiResultReader to the next result and returns true if a result is available.
func (mrr *MultiResultReader) NextResult() bool {
	//for !mrr.closed && mrr.err == nil {
	//	msg, err := mrr.receiveMessage()
	//	if err != nil {
	//		return false
	//	}
	//
	//	switch msg := msg.(type) {
	//	case *pgproto3.RowDescription:
	//		mrr.fbConn.resultReader = ResultReader{
	//			fbConn:            mrr.fbConn,
	//			multiResultReader: mrr,
	//			ctx:               mrr.ctx,
	//			fieldDescriptions: mrr.fbConn.convertRowDescription(mrr.fbConn.fieldDescriptions[:], msg),
	//		}
	//
	//		mrr.rr = &mrr.fbConn.resultReader
	//		return true
	//	case *pgproto3.CommandComplete:
	//		mrr.fbConn.resultReader = ResultReader{
	//			commandTag:       mrr.fbConn.makeCommandTag(msg.QueryStatus),
	//			commandConcluded: true,
	//			closed:           true,
	//		}
	//		mrr.rr = &mrr.fbConn.resultReader
	//		return true
	//	case *pgproto3.EmptyQueryResponse:
	//		return false
	//	}
	//}
	//
	//return false
	panic("implement me")
}

// ResultReader returns the current ResultReader.
func (mrr *MultiResultReader) ResultReader() *ResultReader {
	return mrr.rr
}

// Close closes the MultiResultReader and returns the first error that occurred during the MultiResultReader's use.
func (mrr *MultiResultReader) Close() error {
	for !mrr.closed {
		_, err := mrr.receiveMessage()
		if err != nil {
			return mrr.err
		}
	}

	return mrr.err
}

// ResultReader is a reader for the result of a single query.
type ResultReader struct {
	fbConn *FbConn
	ctx    context.Context

	fieldDescriptions []FieldDescription
	rowValues         [][]byte
	queryStatus       QueryStatus
	commandConcluded  bool
	closed            bool
	err               error

	statementDescription *StatementDescription
	singleton            *fbproto.SQLData
	singletonResult      bool
	cursorOpened         bool
	fetchExecuted        bool
	currentFetch         [][][]byte
	currentFetchPos      int
	eoc                  bool
}

// Result is the saved query response that is returned by calling Read on a ResultReader.
type Result struct {
	FieldDescriptions []FieldDescription
	Rows              [][][]byte
	QueryStatus       QueryStatus
	Err               error
}

// Read saves the query response to a Result.
func (rr *ResultReader) Read() *Result {
	br := &Result{}

	for rr.NextRow() {
		if br.FieldDescriptions == nil {
			br.FieldDescriptions = make([]FieldDescription, len(rr.FieldDescriptions()))
			copy(br.FieldDescriptions, rr.FieldDescriptions())
		}

		values := rr.Values()
		row := make([][]byte, len(values))
		for i := range row {
			if values[i] != nil {
				row[i] = make([]byte, len(values[i]))
				copy(row[i], values[i])
			}
		}
		br.Rows = append(br.Rows, row)
	}

	br.QueryStatus, br.Err = rr.Close()

	return br
}

// ReadAll for compatibility with pgx code
// It will always return single result from ResultReader and its error
func (rr *ResultReader) ReadAll() ([]*Result, error) {
	var results []*Result
	results = append(results, rr.Read())
	return results, results[0].Err
}

// NextRow advances the ResultReader to the next row and returns true if a row is available.
func (rr *ResultReader) NextRow() bool {
	var err error
	if rr.singletonResult {
		if rr.singleton == nil {
			return false
		}
		rr.rowValues = rr.singleton.RowData
		rr.singleton = nil
		return true
	}

	if !rr.cursorOpened {
		return false
	}

	if rr.currentFetch != nil {
		if rr.currentFetchPos+1 < len(rr.currentFetch) {
			rr.currentFetchPos += 1
			rr.rowValues = rr.currentFetch[rr.currentFetchPos]
			return true
		}
	}

	if rr.eoc {
		return false
	}

	var rowBlr []byte
	if !rr.fetchExecuted {
		rowBlr = rr.statementDescription.StatementInfo.GetRowBlr()
	}
	if err = rr.fbConn.client.WriteMessage(&fbproto.OpFetch{
		StatementHandle: rr.statementDescription.StatementHandle,
		RowBlr:          rowBlr,
		Messages:        400,
	}); err != nil {
		return false
	}
	if err = rr.fbConn.writer.Flush(); err != nil {
		return false
	}
	rr.fetchExecuted = true

	var fetchResp *fbproto.OpFetchResponse
	msg, err := rr.fbConn.receiveMessage()
	if err != nil {
		rr.fbConn.asyncClose()
		rr.err = err
		return false
	}
	switch m := msg.(type) {
	case *fbproto.OpFetchResponse:
		fetchResp = m
	case *fbproto.ErrorResponse:
		rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(m))
		return false
	default:
		panic(fmt.Sprintf("wrong response message type %T instead expected OpFetchResponse or ErrorResponse", m))
	}
	rr.currentFetch = nil
	rr.currentFetchPos = 0

	if fetchResp.EndOfCursor() {
		rr.eoc = true
		return false
	}

	for fetchResp.MoreRows() {
		row, msg, err := rr.fbConn.client.ReceiveSQLData(rr.statementDescription.StatementInfo.FieldDescriptors)
		if err != nil {
			rr.fbConn.asyncClose()
			rr.concludeCommand(QueryStatus{}, err)
			return false
		}
		switch m := msg.(type) {
		case *fbproto.OpFetchResponse:
			fetchResp = m
		case *fbproto.ErrorResponse:
			rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(m))
			return false
		default:
			panic(fmt.Sprintf("wrong response message type %T instead expected OpFetchResponse or ErrorResponse", m))
		}
		rr.currentFetch = append(rr.currentFetch, row.RowData)
	}
	rr.eoc = fetchResp.EndOfCursor()
	rr.currentFetchPos = 0
	rr.rowValues = rr.currentFetch[rr.currentFetchPos]

	return true
}

// FieldDescriptions returns the field descriptions for the current result set. The returned slice is only valid until
// the ResultReader is closed. It may return nil (for example, if the query did not return a result set or an error was
// encountered.)
func (rr *ResultReader) FieldDescriptions() []FieldDescription {
	return rr.fieldDescriptions
}

// Values returns the current row data. NextRow must have been previously been called. The returned [][]byte is only
// valid until the next NextRow call or the ResultReader is closed.
func (rr *ResultReader) Values() [][]byte {
	return rr.rowValues
}

// Close consumes any remaining result data and returns the command tag or
// error.
func (rr *ResultReader) Close() (QueryStatus, error) {
	if !rr.closed {
		rr.fbConn.unlock()
		rr.fbConn.contextWatcher.Unwatch()
	}
	return rr.queryStatus, rr.err
}

func (rr *ResultReader) concludeCommand(queryStatus QueryStatus, err error) {
	// Keep the first error that is recorded. Store the error before checking if the command is already concluded to
	// allow for receiving an error after CommandComplete but before ReadyForQuery.
	if err != nil && rr.err == nil {
		rr.err = err
	}

	if rr.commandConcluded {
		return
	}

	rr.queryStatus = queryStatus
	rr.rowValues = nil
	rr.commandConcluded = true
}

// Batch is a collection of queries that can be sent to the PostgreSQL server in a single round-trip.
type Batch struct {
	buf []byte
	err error
}

// ExecParams appends an ExecParams command to the batch. See FbConn.ExecParams for parameter descriptions.
func (batch *Batch) ExecParams(sql string, paramValues [][]byte, paramOIDs []uint32, paramFormats []int16, resultFormats []int16) {
	//if batch.err != nil {
	//	return
	//}
	//
	//batch.buf, batch.err = (&pgproto3.Parse{Query: sql, ParameterOIDs: paramOIDs}).Encode(batch.buf)
	//if batch.err != nil {
	//	return
	//}
	//batch.ExecPrepared("", paramValues, paramFormats, resultFormats)
}

// ExecPrepared appends an ExecPrepared e command to the batch. See FbConn.ExecPrepared for parameter descriptions.
func (batch *Batch) ExecPrepared(stmtName string, paramValues [][]byte, paramFormats []int16, resultFormats []int16) {
	//if batch.err != nil {
	//	return
	//}
	//
	//batch.buf, batch.err = (&pgproto3.Bind{PreparedStatement: stmtName, ParameterFormatCodes: paramFormats, Parameters: paramValues, ResultFormatCodes: resultFormats}).Encode(batch.buf)
	//if batch.err != nil {
	//	return
	//}
	//
	//batch.buf, batch.err = (&pgproto3.Describe{ObjectType: 'P'}).Encode(batch.buf)
	//if batch.err != nil {
	//	return
	//}
	//
	//batch.buf, batch.err = (&pgproto3.Execute{}).Encode(batch.buf)
	//if batch.err != nil {
	//	return
	//}
}

// ExecBatch executes all the queries in batch in a single round-trip. Execution is implicitly transactional unless a
// transaction is already in progress or SQL contains transaction control statements. This is a simpler way of executing
// multiple queries in a single round trip than using pipeline mode.
func (fbConn *FbConn) ExecBatch(ctx context.Context, batch *Batch) *MultiResultReader {
	//if batch.err != nil {
	//	return &MultiResultReader{
	//		closed: true,
	//		err:    batch.err,
	//	}
	//}
	//
	//if err := fbConn.lock(); err != nil {
	//	return &MultiResultReader{
	//		closed: true,
	//		err:    err,
	//	}
	//}
	//
	//fbConn.multiResultReader = MultiResultReader{
	//	fbConn: fbConn,
	//	ctx:    ctx,
	//}
	//multiResult := &fbConn.multiResultReader
	//
	//if ctx != context.Background() {
	//	select {
	//	case <-ctx.Done():
	//		multiResult.closed = true
	//		multiResult.err = newContextAlreadyDoneError(ctx)
	//		fbConn.unlock()
	//		return multiResult
	//	default:
	//	}
	//	fbConn.contextWatcher.Watch(ctx)
	//}
	//
	//batch.buf, batch.err = (&pgproto3.Sync{}).Encode(batch.buf)
	//if batch.err != nil {
	//	fbConn.contextWatcher.Unwatch()
	//	multiResult.err = normalizeTimeoutError(multiResult.ctx, batch.err)
	//	multiResult.closed = true
	//	fbConn.asyncClose()
	//	return multiResult
	//}
	//
	//fbConn.enterPotentialWriteReadDeadlock()
	//defer fbConn.exitPotentialWriteReadDeadlock()
	//_, err := fbConn.conn.Write(batch.buf)
	//if err != nil {
	//	fbConn.contextWatcher.Unwatch()
	//	multiResult.err = normalizeTimeoutError(multiResult.ctx, err)
	//	multiResult.closed = true
	//	fbConn.asyncClose()
	//	return multiResult
	//}
	//
	//return multiResult
	panic("implement me")
}

// EscapeString escapes a string such that it can safely be interpolated into a SQL command string. It does not include
// the surrounding single quotes.
//
// The current implementation requires that standard_conforming_strings=on and client_encoding="UTF8". If these
// conditions are not met an error will be returned. It is possible these restrictions will be lifted in the future.
func (fbConn *FbConn) EscapeString(s string) (string, error) {
	//FIXME: need to check anything for RDB?
	//if fbConn.ParameterStatus("client_encoding") != "UTF8" {
	//	return "", errors.New("EscapeString must be run with client_encoding=UTF8")
	//}

	return strings.Replace(s, "'", "''", -1), nil
}

// CheckConn checks the underlying connection without writing any bytes. This is currently implemented by doing a read
// with a very short deadline. This can be useful because a TCP connection can be broken such that a write will appear
// to succeed even though it will never actually reach the server. Reading immediately before a write will detect this
// condition. If this is done immediately before sending a query it reduces the chances a query will be sent that fails
// without the client knowing whether the server received it or not.
//
// Deprecated: CheckConn is deprecated in favor of Ping. CheckConn cannot detect all types of broken connections where
// the write would still appear to succeed. Prefer Ping unless on a high latency connection.
func (fbConn *FbConn) CheckConn() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := fbConn.ReceiveMessage(ctx)
	if err != nil {
		if !Timeout(err) {
			return err
		}
	}

	return nil
}

// Ping pings the server
func (fbConn *FbConn) Ping(ctx context.Context) error {
	var err error
	if err = fbConn.lock(); err != nil {
		return err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	if err = fbConn.client.WriteMessage(&fbproto.OpPing{}); err != nil {
		fbConn.asyncClose()
		return err
	}

	if err = fbConn.writer.Flush(); err != nil {
		fbConn.conn.Close()
		return err
	}

	_, err = fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return err
	}

	return nil
}

// makeQueryStatus makes a QueryStatus. It does not retain a reference to buf or buf's underlying memory.
func (fbConn *FbConn) makeQueryStatus(statementType int) QueryStatus {
	return QueryStatus{statementType: statementType}
}

// enterPotentialWriteReadDeadlock must be called before a write that could deadlock if the server is simultaneously
// blocked writing to us.
func (fbConn *FbConn) enterPotentialWriteReadDeadlock() {
	// The time to wait is somewhat arbitrary. A Write should only take as long as the syscall and memcpy to the OS
	// outbound network buffer unless the buffer is full (which potentially is a block). It needs to be long enough for
	// the normal case, but short enough not to kill performance if a block occurs.
	//
	// In addition, on Windows the default timer resolution is 15.6ms. So setting the timer to less than that is
	// ineffective.
	if fbConn.slowWriteTimer.Reset(15 * time.Millisecond) {
		panic("BUG: slow write timer already active")
	}
}

// exitPotentialWriteReadDeadlock must be called after a call to enterPotentialWriteReadDeadlock.
func (fbConn *FbConn) exitPotentialWriteReadDeadlock() {
	//if !fbConn.slowWriteTimer.Stop() {
	//	// The timer starts its function in a separate goroutine. It is necessary to ensure the background reader has
	//	// started before calling Stop. Otherwise, the background reader may not be stopped. That on its own is not a
	//	// serious problem. But what is a serious problem is that the background reader may start at an inopportune time in
	//	// a subsequent query. For example, if a subsequent query was canceled then a deadline may be set on the net.Conn to
	//	// interrupt an in-progress read. After the read is interrupted, but before the deadline is cleared, the background
	//	// reader could start and read a deadline error. Then the next query would receive the an unexpected deadline error.
	//	<-fbConn.bgReaderStarted
	//	fbConn.bgReader.Stop()
	//}
}

func (fbConn *FbConn) flushWithPotentialWriteReadDeadlock() error {
	fbConn.enterPotentialWriteReadDeadlock()
	defer fbConn.exitPotentialWriteReadDeadlock()
	err := fbConn.writer.Flush()
	return err
}

// SyncConn prepares the underlying net.Conn for direct use. FbConn may internally buffer reads or use goroutines for
// background IO. This means that any direct use of the underlying net.Conn may be corrupted if a read is already
// buffered or a read is in progress. SyncConn drains read buffers and stops background IO. In some cases this may
// require sending a ping to the server. ctx can be used to cancel this operation. This should be called before any
// operation that will use the underlying net.Conn directly. e.g. Before Conn() or Hijack().
//
// This should not be confused with the PostgreSQL protocol Sync message.
func (fbConn *FbConn) SyncConn(ctx context.Context) error {
	//for i := 0; i < 10; i++ {
	//	if fbConn.bgReader.Status() == bgreader.StatusStopped && fbConn.frontend.ReadBufferLen() == 0 {
	//		return nil
	//	}
	//
	//	err := fbConn.Ping(ctx)
	//	if err != nil {
	//		return fmt.Errorf("SyncConn: Ping failed while syncing conn: %w", err)
	//	}
	//}
	//
	//// This should never happen. Only way I can imagine this occurring is if the server is constantly sending data such as
	//// LISTEN/NOTIFY or log notifications such that we never can get an empty buffer.
	//return errors.New("SyncConn: conn never synchronized")
	panic("implement me")
}

// CustomData returns a map that can be used to associate custom data with the connection.
func (fbConn *FbConn) CustomData() map[string]any {
	return fbConn.customData
}

// HijackedConn is the result of hijacking a connection.
//
// Due to the necessary exposure of internal implementation details, it is not covered by the semantic versioning
// compatibility.
type HijackedConn struct {
	//Conn              net.Conn
	//PID               uint32            // backend pid
	//SecretKey         uint32            // key to use to send a cancel query message to the server
	//ParameterStatuses map[string]string // parameters that have been reported by the server
	//TxStatus          byte
	//Client          *pgproto3.Client
	//Config            *Config
	//CustomData        map[string]any
}

// Hijack extracts the internal connection data. fbConn must be in an idle state. SyncConn should be called immediately
// before Hijack. fbConn is unusable after hijacking. Hijacking is typically only useful when using fbconn to establish
// a connection, but taking complete control of the raw connection after that (e.g. a load balancer or proxy).
//
// Due to the necessary exposure of internal implementation details, it is not covered by the semantic versioning
// compatibility.
func (fbConn *FbConn) Hijack() (*HijackedConn, error) {
	//if err := fbConn.lock(); err != nil {
	//	return nil, err
	//}
	//fbConn.status = connStatusClosed
	//
	//return &HijackedConn{
	//	Conn:              fbConn.conn,
	//	PID:               fbConn.pid,
	//	SecretKey:         fbConn.secretKey,
	//	ParameterStatuses: fbConn.parameterStatuses,
	//	TxStatus:          fbConn.txStatus,
	//	Client:          fbConn.frontend,
	//	Config:            fbConn.config,
	//	CustomData:        fbConn.customData,
	//}, nil
	panic("implement me")
}

// Construct created a FbConn from an already established connection to a PostgreSQL server. This is the inverse of
// FbConn.Hijack. The connection must be in an idle state.
//
// hc.Client is replaced by a new pgproto3.Client built by hc.Config.BuildFrontend.
//
// Due to the necessary exposure of internal implementation details, it is not covered by the semantic versioning
// compatibility.
func Construct(hc *HijackedConn) (*FbConn, error) {
	//fbConn := &FbConn{
	//	conn:              hc.Conn,
	//	pid:               hc.PID,
	//	secretKey:         hc.SecretKey,
	//	parameterStatuses: hc.ParameterStatuses,
	//	txStatus:          hc.TxStatus,
	//	frontend:          hc.Client,
	//	config:            hc.Config,
	//	customData:        hc.CustomData,
	//
	//	status: connStatusIdle,
	//
	//	cleanupDone: make(chan struct{}),
	//}
	//
	//fbConn.contextWatcher = ctxwatch.NewContextWatcher(hc.Config.BuildContextWatcherHandler(fbConn))
	//fbConn.bgReader = bgreader.New(fbConn.conn)
	//fbConn.slowWriteTimer = time.AfterFunc(time.Duration(math.MaxInt64),
	//	func() {
	//		fbConn.bgReader.Start()
	//		fbConn.bgReaderStarted <- struct{}{}
	//	},
	//)
	//fbConn.slowWriteTimer.Stop()
	//fbConn.bgReaderStarted = make(chan struct{})
	//fbConn.frontend = hc.Config.BuildFrontend(fbConn.bgReader, fbConn.conn)
	//
	//return fbConn, nil
	panic("implement me")
}

// Pipeline represents a connection in pipeline mode.
//
// SendPrepare, SendQueryParams, and SendQueryPrepared queue requests to the server. These requests are not written until
// pipeline is flushed by Flush or Sync. Sync must be called after the last request is queued. Requests between
// synchronization points are implicitly transactional unless explicit transaction control statements have been issued.
//
// The context the pipeline was started with is in effect for the entire life of the Pipeline.
//
// For a deeper understanding of pipeline mode see the PostgreSQL documentation for the extended query protocol
// (https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-FLOW-EXT-QUERY) and the libpq pipeline mode
// (https://www.postgresql.org/docs/current/libpq-pipeline-mode.html).
type Pipeline struct {
	conn *FbConn
	ctx  context.Context

	state  pipelineState
	err    error
	closed bool
}

// PipelineSync is returned by GetResults when a ReadyForQuery message is received.
type PipelineSync struct{}

// CloseComplete is returned by GetResults when a CloseComplete message is received.
type CloseComplete struct{}

type pipelineRequestType int

const (
	pipelineNil pipelineRequestType = iota
	pipelinePrepare
	pipelineQueryParams
	pipelineQueryPrepared
	pipelineDeallocate
	pipelineSyncRequest
	pipelineFlushRequest
)

type pipelineRequestEvent struct {
	RequestType       pipelineRequestType
	WasSentToServer   bool
	BeforeFlushOrSync bool
}

type pipelineState struct {
	requestEventQueue          list.List
	lastRequestType            pipelineRequestType
	fbErr                      *FbError
	expectedReadyForQueryCount int
}

func (s *pipelineState) Init() {
	s.requestEventQueue.Init()
	s.lastRequestType = pipelineNil
}

func (s *pipelineState) RegisterSendingToServer() {
	for elem := s.requestEventQueue.Back(); elem != nil; elem = elem.Prev() {
		val := elem.Value.(pipelineRequestEvent)
		if val.WasSentToServer {
			return
		}
		val.WasSentToServer = true
		elem.Value = val
	}
}

func (s *pipelineState) registerFlushingBufferOnServer() {
	for elem := s.requestEventQueue.Back(); elem != nil; elem = elem.Prev() {
		val := elem.Value.(pipelineRequestEvent)
		if val.BeforeFlushOrSync {
			return
		}
		val.BeforeFlushOrSync = true
		elem.Value = val
	}
}

func (s *pipelineState) PushBackRequestType(req pipelineRequestType) {
	if req == pipelineNil {
		return
	}

	if req != pipelineFlushRequest {
		s.requestEventQueue.PushBack(pipelineRequestEvent{RequestType: req})
	}
	if req == pipelineFlushRequest || req == pipelineSyncRequest {
		s.registerFlushingBufferOnServer()
	}
	s.lastRequestType = req

	if req == pipelineSyncRequest {
		s.expectedReadyForQueryCount++
	}
}

func (s *pipelineState) ExtractFrontRequestType() pipelineRequestType {
	for {
		elem := s.requestEventQueue.Front()
		if elem == nil {
			return pipelineNil
		}
		val := elem.Value.(pipelineRequestEvent)
		if !(val.WasSentToServer && val.BeforeFlushOrSync) {
			return pipelineNil
		}

		s.requestEventQueue.Remove(elem)
		if val.RequestType == pipelineSyncRequest {
			s.fbErr = nil
		}
		if s.fbErr == nil {
			return val.RequestType
		}
	}
}

func (s *pipelineState) HandleError(err *FbError) {
	s.fbErr = err
}

func (s *pipelineState) HandleReadyForQuery() {
	s.expectedReadyForQueryCount--
}

func (s *pipelineState) PendingSync() bool {
	var notPendingSync bool

	if elem := s.requestEventQueue.Back(); elem != nil {
		val := elem.Value.(pipelineRequestEvent)
		notPendingSync = (val.RequestType == pipelineSyncRequest) && val.WasSentToServer
	} else {
		notPendingSync = (s.lastRequestType == pipelineSyncRequest) || (s.lastRequestType == pipelineNil)
	}

	return !notPendingSync
}

func (s *pipelineState) ExpectedReadyForQuery() int {
	return s.expectedReadyForQueryCount
}

// ensureInternalTransaction starts internal transaction if not started yet and bind it to connection
func (fbConn *FbConn) ensureInternalTransaction(ctx context.Context, tpb []byte) (*FBTransaction, error) {
	if fbConn.internalTransaction == nil || fbConn.internalTransaction.Closed {
		tx, err := fbConn.StartTransactionTPB(ctx, tpb)
		if err != nil {
			return nil, err
		}
		fbConn.internalTransaction = tx
		fbConn.explicitTransaction = false
	}
	return fbConn.internalTransaction, nil
}

// StartTransaction starts new transaction not bound to connection
func (fbConn *FbConn) StartTransaction(ctx context.Context) (*FBTransaction, error) {
	return fbConn.StartTransactionTPB(ctx, nil)
}

// StartTransaction starts new transaction not bound to connection with TPB block
func (fbConn *FbConn) StartTransactionTPB(ctx context.Context, tpb []byte) (*FBTransaction, error) {
	var err error

	if err = fbConn.lock(); err != nil {
		return nil, err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return nil, newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	opTransaction := fbproto.OpTransaction{
		TPB: tpb,
	}
	if err = fbConn.client.WriteMessage(&opTransaction); err != nil {
		return nil, err
	}

	if err = fbConn.writer.Flush(); err != nil {
		return nil, err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	switch m := msg.(type) {
	case *fbproto.OpResponse:
		tx := &FBTransaction{Connection: fbConn, Handle: m.Object, Closed: false}
		fbConn.activeTransactions[tx] = struct{}{}
		fbConn.internalTransaction = tx
		fbConn.explicitTransaction = true
		return tx, nil
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	default:
		return nil, fmt.Errorf("received unexpected message: %v", m)
	}
}

// StartPipeline switches the connection to pipeline mode and returns a *Pipeline. In pipeline mode requests can be sent
// to the server without waiting for a response. Close must be called on the returned *Pipeline to return the connection
// to normal mode. While in pipeline mode, no methods that communicate with the server may be called except
// CancelRequest and Close. ctx is in effect for entire life of the *Pipeline.
//
// Prefer ExecBatch when only sending one group of queries at once.
func (fbConn *FbConn) StartPipeline(ctx context.Context) *Pipeline {
	//if err := fbConn.lock(); err != nil {
	//	pipeline := &Pipeline{
	//		closed: true,
	//		err:    err,
	//	}
	//	pipeline.state.Init()
	//
	//	return pipeline
	//}
	//
	//fbConn.pipeline = Pipeline{
	//	conn: fbConn,
	//	ctx:  ctx,
	//}
	//fbConn.pipeline.state.Init()
	//
	//pipeline := &fbConn.pipeline
	//
	//if ctx != context.Background() {
	//	select {
	//	case <-ctx.Done():
	//		pipeline.closed = true
	//		pipeline.err = newContextAlreadyDoneError(ctx)
	//		fbConn.unlock()
	//		return pipeline
	//	default:
	//	}
	//	fbConn.contextWatcher.Watch(ctx)
	//}
	//
	//return pipeline
	panic("implement me")
}

// SendPrepare is the pipeline version of *FbConn.Prepare.
func (p *Pipeline) SendPrepare(name, sql string, paramOIDs []uint32) {
	//if p.closed {
	//	return
	//}
	//
	//p.conn.frontend.SendParse(&pgproto3.Parse{Name: name, Query: sql, ParameterOIDs: paramOIDs})
	//p.conn.frontend.SendDescribe(&pgproto3.Describe{ObjectType: 'S', Name: name})
	//p.state.PushBackRequestType(pipelinePrepare)
}

// SendDeallocate deallocates a prepared statement.
func (p *Pipeline) SendDeallocate(name string) {
	//if p.closed {
	//	return
	//}
	//
	//p.conn.frontend.SendClose(&pgproto3.Close{ObjectType: 'S', Name: name})
	//p.state.PushBackRequestType(pipelineDeallocate)
}

// SendQueryParams is the pipeline version of *FbConn.QueryParams.
func (p *Pipeline) SendQueryParams(sql string, paramValues [][]byte, paramOIDs []uint32, paramFormats []int16, resultFormats []int16) {
	//if p.closed {
	//	return
	//}
	//
	//p.conn.frontend.SendParse(&pgproto3.Parse{Query: sql, ParameterOIDs: paramOIDs})
	//p.conn.frontend.SendBind(&pgproto3.Bind{ParameterFormatCodes: paramFormats, Parameters: paramValues, ResultFormatCodes: resultFormats})
	//p.conn.frontend.SendDescribe(&pgproto3.Describe{ObjectType: 'P'})
	//p.conn.frontend.SendExecute(&pgproto3.Execute{})
	//p.state.PushBackRequestType(pipelineQueryParams)
}

// SendQueryPrepared is the pipeline version of *FbConn.QueryPrepared.
func (p *Pipeline) SendQueryPrepared(stmtName string, paramValues [][]byte, paramFormats []int16, resultFormats []int16) {
	//if p.closed {
	//	return
	//}
	//
	//p.conn.frontend.SendBind(&pgproto3.Bind{PreparedStatement: stmtName, ParameterFormatCodes: paramFormats, Parameters: paramValues, ResultFormatCodes: resultFormats})
	//p.conn.frontend.SendDescribe(&pgproto3.Describe{ObjectType: 'P'})
	//p.conn.frontend.SendExecute(&pgproto3.Execute{})
	//p.state.PushBackRequestType(pipelineQueryPrepared)
}

// SendFlushRequest sends a request for the server to flush its output buffer.
//
// The server flushes its output buffer automatically as a result of Sync being called,
// or on any request when not in pipeline mode; this function is useful to cause the server
// to flush its output buffer in pipeline mode without establishing a synchronization point.
// Note that the request is not itself flushed to the server automatically; use Flush if
// necessary. This copies the behavior of libpq PQsendFlushRequest.
func (p *Pipeline) SendFlushRequest() {
	//if p.closed {
	//	return
	//}
	//
	//p.conn.frontend.WriteMessage(&pgproto3.Flush{})
	//p.state.PushBackRequestType(pipelineFlushRequest)
}

// SendPipelineSync marks a synchronization point in a pipeline by sending a sync message
// without flushing the send buffer. This serves as the delimiter of an implicit
// transaction and an error recovery point.
//
// Note that the request is not itself flushed to the server automatically; use Flush if
// necessary. This copies the behavior of libpq PQsendPipelineSync.
func (p *Pipeline) SendPipelineSync() {
	//if p.closed {
	//	return
	//}
	//
	//p.conn.frontend.SendSync(&pgproto3.Sync{})
	//p.state.PushBackRequestType(pipelineSyncRequest)
}

// Flush flushes the queued requests without establishing a synchronization point.
func (p *Pipeline) Flush() error {
	//if p.closed {
	//	if p.err != nil {
	//		return p.err
	//	}
	//	return errors.New("pipeline closed")
	//}
	//
	//err := p.conn.flushWithPotentialWriteReadDeadlock()
	//if err != nil {
	//	err = normalizeTimeoutError(p.ctx, err)
	//
	//	p.conn.asyncClose()
	//
	//	p.conn.contextWatcher.Unwatch()
	//	p.conn.unlock()
	//	p.closed = true
	//	p.err = err
	//	return err
	//}
	//
	//p.state.RegisterSendingToServer()
	//return nil
	panic("implement me")
}

// Sync establishes a synchronization point and flushes the queued requests.
func (p *Pipeline) Sync() error {
	p.SendPipelineSync()
	return p.Flush()
}

// GetResults gets the next results. If results are present, results may be a *ResultReader, *StatementDescription, or
// *PipelineSync. If an ErrorResponse is received from the server, results will be nil and err will be a *FbError. If no
// results are available, results and err will both be nil.
func (p *Pipeline) GetResults() (results any, err error) {
	if p.closed {
		if p.err != nil {
			return nil, p.err
		}
		return nil, errors.New("pipeline closed")
	}

	if p.state.ExtractFrontRequestType() == pipelineNil {
		return nil, nil
	}

	return p.getResults()
}

func (p *Pipeline) getResults() (results any, err error) {
	//for {
	//	msg, err := p.conn.receiveMessage()
	//	if err != nil {
	//		p.closed = true
	//		p.err = err
	//		p.conn.asyncClose()
	//		return nil, normalizeTimeoutError(p.ctx, err)
	//	}
	//
	//	switch msg := msg.(type) {
	//	case *pgproto3.RowDescription:
	//		p.conn.resultReader = ResultReader{
	//			fbConn:            p.conn,
	//			pipeline:          p,
	//			ctx:               p.ctx,
	//			fieldDescriptions: p.conn.convertRowDescription(p.conn.fieldDescriptions[:], msg),
	//		}
	//		return &p.conn.resultReader, nil
	//	case *pgproto3.CommandComplete:
	//		p.conn.resultReader = ResultReader{
	//			commandTag:       p.conn.makeCommandTag(msg.QueryStatus),
	//			commandConcluded: true,
	//			closed:           true,
	//		}
	//		return &p.conn.resultReader, nil
	//	case *pgproto3.ParseComplete:
	//		peekedMsg, err := p.conn.peekMessage()
	//		if err != nil {
	//			p.conn.asyncClose()
	//			return nil, normalizeTimeoutError(p.ctx, err)
	//		}
	//		if _, ok := peekedMsg.(*pgproto3.ParameterDescription); ok {
	//			return p.getResultsPrepare()
	//		}
	//	case *pgproto3.CloseComplete:
	//		return &CloseComplete{}, nil
	//	case *pgproto3.ReadyForQuery:
	//		p.state.HandleReadyForQuery()
	//		return &PipelineSync{}, nil
	//	case *pgproto3.ErrorResponse:
	//		fbErr := ErrorResponseToPgError(msg)
	//		p.state.HandleError(fbErr)
	//		return nil, fbErr
	//	}
	//}
	panic("implement me")
}

func (p *Pipeline) getResultsPrepare() (*StatementDescription, error) {
	//psd := &StatementDescription{}
	//
	//for {
	//	msg, err := p.conn.receiveMessage()
	//	if err != nil {
	//		p.conn.asyncClose()
	//		return nil, normalizeTimeoutError(p.ctx, err)
	//	}
	//
	//	switch msg := msg.(type) {
	//	case *pgproto3.ParameterDescription:
	//		psd.ParamOIDs = make([]uint32, len(msg.ParameterOIDs))
	//		copy(psd.ParamOIDs, msg.ParameterOIDs)
	//	case *pgproto3.RowDescription:
	//		psd.Fields = p.conn.convertRowDescription(nil, msg)
	//		return psd, nil
	//
	//	// NoData is returned instead of RowDescription when there is no expected result. e.g. An INSERT without a RETURNING
	//	// clause.
	//	case *pgproto3.NoData:
	//		return psd, nil
	//
	//	// These should never happen here. But don't take chances that could lead to a deadlock.
	//	case *pgproto3.ErrorResponse:
	//		fbErr := ErrorResponseToPgError(msg)
	//		p.state.HandleError(fbErr)
	//		return nil, fbErr
	//	case *pgproto3.CommandComplete:
	//		p.conn.asyncClose()
	//		return nil, errors.New("BUG: received CommandComplete while handling Describe")
	//	case *pgproto3.ReadyForQuery:
	//		p.conn.asyncClose()
	//		return nil, errors.New("BUG: received ReadyForQuery while handling Describe")
	//	}
	//}
	panic("implement me")
}

// Close closes the pipeline and returns the connection to normal mode.
func (p *Pipeline) Close() error {
	//if p.closed {
	//	return p.err
	//}
	//
	//p.closed = true
	//
	//if p.state.PendingSync() {
	//	p.conn.asyncClose()
	//	p.err = errors.New("pipeline has unsynced requests")
	//	p.conn.contextWatcher.Unwatch()
	//	p.conn.unlock()
	//
	//	return p.err
	//}
	//
	//for p.state.ExpectedReadyForQuery() > 0 {
	//	_, err := p.getResults()
	//	if err != nil {
	//		p.err = err
	//		var fbErr *FbError
	//		if !errors.As(err, &fbErr) {
	//			p.conn.asyncClose()
	//			break
	//		}
	//	}
	//}
	//
	//p.conn.contextWatcher.Unwatch()
	//p.conn.unlock()
	//
	//return p.err
	panic("implement me")
}

// DeadlineContextWatcherHandler handles canceled contexts by setting a deadline on a net.Conn.
type DeadlineContextWatcherHandler struct {
	Conn net.Conn

	// DeadlineDelay is the delay to set on the deadline set on net.Conn when the context is canceled.
	DeadlineDelay time.Duration
}

func (h *DeadlineContextWatcherHandler) HandleCancel(ctx context.Context) {
	h.Conn.SetDeadline(time.Now().Add(h.DeadlineDelay))
}

func (h *DeadlineContextWatcherHandler) HandleUnwatchAfterCancel() {
	h.Conn.SetDeadline(time.Time{})
}

// CancelRequestContextWatcherHandler handles canceled contexts by sending a cancel request to the server. It also sets
// a deadline on a net.Conn as a fallback.
type CancelRequestContextWatcherHandler struct {
	Conn *FbConn

	// CancelRequestDelay is the delay before sending the cancel request to the server.
	CancelRequestDelay time.Duration

	// DeadlineDelay is the delay to set on the deadline set on net.Conn when the context is canceled.
	DeadlineDelay time.Duration

	cancelFinishedChan             chan struct{}
	handleUnwatchAfterCancelCalled func()
}

func (h *CancelRequestContextWatcherHandler) HandleCancel(context.Context) {
	//h.cancelFinishedChan = make(chan struct{})
	//var handleUnwatchedAfterCancelCalledCtx context.Context
	//handleUnwatchedAfterCancelCalledCtx, h.handleUnwatchAfterCancelCalled = context.WithCancel(context.Background())
	//
	//deadline := time.Now().Add(h.DeadlineDelay)
	//h.Conn.conn.SetDeadline(deadline)
	//
	//go func() {
	//	defer close(h.cancelFinishedChan)
	//
	//	select {
	//	case <-handleUnwatchedAfterCancelCalledCtx.Done():
	//		return
	//	case <-time.After(h.CancelRequestDelay):
	//	}
	//
	//	cancelRequestCtx, cancel := context.WithDeadline(handleUnwatchedAfterCancelCalledCtx, deadline)
	//	defer cancel()
	//	h.Conn.CancelRequest(cancelRequestCtx)
	//
	//	// CancelRequest is inherently racy. Even though the cancel request has been received by the server at this point,
	//	// it hasn't necessarily been delivered to the other connection. If we immediately return and the connection is
	//	// immediately used then it is possible the CancelRequest will actually cancel our next query. The
	//	// TestCancelRequestContextWatcherHandler Stress test can produce this error without the sleep below. The sleep time
	//	// is arbitrary, but should be sufficient to prevent this error case.
	//	time.Sleep(100 * time.Millisecond)
	//}()
}

func (h *CancelRequestContextWatcherHandler) HandleUnwatchAfterCancel() {
	//h.handleUnwatchAfterCancelCalled()
	//<-h.cancelFinishedChan
	//
	//h.Conn.conn.SetDeadline(time.Time{})
}
