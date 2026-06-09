package fbevent

// DatabaseEvent is a delivered Firebird database event.
type DatabaseEvent struct {
	EventName  string
	EventCount int
}

// ListenerHandle uniquely identifies manager listener registration.
type ListenerHandle uint64

const (
	// InvalidListenerHandle indicates missing or failed registration.
	InvalidListenerHandle ListenerHandle = 0
)

// EventListener receives database events.
type EventListener interface {
	EventOccurred(DatabaseEvent)
}

// EventListenerFunc adapts a function to EventListener.
type EventListenerFunc func(DatabaseEvent)

// EventOccurred calls f(event).
func (f EventListenerFunc) EventOccurred(event DatabaseEvent) {
	f(event)
}
