package cleanupctx

import (
	"context"
	"errors"
	"time"
)

var cleanupGracePeriod time.Duration = 5 * time.Second

// SetCleanupTimeout sets the timeout for the resource cleanup context in the event
// of an error. The default timeout is 5 seconds.
// It must be called before using the driver and must not be called
// concurrently with driver operations.
// The transmitted timeout must be at least 150 ms.
func SetCleanupTimeout(timeout time.Duration) error {
	if timeout >= 150*time.Millisecond {
		cleanupGracePeriod = timeout
		return nil
	}
	return errors.New("timeout must be at least 150ms")
}

// Returns the cleanup timeout, which defaults to 5 seconds but can be changed via [SetCleanupTimeout].
func CleanupTimeout() time.Duration {
	return cleanupGracePeriod
}

// Returns a context for clearing and a cancellation function for it; by default,
// the context timeout is 5 seconds, but it can be changed via [SetCleanupTimeout].
func CleanupContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), cleanupGracePeriod)
}
