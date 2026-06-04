package fbevent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListenWaitForNotification(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	require.NoError(t, m.Listen(context.Background(), "EVT"))

	m.notify(DatabaseEvent{EventName: "EVT", EventCount: 3})

	notification, err := m.WaitForNotification(context.Background())
	require.NoError(t, err)
	require.Equal(t, "EVT", notification.Channel)
	require.Equal(t, 3, notification.Count)
}

func TestWaitForNotificationReturnsBufferedNotificationBeforeContextError(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	require.NoError(t, m.Listen(context.Background(), "EVT"))
	m.notify(DatabaseEvent{EventName: "EVT", EventCount: 1})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	notification, err := m.WaitForNotification(ctx)
	require.NoError(t, err)
	require.Equal(t, "EVT", notification.Channel)
	require.Equal(t, 1, notification.Count)
}

func TestWaitForNotificationContextCanceled(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	notification, err := m.WaitForNotification(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, notification)
}

func TestWaitForNotificationReturnsAsyncError(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()
	expectedErr := errors.New("event channel failed")
	m.setAsyncError(expectedErr)

	notification, err := m.WaitForNotification(context.Background())
	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, notification)
}

func TestUnlistenRemovesNotificationListener(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	require.NoError(t, m.Listen(context.Background(), "EVT"))
	require.NoError(t, m.Unlisten("EVT"))

	m.notify(DatabaseEvent{EventName: "EVT", EventCount: 1})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	notification, err := m.WaitForNotification(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, notification)
}

func TestListenIsIdempotent(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	require.NoError(t, m.Listen(context.Background(), "EVT"))
	require.NoError(t, m.Listen(context.Background(), "EVT"))

	m.mu.Lock()
	defer m.mu.Unlock()
	require.Len(t, m.listeners["EVT"], 2)
	require.Len(t, m.notificationNames, 1)
}

func TestListenReturnsContextError(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.Listen(ctx, "EVT")
	require.ErrorIs(t, err, context.Canceled)
}

func TestNotificationQueueOverflowSetsAsyncErrorAndBackpressures(t *testing.T) {
	t.Parallel()

	m := newConnectedManagerForNotificationTest()
	m.notifications = newNotificationQueueWithLimit(1)
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	require.NoError(t, m.Listen(context.Background(), "EVT"))

	m.notify(DatabaseEvent{EventName: "EVT", EventCount: 1})

	notifyDone := make(chan struct{})
	go func() {
		m.notify(DatabaseEvent{EventName: "EVT", EventCount: 2})
		close(notifyDone)
	}()

	require.Eventually(t, func() bool {
		err := m.LastError()
		return err != nil && err.Error() == "event notification queue overflow: max size 1"
	}, time.Second, 10*time.Millisecond)

	select {
	case <-notifyDone:
		t.Fatal("notification delivery completed while bounded queue was full")
	case <-time.After(100 * time.Millisecond):
	}

	notification, err := m.WaitForNotification(context.Background())
	require.NoError(t, err)
	require.Equal(t, "EVT", notification.Channel)
	require.Equal(t, 1, notification.Count)

	select {
	case <-notifyDone:
	case <-time.After(time.Second):
		t.Fatal("notification delivery did not resume after queue space was freed")
	}

	notification, err = m.WaitForNotification(context.Background())
	require.NoError(t, err)
	require.Equal(t, "EVT", notification.Channel)
	require.Equal(t, 2, notification.Count)
	require.ErrorContains(t, m.LastError(), "event notification queue overflow")
}

func newConnectedManagerForNotificationTest() *Manager {
	return &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
		asyncErrSignal:    make(chan struct{}),
	}
}
