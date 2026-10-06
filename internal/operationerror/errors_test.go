package operationerror_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/internal/operationerror"
)

func TestCleanupPreservesOperationRetryPolicy(t *testing.T) {
	primary := &fbconn.FbError{Code: "40001", Message: "operation failed"}
	cleanup := operationerror.WithRetry(errors.New("cleanup connection is closed"), true)
	for _, test := range []struct {
		name      string
		operation error
		cleanup   error
		retry     bool
	}{
		{"unknown_result", primary, cleanup, false},
		{"successful_operation", nil, cleanup, false},
		{"before_sql", operationerror.WithRetry(primary, true), errors.New("cleanup failed"), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := operationerror.Join(test.operation, test.cleanup)
			assert.Equal(t, test.retry, fbconn.SafeToRetry(err))
			assert.ErrorIs(t, err, test.cleanup)
			if test.operation != nil {
				assert.ErrorIs(t, err, test.operation)
				var fbErr *fbconn.FbError
				require.ErrorAs(t, err, &fbErr)
				assert.Same(t, primary, fbErr)
			}
		})
	}
}

func TestNonRetryableServerErrorRetainsItsType(t *testing.T) {
	err := &fbconn.FbError{Code: "25006", Message: "read-only transaction"}
	assert.Same(t, err, operationerror.WithRetry(err, false))
	assert.False(t, fbconn.SafeToRetry(err))
}
