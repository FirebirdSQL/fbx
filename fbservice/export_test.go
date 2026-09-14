package fbservice

import (
	"context"

	"rdb.red-soft.ru/fbx/fbconn"
)

// ServiceConnection exposes the connection interface only to external tests.
type ServiceConnection = serviceConnection

// StatisticsConnection exposes the statistics connection interface only to external tests.
type StatisticsConnection = statisticsConnection

// SetConnectForTest replaces the connector before any manager operations run.
func SetConnectForTest(manager *Manager, connect func(context.Context, *fbconn.ServiceConfig) (ServiceConnection, error)) {
	manager.connect = connect
}

// SetStatisticsConnectForTest replaces the connector before any statistics operations run.
func SetStatisticsConnectForTest(manager *StatisticsManager, connect func(context.Context, *fbconn.ServiceConfig) (StatisticsConnection, error)) {
	manager.connect = connect
}
