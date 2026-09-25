package fbconn

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbconn/internal/bgreader"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/auth"
	"rdb.red-soft.ru/fbx/fbproto/wirecrypt"
)

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

// FbErrorHandler is called for errors returned from Firebird. Returning true keeps the connection open, and returning
// false requests immediate connection closure. Transport and protocol errors are handled internally and do not invoke
// this callback. The supplied [FbConn] identifies the origin of the error, but the handler must not invoke any query
// method on it.
type FbErrorHandler func(*FbConn, *FbError) bool

func defaultOnFbError(_ *FbConn, err *FbError) bool {
	return !IsConnectionFatal(err)
}

// FbWarningHandler is a function that handles warnings returned from Firebird.
// The *FbConn is provided so the handler is aware of the origin of the warning, but it must not invoke any query method.
type FbWarningHandler func(*FbConn, *FbWarning)

// FbConn is a low-level FirebirdSQL connection handle. It is not safe for concurrent usage.
type FbConn struct {
	conn             net.Conn
	txStatus         byte
	client           *fbproto.Client
	wireCrypt        *wirecrypt.Manager
	bgReader         *bgreader.BGReader
	writer           *bufio.Writer
	writeMu          sync.Mutex
	slowWriteTimer   *time.Timer
	bgReaderStarted  chan struct{}
	writeInProgress  atomic.Bool
	attachmentHandle int
	attachmentID     uint64

	customData map[string]any

	config *Config

	statusMu         sync.Mutex
	status           byte // One of connStatus* constants
	protocolFeatures ProtocolFeatures

	serverVersion      ServerVersion
	serverMajorVersion int

	protocolVersion int
	inlineBlobSize  int

	bufferingReceive    bool
	bufferingReceiveMux sync.Mutex
	bufferingReceiveMsg fbproto.ServerMessage
	bufferingReceiveErr error

	peekedMsg fbproto.ServerMessage

	eventMu      sync.Mutex
	eventChannel *eventChannel
	eventHandles map[int]*EventHandle
	eventHost    string
	eventErr     error
	// eventErrHandler is notified when the first asynchronous event-channel
	// error is captured. It should be non-blocking and thread-safe.
	eventErrHandler func(error)

	// Reusable / preallocated resources
	contextWatcher *ctxwatch.ContextWatcher
	contextHandler ctxwatch.Handler

	cleanupDone         chan struct{}
	Encoding            *fbproto.Encoding
	internalTransaction *FBTransaction
	explicitTransaction bool
	activeTransactions  map[*FBTransaction]struct{}

	activeStatements map[*Statement]struct{}

	// Tracks whether we've already sent the first op_cont_auth (where p_list
	// must include the client's plugin list). Subsequent op_cont_auth must use
	// an empty p_list.
	authContFirstSent bool

	cleanupDoneOnce sync.Once
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

// CreateDatabase creates a Firebird/RedDatabase database and returns an attached connection to it.
// The connection string accepts the same options as Connect.
func CreateDatabase(ctx context.Context, connString string) (*FbConn, error) {
	config, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	config.CreateDatabase = true

	return ConnectConfig(ctx, config)
}

// DropDatabase connects to a Firebird/RedDatabase database, drops it, and closes the connection.
// The connection string accepts the same options as Connect.
func DropDatabase(ctx context.Context, connString string) error {
	config, err := ParseConfig(connString)
	if err != nil {
		return err
	}
	config.CreateDatabase = false

	fbConn, err := ConnectConfig(ctx, config)
	if err != nil {
		return err
	}
	err = fbConn.DropDatabase(ctx)
	if !fbConn.IsClosed() {
		err = errors.Join(err, fbConn.Close(context.Background()))
	}
	return err
}

// ConnectWithOptions establishes a connection to a Firebird/RedDatabase server using the environment and connString (in URL or keyword/value
// format) and ParseConfigOptions to provide additional configuration. See documentation for [ParseConfig] for details.
// ctx can be used to cancel a connect attempt.
// The connection string accepts the same options as Connect.
func ConnectWithOptions(ctx context.Context, connString string, parseConfigOptions ParseConfigOptions) (*FbConn, error) {
	config, err := ParseConfigWithOptions(connString, parseConfigOptions)
	if err != nil {
		return nil, err
	}

	return ConnectConfig(ctx, config)
}

// CreateDatabaseWithOptions behaves exactly like CreateDatabase with the addition of ParseConfigOptions.
// The connection string accepts the same options as Connect.
func CreateDatabaseWithOptions(ctx context.Context, connString string, parseConfigOptions ParseConfigOptions) (*FbConn, error) {
	config, err := ParseConfigWithOptions(connString, parseConfigOptions)
	if err != nil {
		return nil, err
	}
	config.CreateDatabase = true

	return ConnectConfig(ctx, config)
}

// DropDatabaseWithOptions behaves exactly like DropDatabase with the addition of ParseConfigOptions.
// The connection string accepts the same options as Connect.
func DropDatabaseWithOptions(ctx context.Context, connString string, parseConfigOptions ParseConfigOptions) error {
	config, err := ParseConfigWithOptions(connString, parseConfigOptions)
	if err != nil {
		return err
	}
	config.CreateDatabase = false

	fbConn, err := ConnectConfig(ctx, config)
	if err != nil {
		return err
	}
	err = fbConn.DropDatabase(ctx)
	if !fbConn.IsClosed() {
		err = errors.Join(err, fbConn.Close(context.Background()))
	}
	return err
}

// ConnectConfig establishes a connection to a Firebird/RedDatabase server using config. config must have been constructed with
// [ParseConfig]. ctx can be used to cancel a connect attempt.
//
// If config.Fallbacks are present they will sequentially be tried in case of error establishing network connection. An
// authentication error will terminate the chain of attempts and be returned as the error.
func ConnectConfig(ctx context.Context, config *Config) (*FbConn, error) {
	fbConn, err := connectConfig(ctx, config, connectTargetDatabase)
	if err != nil {
		return nil, err
	}

	if config.AfterConnect != nil {
		err := config.AfterConnect(ctx, fbConn)
		if err != nil {
			_ = fbConn.conn.Close()
			return nil, &ConnectError{Config: config, err: fmt.Errorf("AfterConnect error: %w", err)}
		}
	}

	return fbConn, nil
}

func connectConfig(ctx context.Context, config *Config, target connectTarget) (*FbConn, error) {
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

	fbConn, errs := connectPreferred(ctx, config, connectConfigs, target)
	if len(errs) > 0 {
		allErrors = append(allErrors, errs...)
		return nil, &ConnectError{Config: config, err: errors.Join(allErrors...)}
	}

	return fbConn, nil
}

// CreateDatabaseConfig creates a Firebird/RedDatabase database using config and returns an attached connection to it.
// config must have been constructed with [ParseConfig].
func CreateDatabaseConfig(ctx context.Context, config *Config) (*FbConn, error) {
	config = config.Copy()
	config.CreateDatabase = true
	return ConnectConfig(ctx, config)
}

// DropDatabaseConfig connects using config, drops the attached database, and closes the connection.
// config must have been constructed with [ParseConfig].
func DropDatabaseConfig(ctx context.Context, config *Config) error {
	config = config.Copy()
	config.CreateDatabase = false

	fbConn, err := ConnectConfig(ctx, config)
	if err != nil {
		return err
	}
	err = fbConn.DropDatabase(ctx)
	if !fbConn.IsClosed() {
		err = errors.Join(err, fbConn.Close(context.Background()))
	}
	return err
}

// handleCondAccept applies protocol metadata and processes the server challenge.
// When respond is false, the generated reply remains in authManager for the
// following attach DPB instead of being sent in op_cont_auth.
func (fbConn *FbConn) handleCondAccept(msg *fbproto.OpCondAccept, authManager *auth.AuthManager, respond bool) error {
	if err := fbConn.handleAcceptMetadata(&msg.AcceptMetadata, msg.ServerKeys, authManager); err != nil {
		return err
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

// handleAcceptMetadata records the negotiated protocol version, wire-encryption
// keys, and the server-supported authentication plugin list.
func (fbConn *FbConn) handleAcceptMetadata(metadata *fbproto.AcceptMetadata, serverKeys []byte, authManager *auth.AuthManager) error {
	if metadata.Version != 0 {
		fbConn.protocolFeatures = ProtocolFeaturesForVersion(metadata.Version)
		fbConn.protocolVersion = metadata.Version & fbproto.ProtocolMask
	}
	if metadata.Compress && !fbConn.client.CompressionEnabled() {
		if err := fbConn.client.EnableCompression(); err != nil {
			return fmt.Errorf("failed to enable wire compression: %w", err)
		}
	}

	if fbConn.wireCrypt != nil {
		if err := fbConn.wireCrypt.AddServerKeys(serverKeys); err != nil {
			return err
		}
	}
	if err := authManager.RestrictToServerPlugins(serverKeys); err != nil {
		return err
	}
	return nil
}

// handleAuthResponse validates the final plugin response and starts wire
// encryption when both peers supplied compatible key material. The boolean
// result reports whether an op_crypt acknowledgment is still expected.
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
	if !config.WireCrypt || len(sessionKey) == 0 {
		return false, nil
	}

	initInfo, err := fbConn.enableWireCrypt(sessionKey)
	if err != nil {
		if errors.Is(err, wirecrypt.ErrNoCommonPlugin) {
			return false, nil
		}
		return false, err
	}
	return initInfo != nil, nil
}

// handlePostAttachAuthChallenge processes authentication continuation received
// after op_attach. The plugin list is omitted because it was already advertised
// during the initial negotiation.
func (fbConn *FbConn) handlePostAttachAuthChallenge(authManager *auth.AuthManager, pluginName string, data []byte, keys []byte) error {
	if fbConn.wireCrypt != nil {
		if err := fbConn.wireCrypt.AddServerKeys(keys); err != nil {
			return err
		}
	}

	step, err := authManager.HandleServerChallenge(pluginName, data)
	if err != nil {
		return err
	}
	if step == nil {
		return nil
	}
	return fbConn.sendContAuth(step.Reply, pluginName, "", nil)
}

// finishAttachAuth completes authentication that continues after op_attach and
// does not return until the server supplies the database handle or an error.
func (fbConn *FbConn) finishAttachAuth(config *Config, authManager *auth.AuthManager, attachAction string) error {
	pendingCryptAck := false

	for {
		msg, err := fbConn.receiveMessage()
		if err != nil {
			return fmt.Errorf("%s failed: %w", attachAction, err)
		}

		switch m := msg.(type) {
		case *fbproto.OpContAuth:
			if err := fbConn.handlePostAttachAuthChallenge(authManager, m.PluginName, m.AuthData, m.Keys); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
				return fmt.Errorf("failed to flush opContAuth: %w", err)
			}
			continue
		case *fbproto.OpCondAccept:
			if err := fbConn.handleAcceptMetadata(&m.AcceptMetadata, m.ServerKeys, authManager); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if err := fbConn.handlePostAttachAuthChallenge(authManager, m.PluginName, m.Data, nil); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
				return fmt.Errorf("failed to flush opContAuth: %w", err)
			}
			continue
		case *fbproto.OpDataAccept:
			if err := fbConn.handleAcceptMetadata(&m.AcceptMetadata, m.ServerKeys, authManager); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if m.AuthComplete {
				authManager.SetAuthComplete(true)
				continue
			}
			if err := fbConn.handlePostAttachAuthChallenge(authManager, m.PluginName, m.Data, nil); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
				return fmt.Errorf("failed to flush opContAuth: %w", err)
			}
			continue
		case *fbproto.OpResponse:
			if pendingCryptAck {
				return nil
			}
			continueLoop, err := fbConn.handleAuthResponse(m, config, authManager)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			fbConn.attachmentHandle = m.Object
			if continueLoop {
				pendingCryptAck = true
				if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
					return fmt.Errorf("failed to flush opCrypt: %w", err)
				}
				continue
			}
			return nil
		case *fbproto.ErrorResponse:
			return ErrorResponseToFbError(m)
		default:
			return fmt.Errorf("%s failed: unexpected message type %T", attachAction, msg)
		}
	}
}

