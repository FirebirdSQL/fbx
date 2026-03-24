package fbconn

import (
	"bufio"
	"context"
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

	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbconn/internal/bgreader"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/auth"
	"rdb.red-soft.ru/fbx/fbproto/wirecrypt"
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

// DialFunc is a function that can be used to establish a network connection to a server
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

// FbConn is a low-level FirebirdSQL connection handle. It is not safe for concurrent usage.
type FbConn struct {
	conn            net.Conn
	secretKey       uint32 // key to use to send a cancel query message to the server
	txStatus        byte
	client          *fbproto.Client
	wireCrypt       *wirecrypt.Manager
	bgReader        *bgreader.BGReader
	writer          *bufio.Writer
	slowWriteTimer  *time.Timer
	bgReaderStarted chan struct{}
	dbHandle        int

	customData map[string]any

	config *Config

	status           byte // One of connStatus* constants
	protocolFeatures ProtocolFeatures

	protocolVersion int
	inlineBlobSize  int

	bufferingReceive    bool
	bufferingReceiveMux sync.Mutex
	bufferingReceiveMsg fbproto.ServerMessage
	bufferingReceiveErr error

	peekedMsg fbproto.ServerMessage

	// Reusable / preallocated resources
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

// Connect establishes a connection to a server using the environment and connString (in URL or keyword/value
// format) to provide configuration. See documentation for [ParseConfig] for details. ctx can be used to cancel a
// connect attempt.
func Connect(ctx context.Context, connString string) (*FbConn, error) {
	config, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	return ConnectConfig(ctx, config)
}

// Connect establishes a connection to a Firebird/RedDatabase server using the environment and connString (in URL or keyword/value
// format) and ParseConfigOptions to provide additional configuration. See documentation for [ParseConfig] for details.
// ctx can be used to cancel a connect attempt.
func ConnectWithOptions(ctx context.Context, connString string, parseConfigOptions ParseConfigOptions) (*FbConn, error) {
	config, err := ParseConfigWithOptions(connString, parseConfigOptions)
	if err != nil {
		return nil, err
	}

	return ConnectConfig(ctx, config)
}

// Connect establishes a connection to a Firebird/RedDatabase server using config. config must have been constructed with
// [ParseConfig]. ctx can be used to cancel a connect attempt.
//
// If config.Fallbacks are present they will sequentially be tried in case of error establishing network connection. An
// authentication error will terminate the chain of attempts and be returned as the error.
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
	if msg.Version != 0 {
		fbConn.protocolFeatures = ProtocolFeaturesForVersion(msg.Version)
	}
	authPrintf("[AUTHDBG] cond_accept: plugin=%s data_len=%d\n", msg.PluginName, len(msg.Data))
	authPrintf("%#v\n", msg)
	if msg.Version != 0 {
		fbConn.protocolVersion = msg.Version & fbproto.ProtocolMask
	}

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
			// fbx will try next host even if libpq does not in certain cases (see #2246)
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

	fbConn.status = connStatusConnecting

	fbConn.Encoding, err = fbproto.NewEncoding(config.Encoding)
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
	fbConn.inlineBlobSize = config.MaxInlineBlobSize
	fbConn.client.ConfigureInlineBlobCache(config.MaxInlineBlobSize, config.MaxBlobCacheSize)
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
		SystemHost:       config.SystemHost,
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
		if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
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
	if config.SystemUser != "" {
		dpb.PutString(fbproto.Isc_dpb_os_user, config.SystemUser)
	}
	if config.SystemHost != "" {
		dpb.PutString(fbproto.Isc_dpb_host_name, config.SystemHost)
	}
	if config.ClientVersion != "" {
		dpb.PutString(fbproto.Isc_dpb_client_version, config.ClientVersion)
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
	if fbConn.protocolFeatures.OpExecuteHaveBlobInlineCache {
		dpb.PutInt32(fbproto.Isc_dpb_max_inline_blob_size, int32(config.MaxInlineBlobSize))
		dpb.PutInt32(fbproto.Isc_dpb_max_blob_cache_size, int32(config.MaxBlobCacheSize))
	}
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
	if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
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

	if config.ValidateConnect != nil {
		fbConn.contextWatcher.Unwatch()
		err = config.ValidateConnect(ctx, fbConn)
		if err != nil {
			if _, ok := err.(*NotPreferredError); ignoreNotPreferredErr && ok {
				return fbConn, nil
			}
			fbConn.conn.Close()
			return nil, newPerDialConnectError("ValidateConnect failed", err)
		}
	}

	return fbConn, nil
}

// ReceiveMessage receives one wire protocol message from the server. It must only be used when the
// connection is not busy. e.g. It is an error to call ReceiveMessage while reading the result of a query.
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
		err = normalizeConnClosedError(err)
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

// Client returns the underlying *fbproto.Client
func (fbConn *FbConn) Client() *fbproto.Client {
	return fbConn.client
}

// Close closes a connection. It is safe to call Close on an already closed connection. Close attempts a clean close by
// sending the exit message to Firebird/RedDatabase server. However, this could block so ctx is available to limit the time to wait. The
// underlying net.Conn.Close() will always be called regardless of any other errors.
func (fbConn *FbConn) Close(ctx context.Context) error {
	if fbConn.status == connStatusClosed {
		return nil
	}
	fbConn.status = connStatusClosed
	fbConn.clearAllInlineBlobCache()

	defer close(fbConn.cleanupDone)
	defer fbConn.conn.Close()

	if ctx != context.Background() {
		// Close may be called while a cancellable query is in progress. This will most often be triggered by panic when
		// a defer closes the connection (possibly indirectly via a transaction or a connection pool). Unwatch to end any
		// previous watch. It is safe to Unwatch regardless of whether a watch is already is progress.
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
	fbConn.clearAllInlineBlobCache()

	go func() {
		defer close(fbConn.cleanupDone)
		defer fbConn.conn.Close()
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

// QueryStatus simple query info
type QueryStatus struct {
	statementType int
	rowsAffected  int64
}

// RowsAffected returns the number of rows affected
func (ct QueryStatus) RowsAffected() int64 {
	return ct.rowsAffected
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
	Scale             int32
	Subtype           int32
	Length            uint32
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
		res[i].Scale = int32(src[i].Scale)
		res[i].Subtype = int32(src[i].Subtype)
		res[i].Length = uint32(src[i].Length)
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
	tx, err := fbConn.ensureInternalTransaction(ctx, nil, false)
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
	if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return nil, normalizeTimeoutError(ctx, err)
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
		return nil, normalizeTimeoutError(ctx, err)
	}
	switch m := msg.(type) {
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	}

	statementInfo := StatementInfo{}
	err = fbConn.getInfoStatement(statementHandle, statementInfo.parseInfo)
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
		StatementInfo:   &statementInfo,
	}

	st := &Statement{
		conn:        fbConn,
		Description: desc,
	}
	if tx != fbConn.internalTransaction {
		st.transaction = tx
	}
	return st, nil
}

// CancelRequest sends a cancel request to the server. It returns an error if unable to deliver the cancel
// request, but lack of an error does not ensure that the query was canceled.
func (fbConn *FbConn) CancelRequest(ctx context.Context) error {
	if ctx != context.Background() {
		contextWatcher := ctxwatch.NewContextWatcher(&DeadlineContextWatcherHandler{Conn: fbConn.conn})
		contextWatcher.Watch(ctx)
		defer contextWatcher.Unwatch()
	}

	if err := fbConn.client.WriteMessage(&fbproto.OpCancel{
		Kind: fbproto.Fb_cancel_raise,
	}); err != nil {
		fbConn.asyncClose()
		return err
	}

	if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.asyncClose()
		return err
	}

	return nil
}

// ExecImmediate executes statement immediate inside internal transaction
func (fbConn *FbConn) ExecImmediate(ctx context.Context, sql string) error {
	tx, err := fbConn.ensureInternalTransaction(ctx, nil, false)
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

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			fbConn.unlock()
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

	if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.asyncClose()
		return err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}

	switch m := msg.(type) {
	case *fbproto.OpResponse:
	case *fbproto.ErrorResponse:
		fbConn.unlock()
		return ErrorResponseToFbError(m)
	default:
		panic(fmt.Sprintf("wrong response message type %T instead expected OpResponse or ErrorResponse", m))
	}

	fbConn.unlock()
	fbConn.contextWatcher.Unwatch()
	if tx != nil && tx == fbConn.internalTransaction && fbConn.config.Autocommit && !fbConn.explicitTransaction {
		tx.Commit(ctx)
	}

	return nil
}

// Exec execute a single SQL statement.
//
// If no transaction is provided (see ExecTx / ExecParamsTx), an internal transaction is used.
//
// ResultReader must be closed before FbConn can be used again.
func (fbConn *FbConn) Exec(ctx context.Context, sql string) *ResultReader {
	return fbConn.ExecParams(ctx, sql, nil)
}

func (fbConn *FbConn) ExecTx(ctx context.Context, tx *FBTransaction, sql string) *ResultReader {
	return fbConn.ExecParamsTx(ctx, tx, sql, nil)
}

// ExecParams executes a single SQL statement with already-encoded parameter values.
//
// sql is a SQL command string. It may only contain one query.
// Parameters are positional; the placeholder syntax is defined by the SQL dialect supported by the server.
//
// paramValues are parameter values encoded for the wire protocol / statement parameter layout.
// Higher-level APIs (fbx.Conn) typically build these values using the extended query builder.
//
// ResultReader must be closed before FbConn can be used again.
func (fbConn *FbConn) ExecParams(ctx context.Context, sql string, paramValues [][]byte) *ResultReader {
	tx, err := fbConn.ensureInternalTransaction(ctx, nil, false)
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
		closeConn := true
		var fbErr *FbError
		if SafeToRetry(rr.err) || errors.As(rr.err, &fbErr) {
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
		res.concludeCommand(QueryStatus{}, rr.err)
		return rr
	}
	return rr
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

	if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.conn.Close()
		return err
	}

	return nil
}

