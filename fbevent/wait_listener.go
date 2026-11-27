package fbevent

import (
	"context"
	"errors"
	"sync"
)

type oneTimeEventListener struct {
	mu    sync.Mutex
	once  sync.Once
	done  chan struct{}
	count int
}

func newOneTimeEventListener() *oneTimeEventListener {
	return &oneTimeEventListener{
		done: make(chan struct{}),
	}
}

func (l *oneTimeEventListener) EventOccurred(event DatabaseEvent) {
	l.once.Do(func() {
		l.mu.Lock()
		l.count = event.EventCount
		l.mu.Unlock()
		close(l.done)
	})
}

func (l *oneTimeEventListener) await(
	ctx context.Context,
	asyncErrSignal <-chan struct{},
	lastAsyncErr func() error,
) (int, error) {
	select {
	case <-l.done:
		l.mu.Lock()
		count := l.count
		l.mu.Unlock()
		return count, nil
	case <-asyncErrSignal:
		if lastAsyncErr != nil {
			if err := lastAsyncErr(); err != nil {
				return -1, err
			}
		}
		return -1, errors.New("asynchronous event manager error")
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}