// enableWireCrypt selects a mutually supported wire-encryption plugin, sends
// op_crypt, and installs the resulting stream codec on the protocol client.
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
	// op_crypt itself must use the current (compressed, but not encrypted) transport.
	// Flush it all the way to the socket before installing the cipher below zlib.
	if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
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

// shouldRespondToDataAccept reports whether op_accept_data requires another
// op_cont_auth exchange. Without wire-encryption keys, its generated plugin
// reply is carried by the following attach DPB.
func shouldRespondToDataAccept(msg *fbproto.OpDataAccept) bool {
	if msg == nil {
		return true
	}
	if msg.AuthComplete {
		return false
	}
	return hasWireCryptKeys(msg.ServerKeys)
}

// hasWireCryptKeys reports whether p_acpt_keys contains at least one complete
// wire-encryption key type and plugin advertisement.
func hasWireCryptKeys(serverKeys []byte) bool {
	keys, err := wirecrypt.ParseKnownServerKeys(serverKeys)
	if err != nil {
		return false
	}
	for _, key := range keys {
		if key.Type != "" && len(key.Plugins) > 0 {
			return true
		}
	}
	return false
}

// sendContAuth writes one authentication continuation packet. pluginList is
// normally populated only for the first continuation sent before attach.
func (fbConn *FbConn) sendContAuth(authData []byte, pluginName string, pluginList string, keys []byte) error {
	dataCopy := make([]byte, len(authData))
	copy(dataCopy, authData)
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
func connectPreferred(
	ctx context.Context,
	config *Config,
	connectOneConfigs []*connectOneConfig,
	target connectTarget,
) (*FbConn, []error) {
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

		fbConn, err := connectOne(ctx, config, c, false, target)
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
		fallbackCtx := octx
		if config.ConnectTimeout != 0 {
			var cancel context.CancelFunc
			fallbackCtx, cancel = context.WithTimeout(octx, config.ConnectTimeout)
			defer cancel()
		}
		fbConn, err := connectOne(fallbackCtx, config, fallbackConnectOneConfig, true, target)
		if err == nil {
			return fbConn, nil
		}
		allErrors = append(allErrors, err)
	}

	return nil, allErrors
}