// ResultReader is a reader for the result of a single query.
type ResultReader struct {
	fbConn *FbConn
	ctx    context.Context

	fieldDescriptions []FieldDescription
	rowValues         [][]byte
	bufferedRows      []bufferedRow
	bufferedCols      []bufferedCol
	bufferedArena     []byte
	bufferedNext      int
	queryStatus       QueryStatus
	commandConcluded  bool
	closed            bool
	err               error

	statementDescription *StatementDescription
	singleton            *fbproto.SQLData
	singletonResult      bool
	cursorOpened         bool
	fetchExecuted        bool
	fetchResp            *fbproto.OpFetchResponse
	eoc                  bool
	rowsInfoPopulated    bool

	boundTransaction *FBTransaction
}

const bufferedColNullLength = -1

type bufferedRow struct {
	colStart int
}

type bufferedCol struct {
	start  int
	length int
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

// ReadAll reads the entire result set into memory
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
	if rr.popBufferedRow() {
		return true
	}

	if !rr.cursorOpened {
		return false
	}

	if rr.eoc {
		return false
	}

	if rr.fetchResp == nil || !rr.fetchResp.MoreRows() {
		if rr.fetchResp != nil && rr.fetchResp.EndOfCursor() {
			rr.eoc = true
			return false
		}

		var rowBlr []byte
		if !rr.fetchExecuted {
			rowBlr = rr.statementDescription.StatementInfo.GetRowBlr()
		}
		fetchBatchSize := DefaultFetchBatchSize
		if rr.fbConn != nil && rr.fbConn.config != nil && rr.fbConn.config.FetchBatchSize > 0 {
			fetchBatchSize = rr.fbConn.config.FetchBatchSize
		}
		if err = rr.fbConn.client.WriteMessage(&fbproto.OpFetch{
			StatementHandle: rr.statementDescription.StatementHandle,
			RowBlr:          rowBlr,
			Messages:        fetchBatchSize,
		}); err != nil {
			return false
		}
		if err = rr.fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
			return false
		}
		rr.fetchExecuted = true

		msg, err := rr.fbConn.receiveMessage()
		if err != nil {
			rr.fbConn.asyncClose()
			rr.concludeCommand(QueryStatus{}, normalizeTimeoutError(rr.ctx, err))
			rr.fbConn.contextWatcher.Unwatch()
			return false
		}
		switch m := msg.(type) {
		case *fbproto.OpFetchResponse:
			rr.fetchResp = m
		case *fbproto.ErrorResponse:
			rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(m))
			return false
		default:
			panic(fmt.Sprintf("wrong response message type %T instead expected OpFetchResponse or ErrorResponse", m))
		}

		if rr.fetchResp.EndOfCursor() {
			rr.eoc = true
			return false
		}
		if !rr.fetchResp.MoreRows() {
			return false
		}
	}
	if !rr.bufferCurrentFetchRows() {
		return false
	}
	return rr.popBufferedRow()
}

