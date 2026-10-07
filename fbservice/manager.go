package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"rdb.red-soft.ru/fbx/fbconn"
)

const serviceCloseTimeout = 5 * time.Second

type serviceConnection interface {
	Backup(context.Context, string, string, fbconn.BackupOptions) error
	Restore(context.Context, string, string, fbconn.RestoreOptions) error
	Statistics(context.Context, string, fbconn.StatisticsOptions) error
	Repair(context.Context, string, fbconn.RepairOptions) error
	SetDatabaseProperties(context.Context, string, fbconn.DatabaseProperties) error
	LimboTransactions(context.Context, string) ([]int64, error)
	LimboTransactionDetails(context.Context, string) ([]fbconn.LimboTransaction, error)
	ResolveLimboTransaction(context.Context, string, int64, fbconn.LimboTransactionAction, io.Writer) error
	serviceCloser
}

type serviceCloser interface {
	Close(context.Context) error
}

type connectServiceFunc func(context.Context, *fbconn.ServiceConfig) (serviceConnection, error)

// Manager runs Firebird backup, restore, statistics and database maintenance services.
//
// A Manager is safe for concurrent use. Each operation opens and closes its
// own service connection. Manager does not limit the number of concurrent
// service actions; callers are responsible for limiting them according to the
// server's capacity.
type Manager struct {
	config  *Config
	connect connectServiceFunc
}

// NewManager parses connString with [ParseConfig] and returns a service manager.
//
// Construction does not open a connection. Each operation
// opens its own service connection using the context passed to that operation.
func NewManager(connString string) (*Manager, error) {
	config, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	return NewManagerConfig(config)
}

// NewManagerConfig returns a service manager that uses config for new connections.
//
// Config must be non-nil and created by [ParseConfig]. NewManagerConfig copies
// config. Later changes to config do not affect the manager.
//
// Construction does not open a connection. Each operation
// opens its own service connection using the context passed to that operation.
func NewManagerConfig(config *Config) (*Manager, error) {
	if _, err := config.toConnectionConfig(); err != nil {
		return nil, err
	}
	return &Manager{
		config:  config.Copy(),
		connect: connectService,
	}, nil
}

func connectService(ctx context.Context, config *fbconn.ServiceConfig) (serviceConnection, error) {
	return fbconn.ConnectServiceConfig(ctx, config)
}

func (m *Manager) open(ctx context.Context, expectedDatabase string) (serviceConnection, error) {
	if m == nil || m.config == nil {
		return nil, errors.New("service manager config must not be nil")
	}
	config, err := m.config.toServiceConfig(expectedDatabase)
	if err != nil {
		return nil, err
	}
	return m.connect(ctx, config)
}

func closeService(conn serviceCloser) error {
	ctx, cancel := context.WithTimeout(context.Background(), serviceCloseTimeout)
	defer cancel()
	if err := conn.Close(ctx); err != nil {
		return fmt.Errorf("close service manager connection: %w", err)
	}
	return nil
}
