package event

import (
	"context"
	"errors"
	"sync"
	"time"
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

func (l *oneTimeEventListener) await(ctx context.Context, timeout time.Duration) (int, error) {
	return l.awaitWithAsyncError(ctx, timeout, nil, nil)
}

func (l *oneTimeEventListener) awaitWithAsyncError(
	ctx context.Context,
	timeout time.Duration,
	asyncErrSignal <-chan struct{},
	lastAsyncErr func() error,
) (int, error) {
	if timeout == 0 {
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

	timer := time.NewTimer(timeout)
	defer timer.Stop()

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
	case <-timer.C:
		return -1, nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}
