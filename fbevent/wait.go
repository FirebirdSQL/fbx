package fbevent

import (
	"context"
	"errors"
	"time"
)

// WaitForEvent waits for one event occurrence for eventName.
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
