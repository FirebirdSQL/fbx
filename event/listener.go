package event

// DatabaseEvent is a delivered Firebird database event.
type DatabaseEvent struct {
	EventName  string
	EventCount int
}

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
