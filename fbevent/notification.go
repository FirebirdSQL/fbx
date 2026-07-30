package fbevent

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

const defaultMaxNotificationQueue = 4096

// Notification is an event notification returned by
// [Manager.WaitForNotification].
//
// Firebird events do not carry a payload.
type Notification struct {
	// Channel is the Firebird event name.
	Channel string

	// Count is the event count delta reported by the server.
	Count int
}

// Listen starts listening for eventName and makes future events available via
// [Manager.WaitForNotification].
//
// The manager must be connected and eventName must be non-empty. Calling Listen
// repeatedly for the same eventName is idempotent. A nil ctx is treated as
// [context.Background].
func (m *Manager) Listen(ctx context.Context, eventName string) error {
	if eventName == "" {
		return errors.New("eventName must not be empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.notificationOpMu.Lock()
	defer m.notificationOpMu.Unlock()

	queue := m.ensureNotificationQueue()

	m.mu.Lock()
	if m.notificationNames == nil {
		m.notificationNames = make(map[string]ListenerHandle)
	}
	if _, exists := m.notificationNames[eventName]; exists {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	listener := EventListenerFunc(func(event DatabaseEvent) {
		queue.push(m, &Notification{
			Channel: event.EventName,
			Count:   event.EventCount,
		})
	})

	listenerCloser, err := m.AddEventListener(eventName, listener)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if m.notificationNames == nil {
		m.notificationNames = make(map[string]ListenerHandle)
	}
	m.notificationNames[eventName] = listenerCloser.Handle()
	m.mu.Unlock()

	return nil
}

// Unlisten stops a [Manager.Listen] registration for eventName.
//
// It is safe to call Unlisten even if eventName is not currently listened for.
// An empty eventName returns an error.
func (m *Manager) Unlisten(eventName string) error {
	if eventName == "" {
		return errors.New("eventName must not be empty")
	}

	m.notificationOpMu.Lock()
	defer m.notificationOpMu.Unlock()

	m.mu.Lock()
	handle, exists := m.notificationNames[eventName]
	m.mu.Unlock()
	if !exists {
		return nil
	}

	if err := m.RemoveEventListener(handle); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.notificationNames, eventName)
	m.mu.Unlock()

	return nil
}

// WaitForNotification waits for the next notification registered by
// [Manager.Listen].
//
// WaitForNotification returns an already buffered notification immediately, even
// if ctx is already canceled. If no notification is buffered, it waits until a
// notification arrives, ctx is done, the manager disconnects, or an asynchronous
// manager error occurs. A nil ctx is treated as [context.Background].
func (m *Manager) WaitForNotification(ctx context.Context) (*Notification, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	queue := m.ensureNotificationQueue()

	for {
		if notification := queue.pop(); notification != nil {
			return notification, nil
		}

		m.mu.Lock()
		connected := m.connected
		asyncErr := m.asyncErr
		asyncErrSignal := m.asyncErrSignal
		m.mu.Unlock()

		if !connected {
			return nil, errors.New("manager disconnected")
		}
		if asyncErr != nil {
			return nil, asyncErr
		}

		select {
		case notification := <-queue.events:
			return notification, nil
		case <-queue.done:
		case <-asyncErrSignal:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (m *Manager) ensureNotificationQueue() *notificationQueue {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.notifications == nil {
		m.notifications = newNotificationQueue()
	}
	return m.notifications
}

type notificationQueue struct {
	events       chan *Notification
	done         chan struct{}
	closeOnce    sync.Once
	overflowOnce sync.Once
}

func newNotificationQueue() *notificationQueue {
	return newNotificationQueueWithLimit(defaultMaxNotificationQueue)
}

func newNotificationQueueWithLimit(maxQueueSize int) *notificationQueue {
	if maxQueueSize <= 0 {
		maxQueueSize = defaultMaxNotificationQueue
	}
	return &notificationQueue{
		events: make(chan *Notification, maxQueueSize),
		done:   make(chan struct{}),
	}
}

func (q *notificationQueue) push(manager *Manager, notification *Notification) {
	if notification == nil {
		return
	}

	select {
	case q.events <- notification:
		return
	case <-q.done:
		return
	default:
	}

	q.overflowOnce.Do(func() {
		manager.setAsyncError(fmt.Errorf("event notification queue overflow: max size %d", cap(q.events)))
	})

	select {
	case q.events <- notification:
	case <-q.done:
	}
}

func (q *notificationQueue) pop() *Notification {
	select {
	case notification := <-q.events:
		return notification
	default:
		return nil
	}
}

func (q *notificationQueue) close() {
	q.closeOnce.Do(func() {
		close(q.done)
		for {
			select {
			case <-q.events:
			default:
				return
			}
		}
	})
}