// connectOne makes one connection attempt to a single host.
func connectOne(ctx context.Context, config *Config, connectConfig *connectOneConfig,
	ignoreNotPreferredErr bool, target connectTarget,
) (*FbConn, error) {
	fbConn := new(FbConn)
	fbConn.config = config
	fbConn.eventHost = connectConfig.originalHostname
	if fbConn.eventHost == "" {
		fbConn.eventHost = config.Host
	}
	if isAbsolutePath(fbConn.eventHost) {
		fbConn.eventHost = "localhost"
	}
	fbConn.cleanupDone = make(chan struct{})
	fbConn.customData = make(map[string]any)
	fbConn.activeTransactions = make(map[*FBTransaction]struct{})
	fbConn.activeStatements = make(map[*Statement]struct{})

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

	fbConn.contextHandler = config.BuildContextWatcherHandler(fbConn)
	fbConn.contextWatcher = ctxwatch.NewContextWatcher(fbConn.contextHandler)
	fbConn.contextWatcher.Watch(ctx)
	defer fbConn.contextWatcher.Unwatch()

	fbConn.status = connStatusConnecting

	fbConn.Encoding, err = fbproto.NewEncoding(config.Encoding)
	if err != nil {
		closeErr := fbConn.conn.Close()
		return nil, newPerDialConnectError("encoding setup failed", errors.Join(err, closeErr))
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
	if config.BuildFrontend != nil {
		fbConn.client = config.BuildFrontend(fbConn.bgReader, fbConn.writer)
	} else {
		fbConn.client = fbproto.NewClient(fbConn.bgReader, fbConn.writer, fbConn.Encoding)
	}
	fbConn.inlineBlobSize = config.MaxInlineBlobSize
	fbConn.client.ConfigureInlineBlobCache(config.MaxInlineBlobSize, config.MaxBlobCacheSize)
	fbConn.wireCrypt = wirecrypt.NewDefaultManager()

	pluginConfig := auth.PluginConfig{User: config.User, Password: config.Password, ConstructorOptions: config.ConstructorOptions}
	authManager, err := auth.NewAuthManager(pluginConfig, config.AuthPluginList)
	if err != nil {
		_ = fbConn.conn.Close()
		return nil, newPerDialConnectError("authentication setup failed", err)
	}
	initialHandshake := authManager.BeginHandshake()
	if initialHandshake == nil {
		_ = fbConn.conn.Close()
		return nil, newPerDialConnectError("authentication setup failed", fmt.Errorf("no authentication plugins available"))
	}

	connectObjectName := config.Database
	if target == connectTargetService {
		connectObjectName = defaultServiceName
	}

	opConnect := fbproto.OpConnect{
		Operation:        fbproto.Op_attach,
		Database:         connectObjectName,
		User:             config.User,
		SystemUser:       config.SystemUser,
		SystemHost:       config.SystemHost,
		Password:         config.Password,
		AuthPluginName:   initialHandshake.PluginName,
		AuthPluginsList:  initialHandshake.PluginList,
		AuthSpecificData: initialHandshake.SpecificData,
		Options:          nil,
		WireCrypt:        config.WireCrypt,
		Compress:         config.WireCompression,
		Encoding:         fbConn.Encoding,
	}
	if target == connectTargetDatabase && config.CreateDatabase {
		opConnect.Operation = fbproto.Op_create
	}

	if err = fbConn.client.WriteMessage(&opConnect); err != nil {
		_ = fbConn.conn.Close()
		return nil, newPerDialConnectError("failed to send opConnect", err)
	}

	// Track if we've just enabled wire encryption and need to consume
	// the server's acknowledgment response for op_crypt.
	pendingCryptAck := false

connectLoop:
	for {
		if err = fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
			_ = fbConn.conn.Close()
			return nil, newPerDialConnectError("failed to flush opConnect", err)
		}

		msg, err := fbConn.receiveMessage()
		if err != nil {
			_ = fbConn.conn.Close()
			return nil, newPerDialConnectError("authentication failed", err)
		}

		switch msg := msg.(type) {
		case *fbproto.OpAccept:
			if err := fbConn.handleAcceptMetadata(&msg.AcceptMetadata, nil, authManager); err != nil {
				_ = fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			break connectLoop
		case *fbproto.OpContAuth:
			fake := &fbproto.OpCondAccept{Encoding: fbConn.Encoding}
			fake.PluginName = msg.PluginName
			fake.Data = msg.AuthData
			fake.ServerKeys = msg.Keys
			if err := fbConn.handleCondAccept(fake, authManager, true); err != nil {
				_ = fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			continue connectLoop
		case *fbproto.OpCondAccept:
			// op_cond_accept always requests an authentication continuation,
			// including an empty challenge that selects another plugin. The
			// exchange must finish here so wire encryption starts before attach.
			if err := fbConn.handleCondAccept(msg, authManager, true); err != nil {
				_ = fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			continue connectLoop
		case *fbproto.OpResponse:
			if pendingCryptAck {
				// This is the ack for op_crypt; consume it and finish handshake
				break connectLoop
			}
			continueLoop, err := fbConn.handleAuthResponse(msg, config, authManager)
			if err != nil {
				_ = fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			if continueLoop {
				// We enabled encryption; expect a crypt ack next
				pendingCryptAck = true
				continue connectLoop
			}
			break connectLoop
		case *fbproto.OpDataAccept:
			if msg.AuthComplete {
				if err := fbConn.handleAcceptMetadata(&msg.AcceptMetadata, msg.ServerKeys, authManager); err != nil {
					_ = fbConn.conn.Close()
					return nil, newPerDialConnectError("authentication failed", err)
				}
				authManager.SetAuthComplete(true)
				break connectLoop
			}
			respond := shouldRespondToDataAccept(msg)
			if err := fbConn.handleCondAccept(&msg.OpCondAccept, authManager, respond); err != nil {
				_ = fbConn.conn.Close()
				return nil, newPerDialConnectError("authentication failed", err)
			}
			if !respond {
				break connectLoop
			}
			continue connectLoop
		case *fbproto.ErrorResponse:
			_ = fbConn.conn.Close()
			return nil, newPerDialConnectError("authentication failed", ErrorResponseToFbError(msg))
		default:
			_ = fbConn.conn.Close()
			return nil, newPerDialConnectError("authentication failed", fmt.Errorf("unexpected message type %T", msg))
		}
	}

	if fbConn.wireCrypt != nil {
		fbConn.wireCrypt.ClearServerKeys()
	}

	if target == connectTargetService {
		if err := fbConn.attachService(config, authManager); err != nil {
			_ = fbConn.conn.Close()
			return nil, newPerDialConnectError("opServiceAttach failed", err)
		}
		fbConn.status = connStatusIdle
		return fbConn, nil
	}

	var dpb *fbproto.DPBWriterV2
	if config.DPB == nil {
		dpb = fbproto.NewDPBWriterV2()
	} else {
		dpb = fbproto.NewDPBWriterV2FromBytes(config.DPB)
	}
	dpb.PutInt32(fbproto.Isc_dpb_sql_dialect, 3)
	if config.CreateDatabase {
		dpb.PutInt32(fbproto.Isc_dpb_set_db_sql_dialect, 3)
	}
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
		app, err = os.Executable()
		if err == nil {
			if len(app) > 255 {
				app = app[:255]
			}
		} else {
			app = "fbx"
		}
	}

	dpb.PutString(fbproto.Isc_dpb_process_name, app)
	if searchPath, ok := config.RuntimeParams["search_path"]; ok {
		dpb.PutString(fbproto.Isc_dpb_search_path, searchPath)
	}
	if timezone, ok := config.RuntimeParams["timezone"]; ok {
		dpb.PutString(fbproto.Isc_dpb_session_time_zone, timezone)
	}
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
		dpb.PutTag(fbproto.Isc_dpb_specific_auth_data)
		dpb.PutBuffer(authSpecific)
	}

	var attachOrCreate fbproto.ClientMessage
	attachAction := "opAttach"
	if config.CreateDatabase {
		attachAction = "opCreate"
		attachOrCreate = &fbproto.OpCreate{
			File: config.Database,
			DPB:  dpb.Bytes(),
		}
	} else {
		attachOrCreate = &fbproto.OpAttach{
			File: config.Database,
			DPB:  dpb.Bytes(),
		}
	}

	if err = fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(attachOrCreate)
	}); err != nil {
		_ = fbConn.conn.Close()
		return nil, newPerDialConnectError("failed to flush "+attachAction, err)
	}

	if err := fbConn.finishAttachAuth(config, authManager, attachAction); err != nil {
		_ = fbConn.conn.Close()
		return nil, newPerDialConnectError(attachAction+" failed", err)
	}

	fbConn.status = connStatusIdle

	err = fbConn.getDatabaseInfo([]byte{
		fbproto.Isc_info_firebird_version,
		fbproto.Isc_info_attachment_id,
		fbproto.Isc_info_end,
	}, fbConn.parseDatabaseInfo)

	if err != nil {
		_ = fbConn.conn.Close()
		return nil, newPerDialConnectError("error while trying to get database info", err)
	}

	if config.ValidateConnect != nil {
		fbConn.contextWatcher.Unwatch()
		err = config.ValidateConnect(ctx, fbConn)
		if err != nil {
			if _, ok := err.(*NotPreferredError); !ignoreNotPreferredErr || !ok {
				_ = fbConn.conn.Close()
				return nil, newPerDialConnectError("ValidateConnect failed", err)
			}
		}
	}

	return fbConn, nil
}