func (rr *ResultReader) popBufferedRow() bool {
	if rr.bufferedNext >= len(rr.bufferedRows) {
		return false
	}

	colCount := len(rr.fieldDescriptions)
	if cap(rr.rowValues) < colCount {
		rr.rowValues = make([][]byte, colCount)
	} else {
		rr.rowValues = rr.rowValues[:colCount]
	}

	row := rr.bufferedRows[rr.bufferedNext]
	cols := rr.bufferedCols[row.colStart : row.colStart+colCount]
	for i := 0; i < colCount; i++ {
		col := cols[i]
		if col.length == bufferedColNullLength {
			rr.rowValues[i] = nil
			continue
		}
		if col.length == 0 {
			// Keep non-NULL empty values distinguishable from NULL.
			rr.rowValues[i] = []byte{}
			continue
		}
		end := col.start + col.length
		rr.rowValues[i] = rr.bufferedArena[col.start:end:end]
	}

	rr.bufferedNext++
	if rr.bufferedNext == len(rr.bufferedRows) {
		rr.clearBufferedFetch()
	}
	return true
}

func (rr *ResultReader) bufferCurrentFetchRows() bool {
	rr.clearBufferedFetch()
	colCount := len(rr.fieldDescriptions)

	for rr.fetchResp.MoreRows() {
		row, msg, err := rr.fbConn.client.ReceiveSQLData(rr.statementDescription.StatementInfo.FieldDescriptors)
		if err != nil {
			rr.fbConn.asyncClose()
			rr.concludeCommand(QueryStatus{}, err)
			return false
		}
		switch m := msg.(type) {
		case *fbproto.OpFetchResponse:
			rr.fetchResp = m
		case *fbproto.ErrorResponse:
			rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(m))
			return false
		default:
			panic(fmt.Sprintf("wrong response message type %T instead expected OpFetchResponse or ErrorResponse", m))
		}

		rowStart := len(rr.bufferedCols)
		for i := 0; i < colCount; i++ {
			col := row.RowData[i]
			if col == nil {
				rr.bufferedCols = append(rr.bufferedCols, bufferedCol{length: bufferedColNullLength})
				continue
			}
			start := len(rr.bufferedArena)
			rr.bufferedArena = append(rr.bufferedArena, col...)
			rr.bufferedCols = append(rr.bufferedCols, bufferedCol{start: start, length: len(col)})
		}
		rr.bufferedRows = append(rr.bufferedRows, bufferedRow{colStart: rowStart})

		if rr.fetchResp.EndOfCursor() {
			rr.eoc = true
		}
	}
	if rr.fetchResp.EndOfCursor() {
		rr.eoc = true
	}
	return true
}

