package fbconn

import (
	"context"
	"errors"

	"github.com/firebirdsql/fbx/internal/operationerror"
)

// serviceContextError retains the original failure and its retry policy when
// cancellation is reported as a server or transport error.
func serviceContextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	ctxErr := ctx.Err()
	if ctxErr == nil {
		return normalizeTimeoutError(ctx, err)
	}
	if !errors.Is(err, ctxErr) {
		err = operationerror.Join(err, ctxErr)
	}
	if ctxErr == context.DeadlineExceeded && !Timeout(err) {
		return &errTimeout{err: err}
	}
	return err
}
