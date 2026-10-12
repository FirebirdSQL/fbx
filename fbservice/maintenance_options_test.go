package fbservice_test

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestManagerRepairIgnoreChecksums(t *testing.T) {
	for name, tt := range map[string]struct {
		operation fbservice.RepairOperation
		wantErr   bool
	}{
		"validate":     {operation: fbservice.RepairValidate},
		"sweep":        {operation: fbservice.RepairSweep},
		"mend":         {operation: fbservice.RepairMend},
		"kill shadows": {operation: fbservice.RepairKillShadows, wantErr: true},
		"ODS upgrade":  {operation: fbservice.RepairUpgradeODS, wantErr: true},
		"ICU repair":   {operation: fbservice.RepairFixICU, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			manager, conn := newTestManager(t)
			err := manager.Repair(context.Background(), "db", fbservice.RepairOptions{
				Operation: tt.operation, IgnoreChecksums: true, Output: io.Discard,
			})
			if tt.wantErr {
				require.Error(t, err)
				require.False(t, conn.opened)
				return
			}
			require.NoError(t, err)
			require.True(t, conn.opened)
			require.True(t, conn.repairOptions.IgnoreChecksums)
			require.True(t, conn.closed)
		})
	}
}

func TestManagerMaintenanceOutputRequired(t *testing.T) {
	for name, call := range map[string]maintenanceCall{
		"validate without options": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Validate(ctx, db)
		},
		"validate with nil output": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Validate(ctx, db, fbservice.ValidateOptions{ReadOnly: true})
		},
		"repair validate": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Repair(ctx, db, fbservice.RepairOptions{Operation: fbservice.RepairValidate})
		},
		"repair mend": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Repair(ctx, db, fbservice.RepairOptions{Operation: fbservice.RepairMend})
		},
		"limbo recovery": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.ResolveLimboTransaction(ctx, db, 42, fbservice.LimboRecover, nil)
		},
		"checked limbo recovery": func(ctx context.Context, m *fbservice.Manager, db string) error {
			_, err := m.ResolveLimboTransactionChecked(ctx, db, 42, fbservice.LimboRecover, nil)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			manager, conn := newTestManager(t)
			err := call(context.Background(), manager, "db")
			require.ErrorContains(t, err, "output must not be nil")
			require.False(t, conn.opened, "reject missing output before starting the service action")
		})
	}
}
