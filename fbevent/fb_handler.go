package fbevent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/internal/eventctx"
)

type fbEventHandler struct {
	manager   *Manager
	conn      *fbconn.FbConn
	stop      <-chan struct{}
	eventName string
	handle    *fbconn.EventHandle
	baseline  chan struct{}

	mu         sync.Mutex // serializes queue and cancel operations
	registered bool
}

func (h *fbEventHandler) registerLocked(ctx context.Context, channelErr <-chan error) error {
	select {
	case <-h.stop:
		return errors.New("manager disconnected")
	default:
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	waitCtx, cancel := context.WithTimeout(ctx, managerOperationTimeout)
	defer cancel()
	if err := h.queue(waitCtx); err != nil {
		return err
	}

	select {
	case <-h.baseline:
	case err := <-channelErr:
		return err
	case <-h.stop:
		return errors.New("manager disconnected")
	case <-waitCtx.Done():
		return waitCtx.Err()
	}

	if err := waitCtx.Err(); err != nil {
		return err
	}
	// The first notification supplies the counter for the live registration.
	if err := h.conn.CountEvents(h.handle); err != nil {
		return err
	}
	if err := h.queue(waitCtx); err != nil {
		return err
	}
	h.registered = true
	return ctx.Err()
}

func (h *fbEventHandler) queue(ctx context.Context) error {
	err := h.conn.QueueEvent(eventctx.Queue{Context: ctx, ResponseTimeout: managerOperationTimeout}, h.handle)
	if ctx.Err() != nil && fbconn.SafeToRetry(err) && errors.Is(err, ctx.Err()) {
		return ctx.Err()
	}
	return err
}

func (h *fbEventHandler) unregister(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.unregisterLocked(ctx)
}

func (h *fbEventHandler) unregisterLocked(ctx context.Context) error {
	if !h.registered {
		select {
		case <-h.baseline:
			// Firebird consumed the initial request and no live request was queued.
			return nil
		default:
		}
	}
	// The initial request can still need cancellation before the subscription is ready.
	if h.conn != nil && (h.handle == nil || (h.handle.EventID() != 0 && !h.conn.IsClosed())) {
		opCtx, cancel := h.manager.operationContext(ctx)
		defer cancel()
		if err := h.conn.CancelEvent(opCtx, h.handle); err != nil {
			return err
		}
	}
	h.registered = false
	return nil
}

func (h *fbEventHandler) EventOccurred(handle *fbconn.EventHandle) {
	// Firebird consumes each queued request once. Only register can queue the
	// next request after receiving this first counter.
	select {
	case <-h.baseline:
	default:
		close(h.baseline)
		return
	}
	h.mu.Lock()
	if !h.registered {
		h.mu.Unlock()
		return
	}
	countErr := h.conn.CountEvents(handle)
	h.mu.Unlock()

	if countErr != nil {
		h.setError(fmt.Errorf("count events failed for %q: %w", h.eventName, countErr))
		return
	}

	if err := h.reRegister(); err != nil {
		h.setError(fmt.Errorf("re-register failed for %q: %w", h.eventName, err))
	}
	// Keep server-side subscription active before potentially blocking on
	// manager event queue backpressure.
	h.manager.enqueueEvent(h, DatabaseEvent{
		EventName:  handle.EventName(),
		EventCount: handle.EventCount(),
	})
}

func (h *fbEventHandler) reRegister() error {
	const maxAttempts = 3
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-h.stop:
			return nil
		default:
		}
		h.mu.Lock()
		if !h.registered {
			h.mu.Unlock()
			return nil
		}
		ctx, cancel := h.manager.operationContext(context.Background())
		lastErr = h.queue(ctx)
		cancel()
		h.mu.Unlock()
		if lastErr == nil {
			return nil
		}
		if attempt+1 < maxAttempts {
			select {
			case <-h.stop:
				return nil
			case <-time.After(time.Duration(25*(attempt+1)) * time.Millisecond):
			}
		}
	}
	return lastErr
}

func (h *fbEventHandler) setError(err error) {
	m := h.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handlers[h.eventName] == h {
		m.setAsyncErrorLocked(err)
	}
}
