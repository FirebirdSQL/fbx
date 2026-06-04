package fbevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
)

func (m *Manager) currentConn() *fbconn.FbConn {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected || m.conn == nil || m.conn.IsClosed() {
		return nil
	}
	return m.conn.FbConn()
}

func (m *Manager) bindConnEventErrorHandler(conn *fbx.Conn) {
	if conn == nil || conn.IsClosed() {
		return
	}
	conn.FbConn().SetEventErrorHandler(m.onConnEventError)
}

func (m *Manager) unbindConnEventErrorHandler(conn *fbx.Conn) {
	if conn == nil || conn.IsClosed() {
		return
	}
	conn.FbConn().SetEventErrorHandler(nil)
}

func (m *Manager) onConnEventError(err error) {
	if err == nil {
		return
	}
	m.setAsyncError(fmt.Errorf("event channel failed: %w", err))
}

func (m *Manager) setAsyncError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	m.setAsyncErrorLocked(err)
	m.mu.Unlock()
}

func (m *Manager) asyncErrorSignal() <-chan struct{} {
	m.mu.Lock()
	ch := m.asyncErrSignal
	m.mu.Unlock()
	return ch
}

func (m *Manager) startConnLifecycleWatcher(conn *fbx.Conn, stop <-chan struct{}, generation uint64) {
	if conn == nil {
		return
	}
	cleanupDone := conn.FbConn().CleanupDone()
	go func() {
		select {
		case <-stop:
			return
		case <-cleanupDone:
		}

		// Explicit manager disconnect closes stop channel first; do not treat
		// that path as an asynchronous lifecycle failure.
		select {
		case <-stop:
			return
		default:
		}

		m.handleConnLifecycleTermination(conn, generation)
	}()
}

func (m *Manager) handleConnLifecycleTermination(conn *fbx.Conn, generation uint64) {
	m.mu.Lock()
	if generation != m.lifecycleWatchGen || !m.connected || m.conn != conn {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	lifecycleErr := errors.New("underlying connection detached")
	if err := m.disconnect(context.Background(), false, false, nil); err != nil {
		lifecycleErr = errors.Join(lifecycleErr, fmt.Errorf("manager disconnect failed after detaching: %w", err))
	}
	m.setAsyncError(lifecycleErr)
}

func (m *Manager) setAsyncErrorLocked(err error) {
	if err == nil || m.asyncErr != nil {
		return
	}
	m.asyncErr = err
	closeSignalIfOpen(m.asyncErrSignal)
}

func closeSignalIfOpen(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case <-ch:
		return
	default:
		close(ch)
	}
}

func (m *Manager) operationContext(parent context.Context) (context.Context, context.CancelFunc) {
	return contextWithBoundedTimeout(parent, managerOperationTimeout)
}

func (m *Manager) dispatcherStopContext(parent context.Context) (context.Context, context.CancelFunc) {
	return contextWithBoundedTimeout(parent, managerDispatcherShutdownTimeout)
}

func contextWithBoundedTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}