func (fbConn *FbConn) parseDatabaseInfo(buf []byte) error {
	r := NewServerInfoBufferReader(buf)
	for {
		item, err := r.ReadByte()
		if err != nil {
			return err
		}

		switch item {
		case fbproto.Isc_info_end:
			return nil
		case fbproto.Isc_info_firebird_version:
			v, err := r.ReadStringValue()
			if err != nil {
				return err
			}
			fbConn.serverVersion, err = parseServerVersion(v)
			if err != nil {
				return err
			}
			fbConn.serverMajorVersion = fbConn.serverVersion.Major
		case fbproto.Isc_info_attachment_id:
			attachmentID, err := r.ReadInt64Value()
			if err != nil {
				return err
			}
			fbConn.attachmentID = uint64(attachmentID)
		case fbproto.Isc_info_truncated:
			return InfoTruncationError{0}
		default:
			return fmt.Errorf("unknown database info item: %d", item)
		}
	}
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

type requestStartHandler interface {
	HandleBeforeRequest() error
}

func (fbConn *FbConn) beforeRequest() error {
	if fbConn == nil || fbConn.contextHandler == nil {
		return nil
	}

	handler, ok := fbConn.contextHandler.(requestStartHandler)
	if !ok {
		return nil
	}

	if err := handler.HandleBeforeRequest(); err != nil {
		fbConn.asyncClose()
		return err
	}

	return nil
}

func (fbConn *FbConn) withWriteLock(writeFn func() error) error {
	fbConn.writeMu.Lock()
	defer fbConn.writeMu.Unlock()
	return writeFn()
}

func (fbConn *FbConn) withWriteLockAndFlush(writeFn func() error) error {
	return fbConn.withWriteLock(func() error {
		if err := writeFn(); err != nil {
			return err
		}
		return fbConn.flushWithPotentialWriteReadDeadlockLocked()
	})
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
		if !isNetErr || !netErr.Timeout() {
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

	if err := fbConn.handleServerMessage(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (fbConn *FbConn) receiveSQLData(fields []fbproto.FieldDescriptor) (*fbproto.SQLData, fbproto.ServerMessage, error) {
	row, msg, err := fbConn.client.ReceiveSQLData(fields)
	if err != nil {
		return nil, nil, err
	}
	if err := fbConn.handleServerMessage(msg); err != nil {
		return nil, nil, err
	}
	return row, msg, nil
}

func (fbConn *FbConn) handleServerMessage(msg fbproto.ServerMessage) error {
	switch msg := msg.(type) {
	case *fbproto.ErrorResponse:
		err := ErrorResponseToFbError(msg)
		if !fbConn.handleFbError(err) {
			return err
		}
	case *fbproto.OpResponse:
		if msg.Warnings && fbConn.config.OnFbWarning != nil {
			fbConn.config.OnFbWarning(fbConn, OpResponseToFbWarning(msg))
		}
	}
	return nil
}

func (fbConn *FbConn) handleFbError(err *FbError) bool {
	keepOpen := true
	if handler := fbConn.config.OnFbError; handler != nil {
		keepOpen = handler(fbConn, err)
	}
	if !keepOpen {
		fbConn.asyncClose()
	}
	return keepOpen
}

// Conn returns the underlying net.Conn. This rarely necessary. If the connection will be directly used for reading or
// writing then SyncConn should usually be called before Conn.
func (fbConn *FbConn) Conn() net.Conn {
	return fbConn.conn
}

// AttachmentID returns the server-side attachment identifier for this connection.
// It is the same value exposed by MON$ATTACHMENT_ID and CURRENT_CONNECTION.
func (fbConn *FbConn) AttachmentID() uint64 {
	return fbConn.attachmentID
}

// ServerVersion returns the server version discovered while establishing the
// connection. It does not perform a request to the server.
func (fbConn *FbConn) ServerVersion() ServerVersion {
	return fbConn.serverVersion
}

// ActiveTx returns true if connection have active transactions
func (fbConn *FbConn) ActiveTx() bool {
	return len(fbConn.activeTransactions) > 0
}

// ActiveExplicitTx returns true if the connection has an active user-managed transaction.
func (fbConn *FbConn) ActiveExplicitTx() bool {
	return fbConn.explicitTransaction && fbConn.internalTransaction != nil && !fbConn.internalTransaction.Closed
}

// Client returns the underlying *fbproto.Client
func (fbConn *FbConn) Client() *fbproto.Client {
	return fbConn.client
}

// WireCompressionEnabled reports whether wire compression was negotiated with the server.
func (fbConn *FbConn) WireCompressionEnabled() bool {
	return fbConn != nil && fbConn.client != nil && fbConn.client.CompressionEnabled()
}

// Close closes a connection. It is safe to call Close on an already closed connection. Close attempts a clean close by
// sending the exit message to Firebird/RedDatabase server. However, this could block so ctx is available to limit the time to wait. The
// underlying net.Conn.Close() will always be called regardless of any other errors.
func (fbConn *FbConn) Close(ctx context.Context) (err error) {
	fbConn.statusMu.Lock()
	if fbConn.status == connStatusClosed {
		fbConn.statusMu.Unlock()
		return nil
	}
	fbConn.status = connStatusClosed
	fbConn.statusMu.Unlock()
	fbConn.clearAllInlineBlobCache()
	fbConn.closeEventChannel()

	defer fbConn.closeCleanupDone()
	defer func() {
		err = errors.Join(err, fbConn.conn.Close())
		fbConn.client.WaitForCompressionRead()
	}()

	if ctx != context.Background() {
		// Close may be called while a cancellable query is in progress. This will most often be triggered by panic when
		// a defer closes the connection (possibly indirectly via a transaction or a connection pool). Unwatch to end any
		// previous watch. It is safe to Unwatch regardless of whether a watch is already is progress.
		fbConn.contextWatcher.Unwatch()

		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	// Ignore errors while terminating the protocol; the underlying connection
	// is closed below regardless of whether the server accepted the messages.
	_ = fbConn.withWriteLock(func() error {
		terminateErr := fbConn.client.WriteMessage(&fbproto.OpDetach{})
		if terminateErr == nil {
			terminateErr = fbConn.client.WriteMessage(&fbproto.OpDisconnect{})
		}
		if terminateErr == nil {
			terminateErr = fbConn.flushWithPotentialWriteReadDeadlockLocked()
		}
		return errors.Join(terminateErr, fbConn.client.CloseCompression())
	})
	// Firebird replies to op_detach, but op_disconnect is terminal and has no response.
	// Close is best-effort, so it closes the socket instead of waiting for detach acknowledgement.
	return nil
}

// DropDatabase drops the currently attached database and closes the connection.
func (fbConn *FbConn) DropDatabase(ctx context.Context) (err error) {
	if err = fbConn.lock(); err != nil {
		return err
	}
	defer fbConn.unlock()

	if err = fbConn.beforeRequest(); err != nil {
		return err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	dropStarted := false
	defer func() {
		if !dropStarted {
			return
		}
		fbConn.status = connStatusClosed
		fbConn.clearAllInlineBlobCache()
		compressionErr := fbConn.withWriteLock(func() error {
			return fbConn.client.CloseCompression()
		})
		closeErr := fbConn.conn.Close()
		fbConn.client.WaitForCompressionRead()
		fbConn.closeCleanupDone()
		err = errors.Join(err, closeErr, compressionErr)
	}()

	dropStarted = true
	if err = fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(&fbproto.OpDropDatabase{})
	}); err != nil {
		return err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
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

// asyncClose marks the connection as closed and asynchronously sends a cancel query message and closes the underlying
// connection.
func (fbConn *FbConn) asyncClose() {
	fbConn.statusMu.Lock()
	if fbConn.status == connStatusClosed {
		fbConn.statusMu.Unlock()
		return
	}
	fbConn.status = connStatusClosed
	fbConn.statusMu.Unlock()
	fbConn.clearAllInlineBlobCache()
	fbConn.closeEventChannel()

	go func() {
		defer fbConn.closeCleanupDone()
		_ = fbConn.conn.Close()
		_ = fbConn.withWriteLock(func() error {
			return fbConn.client.CloseCompression()
		})
		fbConn.client.WaitForCompressionRead()
	}()
}

func (fbConn *FbConn) closeCleanupDone() {
	fbConn.cleanupDoneOnce.Do(func() {
		close(fbConn.cleanupDone)
	})
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
	fbConn.statusMu.Lock()
	s := fbConn.status
	fbConn.statusMu.Unlock()
	return s < connStatusIdle
}

// IsBusy reports if the connection is busy.
func (fbConn *FbConn) IsBusy() bool {
	fbConn.statusMu.Lock()
	s := fbConn.status
	fbConn.statusMu.Unlock()
	return s == connStatusBusy
}

// lock locks the connection.
func (fbConn *FbConn) lock() error {
	fbConn.statusMu.Lock()
	defer fbConn.statusMu.Unlock()
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
	fbConn.statusMu.Lock()
	defer fbConn.statusMu.Unlock()
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
	Schema            string
	Nullable          bool
	ArrayData         *Array
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
		res[i].Schema = src[i].Schema
		res[i].Nullable = src[i].Nullable
	}

	return res
}

// Prepare creates a connection-scoped prepared statement using the current
// internal transaction to resolve its metadata. Statement.Exec uses the
// connection's current internal transaction on each execution.
func (fbConn *FbConn) Prepare(ctx context.Context, sql string) (*Statement, error) {
	tx, err := fbConn.ensureInternalTransaction(ctx, nil, false)
	if err != nil {
		return nil, err
	}
	st, err := fbConn.prepareTx(ctx, tx, sql, statementScopeConnection)
	if err != nil {
		if rbErr := fbConn.rollbackInternalTxIfAutocommitOnError(ctx, tx, err); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback internal tx: %w", rbErr))
		}
		return nil, err
	}
	return st, nil
}

type statementScope uint8

const (
	statementScopeTransaction statementScope = iota
	statementScopeConnection
)

// PrepareTx creates a prepared statement bound to tx.
func (fbConn *FbConn) PrepareTx(ctx context.Context, tx *FBTransaction, sql string) (*Statement, error) {
	return fbConn.prepareTx(ctx, tx, sql, statementScopeTransaction)
}

// PrepareTxForConnection creates a connection-scoped prepared statement using
// tx to resolve its metadata. [Statement.Exec] uses the connection's current
// internal transaction, while [Statement.ExecTx] uses its explicit transaction.
// Callers must only retain the statement while its metadata remains valid
// outside tx.
func (fbConn *FbConn) PrepareTxForConnection(ctx context.Context, tx *FBTransaction, sql string) (*Statement, error) {
	return fbConn.prepareTx(ctx, tx, sql, statementScopeConnection)
}

func (fbConn *FbConn) prepareTx(ctx context.Context, tx *FBTransaction, sql string, scope statementScope) (*Statement, error) {
	if err := fbConn.lock(); err != nil {
		return nil, err
	}
	defer fbConn.unlock()

	if err := fbConn.beforeRequest(); err != nil {
		return nil, err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return nil, newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	return fbConn.internalPrepareTx(ctx, tx, sql, scope)
}

func (fbConn *FbConn) internalPrepareTx(
	ctx context.Context,
	tx *FBTransaction,
	sql string,
	scope statementScope,
) (*Statement, error) {
	var err error

	if err = fbConn.withWriteLockAndFlush(func() error {
		if err := fbConn.client.WriteMessage(&fbproto.OpAllocateStatement{}); err != nil {
			return err
		}
		return fbConn.client.WriteMessage(&fbproto.OpPrepareStatement{
			Encoding:          fbConn.Encoding,
			TransactionHandle: tx.Handle,
			StatementHandle:   0xFFFF,
			Dialect:           3,
			Query:             sql,
		})
	}); err != nil {
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
		if fbConn.IsClosed() {
			return nil, err
		}
		if cleanupErr := fbConn.freeStatementHandleLocked(statementHandle); cleanupErr != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, err
	}

	desc := &StatementDescription{
		SQL:             sql,
		Fields:          fbConn.convertRowDescription(statementInfo.FieldDescriptors),
		Parameters:      fbConn.convertRowDescription(statementInfo.ParametersDescriptors),
		StatementHandle: statementHandle,
		StatementInfo:   &statementInfo,
	}

	var tables, fields []string
	var hasArrays bool
	collectArrayIdentifiers := func(description FieldDescription) {
		if !description.IsArray() {
			return
		}
		hasArrays = true
		if description.OriginalTableName != "" && !slices.Contains(tables, description.OriginalTableName) {
			tables = append(tables, description.OriginalTableName)
		}
		if description.OriginalName != "" && !slices.Contains(fields, description.OriginalName) {
			fields = append(fields, description.OriginalName)
		}
	}

	for _, field := range desc.Fields {
		collectArrayIdentifiers(field)
	}
	for _, param := range desc.Parameters {
		collectArrayIdentifiers(param)
	}

	if hasArrays {
		arrays := map[metadataMapKey]*Array{}
		if len(tables) > 0 && len(fields) > 0 {
			arrays, err = getArrayInfo(fbConn, ctx, tx, fields, tables)
		}
		if err != nil {
			if cleanupErr := fbConn.freeStatementHandleLocked(statementHandle); cleanupErr != nil {
				return nil, errors.Join(err, cleanupErr)
			}
			return nil, err
		}

		err = attachArrayMetadata(desc.Fields, arrays, "result field")
		if err == nil {
			err = attachArrayMetadata(desc.Parameters, arrays, "parameter")
		}
		if err != nil {
			if cleanupErr := fbConn.freeStatementHandleLocked(statementHandle); cleanupErr != nil {
				return nil, errors.Join(err, cleanupErr)
			}
			return nil, err
		}
	}

	st := &Statement{
		conn:        fbConn,
		Description: desc,
	}
	if scope == statementScopeTransaction {
		st.transaction = tx
	}
	fbConn.activeStatements[st] = struct{}{}
	return st, nil
}

// CancelRequest sends a cancel request to the server. It returns an error if unable to deliver the cancel
// request, but lack of an error does not ensure that the query was canceled. Delivery errors do not close the
// connection; callers that consider a failed cancel request fatal must close the connection themselves.
func (fbConn *FbConn) CancelRequest(ctx context.Context) error {
	return fbConn.sendCancelOperation(ctx, fbproto.Fb_cancel_raise)
}

func (fbConn *FbConn) sendCancelOperation(ctx context.Context, kind int) error {
	if ctx != context.Background() {
		contextWatcher := ctxwatch.NewContextWatcher(&DeadlineContextWatcherHandler{Conn: fbConn.conn})
		contextWatcher.Watch(ctx)
		defer contextWatcher.Unwatch()
	}

	if err := fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(&fbproto.OpCancel{Kind: kind})
	}); err != nil {
		return err
	}

	return nil
}

// ExecImmediate executes statement immediate inside internal transaction
func (fbConn *FbConn) ExecImmediate(ctx context.Context, sql string) error {
	tx, err := fbConn.ensureInternalTransaction(ctx, nil, false)
	if err != nil {
		return err
	}
	if err := fbConn.ExecImmediateTx(ctx, tx, sql); err != nil {
		if rbErr := fbConn.rollbackInternalTxIfAutocommitOnError(ctx, tx, err); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback internal tx: %w", rbErr))
		}
		return err
	}
	return nil
}

// ExecImmediateTx executes statement immediate inside specified transaction
func (fbConn *FbConn) ExecImmediateTx(ctx context.Context, tx *FBTransaction, sql string) error {
	var err error
	watched := false

	if err = fbConn.lock(); err != nil {
		return err
	}

	if err = fbConn.beforeRequest(); err != nil {
		fbConn.unlock()
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
		watched = true
	}

	txHandle := 0
	if tx != nil {
		txHandle = tx.Handle
	}
	if err = fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(&fbproto.OpExecuteImmediate{
			Encoding:          fbConn.Encoding,
			TransactionHandle: txHandle,
			Dialect:           3,
			Query:             sql,
		})
	}); err != nil {
		if watched {
			fbConn.contextWatcher.Unwatch()
		}
		fbConn.unlock()
		fbConn.asyncClose()
		return err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		if watched {
			fbConn.contextWatcher.Unwatch()
		}
		fbConn.unlock()
		fbConn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}

	switch m := msg.(type) {
	case *fbproto.OpResponse:
	case *fbproto.ErrorResponse:
		if watched {
			fbConn.contextWatcher.Unwatch()
		}
		fbConn.unlock()
		return ErrorResponseToFbError(m)
	default:
		if watched {
			fbConn.contextWatcher.Unwatch()
		}
		fbConn.unlock()
		panic(fmt.Sprintf("wrong response message type %T instead expected OpResponse or ErrorResponse", m))
	}

	if watched {
		fbConn.contextWatcher.Unwatch()
	}
	fbConn.unlock()
	if tx != nil && tx == fbConn.internalTransaction && fbConn.config.Autocommit && !fbConn.explicitTransaction {
		return tx.Commit(ctx)
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
		res := ResultReader{
			fbConn: fbConn,
			ctx:    ctx,
			closed: true,
		}
		res.concludeCommand(QueryStatus{}, err)
		return &res
	}
	return fbConn.ExecParamsTx(ctx, tx, sql, paramValues)
}

