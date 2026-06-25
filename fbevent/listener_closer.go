package fbevent

import "sync"

// ListenerCloser removes one listener registration from a [Manager].
type ListenerCloser struct {
	manager *Manager
	handle  ListenerHandle

	closeOnce sync.Once
	closeErr  error
}

func newListenerCloser(manager *Manager, handle ListenerHandle) *ListenerCloser {
	return &ListenerCloser{
		manager: manager,
		handle:  handle,
	}
}

// Handle returns the underlying listener registration handle.
//
// Handle returns [InvalidListenerHandle] when called on a nil closer.
func (r *ListenerCloser) Handle() ListenerHandle {
	if r == nil {
		return InvalidListenerHandle
	}
	return r.handle
}

// Close removes the listener registration.
//
// Close is safe to call multiple times. It returns nil when called on a nil
// closer.
func (r *ListenerCloser) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		if r.manager == nil || r.handle == InvalidListenerHandle {
			return
		}
		r.closeErr = r.manager.RemoveEventListener(r.handle)
	})
	return r.closeErr
}
