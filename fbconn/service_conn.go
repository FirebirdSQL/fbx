package fbconn

import (
	"context"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/auth"
)

const defaultServiceName = "service_mgr"

type connectTarget byte

const (
	connectTargetDatabase connectTarget = iota
	connectTargetService
)

// ServiceConn is a low-level connection to the Firebird service manager.
//
// A ServiceConn is not safe for concurrent use. The caller must call
// [ServiceConn.Close] when the connection is no longer needed.
type ServiceConn struct {
	conn         *FbConn
	actionActive bool
}

// ServiceConfig contains settings used to establish a connection to the
// Firebird service manager.
type ServiceConfig struct {
	// ConnectionConfig contains the shared network, authentication, encryption,
	// timeout, and fallback settings. It must have been constructed with
	// [ParseConfig] or [ParseConfigWithOptions].
	ConnectionConfig *Config

	// ExpectedDatabase identifies the database used to select authentication
	// rules. It does not attach the service connection to that database.
	ExpectedDatabase string
}

// ConnectService establishes a connection to the Firebird service manager
// using the environment and connString for configuration. It parses connString
// with [ParseConfig].
//
// The database setting, when present, is sent to the service manager as the
// expected database for authentication; ConnectService does not attach to that
// database. The context controls the connection attempt.
func ConnectService(ctx context.Context, connString string) (*ServiceConn, error) {
	config, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	return ConnectServiceConfig(ctx, &ServiceConfig{
		ConnectionConfig: config,
		ExpectedDatabase: config.Database,
	})
}

// ConnectServiceConfig establishes a connection to the Firebird service
// manager using config.
//
// ConnectServiceConfig rejects database-only settings such as
// [Config.ValidateConnect], [Config.AfterConnect], [Config.CreateDatabase],
// and [Config.DPB]. The context controls the connection attempt.
func ConnectServiceConfig(ctx context.Context, config *ServiceConfig) (*ServiceConn, error) {
	connectionConfig, err := prepareServiceConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	fbConn, err := connectConfig(ctx, connectionConfig, connectTargetService)
	if err != nil {
		return nil, err
	}
	return &ServiceConn{conn: fbConn}, nil
}

func prepareServiceConnectionConfig(config *ServiceConfig) (*Config, error) {
	if config == nil {
		return nil, errors.New("service config must not be nil")
	}
	if config.ConnectionConfig == nil {
		return nil, errors.New("service connection config must not be nil")
	}

	connectionConfig := config.ConnectionConfig
	switch {
	case connectionConfig.ValidateConnect != nil:
		return nil, errors.New("service connections do not support Config.ValidateConnect")
	case connectionConfig.AfterConnect != nil:
		return nil, errors.New("service connections do not support Config.AfterConnect")
	case connectionConfig.CreateDatabase:
		return nil, errors.New("service connections do not support Config.CreateDatabase")
	case connectionConfig.DPB != nil:
		return nil, errors.New("service connections do not support Config.DPB")
	}

	connectionConfig = connectionConfig.Copy()
	connectionConfig.Database = config.ExpectedDatabase
	return connectionConfig, nil
}

func (fbConn *FbConn) attachService(config *Config, authManager *auth.AuthManager) error {
	spb := buildServiceAttachSPB(config, authManager)
	if err := fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(&fbproto.OpServiceAttach{
			Encoding:    fbConn.Encoding,
			ServiceName: defaultServiceName,
			SPB:         spb,
		})
	}); err != nil {
		return fmt.Errorf("failed to flush opServiceAttach: %w", err)
	}
	return fbConn.finishAttachAuth(config, authManager, "opServiceAttach")
}

func buildServiceAttachSPB(config *Config, authManager *auth.AuthManager) []byte {
	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_spb_version3)

	putServiceAttachString(spb, fbproto.Isc_spb_user_name, config.User)
	putServiceAttachString(spb, fbproto.Isc_spb_sql_role_name, config.Role)
	putServiceAttachString(spb, fbproto.Isc_spb_os_user, config.SystemUser)
	putServiceAttachString(spb, fbproto.Isc_spb_host_name, config.SystemHost)
	putServiceAttachString(spb, fbproto.Isc_spb_client_version, config.ClientVersion)
	putServiceAttachString(spb, fbproto.Isc_spb_expected_db, config.Database)
	putServiceAttachString(spb, fbproto.Isc_spb_auth_plugin_list, authManager.BuildList())

	if plugin := authManager.GetCurrentPlugin(); plugin != nil {
		putServiceAttachString(spb, fbproto.Isc_spb_auth_plugin_name, plugin.Name())
	}
	if authSpecific := authManager.AuthSpecificData(); len(authSpecific) > 0 {
		putServiceAttachBytes(spb, fbproto.Isc_spb_specific_auth_data, authSpecific)
	}
	putServiceAttachTag(spb, fbproto.Isc_spb_utf8_filename)

	return spb.Bytes()
}

func putServiceAttachString(spb *fbproto.XPBWriter, tag byte, value string) {
	if value == "" {
		return
	}
	putServiceAttachBytes(spb, tag, []byte(value))
}

func putServiceAttachBytes(spb *fbproto.XPBWriter, tag byte, value []byte) {
	spb.AppendByte(tag)
	spb.AppendInt32(int32(len(value)))
	spb.AppendBytes(value)
}

func putServiceAttachTag(spb *fbproto.XPBWriter, tag byte) {
	spb.AppendByte(tag)
	spb.AppendInt32(0)
}

// Close attempts to detach from the service manager and always closes the
// underlying network connection. It is safe to call Close more than once.
// The context limits the time available for the clean detach.
func (c *ServiceConn) Close(ctx context.Context) error {
	if c == nil || c.conn == nil {
		return nil
	}

	fbConn := c.conn
	fbConn.statusMu.Lock()
	if fbConn.status == connStatusClosed {
		fbConn.statusMu.Unlock()
		return nil
	}
	fbConn.status = connStatusClosed
	fbConn.statusMu.Unlock()

	defer fbConn.closeCleanupDone()

	if ctx != context.Background() {
		fbConn.contextWatcher.Unwatch()
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	detachErr := fbConn.withWriteLockAndFlush(func() error {
		if err := fbConn.client.WriteMessage(&fbproto.OpServiceDetach{
			ServiceHandle: fbConn.attachmentHandle,
		}); err != nil {
			return err
		}
		return fbConn.client.WriteMessage(&fbproto.OpDisconnect{})
	})
	if detachErr != nil {
		detachErr = fmt.Errorf("detach from service manager: %w", detachErr)
	}
	closeErr := fbConn.conn.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("close service manager connection: %w", closeErr)
	}
	return errors.Join(detachErr, closeErr)
}
