package fbservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"rdb.red-soft.ru/fbx/fbconn"
)

const serviceCloseTimeout = 5 * time.Second

type serviceConnection interface {
	Backup(context.Context, string, string, fbconn.BackupOptions) error
	Restore(context.Context, string, string, fbconn.RestoreOptions) error
	Close(context.Context) error
}

type connectServiceFunc func(context.Context, *fbconn.ServiceConfig) (serviceConnection, error)

// Manager runs Firebird service manager operations.
//
// A Manager is safe for concurrent use. Each operation opens and closes its
// own service connection. Manager does not limit the number of concurrent
// service actions; callers are responsible for limiting them according to the
// server's capacity.
type Manager struct {
	config  *Config
	connect connectServiceFunc
}

// NewManager returns a service manager that uses config for new connections.
//
// NewManager copies config. Later changes to config do not affect the manager.
// If config is nil, operations on the returned manager fail.
func NewManager(config *Config) *Manager {
	return &Manager{
		config:  config.Copy(),
		connect: connectService,
	}
}

func connectService(ctx context.Context, config *fbconn.ServiceConfig) (serviceConnection, error) {
	return fbconn.ConnectServiceConfig(ctx, config)
}

func (m *Manager) open(ctx context.Context, expectedDatabase string) (serviceConnection, error) {
	if m == nil || m.config == nil {
		return nil, errors.New("service manager config must not be nil")
	}
	connectionConfig, err := m.config.toConnectionConfig()
	if err != nil {
		return nil, err
	}
	return m.connect(ctx, &fbconn.ServiceConfig{
		ConnectionConfig: connectionConfig,
		ExpectedDatabase: expectedDatabase,
	})
}

func closeService(conn serviceConnection) error {
	ctx, cancel := context.WithTimeout(context.Background(), serviceCloseTimeout)
	defer cancel()
	if err := conn.Close(ctx); err != nil {
		return fmt.Errorf("close service manager connection: %w", err)
	}
	return nil
}
