package fbservice_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

func (c *fakeServiceConnection) Repair(_ context.Context, database string, options fbconn.RepairOptions) error {
	c.database, c.repairOptions = database, options
	return c.maintenanceErr
}
func (c *fakeServiceConnection) SetDatabaseProperties(_ context.Context, database string, options fbconn.DatabaseProperties) error {
	c.database, c.properties = database, options
	return c.maintenanceErr
}
func (c *fakeServiceConnection) LimboTransactions(_ context.Context, database string) ([]int64, error) {
	c.database = database
	return []int64{42, 1 << 40}, c.maintenanceErr
}
func (c *fakeServiceConnection) LimboTransactionDetails(_ context.Context, database string) ([]fbconn.LimboTransaction, error) {
	c.database = database
	return c.limboDetails, c.maintenanceErr
}
func (c *fakeServiceConnection) ResolveLimboTransaction(_ context.Context, database string, id int64, action fbconn.LimboTransactionAction, output io.Writer) error {
	c.database, c.transactionID, c.limboAction, c.limboOutput = database, id, action, output
	return c.maintenanceErr
}
func (c *concurrentServiceConnection) Repair(_ context.Context, database string, _ fbconn.RepairOptions) error {
	return c.recordOperation(database)
}
func (c *concurrentServiceConnection) SetDatabaseProperties(_ context.Context, database string, _ fbconn.DatabaseProperties) error {
	return c.recordOperation(database)
}
func (c *concurrentServiceConnection) LimboTransactions(_ context.Context, database string) ([]int64, error) {
	return nil, c.recordOperation(database)
}
func (c *concurrentServiceConnection) LimboTransactionDetails(_ context.Context, database string) ([]fbconn.LimboTransaction, error) {
	return nil, c.recordOperation(database)
}
func (c *concurrentServiceConnection) ResolveLimboTransaction(_ context.Context, database string, _ int64, _ fbconn.LimboTransactionAction, _ io.Writer) error {
	return c.recordOperation(database)
}

func TestManagerRepair(t *testing.T) {
	for _, op := range []fbservice.RepairOperation{fbservice.RepairValidate, fbservice.RepairSweep, fbservice.RepairMend, fbservice.RepairKillShadows, fbservice.RepairUpgradeODS, fbservice.RepairFixICU} {
		manager, conn := newTestManager(t)
		options := fbservice.RepairOptions{Operation: op, Output: io.Discard}
		if op == fbservice.RepairValidate {
			options.Full, options.ReadOnly, options.IgnoreChecksums = true, true, true
		}
		if op == fbservice.RepairSweep || op == fbservice.RepairFixICU {
			options.ParallelWorkers = 1
		}
		if op == fbservice.RepairSweep {
			options.SweepTimeout = 2147483
		}
		require.NoError(t, manager.Repair(context.Background(), "db", options))
		require.Equal(t, fbconn.RepairOperation(op), conn.repairOptions.Operation)
		require.Equal(t, options.Full, conn.repairOptions.Full)
		require.Equal(t, options.ReadOnly, conn.repairOptions.ReadOnly)
		require.Equal(t, options.IgnoreChecksums, conn.repairOptions.IgnoreChecksums)
		require.Equal(t, options.ParallelWorkers, conn.repairOptions.ParallelWorkers)
		require.Equal(t, options.SweepTimeout, conn.repairOptions.SweepTimeout)
		require.Equal(t, io.Discard, conn.repairOptions.Output)
		require.Equal(t, "db", conn.expectedDatabase)
		require.Equal(t, "db", conn.database)
		require.True(t, conn.closed)
	}
}