func (fbConn *FbConn) ExecParamsTx(ctx context.Context, tx *FBTransaction, sql string, paramValues [][]byte) *ResultReader {
	st, err := fbConn.PrepareTx(ctx, tx, sql)
	if err != nil {
		if rbErr := fbConn.rollbackInternalTxIfAutocommitOnError(ctx, tx, err); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback internal tx: %w", rbErr))
		}
		res := ResultReader{
			fbConn: fbConn,
			ctx:    ctx,
			closed: true,
		}
		res.concludeCommand(QueryStatus{}, err)
		return &res
	}

	rr := st.Exec(ctx, paramValues)
	if rr.err != nil {
		cleanupCtx, cancel := cleanupContext(ctx)
		_ = st.Deallocate(cleanupCtx)
		cancel()

		if rbErr := fbConn.rollbackInternalTxIfAutocommitOnError(ctx, tx, rr.err); rbErr != nil {
			rr.err = errors.Join(rr.err, fmt.Errorf("rollback internal tx: %w", rbErr))
		}
		return rr
	}
	rr.ownsStatement = st
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

	if err = fbConn.beforeRequest(); err != nil {
		return err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	if err = fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(&fbproto.OpDisconnect{})
	}); err != nil {
		fbConn.asyncClose()
		return err
	}

	return nil
}

func (fbConn *FbConn) getArrayMetadata(ctx context.Context, tx *FBTransaction, serverVersion int, tableNames, fieldNames []string) ([]map[string][]byte, []dimensionsMapKey, error) {
	if len(tableNames) == 0 || len(fieldNames) == 0 {
		return nil, nil, nil
	}

	tables := ""
	for _, table := range tableNames {
		tables += "'" + strings.ReplaceAll(table, "'", "''") + "',"
	}
	fields := ""
	for _, field := range fieldNames {
		fields += "'" + strings.ReplaceAll(field, "'", "''") + "',"
	}

	var queryString string
	if serverVersion < 6 {
		queryString = fmt.Sprintf(
			`SELECT
    Y.RDB$FIELD_TYPE,
    Y.RDB$FIELD_SCALE,
    Y.RDB$FIELD_LENGTH,
    CASE
		WHEN Y.RDB$FIELD_TYPE IN (%d, %d) THEN Y.RDB$CHARACTER_SET_ID
		ELSE Y.RDB$FIELD_SUB_TYPE
    END AS RDB$FIELD_SUB_TYPE,
    X.RDB$FIELD_SOURCE,
    X.RDB$RELATION_NAME,
    X.RDB$FIELD_NAME
		FROM RDB$RELATION_FIELDS X, RDB$FIELDS Y
		WHERE X.RDB$FIELD_SOURCE = Y.RDB$FIELD_NAME
		  AND X.RDB$RELATION_NAME IN (%s)
		  AND X.RDB$FIELD_NAME IN (%s)`,
			fbproto.Isc_blr_text,
			fbproto.Isc_blr_varying,
			tables[:len(tables)-1],
			fields[:len(fields)-1],
		)
	} else {
		queryString = fmt.Sprintf(
			`SELECT
    Y.RDB$FIELD_TYPE,
    Y.RDB$FIELD_SCALE,
    Y.RDB$FIELD_LENGTH,
    CASE
		WHEN Y.RDB$FIELD_TYPE IN (%d, %d) THEN Y.RDB$CHARACTER_SET_ID
		ELSE Y.RDB$FIELD_SUB_TYPE
    END AS RDB$FIELD_SUB_TYPE,
    X.RDB$FIELD_SOURCE,
    X.RDB$RELATION_NAME,
    X.RDB$FIELD_NAME,
    X.RDB$SCHEMA_NAME
	FROM RDB$RELATION_FIELDS X, RDB$FIELDS Y
		WHERE X.RDB$FIELD_SOURCE = Y.RDB$FIELD_NAME
		  AND X.RDB$SCHEMA_NAME = Y.RDB$SCHEMA_NAME
		  AND X.RDB$RELATION_NAME IN (%s)
		  AND X.RDB$FIELD_NAME IN (%s)`,
			fbproto.Isc_blr_text,
			fbproto.Isc_blr_varying,
			tables[:len(tables)-1],
			fields[:len(fields)-1],
		)
	}

	st, err := fbConn.internalPrepareTx(ctx, tx, queryString, statementScopeTransaction)

	if err != nil {
		return nil, nil, err
	}

	defer func() {
		cleanupCtx, cancel := cleanupContext(ctx)
		_ = st.deallocateLocked(cleanupCtx)
		cancel()
	}()

	rr := &ResultReader{
		fbConn:           fbConn,
		ctx:              ctx,
		boundTransaction: tx,
	}

	result := st.internalExecTx(ctx, tx, rr, nil).readWithoutCommit()

	if result.Err != nil {
		return nil, nil, result.Err
	}

	metadataList := make([]map[string][]byte, 0)
	sources := []dimensionsMapKey{}
	for _, row := range result.Rows {

		metadata := make(map[string][]byte)

		metadata["DATATYPE"] = row[0]
		metadata["SCALE"] = row[1]
		metadata["LENGTH"] = row[2]
		if row[3] == nil {
			metadata["SUBTYPE"] = make([]byte, 4)
		} else {
			metadata["SUBTYPE"] = row[3]
		}
		metadata["SOURCE"] = row[4]
		var key dimensionsMapKey

		metadata["TABLE"] = row[5]
		metadata["FIELD"] = row[6]
		if serverVersion < 6 {
			key = dimensionsMapKey{source: strings.TrimSpace(string(row[4]))}
		} else {
			metadata["SCHEMA"] = row[7]
			key = dimensionsMapKey{source: strings.TrimSpace(string(row[4])), schema: strings.TrimSpace(string(row[7]))}
		}
		sources = append(sources, key)

		metadataList = append(metadataList, metadata)
	}

	return metadataList, sources, nil
}