func (rr *ResultReader) clearBufferedFetch() {
	rr.bufferedRows = rr.bufferedRows[:0]
	rr.bufferedCols = rr.bufferedCols[:0]
	rr.bufferedArena = rr.bufferedArena[:0]
	rr.bufferedNext = 0
}

func (rr *ResultReader) drainCurrentFetch() {
	if !rr.cursorOpened || rr.eoc || rr.fetchResp == nil {
		return
	}
	for rr.fetchResp.MoreRows() {
		_, msg, err := rr.fbConn.client.ReceiveSQLData(rr.statementDescription.StatementInfo.FieldDescriptors)
		if err != nil {
			rr.fbConn.asyncClose()
			rr.concludeCommand(QueryStatus{}, err)
			return
		}
		switch m := msg.(type) {
		case *fbproto.OpFetchResponse:
			rr.fetchResp = m
		case *fbproto.ErrorResponse:
			rr.concludeCommand(QueryStatus{}, ErrorResponseToFbError(m))
			return
		default:
			panic(fmt.Sprintf("wrong response message type %T instead expected OpFetchResponse or ErrorResponse", m))
		}
	}
	if rr.fetchResp.EndOfCursor() {
		rr.eoc = true
	}
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
		if rr.err == nil {
			rr.drainCurrentFetch()
			if rr.err == nil {
				if err := rr.populateRowsAffected(); err != nil {
					rr.err = err
				}
			}
		}
		rr.clearBufferedFetch()
		rr.closed = true
		rr.fbConn.unlock()
		rr.fbConn.contextWatcher.Unwatch()

		if rr.boundTransaction != nil && rr.boundTransaction == rr.fbConn.internalTransaction && rr.fbConn.config.Autocommit && !rr.fbConn.explicitTransaction {
			if err := rr.boundTransaction.Commit(context.Background()); err != nil && rr.err == nil {
				rr.err = err
			}
		}
	}
	return rr.queryStatus, rr.err
}