func TestManagerRepairParallelWorkersLimits(t *testing.T) {
	for name, operation := range map[string]fbservice.RepairOperation{
		"sweep": fbservice.RepairSweep,
		"ICU":   fbservice.RepairFixICU,
	} {
		t.Run(name, func(t *testing.T) {
			for _, tt := range []struct {
				name    string
				workers int
				wantErr bool
			}{
				{name: "negative", workers: -1, wantErr: true},
				{name: "default", workers: 0},
				{name: "minimum", workers: 1},
				{name: "maximum", workers: 32767},
				{name: "above maximum", workers: 32768, wantErr: true},
				{name: "would wrap to one", workers: 65537, wantErr: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					manager, conn := newTestManager(t)
					err := manager.Repair(context.Background(), "db", fbservice.RepairOptions{
						Operation: operation, ParallelWorkers: tt.workers,
					})
					if tt.wantErr {
						require.ErrorContains(t, err, "repair parallel workers must be between 0 and 32767")
						require.False(t, conn.opened)
						return
					}
					require.NoError(t, err)
					require.True(t, conn.opened)
					require.Equal(t, fbconn.RepairOperation(operation), conn.repairOptions.Operation)
					require.Equal(t, tt.workers, conn.repairOptions.ParallelWorkers)
					require.True(t, conn.closed)
				})
			}
		})
	}
}

func TestManagerProperties(t *testing.T) {
	for name, options := range map[string]fbservice.DatabaseProperties{
		"read only":  {AccessMode: fbservice.DatabaseAccessModeReadOnly},
		"read write": {AccessMode: fbservice.DatabaseAccessModeReadWrite},
		"async":      {WriteMode: fbservice.DatabaseWriteModeAsync},
		"sync":       {WriteMode: fbservice.DatabaseWriteModeSync},
		"cache":      {PageBuffers: new(1234)}, "reset cache": {PageBuffers: new(0)},
		"sweep": {SweepInterval: new(1234)}, "disable sweep": {SweepInterval: new(0)},
		"dialect 1": {SQLDialect: 1}, "dialect 3": {SQLDialect: 3},
		"reserve": {ReserveSpace: new(true)}, "full pages": {ReserveSpace: new(false)},
		"shutdown": {Shutdown: &fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce, Timeout: 17}},
		"online":   {OnlineMode: fbservice.DatabaseModeNormal}, "activate": {ActivateShadow: true},
	} {
		t.Run(name, func(t *testing.T) {
			manager, conn := newTestManager(t)
			options.Output = io.Discard
			require.NoError(t, manager.SetDatabaseProperties(context.Background(), "db", options))
			got := conn.properties
			require.Equal(t, fbconn.DatabaseAccessMode(options.AccessMode), got.AccessMode)
			require.Equal(t, fbconn.DatabaseWriteMode(options.WriteMode), got.WriteMode)
			require.Equal(t, options.PageBuffers, got.PageBuffers)
			require.Equal(t, options.SweepInterval, got.SweepInterval)
			require.Equal(t, options.SQLDialect, got.SQLDialect)
			require.Equal(t, options.ReserveSpace, got.ReserveSpace)
			require.Equal(t, fbconn.DatabaseMode(options.OnlineMode), got.OnlineMode)
			require.Equal(t, options.ActivateShadow, got.ActivateShadow)
			if options.Shutdown != nil {
				require.Equal(t, &fbconn.ShutdownOptions{Mode: fbconn.DatabaseModeFull, Method: fbconn.ShutdownForce, Timeout: 17}, got.Shutdown)
			}
			require.Equal(t, io.Discard, got.Output)
			require.Equal(t, "db", conn.database)
			require.True(t, conn.closed)
		})
	}
}

