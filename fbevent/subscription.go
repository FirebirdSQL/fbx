package fbevent

import (
	"errors"
	"sync"
)

// EventSubscription is a channel-based event subscription.
type EventSubscription struct {
	C <-chan DatabaseEvent

	manager *Manager
	handle  ListenerHandle
	events  *channelEventListener

	closeOnce sync.Once
	closeErr  error
}

// Subscribe subscribes to eventName and returns a channel-based subscription.
//
// Events are delivered sequentially by the Manager delivery loop. If the
// subscription channel is full, delivery of later callbacks and subscriptions
// for the same Manager waits until the user receives from the channel or closes
// the subscription.
func (m *Manager) Subscribe(eventName string, buffer int) (*EventSubscription, error) {
	if buffer < 0 {
		return nil, errors.New("buffer must not be negative")
	}

	listener := newChannelEventListener(buffer)
	handle, err := m.AddEventListener(eventName, listener)
	if err != nil {
		listener.close()
		return nil, err
	}

	return &EventSubscription{
		C:       listener.events,
		manager: m,
		handle:  handle,
		events:  listener,
	}, nil
}

// Events returns the subscription event channel.
func (s *EventSubscription) Events() <-chan DatabaseEvent {
	if s == nil || s.events == nil {
		return nil
	}
	return s.C
}

// Handle returns the underlying listener registration handle.
func (s *EventSubscription) Handle() ListenerHandle {
	if s == nil {
		return InvalidListenerHandle
	}
	return s.handle
}

// Close unsubscribes from the event and closes the event channel.
func (s *EventSubscription) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if s.manager == nil || s.handle == InvalidListenerHandle {
			if s.events != nil {
				s.events.close()
			}
			return
		}
		s.closeErr = s.manager.RemoveEventListener(s.handle)
		if s.closeErr == nil && s.events != nil {
			s.events.close()
		}
	})
	return s.closeErr
}

type closeableEventListener interface {
	close()
}

func closeListenerIfNeeded(listener EventListener) {
	if listener == nil {
		return
	}
	if closeable, ok := listener.(closeableEventListener); ok {
		closeable.close()
	}
}

type channelEventListener struct {
	events chan DatabaseEvent
	done   chan struct{}

	mu     sync.Mutex
	cond   *sync.Cond
	closed bool
	active int
}

func newChannelEventListener(buffer int) *channelEventListener {
	l := &channelEventListener{
		events: make(chan DatabaseEvent, buffer),
		done:   make(chan struct{}),
	}
	l.cond = sync.NewCond(&l.mu)
	return l
}

func (l *channelEventListener) EventOccurred(event DatabaseEvent) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	l.active++
	events := l.events
	done := l.done
	l.mu.Unlock()

	defer func() {
		l.mu.Lock()
		l.active--
		if l.active == 0 {
			l.cond.Broadcast()
		}
		l.mu.Unlock()
	}()

	select {
	case events <- event:
	case <-done:
	}
}

func (l *channelEventListener) close() {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	l.closed = true
	close(l.done)
	for l.active > 0 {
		l.cond.Wait()
	}
	close(l.events)
	l.mu.Unlock()
}
