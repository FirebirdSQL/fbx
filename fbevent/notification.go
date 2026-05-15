package fbevent

import (
	"context"
	"errors"
	"sync"
)

// Notification is a pgx-like event notification returned by WaitForNotification.
//
// Firebird events do not carry a payload. Channel contains the Firebird event
// name and Count contains the event count delta reported by the server.
type Notification struct {
	Channel string
	Count   int
}

// Listen starts listening for eventName and makes future events available via
// WaitForNotification. Calling Listen repeatedly for the same eventName is
// idempotent.
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
		queue.push(&Notification{
			Channel: event.EventName,
			Count:   event.EventCount,
		})
	})

	handle, err := m.AddEventListener(eventName, listener)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if m.notificationNames == nil {
		m.notificationNames = make(map[string]ListenerHandle)
	}
	m.notificationNames[eventName] = handle
	m.mu.Unlock()

	return nil
}

// Unlisten stops a Listen registration for eventName. It is safe to call even
// if eventName is not currently listened for.
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

// WaitForNotification waits for the next notification registered by Listen.
// It returns an already buffered notification immediately, even if ctx is
// already canceled.
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
		case <-queue.signal:
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
	mu     sync.Mutex
	queue  []*Notification
	signal chan struct{}
}

func newNotificationQueue() *notificationQueue {
	return &notificationQueue{
		signal: make(chan struct{}, 1),
	}
}

func (q *notificationQueue) push(notification *Notification) {
	if notification == nil {
		return
	}

	q.mu.Lock()
	q.queue = append(q.queue, notification)
	q.mu.Unlock()

	select {
	case q.signal <- struct{}{}:
	default:
	}
}

func (q *notificationQueue) pop() *Notification {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		return nil
	}

	notification := q.queue[0]
	copy(q.queue, q.queue[1:])
	q.queue[len(q.queue)-1] = nil
	q.queue = q.queue[:len(q.queue)-1]
	return notification
}

func (q *notificationQueue) clear() {
	q.mu.Lock()
	for i := range q.queue {
		q.queue[i] = nil
	}
	q.queue = nil
	q.mu.Unlock()
}
