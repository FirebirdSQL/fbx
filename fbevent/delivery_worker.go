package fbevent

import (
	"fmt"
	"sync"
)

// eventDeliveryWorker delivers events to manager listeners sequentially.
// The queue is bounded so a stuck listener cannot grow memory without limit.
type eventDeliveryWorker struct {
	manager      *Manager
	queue        chan DatabaseEvent
	stop         chan struct{}
	overflowOnce sync.Once
}

func newEventDeliveryWorker(manager *Manager) *eventDeliveryWorker {
	return newEventDeliveryWorkerWithQueueSize(manager, defaultMaxEventDeliveryQueue)
}

func newEventDeliveryWorkerWithQueueSize(manager *Manager, maxQueueSize int) *eventDeliveryWorker {
	if maxQueueSize <= 0 {
		maxQueueSize = defaultMaxEventDeliveryQueue
	}
	return &eventDeliveryWorker{
		manager: manager,
		queue:   make(chan DatabaseEvent, maxQueueSize),
		stop:    make(chan struct{}),
	}
}

func (w *eventDeliveryWorker) send(event DatabaseEvent) {
	select {
	case w.queue <- event:
		return
	case <-w.stop:
		return
	default:
	}

	w.overflowOnce.Do(func() {
		w.manager.setAsyncError(fmt.Errorf("event delivery queue overflow: max size %d", cap(w.queue)))
	})

	select {
	case w.queue <- event:
	case <-w.stop:
	}
}

func (w *eventDeliveryWorker) run() {
	for {
		select {
		case <-w.stop:
			return
		case event := <-w.queue:
			w.manager.notify(event)
		}
	}
}
