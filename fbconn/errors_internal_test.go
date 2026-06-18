package fbconn

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnLockErrorUnwrapsErrConnClosed(t *testing.T) {
	require.True(t, errors.Is(&connLockError{status: "conn closed"}, ErrConnClosed))
	require.False(t, errors.Is(&connLockError{status: "conn busy"}, ErrConnClosed))
}
