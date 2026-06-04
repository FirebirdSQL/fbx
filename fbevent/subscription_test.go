package fbevent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscribeDeliversEventsOnChannel(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	sub, err := m.Subscribe("EVT", 1)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, sub.Close())
	}()

	m.notify(DatabaseEvent{EventName: "EVT", EventCount: 3})

	select {
	case event := <-sub.Events():
		require.Equal(t, "EVT", event.EventName)
		require.Equal(t, 3, event.EventCount)
	case <-time.After(time.Second):
		t.Fatal("subscription did not receive event")
	}
}

func TestSubscriptionCloseRemovesListenerAndClosesChannel(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
	existingHandle := addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	sub, err := m.Subscribe("EVT", 1)
	require.NoError(t, err)

	require.NoError(t, sub.Close())

	select {
	case _, ok := <-sub.Events():
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("subscription channel was not closed")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	require.Len(t, m.listeners["EVT"], 1)
	require.Equal(t, existingHandle, m.listeners["EVT"][0].handle)
}

func TestSubscriptionCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	sub, err := m.Subscribe("EVT", 1)
	require.NoError(t, err)

	require.NoError(t, sub.Close())
	require.NoError(t, sub.Close())
}

func TestRemoveEventListenerClosesSubscriptionChannel(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	sub, err := m.Subscribe("EVT", 1)
	require.NoError(t, err)

	require.NoError(t, m.RemoveEventListener(sub.Handle()))

	select {
	case _, ok := <-sub.C:
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("subscription channel was not closed by RemoveEventListener")
	}
}

func TestDisconnectClosesSubscriptionChannel(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	sub, err := m.Subscribe("EVT", 1)
	require.NoError(t, err)

	require.NoError(t, m.Disconnect(context.Background()))

	select {
	case _, ok := <-sub.C:
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("subscription channel was not closed by Disconnect")
	}
}

func TestSubscriptionCloseUnblocksPendingDelivery(t *testing.T) {
	t.Parallel()

	dummyCalled := make(chan struct{})
	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {
		close(dummyCalled)
	}))

	sub, err := m.Subscribe("EVT", 0)
	require.NoError(t, err)

	delivery := newEventDeliveryWorker(m)
	go delivery.run()
	defer close(delivery.stop)

	delivery.send(DatabaseEvent{EventName: "EVT", EventCount: 1})

	select {
	case <-dummyCalled:
	case <-time.After(time.Second):
		t.Fatal("delivery did not reach first listener")
	}
	require.Eventually(t, func() bool {
		sub.events.mu.Lock()
		defer sub.events.mu.Unlock()
		return sub.events.active == 1
	}, time.Second, 10*time.Millisecond)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- sub.Close()
	}()

	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("subscription close blocked on pending delivery")
	}
}
