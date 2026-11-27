package fbevent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOneTimeEventListenerAwaitDeadline(t *testing.T) {
	t.Parallel()

	listener := newOneTimeEventListener()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	count, err := listener.await(ctx, nil, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, -1, count)
}

func TestOneTimeEventListenerAwaitEvent(t *testing.T) {
	t.Parallel()

	listener := newOneTimeEventListener()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go listener.EventOccurred(DatabaseEvent{EventName: "TEST_EVENT", EventCount: 7})

	count, err := listener.await(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 7, count)
}

func TestOneTimeEventListenerAwaitContextDone(t *testing.T) {
	t.Parallel()

	listener := newOneTimeEventListener()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	count, err := listener.await(ctx, nil, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, -1, count)
}
