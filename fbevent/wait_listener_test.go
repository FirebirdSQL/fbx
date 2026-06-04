package fbevent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOneTimeEventListenerAwaitTimeout(t *testing.T) {
	t.Parallel()

	listener := newOneTimeEventListener()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	count, err := listener.await(ctx, 20*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, -1, count)
}

func TestOneTimeEventListenerAwaitEvent(t *testing.T) {
	t.Parallel()

	listener := newOneTimeEventListener()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go listener.EventOccurred(DatabaseEvent{EventName: "TEST_EVENT", EventCount: 7})

	count, err := listener.await(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, 7, count)
}

func TestOneTimeEventListenerAwaitContextDone(t *testing.T) {
	t.Parallel()

	listener := newOneTimeEventListener()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	count, err := listener.await(ctx, time.Second)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, -1, count)
}