func (rr *ResultReader) populateRowsAffected() error {
	if rr.rowsInfoPopulated || rr.statementDescription == nil || rr.fbConn == nil || rr.err != nil {
		rr.rowsInfoPopulated = true
		return nil
	}

	err := rr.fbConn.getInfoSQlRecords(
		rr.statementDescription.StatementHandle,
		rr.statementDescription.StatementInfo.Counts.parseInfo,
	)
	if err != nil {
		rr.rowsInfoPopulated = true
		return err
	}

	rows := rr.statementDescription.GetAffectedCount()

	rr.queryStatus.rowsAffected = rows
	rr.rowsInfoPopulated = true

	return nil
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

	if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.conn.Close()
		return err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}

	switch m := msg.(type) {
	case *fbproto.OpResponse:
		return nil
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		return fmt.Errorf("received unexpected message: %v", m)
	}
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
	if !fbConn.slowWriteTimer.Stop() {
		// The timer starts its function in a separate goroutine. It is necessary to ensure the background reader has
		// started before calling Stop. Otherwise, the background reader may not be stopped. That on its own is not a
		// serious problem. But what is a serious problem is that the background reader may start at an inopportune time in
		// a subsequent query. For example, if a subsequent query was canceled then a deadline may be set on the net.Conn to
		// interrupt an in-progress read. After the read is interrupted, but before the deadline is cleared, the background
		// reader could start and read a deadline error. Then the next query would receive the an unexpected deadline error.
		<-fbConn.bgReaderStarted
		fbConn.bgReader.Stop()
	}
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
	//Client          *fbproto3.Client
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

// Construct created a FbConn from an already established network connection. This is the inverse of
// FbConn.Hijack. The connection must be in an idle state.
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

// ensureInternalTransaction starts internal transaction if not started yet and bind it to connection.
// If locked is true, the caller must already hold the connection lock.
func (fbConn *FbConn) ensureInternalTransaction(ctx context.Context, tpb []byte, locked bool) (*FBTransaction, error) {
	if fbConn.internalTransaction == nil || fbConn.internalTransaction.Closed {

		var txTpb []byte
		if tpb != nil {
			txTpb = tpb
		} else {
			if fbConn.config.DefaultTransactionReadOnly {
				w := fbproto.NewTPBWriterV3()
				w.PutTag(fbproto.Isc_tpb_read)
				txTpb = w.Bytes()
			}
		}
		tx, err := fbConn.startTransactionTPB(ctx, txTpb, locked)
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
	return fbConn.startTransactionTPB(ctx, tpb, false)
}

func (fbConn *FbConn) startTransactionTPB(ctx context.Context, tpb []byte, locked bool) (*FBTransaction, error) {
	var err error

	if !locked {
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
	} else {
		if !fbConn.IsBusy() {
			return nil, &connLockError{status: "conn is not locked"}
		}
		if ctx == context.Background() {
			// already locked, no context handling needed
		} else {
			select {
			case <-ctx.Done():
				return nil, newContextAlreadyDoneError(ctx)
			default:
			}
		}
	}

	opTransaction := fbproto.OpTransaction{
		TPB: tpb,
	}
	if err = fbConn.client.WriteMessage(&opTransaction); err != nil {
		return nil, err
	}

	if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		return nil, err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return nil, normalizeTimeoutError(ctx, err)
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
	h.cancelFinishedChan = make(chan struct{})
	var handleUnwatchedAfterCancelCalledCtx context.Context
	handleUnwatchedAfterCancelCalledCtx, h.handleUnwatchAfterCancelCalled = context.WithCancel(context.Background())

	deadline := time.Now().Add(h.DeadlineDelay)
	h.Conn.conn.SetDeadline(deadline)

	go func() {
		defer close(h.cancelFinishedChan)

		select {
		case <-handleUnwatchedAfterCancelCalledCtx.Done():
			return
		case <-time.After(h.CancelRequestDelay):
		}

		cancelRequestCtx, cancel := context.WithDeadline(handleUnwatchedAfterCancelCalledCtx, deadline)
		defer cancel()
		h.Conn.CancelRequest(cancelRequestCtx)

		// CancelRequest is inherently racy. Even though the cancel request has been received by the server at this point,
		// it hasn't necessarily been delivered to the other connection. If we immediately return and the connection is
		// immediately used then it is possible the CancelRequest will actually cancel our next query. The
		// TestCancelRequestContextWatcherHandler Stress test can produce this error without the sleep below. The sleep time
		// is arbitrary, but should be sufficient to prevent this error case.
		time.Sleep(100 * time.Millisecond)
	}()
}

func (h *CancelRequestContextWatcherHandler) HandleUnwatchAfterCancel() {
	h.handleUnwatchAfterCancelCalled()
	<-h.cancelFinishedChan

	h.Conn.conn.SetDeadline(time.Time{})
}
