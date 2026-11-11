package fbservice_test

import (
	"context"
	"io"
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

func TestManagerMaintenanceActions(t *testing.T) {
	ctx := context.Background()
	t.Run("validate", func(t *testing.T) {
		manager, conn := newTestManager(t)
		require.NoError(t, manager.Validate(ctx, "db", fbservice.ValidateOptions{Output: io.Discard}))
		require.Equal(t, fbconn.RepairOptions{Operation: fbconn.RepairValidate, Output: io.Discard}, conn.repairOptions)
		require.True(t, conn.closed)

		manager, conn = newTestManager(t)
		require.NoError(t, manager.Validate(ctx, "db", fbservice.ValidateOptions{
			Full: true, ReadOnly: true, IgnoreChecksums: true, Output: io.Discard,
		}))
		require.Equal(t, fbconn.RepairOptions{
			Operation: fbconn.RepairValidate, Full: true, ReadOnly: true, IgnoreChecksums: true, Output: io.Discard,
		}, conn.repairOptions)
		require.Equal(t, "db", conn.expectedDatabase)
		require.Equal(t, "db", conn.database)
		require.True(t, conn.closed)
	})
	t.Run("sweep", func(t *testing.T) {
		manager, conn := newTestManager(t)
		require.NoError(t, manager.Sweep(ctx, "db"))
		require.Equal(t, fbconn.RepairOptions{Operation: fbconn.RepairSweep}, conn.repairOptions)
		require.True(t, conn.closed)

		manager, conn = newTestManager(t)
		require.NoError(t, manager.Sweep(ctx, "db", fbservice.SweepOptions{
			IgnoreChecksums: true, ParallelWorkers: 32767, Timeout: 2147483, Output: io.Discard,
		}))
		require.Equal(t, fbconn.RepairOptions{
			Operation: fbconn.RepairSweep, IgnoreChecksums: true, ParallelWorkers: 32767, SweepTimeout: 2147483, Output: io.Discard,
		}, conn.repairOptions)
		require.Equal(t, "db", conn.expectedDatabase)
		require.Equal(t, "db", conn.database)
		require.True(t, conn.closed)
	})
	t.Run("shutdown", func(t *testing.T) {
		manager, conn := newTestManager(t)
		require.NoError(t, manager.Shutdown(ctx, "db", fbservice.ShutdownOptions{
			Mode: fbservice.DatabaseModeSingle, Method: fbservice.ShutdownDenyTransactions, Timeout: 17,
		}))
		require.Equal(t, fbconn.DatabaseProperties{
			Shutdown: &fbconn.ShutdownOptions{
				Mode: fbconn.DatabaseModeSingle, Method: fbconn.ShutdownDenyTransactions, Timeout: 17,
			},
		}, conn.properties)
		require.Equal(t, "db", conn.expectedDatabase)
		require.Equal(t, "db", conn.database)
		require.True(t, conn.closed)
	})
}

func TestManagerMaintenanceActionsValidation(t *testing.T) {
	ctx := context.Background()
	for name, call := range map[string]maintenanceCall{
		"multiple validate options": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Validate(ctx, db, fbservice.ValidateOptions{}, fbservice.ValidateOptions{})
		},
		"multiple sweep options": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Sweep(ctx, db, fbservice.SweepOptions{}, fbservice.SweepOptions{})
		},
		"negative sweep workers": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Sweep(ctx, db, fbservice.SweepOptions{ParallelWorkers: -1})
		},
		"overflowing sweep workers": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Sweep(ctx, db, fbservice.SweepOptions{ParallelWorkers: 32768})
		},
		"negative sweep timeout": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Sweep(ctx, db, fbservice.SweepOptions{Timeout: -1})
		},
		"overflowing sweep timeout": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Sweep(ctx, db, fbservice.SweepOptions{Timeout: 2147484})
		},
		"unset shutdown mode and method": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Shutdown(ctx, db, fbservice.ShutdownOptions{})
		},
		"invalid shutdown mode": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Shutdown(ctx, db, fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeNormal, Method: fbservice.ShutdownForce})
		},
		"unset shutdown method": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Shutdown(ctx, db, fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull})
		},
		"invalid shutdown method": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Shutdown(ctx, db, fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: 255})
		},
		"negative shutdown timeout": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Shutdown(ctx, db, fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce, Timeout: -1})
		},
	} {
		t.Run(name, func(t *testing.T) {
			manager, conn := newTestManager(t)
			require.Error(t, call(ctx, manager, "db"))
			require.False(t, conn.opened, "reject invalid requests before connecting")
		})
	}
}
