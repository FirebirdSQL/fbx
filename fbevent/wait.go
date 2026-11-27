package fbevent

import (
	"context"
	"errors"
	"time"
)

var errEventWaitTimeout = errors.New("event wait timeout")

// WaitForEvent waits for one occurrence of eventName.
//
// WaitForEvent temporarily registers a listener, waits for the first matching
// event, and removes the listener before returning. It returns the event count
// reported by the server. If timeout is zero, WaitForEvent waits until an event
// arrives, ctx is done, or an asynchronous manager error occurs. If timeout is
// positive and no event arrives before it expires, WaitForEvent returns -1 and a
// nil error. The method finishes any in-flight registration or cancellation
// request before it returns.
func (m *Manager) WaitForEvent(ctx context.Context, eventName string, timeout time.Duration) (int, error) {
	if timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(ctx, timeout, errEventWaitTimeout)
		defer cancel()
	}

	count := -1
	var removeErr error
	listener := newOneTimeEventListener()
	listenerCloser, err := m.addEventListener(ctx, eventName, listener)
	if err == nil {
		count, err = listener.await(ctx, m.asyncErrorSignal(), m.LastError)
		removeErr = listenerCloser.Close()
	}
	if err == context.DeadlineExceeded && context.Cause(ctx) == errEventWaitTimeout {
		err = nil
	}
	if removeErr != nil {
		return -1, errors.Join(err, removeErr)
	}
	return count, err
}
