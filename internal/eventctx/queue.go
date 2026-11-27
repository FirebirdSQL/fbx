// Package eventctx carries the event manager's response timeout across the
// boundary between fbevent and fbconn.
package eventctx

import (
	"context"
	"time"
)

// Queue uses Context while waiting for the connection and ResponseTimeout
// for each request once it can be sent.
type Queue struct {
	context.Context
	ResponseTimeout time.Duration
}

// ForResponse lets an in-flight manager request finish independently of the
// listener's deadline. Low-level callers keep their original context.
func ForResponse(ctx context.Context) (context.Context, context.CancelFunc) {
	if queue, ok := ctx.(Queue); ok {
		return context.WithTimeout(context.WithoutCancel(ctx), queue.ResponseTimeout)
	}
	return ctx, func() {}
}
