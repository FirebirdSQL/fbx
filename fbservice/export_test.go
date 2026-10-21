package fbservice

import (
	"context"

	"github.com/firebirdsql/fbx/fbconn"
)

// ServiceConnection exposes the connection interface only to external tests.
type ServiceConnection = serviceConnection

// SetConnectForTest replaces the connector before any manager operations run.
func SetConnectForTest(manager *Manager, connect func(context.Context, *fbconn.ServiceConfig) (ServiceConnection, error)) {
	manager.connect = connect
}
