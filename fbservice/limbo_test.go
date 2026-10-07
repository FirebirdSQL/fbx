package fbservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbservice"
)

func TestManagerLimboTransactionDetails(t *testing.T) {
	manager, conn := newTestManager(t)
	want := []fbservice.LimboTransaction{
		{ID: 42},
		{
			ID: 1 << 40, Distributed: true, Advice: fbservice.LimboAdviceRollback,
			Participants: []fbservice.LimboParticipant{
				{ID: 7, State: fbservice.LimboStatePrepared, HostSite: "coordinator", RemoteSite: "remote", DatabasePath: "/data/one.fdb"},
				{ID: 1 << 40, State: fbservice.LimboStateRolledBack, DatabasePath: "/data/two.fdb"},
				{DatabasePath: "/data/unavailable.fdb"},
			},
		},
	}
	conn.limboDetails = want
	got, err := manager.LimboTransactionDetails(context.Background(), "db")
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, "db", conn.database)
	require.Equal(t, "db", conn.expectedDatabase)
	require.True(t, conn.closed)
	require.Equal(t, fbservice.LimboStateUnknown, got[1].Participants[2].State)
	require.Equal(t, fbservice.LimboAdviceUnknown, got[0].Advice)

	// Even complete details are discarded if action or connection cleanup fails.
	for _, actionFails := range []bool{false, true} {
		manager, conn = newTestManager(t)
		conn.limboDetails = want
		conn.closeErr = errors.New("detach failed")
		if actionFails {
			conn.maintenanceErr = errors.New("query failed")
		}
		got, err = manager.LimboTransactionDetails(context.Background(), "db")
		require.Nil(t, got)
		require.ErrorIs(t, err, conn.closeErr)
		if actionFails {
			require.ErrorIs(t, err, conn.maintenanceErr)
		}
		require.True(t, conn.closed)
	}
}