func (fbConn *FbConn) getArrayDimensions(ctx context.Context, tx *FBTransaction, serverVersion int, sources []dimensionsMapKey) (map[dimensionsMapKey][][]int, error) {
	if len(sources) == 0 {
		return map[dimensionsMapKey][][]int{}, nil
	}

	sourcesString := ""
	schemasString := ""
	for _, source := range sources {
		sourcesString += "'" + strings.ReplaceAll(string(source.source), "'", "''") + "',"
		schemasString += "'" + strings.ReplaceAll(string(source.schema), "'", "''") + "',"
	}

	var queryString string
	if serverVersion < 6 {
		queryString = fmt.Sprintf(
			`SELECT
	X.RDB$LOWER_BOUND,
	X.RDB$UPPER_BOUND,
	X.RDB$FIELD_NAME
	FROM RDB$FIELD_DIMENSIONS X
		WHERE X.RDB$FIELD_NAME IN (%s)
		ORDER BY X.RDB$FIELD_NAME, X.RDB$DIMENSION`,
			sourcesString[:len(sourcesString)-1],
		)
	} else {
		queryString = fmt.Sprintf(
			`SELECT
    X.RDB$LOWER_BOUND,
    X.RDB$UPPER_BOUND,
    X.RDB$FIELD_NAME,
    X.RDB$SCHEMA_NAME
    FROM RDB$FIELD_DIMENSIONS X
		WHERE X.RDB$FIELD_NAME IN (%s) AND X.RDB$SCHEMA_NAME IN (%s)
        ORDER BY X.RDB$FIELD_NAME, X.RDB$SCHEMA_NAME, X.RDB$DIMENSION`,
			sourcesString[:len(sourcesString)-1],
			schemasString[:len(schemasString)-1],
		)
	}

	st, err := fbConn.internalPrepareTx(ctx, tx, queryString, statementScopeTransaction)

	if err != nil {
		return nil, err
	}

	defer func() {
		cleanupCtx, cancel := cleanupContext(ctx)
		_ = st.deallocateLocked(cleanupCtx)
		cancel()
	}()

	rr := &ResultReader{
		fbConn:           fbConn,
		ctx:              ctx,
		boundTransaction: tx,
	}

	result := st.internalExecTx(ctx, tx, rr, nil).readWithoutCommit()

	if result.Err != nil {
		return nil, result.Err
	}

	dimensions := map[dimensionsMapKey][][]int{}
	for _, row := range result.Rows {
		key := dimensionsMapKey{source: strings.TrimSpace(string(row[2]))} // row[2] - field name
		if serverVersion >= 6 {
			key.schema = strings.TrimSpace(string(row[3])) // row[3] - schema name
		}
		dimensions[key] = append(
			dimensions[key],
			[]int{
				int(int32(binary.BigEndian.Uint32(row[0]))), // lower bound
				int(int32(binary.BigEndian.Uint32(row[1]))), // upper bound
			},
		)
	}

	return dimensions, nil
}

func (fbConn *FbConn) getArray(ctx context.Context, tx *FBTransaction, arrayId int64, datatype uint32, length, l int, sdl []byte) ([]byte, error) {
	if tx == nil || tx.Closed {
		return nil, ErrTransactionClosed
	}

	opGetSlice := fbproto.OpGetSlice{
		TransactionHandle: tx.Handle,
		ArrayHandle:       arrayId,
		SliceLength:       length,
		SliceDescriptor:   sdl,
	}

	if err := fbConn.client.WriteMessage(&opGetSlice); err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	msg, err := fbConn.client.ReadSlice(datatype, l)

	if err != nil {
		fbConn.asyncClose()
		return nil, normalizeTimeoutError(ctx, err)
	}
	if err := fbConn.handleServerMessage(msg); err != nil {
		return nil, err
	}

	switch m := msg.(type) {
	case *fbproto.OpSliceResponse:
		return m.SliceData, nil
	case *fbproto.OpResponse:
		return nil, fmt.Errorf("array getting error: %s", m.Message)
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	default:
		return nil, fmt.Errorf("unexpected response type %T", msg)
	}
}

func (fbConn *FbConn) putArray(ctx context.Context, tx *FBTransaction, array []byte, length int, sdl []byte) ([]byte, error) {
	if tx == nil || tx.Closed {
		return nil, ErrTransactionClosed
	}

	opPutSlice := fbproto.OpPutSlice{
		TransactionHandle: tx.Handle,
		ArrayHandle:       0,
		SliceLength:       length,
		SliceDescriptor:   sdl,
		Slice:             array,
	}

	if err := fbConn.client.WriteMessage(&opPutSlice); err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.asyncClose()
		return nil, err
	}

	msg, err := fbConn.receiveMessage()

	if err != nil {
		fbConn.asyncClose()
		return nil, normalizeTimeoutError(ctx, err)
	}

	switch m := msg.(type) {
	case *fbproto.OpResponse:
		if m.BlobId == 0 {
			return nil, fmt.Errorf("array putting error: %s", m.Message)
		}
		bufId := make([]byte, 8)
		binary.BigEndian.PutUint64(bufId, uint64(m.BlobId))
		return bufId, nil
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	default:
		return nil, fmt.Errorf("unexpected response type %T", msg)
	}
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

	ownsStatement *Statement
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

func (rr *ResultReader) readWithoutCommit() *Result {
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

	br.QueryStatus, br.Err = rr.closeWithoutCommit()

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
	if rr.closed || rr.err != nil {
		return false
	}
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
			rowBlr, err = rr.statementDescription.StatementInfo.GetRowBlr()
			if err != nil {
				rr.fbConn.asyncClose()
				rr.concludeCommand(QueryStatus{}, normalizeTimeoutError(rr.ctx, normalizeConnClosedError(err)))
				return false
			}
		}
		fetchBatchSize := DefaultFetchBatchSize
		if rr.fbConn != nil && rr.fbConn.config != nil && rr.fbConn.config.FetchBatchSize > 0 {
			fetchBatchSize = rr.fbConn.config.FetchBatchSize
		}
		if err = rr.fbConn.withWriteLockAndFlush(func() error {
			return rr.fbConn.client.WriteMessage(&fbproto.OpFetch{
				StatementHandle: rr.statementDescription.StatementHandle,
				RowBlr:          rowBlr,
				Messages:        fetchBatchSize,
			})
		}); err != nil {
			rr.fbConn.asyncClose()
			rr.concludeCommand(QueryStatus{}, normalizeTimeoutError(rr.ctx, normalizeConnClosedError(err)))
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

	var rowsBuf [][][]byte

	for rr.fetchResp.MoreRows() {
		rowBuf := [][]byte{}
		row, msg, err := rr.fbConn.receiveSQLData(rr.statementDescription.StatementInfo.FieldDescriptors)
		if err != nil {
			rr.fbConn.asyncClose()
			rr.concludeCommand(QueryStatus{}, normalizeTimeoutError(rr.ctx, err))
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

		for i := 0; i < colCount; i++ {
			col := make([]byte, 0)
			if row.RowData[i] == nil {
				rowBuf = append(rowBuf, nil)
				continue
			}
			col = append(col, row.RowData[i]...)
			rowBuf = append(rowBuf, col)
		}
		rowsBuf = append(rowsBuf, rowBuf)

		if rr.fetchResp.EndOfCursor() {
			rr.eoc = true
		}
	}
	if rr.fetchResp.EndOfCursor() {
		rr.eoc = true
	}
	for _, row := range rowsBuf {
		rowStart := len(rr.bufferedCols)
		for j, elem := range row {
			if IsArrayType(uint32(rr.fieldDescriptions[j].Type)) {
				array, err := getArray(rr.fbConn, rr.ctx, rr.boundTransaction, rr.fieldDescriptions[j].ArrayData, elem)
				if err != nil {
					rr.concludeCommand(QueryStatus{}, err)
					return false
				}
				elem = array
			}
			if elem == nil {
				rr.bufferedCols = append(rr.bufferedCols, bufferedCol{length: bufferedColNullLength})
				continue
			}
			start := len(rr.bufferedArena)
			rr.bufferedArena = append(rr.bufferedArena, elem...)
			rr.bufferedCols = append(rr.bufferedCols, bufferedCol{start: start, length: len(elem)})
		}
		rr.bufferedRows = append(rr.bufferedRows, bufferedRow{colStart: rowStart})
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
		_, msg, err := rr.fbConn.receiveSQLData(rr.statementDescription.StatementInfo.FieldDescriptors)
		if err != nil {
			rr.fbConn.asyncClose()
			rr.concludeCommand(QueryStatus{}, normalizeTimeoutError(rr.ctx, err))
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

// SetCallerErr propagates an error from upstream callers (e.g., higher-level row
// scan processing in fbx that detects a failure not visible to fbconn) into the ResultReader.
func (rr *ResultReader) SetCallerErr(err error) {
	if err != nil && rr.err == nil {
		rr.err = err
	}
}

// cleanupGracePeriod bounds cleanup network calls (e.g. op_free_statement) that
// must run even after the caller's ctx has been cancelled.
const cleanupGracePeriod = 5 * time.Second

// cleanupContext detaches from parent cancellation but preserves Values, with a
// bounded timeout for the network call.
func cleanupContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), cleanupGracePeriod)
}

// rollbackInternalTxIfAutocommitOnError rolls back the connection's implicit internal
// transaction on an error path. No-op when tx is user-supplied, already closed,
// Autocommit is disabled, or opErr is SafeToRetry (the operation failed before
// reaching the server, so server-side tx state cannot have changed).
func (fbConn *FbConn) rollbackInternalTxIfAutocommitOnError(ctx context.Context, tx *FBTransaction, opErr error) error {
	if tx == nil || tx != fbConn.internalTransaction || tx.Closed {
		return nil
	}
	if !fbConn.config.Autocommit || fbConn.explicitTransaction {
		return nil
	}
	if SafeToRetry(opErr) {
		return nil
	}
	cleanupCtx, cancel := cleanupContext(ctx)
	defer cancel()
	if err := tx.Rollback(cleanupCtx); err != nil {
		// A failed rollback leaves the state of the implicit transaction unknown,
		// so the connection must not be reused even when the error is not fatal.
		fbConn.asyncClose()
		return err
	}
	return nil
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
		rr.fbConn.contextWatcher.Unwatch()
		rr.fbConn.unlock()

		if rr.ownsStatement != nil && !rr.ownsStatement.closed {
			ctx, cancel := cleanupContext(rr.ctx)
			if dErr := rr.ownsStatement.Deallocate(ctx); dErr != nil && rr.err == nil {
				rr.err = dErr
			}
			cancel()
		}

		if rr.boundTransaction != nil && rr.boundTransaction == rr.fbConn.internalTransaction && rr.fbConn.config.Autocommit && !rr.fbConn.explicitTransaction {
			if rr.err != nil {
				if rbErr := rr.fbConn.rollbackInternalTxIfAutocommitOnError(rr.ctx, rr.boundTransaction, rr.err); rbErr != nil {
					rr.err = errors.Join(rr.err, fmt.Errorf("rollback internal tx: %w", rbErr))
				}
			} else {
				if err := rr.boundTransaction.Commit(context.Background()); err != nil {
					rr.err = err
				}
			}
		}
	}
	return rr.queryStatus, rr.err
}