func TestManagerMaintenanceValidation(t *testing.T) {
	ctx := context.Background()
	for _, options := range []fbservice.RepairOptions{
		{}, {Operation: 255}, {Operation: fbservice.RepairSweep, Full: true},
		{Operation: fbservice.RepairMend, ReadOnly: true}, {Operation: fbservice.RepairFixICU, IgnoreChecksums: true},
		{Operation: fbservice.RepairValidate, ParallelWorkers: 1}, {Operation: fbservice.RepairSweep, ParallelWorkers: -1},
		{Operation: fbservice.RepairSweep, SweepTimeout: -1}, {Operation: fbservice.RepairSweep, SweepTimeout: 2147484},
		{Operation: fbservice.RepairFixICU, SweepTimeout: 1}, {Operation: fbservice.RepairValidate, SweepTimeout: 1},
	} {
		manager, conn := newTestManager(t)
		options.Output = io.Discard
		require.Error(t, manager.Repair(ctx, "db", options))
		require.False(t, conn.opened)
	}
	for _, options := range []fbservice.DatabaseProperties{
		{}, {AccessMode: 255}, {WriteMode: 255}, {OnlineMode: fbservice.DatabaseModeFull},
		{SQLDialect: 2}, {SQLDialect: -1}, {PageBuffers: new(-1)}, {PageBuffers: new(49)}, {SweepInterval: new(-1)},
		{AccessMode: fbservice.DatabaseAccessModeReadOnly, SQLDialect: 3},
		{Shutdown: &fbservice.ShutdownOptions{}},
		{Shutdown: &fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeNormal, Method: fbservice.ShutdownForce}},
		{Shutdown: &fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: 255}},
		{Shutdown: &fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce, Timeout: -1}},
	} {
		manager, conn := newTestManager(t)
		require.Error(t, manager.SetDatabaseProperties(ctx, "db", options))
		require.False(t, conn.opened)
	}
	for _, id := range []int64{-1, 0} {
		manager, conn := newTestManager(t)
		require.Error(t, manager.ResolveLimboTransaction(ctx, "db", id, fbservice.LimboCommit, nil))
		require.False(t, conn.opened)
		result, err := manager.ResolveLimboTransactionChecked(ctx, "db", id, fbservice.LimboCommit, nil)
		require.Error(t, err)
		require.Equal(t, fbservice.LimboResolution{}, result)
		require.False(t, conn.opened)
	}
	manager, conn := newTestManager(t)
	require.Error(t, manager.ResolveLimboTransaction(ctx, "db", 1, 255, nil))
	require.False(t, conn.opened)
	result, err := manager.ResolveLimboTransactionChecked(ctx, "db", 1, 255, nil)
	require.Error(t, err)
	require.Equal(t, fbservice.LimboResolution{}, result)
	require.False(t, conn.opened)
}

type maintenanceCall func(context.Context, *fbservice.Manager, string) error

func maintenanceCalls() map[string]maintenanceCall {
	return map[string]maintenanceCall{
		"validate": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Validate(ctx, db, fbservice.ValidateOptions{Output: io.Discard})
		},
		"sweep": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Sweep(ctx, db)
		},
		"shutdown": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Shutdown(ctx, db, fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce})
		},
		"repair": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.Repair(ctx, db, fbservice.RepairOptions{Operation: fbservice.RepairValidate, Output: io.Discard})
		},
		"properties": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.SetDatabaseProperties(ctx, db, fbservice.DatabaseProperties{SQLDialect: 3})
		},
		"limbo": func(ctx context.Context, m *fbservice.Manager, db string) error {
			_, err := m.LimboTransactions(ctx, db)
			return err
		},
		"limbo details": func(ctx context.Context, m *fbservice.Manager, db string) error {
			_, err := m.LimboTransactionDetails(ctx, db)
			return err
		},
		"resolve": func(ctx context.Context, m *fbservice.Manager, db string) error {
			return m.ResolveLimboTransaction(ctx, db, 42, fbservice.LimboCommit, nil)
		},
		"resolve checked": func(ctx context.Context, m *fbservice.Manager, db string) error {
			_, err := m.ResolveLimboTransactionChecked(ctx, db, 42, fbservice.LimboCommit, nil)
			return err
		},
	}
}

