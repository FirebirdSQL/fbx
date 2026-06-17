package fbevent

import (
	"context"
	"errors"
	"time"
)

// WaitForEvent waits for one occurrence of eventName.
//
// WaitForEvent temporarily registers a listener, waits for the first matching
// event, and removes the listener before returning. It returns the event count
// reported by the server. If timeout is zero, WaitForEvent waits until an event
// arrives, ctx is done, or an asynchronous manager error occurs. If timeout is
// positive and no event arrives before it expires, WaitForEvent returns -1 and a
// nil error.
func (m *Manager) WaitForEvent(ctx context.Context, eventName string, timeout time.Duration) (int, error) {
	listener := newOneTimeEventListener()
	handle, err := m.AddEventListener(eventName, listener)
	if err != nil {
		return -1, err
	}

	count, awaitErr := listener.awaitWithAsyncError(ctx, timeout, m.asyncErrorSignal(), m.LastError)
	removeErr := m.RemoveEventListener(handle)

	if awaitErr != nil {
		if removeErr != nil {
			return -1, errors.Join(awaitErr, removeErr)
		}
		return -1, awaitErr
	}
	if removeErr != nil {
		return -1, removeErr
	}
	// Keep wait semantics local to this wait operation: once this listener got
	// its event (or timeout), we don't fail the call because of unrelated
	// manager-wide async errors that may have happened earlier.
	return count, nil
}