func (rr *ResultReader) closeWithoutCommit() (QueryStatus, error) {
	if !rr.closed {
		if rr.err == nil {
			rr.drainCurrentFetch()
			if rr.err == nil {
				if err := rr.populateRowsAffected(); err != nil {
					rr.err = err
				}
			}
		}
		rr.closed = true
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

// EscapeString returns s with each single quote doubled for use inside a
// Firebird string literal. It does not add the surrounding single quotes.
func (fbConn *FbConn) EscapeString(s string) (string, error) {
	return strings.ReplaceAll(s, "'", "''"), nil
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

	if err = fbConn.beforeRequest(); err != nil {
		return err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	if err = fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(&fbproto.OpPing{})
	}); err != nil {
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
	fbConn.writeInProgress.Store(true)
}

// exitPotentialWriteReadDeadlock must be called after a call to enterPotentialWriteReadDeadlock.
func (fbConn *FbConn) exitPotentialWriteReadDeadlock() {
	defer fbConn.writeInProgress.Store(false)

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

func (fbConn *FbConn) flushWithPotentialWriteReadDeadlockLocked() error {
	fbConn.enterPotentialWriteReadDeadlock()
	defer fbConn.exitPotentialWriteReadDeadlock()
	err := fbConn.client.Flush()
	return err
}

func (fbConn *FbConn) flushWithPotentialWriteReadDeadlock() error {
	return fbConn.withWriteLock(func() error {
		return fbConn.flushWithPotentialWriteReadDeadlockLocked()
	})
}

// SyncConn prepares the underlying net.Conn for direct use. FbConn may internally buffer reads or use goroutines for
// background IO. This means that any direct use of the underlying net.Conn may be corrupted if a read is already
// buffered or a read is in progress. SyncConn drains read buffers and stops background IO. In some cases this may
// require sending a ping to the server. ctx can be used to cancel this operation. This should be called before any
// operation that will use the underlying net.Conn directly. e.g. Before Conn() or Hijack().
func (fbConn *FbConn) SyncConn(ctx context.Context) error {
	if fbConn.status == connStatusClosed {
		return fmt.Errorf("SyncConn: %w", ErrConnClosed)
	}

	for i := 0; i < 10; i++ {
		if fbConn.bgReader.Status() == bgreader.StatusStopped &&
			fbConn.bgReader.Buffered() == 0 &&
			fbConn.client.LazyCounter() == 0 &&
			fbConn.peekedMsg == nil &&
			fbConn.bufferingReceiveMsg == nil {
			return nil
		}

		if err := fbConn.Ping(ctx); err != nil {
			return fmt.Errorf("SyncConn: Ping failed while syncing conn: %w", err)
		}
	}

	// Reached only if the server keeps pushing async data (e.g. events) and the read buffer never drains.
	return errors.New("SyncConn: conn never synchronized")
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
	Conn               net.Conn
	DBHandle           int // server-side attachment handle
	AttachmentID       uint64
	TxStatus           byte
	Client             *fbproto.Client    // reused as-is by Construct: holds active wire-crypt codec, lazyCounter and Encoding state
	WireCrypt          *wirecrypt.Manager // wire-crypt plugin factory; not the active cipher (cipher lives inside Client)
	ProtocolVersion    int
	ProtocolFeatures   ProtocolFeatures
	ServerVersion      ServerVersion
	ServerMajorVersion int
	InlineBlobSize     int
	Config             *Config
	CustomData         map[string]any
}

// Hijack extracts the internal connection data. fbConn must be in an idle state with no active transactions,
// prepared statements, or other server-side resources; otherwise an error is returned. SyncConn should be called
// immediately before Hijack. fbConn is unusable after hijacking. Hijacking is typically only useful when using
// fbconn to establish a connection, but taking complete control of the raw connection after that (e.g. a load
// balancer or proxy).
//
// Due to the necessary exposure of internal implementation details, it is not covered by the semantic versioning
// compatibility.
func (fbConn *FbConn) Hijack() (*HijackedConn, error) {
	if err := fbConn.lock(); err != nil {
		return nil, err
	}

	// Live server-side handles must be released before handing the socket over; otherwise they leak until detach.
	hasLiveInternalTx := fbConn.internalTransaction != nil && !fbConn.internalTransaction.Closed
	if hasLiveInternalTx || len(fbConn.activeTransactions) > 0 || len(fbConn.activeStatements) > 0 {
		fbConn.unlock()
		return nil, errors.New("Hijack: connection has active transactions or prepared statements")
	}

	fbConn.client.ClearAllInlineBlobCache()

	fbConn.status = connStatusClosed

	return &HijackedConn{
		Conn:               fbConn.conn,
		DBHandle:           fbConn.attachmentHandle,
		AttachmentID:       fbConn.attachmentID,
		TxStatus:           fbConn.txStatus,
		Client:             fbConn.client,
		WireCrypt:          fbConn.wireCrypt,
		ProtocolVersion:    fbConn.protocolVersion,
		ProtocolFeatures:   fbConn.protocolFeatures,
		ServerVersion:      fbConn.serverVersion,
		ServerMajorVersion: fbConn.serverMajorVersion,
		InlineBlobSize:     fbConn.inlineBlobSize,
		Config:             fbConn.config,
		CustomData:         fbConn.customData,
	}, nil
}

// Construct creates a FbConn from a previously hijacked connection. This is the inverse of FbConn.Hijack.
// The connection must be in an idle state.
//
// Due to the necessary exposure of internal implementation details, it is not covered by the semantic versioning
// compatibility.
func Construct(hc *HijackedConn) (*FbConn, error) {
	serverVersion := hc.ServerVersion
	if serverVersion.Major == 0 {
		serverVersion.Major = hc.ServerMajorVersion
	}

	fbConn := &FbConn{
		conn:               hc.Conn,
		attachmentHandle:   hc.DBHandle,
		attachmentID:       hc.AttachmentID,
		txStatus:           hc.TxStatus,
		client:             hc.Client,
		wireCrypt:          hc.WireCrypt,
		protocolVersion:    hc.ProtocolVersion,
		protocolFeatures:   hc.ProtocolFeatures,
		serverVersion:      serverVersion,
		serverMajorVersion: serverVersion.Major,
		inlineBlobSize:     hc.InlineBlobSize,
		config:             hc.Config,
		customData:         hc.CustomData,
		Encoding:           hc.Client.Encoding,

		status:             connStatusIdle,
		cleanupDone:        make(chan struct{}),
		activeTransactions: make(map[*FBTransaction]struct{}),
		activeStatements:   make(map[*Statement]struct{}),
	}

	fbConn.contextHandler = hc.Config.BuildContextWatcherHandler(fbConn)
	fbConn.contextWatcher = ctxwatch.NewContextWatcher(fbConn.contextHandler)

	fbConn.bgReader = bgreader.New(fbConn.conn)
	fbConn.bgReaderStarted = make(chan struct{})
	fbConn.slowWriteTimer = time.AfterFunc(time.Duration(math.MaxInt64),
		func() {
			fbConn.bgReader.Start()
			fbConn.bgReaderStarted <- struct{}{}
		},
	)
	fbConn.slowWriteTimer.Stop()

	fbConn.writer = bufio.NewWriter(fbConn.conn)

	fbConn.client.Rebind(fbConn.bgReader, fbConn.writer)

	return fbConn, nil
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

// StartTransactionTPB starts new transaction not bound to connection with TPB block
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

		if err := fbConn.beforeRequest(); err != nil {
			return nil, err
		}

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
	if err = fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(&opTransaction)
	}); err != nil {
		fbConn.asyncClose()
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
	if err := h.Conn.SetDeadline(time.Now().Add(h.DeadlineDelay)); err != nil {
		_ = h.Conn.Close()
	}
}

func (h *DeadlineContextWatcherHandler) HandleUnwatchAfterCancel() {
	if err := h.Conn.SetDeadline(time.Time{}); err != nil {
		_ = h.Conn.Close()
	}
}

// CancelRequestContextWatcherHandler manages cancel state around each watched request and handles canceled contexts by
// sending a cancel request to the server. It also sets a deadline on a net.Conn as a fallback.
type CancelRequestContextWatcherHandler struct {
	Conn *FbConn

	// CancelRequestDelay is the delay before sending the cancel request to the server.
	CancelRequestDelay time.Duration

	// DeadlineDelay is the delay to set on the deadline set on net.Conn when the context is canceled.
	DeadlineDelay time.Duration

	// CancelResetTimeout is the timeout for cancel state operations (enable/disable).
	CancelResetTimeout time.Duration

	cancelFinishedChan             chan struct{}
	handleUnwatchAfterCancelCalled func()
	requestCancelEnabled           atomic.Bool
	pendingCancelReset             atomic.Bool
	sendCancelOperationFunc        func(context.Context, int) error
}

func (h *CancelRequestContextWatcherHandler) HandleBeforeRequest() error {
	if !h.pendingCancelReset.Load() || !h.canManageCancelState() {
		return nil
	}

	if err := h.sendCancelOperationWithTimeout(fbproto.Fb_cancel_disable); err != nil {
		return err
	}
	if err := h.sendCancelOperationWithTimeout(fbproto.Fb_cancel_enable); err != nil {
		return err
	}

	h.pendingCancelReset.Store(false)
	return nil
}

func (h *CancelRequestContextWatcherHandler) HandleWatch(context.Context) {
	if !h.canManageCancelState() {
		return
	}

	h.requestCancelEnabled.Store(false)

	if err := h.sendCancelOperationWithTimeout(fbproto.Fb_cancel_enable); err != nil {
		h.Conn.asyncClose()
		return
	}

	h.requestCancelEnabled.Store(true)
}

func (h *CancelRequestContextWatcherHandler) HandleCancel(context.Context) {
	h.cancelFinishedChan = make(chan struct{})
	var handleUnwatchedAfterCancelCalledCtx context.Context
	handleUnwatchedAfterCancelCalledCtx, h.handleUnwatchAfterCancelCalled = context.WithCancel(context.Background())

	now := time.Now()
	deadline := now.Add(h.DeadlineDelay)
	if !deadline.After(now) {
		// Keep a small fallback window to deliver op_cancel even when
		// DeadlineDelay is zero or negative.
		deadline = now.Add(100 * time.Millisecond)
	}
	if err := h.Conn.conn.SetDeadline(deadline); err != nil {
		close(h.cancelFinishedChan)
		h.Conn.asyncClose()
		return
	}

	go func() {
		defer close(h.cancelFinishedChan)

		if h.CancelRequestDelay > 0 {
			select {
			case <-handleUnwatchedAfterCancelCalledCtx.Done():
				return
			case <-time.After(h.CancelRequestDelay):
			}
		}

		cancelRequestParentCtx := context.Background()
		if h.CancelRequestDelay == 0 {
			// If a write is currently in progress, wait briefly for it to finish.
			// This avoids racing a concurrent write+flush path while preserving a
			// best-effort cancel attempt in zero-delay mode.
			for h.Conn.writeInProgress.Load() {
				if time.Now().After(deadline) {
					return
				}
				time.Sleep(time.Millisecond)
			}
		}

		cancelRequestCtx, cancel := context.WithDeadline(cancelRequestParentCtx, deadline)
		defer cancel()
		if err := h.sendCancelRequest(cancelRequestCtx); err != nil {
			// Ignore timeout/cancel outcomes for cancel request itself; they only
			// mean that the deadline fallback should win.
			var netErr net.Error
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
				return
			}
			// A failed cancel request can leave the connection state ambiguous, so prevent further reuse.
			h.Conn.asyncClose()
		}
	}()
}

func (h *CancelRequestContextWatcherHandler) HandleUnwatch() {
	if !h.requestCancelEnabled.Load() {
		return
	}
	h.requestCancelEnabled.Store(false)

	if err := h.sendCancelOperationWithTimeout(fbproto.Fb_cancel_disable); err != nil {
		h.Conn.asyncClose()
		return
	}
}

func (h *CancelRequestContextWatcherHandler) canManageCancelState() bool {
	if h == nil || h.Conn == nil {
		return false
	}
	if h.sendCancelOperationFunc != nil {
		return true
	}
	// During connect handshake ContextWatcher may be active before fbConn.client is initialized.
	// Skip per-request enable/disable until attachment is fully established.
	if h.Conn.client == nil {
		return false
	}
	if h.Conn.status < connStatusIdle {
		return false
	}
	return true
}

func (h *CancelRequestContextWatcherHandler) HandleUnwatchAfterCancel() {
	h.handleUnwatchAfterCancelCalled()
	<-h.cancelFinishedChan

	if h.requestCancelEnabled.Load() && !h.Conn.IsClosed() {
		h.pendingCancelReset.Store(true)
	}

	if err := h.Conn.conn.SetDeadline(time.Time{}); err != nil {
		h.Conn.asyncClose()
	}
}

func (h *CancelRequestContextWatcherHandler) sendCancelOperationWithTimeout(kind int) error {
	timeout := h.CancelResetTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if h.sendCancelOperationFunc != nil {
		return h.sendCancelOperationFunc(ctx, kind)
	}

	return h.Conn.sendCancelOperation(ctx, kind)
}

func (h *CancelRequestContextWatcherHandler) sendCancelRequest(ctx context.Context) error {
	if h.sendCancelOperationFunc != nil {
		return h.sendCancelOperationFunc(ctx, fbproto.Fb_cancel_raise)
	}
	return h.Conn.CancelRequest(ctx)
}

func getArrayInfo(fbConn *FbConn, ctx context.Context, tx *FBTransaction, fields, tables []string) (map[metadataMapKey]*Array, error) {
	metadataList, sources, err := fbConn.getArrayMetadata(ctx, tx, fbConn.serverMajorVersion, tables, fields)
	if err != nil {
		return nil, err
	}
	dimensions, err := fbConn.getArrayDimensions(ctx, tx, fbConn.serverMajorVersion, sources)
	if err != nil {
		return nil, err
	}

	arrays := map[metadataMapKey]*Array{}

	for _, metadata := range metadataList {
		arr := Array{}
		arr.FieldName = strings.TrimSpace(string(metadata["FIELD"]))
		arr.TableName = strings.TrimSpace(string(metadata["TABLE"]))
		if fbConn.serverMajorVersion >= 6 {
			arr.Schema = strings.TrimSpace(string(metadata["SCHEMA"]))
		}
		arr.Scale = int32(binary.BigEndian.Uint32(metadata["SCALE"]))
		arr.BlrTypeId = binary.BigEndian.Uint32(metadata["DATATYPE"])
		arr.TypeId, err = getSqlTypeFromBlr(binary.BigEndian.Uint32(metadata["DATATYPE"]))
		if err != nil {
			return nil, err
		}
		arr.SubType = int32(binary.BigEndian.Uint32(metadata["SUBTYPE"]))

		arr.Length = binary.BigEndian.Uint32(metadata["LENGTH"])
		if arr.TypeId == fbproto.SQL_TYPE_VARYING || arr.TypeId == fbproto.SQL_TYPE_TEXT {
			name, err := fbproto.CharacterSetIdIntoName(arr.SubType)
			if err != nil {
				return nil, err
			}
			encoding, err := fbproto.NewEncoding(name)
			if err != nil {
				return nil, err
			}
			if encoding.MaxBytesPerChar < fbConn.Encoding.MaxBytesPerChar {
				arr.Length = arr.Length / encoding.MaxBytesPerChar * fbConn.Encoding.MaxBytesPerChar // to read an array of text data when connected in any encoding
				arr.SubType = fbConn.Encoding.CharacterSetId
			}
		}

		metadataKey := metadataMapKey{
			schema: arr.Schema,
			table:  arr.TableName,
			field:  arr.FieldName,
		}

		dimensionsKey := dimensionsMapKey{
			source: strings.TrimSpace(string(metadata["SOURCE"])),
			schema: strings.TrimSpace(string(metadata["SCHEMA"])),
		}
		arrayDimensions, ok := dimensions[dimensionsKey]
		if !ok || len(arrayDimensions) == 0 {
			return nil, fmt.Errorf(
				"array metadata for schema %q, table %q, field %q has no dimensions",
				arr.Schema,
				arr.TableName,
				arr.FieldName,
			)
		}
		for _, dimension := range arrayDimensions {
			arr.Dimensions = append(arr.Dimensions,
				map[string]int{
					"Lower": dimension[0],
					"Upper": dimension[1],
				},
			)
			arr.CurrentDimensions = arr.Dimensions
		}
		arrays[metadataKey] = &arr
	}
	return arrays, nil
}

func attachArrayMetadata(descriptions []FieldDescription, arrays map[metadataMapKey]*Array, target string) error {
	for i := range descriptions {
		if !descriptions[i].IsArray() {
			continue
		}

		key := metadataMapKey{
			schema: descriptions[i].Schema,
			table:  descriptions[i].OriginalTableName,
			field:  descriptions[i].OriginalName,
		}
		arrayData, ok := arrays[key]
		if !ok || arrayData == nil {
			return fmt.Errorf(
				"cannot resolve array metadata for %s %d (schema %q, table %q, field %q)",
				target,
				i+1,
				key.schema,
				key.table,
				key.field,
			)
		}
		descriptions[i].ArrayData = arrayData
	}
	return nil
}
