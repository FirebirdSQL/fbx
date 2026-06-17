package fbevent

// DatabaseEvent describes a Firebird event delivered to a listener.
type DatabaseEvent struct {
	// EventName is the name of the database event.
	EventName string

	// EventCount is the event count delta reported by the server.
	EventCount int
}

// ListenerHandle uniquely identifies a listener registration on a [Manager].
type ListenerHandle uint64

const (
	// InvalidListenerHandle indicates missing or failed registration.
	InvalidListenerHandle ListenerHandle = 0
)

// EventListener receives database events.
type EventListener interface {
	// EventOccurred is called when a matching database event is delivered.
	EventOccurred(DatabaseEvent)
}

// EventListenerFunc adapts a function to the [EventListener] interface.
type EventListenerFunc func(DatabaseEvent)

// EventOccurred calls f(event).
func (f EventListenerFunc) EventOccurred(event DatabaseEvent) {
	f(event)
}
