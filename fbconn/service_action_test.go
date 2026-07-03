package fbconn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestStartServiceActionRejectsActiveAction(t *testing.T) {
	conn := &ServiceConn{
		conn:         &FbConn{},
		actionActive: true,
	}

	err := conn.startServiceAction(context.Background(), []byte{fbproto.Isc_action_svc_backup})

	require.ErrorIs(t, err, errServiceActionActive)
}