func TestManagerMaintenanceLifecycle(t *testing.T) {
	for name, call := range maintenanceCalls() {
		t.Run(name, func(t *testing.T) {
			for _, database := range []string{"", "-?", "d\x00b", strings.Repeat("x", 65536)} {
				manager, conn := newTestManager(t)
				require.Error(t, call(context.Background(), manager, database))
				require.False(t, conn.opened)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			manager, conn := newTestManager(t)
			require.ErrorIs(t, call(ctx, manager, "db"), context.Canceled)
			require.False(t, conn.opened)
			actionErr, closeErr := errors.New("action"), errors.New("close")
			conn.maintenanceErr, conn.closeErr = actionErr, closeErr
			err := call(context.Background(), manager, "db")
			require.ErrorIs(t, err, actionErr)
			require.ErrorIs(t, err, closeErr)
			require.True(t, conn.closed)
			var nilManager *fbservice.Manager
			require.Error(t, call(context.Background(), nilManager, "db"))
		})
	}
}

func TestManagerLimbo(t *testing.T) {
	manager, conn := newTestManager(t)
	ids, err := manager.LimboTransactions(context.Background(), "db")
	require.NoError(t, err)
	require.Equal(t, []int64{42, 1 << 40}, ids)
	require.True(t, conn.closed)
	for _, action := range []fbservice.LimboTransactionAction{fbservice.LimboCommit, fbservice.LimboRollback, fbservice.LimboRecover} {
		require.NoError(t, manager.ResolveLimboTransaction(context.Background(), "db", 1<<40, action, io.Discard))
		require.Equal(t, int64(1<<40), conn.transactionID)
		require.Equal(t, fbconn.LimboTransactionAction(action), conn.limboAction)
		require.Equal(t, io.Discard, conn.limboOutput)
	}
	conn.closeErr = errors.New("detach failed")
	ids, err = manager.LimboTransactions(context.Background(), "db")
	require.ErrorIs(t, err, conn.closeErr)
	require.Nil(t, ids)
}

func TestManagerMaintenanceIntegerLimits(t *testing.T) {
	maxValue := int64(1<<31 - 1)
	if int64(int(maxValue+1)) != maxValue+1 {
		return
	}
	value := int(maxValue + 1)
	for _, options := range []fbservice.DatabaseProperties{
		{PageBuffers: &value}, {SweepInterval: &value},
		{Shutdown: &fbservice.ShutdownOptions{Mode: fbservice.DatabaseModeFull, Method: fbservice.ShutdownForce, Timeout: value}},
	} {
		manager, conn := newTestManager(t)
		require.Error(t, manager.SetDatabaseProperties(context.Background(), "db", options))
		require.False(t, conn.opened)
	}
	manager, conn := newTestManager(t)
	require.Error(t, manager.Repair(context.Background(), "db", fbservice.RepairOptions{Operation: fbservice.RepairSweep, ParallelWorkers: value}))
	require.False(t, conn.opened)
}

func TestManagerMaintenanceConnectionError(t *testing.T) {
	for name, call := range maintenanceCalls() {
		t.Run(name, func(t *testing.T) {
			manager, _ := newTestManager(t)
			expected := errors.New("connection failed")
			fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
				return nil, expected
			})
			require.ErrorIs(t, call(context.Background(), manager, "db"), expected)
		})
	}
}

func TestManagerMaintenanceClosesAfterCancellation(t *testing.T) {
	for name, call := range maintenanceCalls() {
		t.Run(name, func(t *testing.T) {
			manager, _ := newTestManager(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conn := &maintenanceCancelConnection{statisticsCancelConnection: statisticsCancelConnection{cancel: cancel}}
			fbservice.SetConnectForTest(manager, func(context.Context, *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
				return conn, nil
			})
			require.ErrorIs(t, call(ctx, manager, "db"), context.Canceled)
			require.True(t, conn.closed)
			require.NoError(t, conn.closeContextErr)
			require.True(t, conn.closeHasDeadline)
		})
	}
}

type maintenanceCancelConnection struct {
	statisticsCancelConnection
}

func (c *maintenanceCancelConnection) Repair(ctx context.Context, _ string, _ fbconn.RepairOptions) error {
	c.cancel()
	return ctx.Err()
}

func (c *maintenanceCancelConnection) SetDatabaseProperties(ctx context.Context, _ string, _ fbconn.DatabaseProperties) error {
	c.cancel()
	return ctx.Err()
}

func (c *maintenanceCancelConnection) LimboTransactions(ctx context.Context, _ string) ([]int64, error) {
	c.cancel()
	return nil, ctx.Err()
}

func (c *maintenanceCancelConnection) LimboTransactionDetails(ctx context.Context, _ string) ([]fbconn.LimboTransaction, error) {
	c.cancel()
	return nil, ctx.Err()
}

func (c *maintenanceCancelConnection) ResolveLimboTransaction(ctx context.Context, _ string, _ int64, _ fbconn.LimboTransactionAction, _ io.Writer) error {
	c.cancel()
	return ctx.Err()
}
