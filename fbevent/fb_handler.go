package fbevent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"rdb.red-soft.ru/fbx/fbconn"
)

type fbEventHandler struct {
	manager   *Manager
	eventName string
	handle    *fbconn.EventHandle

	// mu protects state transitions and initialized flag.
	mu sync.Mutex
	// opMu serializes register/re-register/unregister network operations so state
	// transitions cannot overlap around server calls.
	opMu sync.Mutex

	initialized bool
	state       eventHandlerState
}

// The zero value represents a handler that exists locally but has not yet been
// registered on the server.
type eventHandlerState uint8

const (
	// eventHandlerStateRegistering means a queue/register call is in progress.
	eventHandlerStateRegistering eventHandlerState = iota + 1
	// eventHandlerStateRegistered means server-side registration is active.
	eventHandlerStateRegistered
	// eventHandlerStateCancelling means cancel call is in progress.
	eventHandlerStateCancelling
	// eventHandlerStateCancelled means handler is permanently cancelled.
	eventHandlerStateCancelled
)

func (h *fbEventHandler) register(ctx context.Context) error {
	h.opMu.Lock()
	defer h.opMu.Unlock()

	h.mu.Lock()
	switch h.state {
	case eventHandlerStateCancelled, eventHandlerStateCancelling:
		h.mu.Unlock()
		return errors.New("trying to register cancelled event handler")
	default:
	}
	prevState := h.state
	h.state = eventHandlerStateRegistering
	handle := h.handle
	h.mu.Unlock()

	conn := h.manager.currentConn()
	if conn == nil {
		h.mu.Lock()
		if h.state == eventHandlerStateRegistering {
			h.state = prevState
		}
		h.mu.Unlock()
		return errors.New("manager disconnected")
	}
	opCtx, cancel := h.manager.operationContext(ctx)
	defer cancel()
	err := conn.QueueEvent(opCtx, handle)

	h.mu.Lock()
	// register can fail; on failure we rollback to previous state instead of
	// pretending cancellation happened.
	if h.state == eventHandlerStateRegistering {
		if err == nil {
			h.state = eventHandlerStateRegistered
		} else {
			h.state = prevState
		}
	}
	h.mu.Unlock()
	return err
}

func (h *fbEventHandler) unregister(ctx context.Context) error {
	return h.unregisterOnConn(ctx, h.manager.currentConn())
}

func (h *fbEventHandler) unregisterOnConn(ctx context.Context, conn *fbconn.FbConn) error {
	h.opMu.Lock()
	defer h.opMu.Unlock()

	h.mu.Lock()
	switch h.state {
	case eventHandlerStateCancelled, eventHandlerStateCancelling:
		h.mu.Unlock()
		return nil
	default:
	}
	prevState := h.state
	// Transition to cancelling before network I/O; if cancel fails we roll back
	// to previous state (typically registered), so failed cancel does not leave
	// handler in a permanently cancelled local state.
	h.state = eventHandlerStateCancelling
	handle := h.handle
	h.mu.Unlock()

	if conn == nil {
		// Connection already unavailable: treat as best-effort local cancellation.
		h.mu.Lock()
		if h.state == eventHandlerStateCancelling {
			h.state = eventHandlerStateCancelled
		}
		h.mu.Unlock()
		return nil
	}
	opCtx, cancel := h.manager.operationContext(ctx)
	defer cancel()
	err := conn.CancelEvent(opCtx, handle)

	h.mu.Lock()
	if h.state == eventHandlerStateCancelling {
		if err == nil {
			h.state = eventHandlerStateCancelled
		} else {
			h.state = prevState
		}
	}
	h.mu.Unlock()
	return err
}

func (h *fbEventHandler) EventOccurred(handle *fbconn.EventHandle) {
	if h.isStopping() {
		return
	}

	// CountEvents must be serialized via opMu: FbConn is not safe for concurrent
	// use, and user goroutines call QueueEvent/CancelEvent while holding opMu.
	h.opMu.Lock()
	conn := h.manager.currentConn()
	if conn == nil {
		h.opMu.Unlock()
		return
	}
	countErr := conn.CountEvents(handle)
	h.opMu.Unlock()

	if countErr != nil {
		h.manager.setAsyncError(fmt.Errorf("count events failed for %q: %w", h.eventName, countErr))
		return
	}

	h.mu.Lock()
	if h.state == eventHandlerStateCancelling || h.state == eventHandlerStateCancelled {
		h.mu.Unlock()
		return
	}
	emitEvent := h.initialized
	h.initialized = true
	h.mu.Unlock()

	if h.isStopping() {
		return
	}
	if err := h.reRegister(handle); err != nil {
		h.manager.setAsyncError(fmt.Errorf("re-register failed for %q: %w", h.eventName, err))
	}
	if !emitEvent || h.isStopping() {
		return
	}
	// Keep server-side subscription active before potentially blocking on
	// manager event queue backpressure.
	h.manager.enqueueEvent(DatabaseEvent{
		EventName:  handle.EventName(),
		EventCount: handle.EventCount(),
	})
}

func (h *fbEventHandler) reRegister(handle *fbconn.EventHandle) error {
	const maxAttempts = 3
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		h.opMu.Lock()

		h.mu.Lock()
		if h.state == eventHandlerStateCancelling || h.state == eventHandlerStateCancelled {
			h.mu.Unlock()
			h.opMu.Unlock()
			return nil
		}
		prevState := h.state
		h.state = eventHandlerStateRegistering
		h.mu.Unlock()

		conn := h.manager.currentConn()
		if conn == nil {
			h.mu.Lock()
			if h.state == eventHandlerStateRegistering {
				h.state = prevState
			}
			h.mu.Unlock()
			h.opMu.Unlock()
			return errors.New("manager disconnected")
		}

		opCtx, cancel := h.manager.operationContext(context.Background())
		err := conn.QueueEvent(opCtx, handle)
		cancel()

		h.mu.Lock()
		if h.state == eventHandlerStateRegistering {
			if err == nil {
				h.state = eventHandlerStateRegistered
			} else {
				h.state = prevState
			}
		}
		h.mu.Unlock()

		if err != nil {
			lastErr = err
			h.opMu.Unlock()
		} else {
			h.opMu.Unlock()
			return nil
		}

		time.Sleep(time.Duration(25*(attempt+1)) * time.Millisecond)
	}

	if lastErr == nil {
		return errors.New("re-register failed for unknown reason")
	}
	return lastErr
}

func (h *fbEventHandler) isStopping() bool {
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	return state == eventHandlerStateCancelling || state == eventHandlerStateCancelled
}
