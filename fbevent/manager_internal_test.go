package fbevent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
)

func addRegisteredListenerForTest(m *Manager, eventName string, listener EventListener) ListenerHandle {
	m.nextListenerID++
	handle := m.nextListenerID
	m.listeners[eventName] = append(m.listeners[eventName], listenerRegistration{
		handle:   handle,
		listener: listener,
	})
	m.listenerEventName[handle] = eventName
	return handle
}

func TestAddEventListenerReturnsUniqueHandles(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
	}

	existing := EventListenerFunc(func(DatabaseEvent) {})
	existingHandle := addRegisteredListenerForTest(m, "EVT", existing)

	first := EventListenerFunc(func(DatabaseEvent) {})
	second := EventListenerFunc(func(DatabaseEvent) {})

	firstHandle, err := m.AddEventListener("EVT", first)
	require.NoError(t, err)

	secondHandle, err := m.AddEventListener("EVT", second)
	require.NoError(t, err)

	require.NotEqual(t, InvalidListenerHandle, firstHandle)
	require.NotEqual(t, InvalidListenerHandle, secondHandle)
	require.NotEqual(t, firstHandle, secondHandle)
	require.NotEqual(t, existingHandle, firstHandle)
	require.NotEqual(t, existingHandle, secondHandle)
	require.Len(t, m.listeners["EVT"], 3)
}

func TestRemoveEventListenerByHandle(t *testing.T) {
	t.Parallel()

	functions := make([]EventListener, 0, 2)
	for _, id := range []int{1, 2} {
		listenerID := id
		functions = append(functions, EventListenerFunc(func(DatabaseEvent) {
			_ = listenerID
		}))
	}

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
	}

	firstHandle := addRegisteredListenerForTest(m, "EVT", functions[0])
	secondHandle := addRegisteredListenerForTest(m, "EVT", functions[1])

	require.NoError(t, m.RemoveEventListener(firstHandle))
	require.Len(t, m.listeners["EVT"], 1)
	require.Equal(t, secondHandle, m.listeners["EVT"][0].handle)
}

func TestIsConnectedManagedDisconnectedDoesNotBlock(t *testing.T) {
	t.Parallel()

	stop := make(chan struct{})
	done := make(chan struct{})

	m := &Manager{
		mode:           managerModeManaged,
		connected:      true,
		listeners:      make(map[string][]listenerRegistration),
		handlers:       make(map[string]*gdsEventHandler),
		eventQueue:     make(chan DatabaseEvent, 1),
		dispatcherStop: stop,
		dispatcherDone: done,
	}

	resultCh := make(chan bool, 1)
	go func() {
		resultCh <- m.IsConnected()
	}()

	select {
	case connected := <-resultCh:
		require.False(t, connected)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("IsConnected blocked while dispatcherDone was not closed")
	}

	select {
	case <-stop:
	default:
		t.Fatal("dispatcher stop channel was not closed")
	}

	m.mu.Lock()
	require.False(t, m.connected)
	require.Nil(t, m.dispatcherStop)
	require.Nil(t, m.dispatcherDone)
	require.Nil(t, m.eventQueue)
	m.mu.Unlock()
}

func TestDisconnectRespectsContextWhenDispatcherDoesNotStop(t *testing.T) {
	t.Parallel()

	stop := make(chan struct{})
	done := make(chan struct{})

	m := &Manager{
		connected:      true,
		listeners:      make(map[string][]listenerRegistration),
		handlers:       make(map[string]*gdsEventHandler),
		eventQueue:     make(chan DatabaseEvent, 1),
		dispatcherStop: stop,
		dispatcherDone: done,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := m.Disconnect(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.ErrorContains(t, err, context.DeadlineExceeded.Error())
	require.Less(t, elapsed, time.Second)

	select {
	case <-stop:
	default:
		t.Fatal("dispatcher stop channel was not closed")
	}
}

func TestOperationContextAddsTimeoutForBackground(t *testing.T) {
	t.Parallel()

	m := &Manager{}
	ctx, cancel := m.operationContext(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	require.True(t, hasDeadline)
}

func TestAddEventListenerCreatesSeparateRegistrationsForSameListener(t *testing.T) {
	t.Parallel()

	listener := EventListenerFunc(func(DatabaseEvent) {})
	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
	}

	firstHandle := addRegisteredListenerForTest(m, "EVT", listener)

	secondHandle, err := m.AddEventListener("EVT", listener)
	require.NoError(t, err)
	require.Len(t, m.listeners["EVT"], 2)
	require.NotEqual(t, firstHandle, secondHandle)
}

func TestAddRemoveEventListenerWithDifferentClosuresFromSameLiteral(t *testing.T) {
	t.Parallel()

	functions := make([]EventListenerFunc, 0, 2)
	for _, id := range []int{1, 2} {
		listenerID := id
		functions = append(functions, EventListenerFunc(func(DatabaseEvent) {
			_ = listenerID
		}))
	}
	first := functions[0]
	second := functions[1]

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
	}

	firstHandle := addRegisteredListenerForTest(m, "EVT", first)
	secondHandle := addRegisteredListenerForTest(m, "EVT", second)

	require.Len(t, m.listeners["EVT"], 2)

	require.NoError(t, m.RemoveEventListener(firstHandle))
	require.Len(t, m.listeners["EVT"], 1)
	require.Equal(t, secondHandle, m.listeners["EVT"][0].handle)
}

func TestWaitForEventReturnsAsyncErrorWithoutTimeout(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
		asyncErrSignal:    make(chan struct{}),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	expectedErr := errors.New("async event channel failed")
	resultCh := make(chan error, 1)

	go func() {
		_, err := m.WaitForEvent(context.Background(), "EVT", 0)
		resultCh <- err
	}()

	require.Eventually(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return len(m.listeners["EVT"]) == 2
	}, time.Second, 10*time.Millisecond)

	m.setAsyncError(expectedErr)

	select {
	case err := <-resultCh:
		require.ErrorIs(t, err, expectedErr)
	case <-time.After(time.Second):
		t.Fatal("WaitForEvent blocked after async error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	require.Len(t, m.listeners["EVT"], 1)
}

func TestOnConnEventErrorUnblocksWaitForEventWithoutTimeout(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
		asyncErrSignal:    make(chan struct{}),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	expectedErr := errors.New("event channel read failed")
	resultCh := make(chan error, 1)

	go func() {
		_, err := m.WaitForEvent(context.Background(), "EVT", 0)
		resultCh <- err
	}()

	require.Eventually(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return len(m.listeners["EVT"]) == 2
	}, time.Second, 10*time.Millisecond)

	m.onConnEventError(expectedErr)

	select {
	case err := <-resultCh:
		require.ErrorIs(t, err, expectedErr)
	case <-time.After(time.Second):
		t.Fatal("WaitForEvent blocked after event channel error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	require.Len(t, m.listeners["EVT"], 1)
}

func TestDisconnectUnblocksWaitForEventWithoutTimeout(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
		asyncErrSignal:    make(chan struct{}),
	}
	addRegisteredListenerForTest(m, "EVT", EventListenerFunc(func(DatabaseEvent) {}))

	resultCh := make(chan error, 1)
	go func() {
		_, err := m.WaitForEvent(context.Background(), "EVT", 0)
		resultCh <- err
	}()

	require.Eventually(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return len(m.listeners["EVT"]) == 2
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, m.Disconnect(context.Background()))

	select {
	case err := <-resultCh:
		require.Error(t, err)
		require.ErrorContains(t, err, "event manager disconnected")
	case <-time.After(time.Second):
		t.Fatal("WaitForEvent blocked after manager disconnect")
	}
}

func TestAddEventListenerSerializesConcurrentFirstRegistration(t *testing.T) {
	t.Parallel()

	first := EventListenerFunc(func(DatabaseEvent) {})
	second := EventListenerFunc(func(DatabaseEvent) {})

	enteredEnsure := make(chan struct{})
	releaseEnsure := make(chan struct{})
	var once sync.Once

	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
		beforeEnsureEventHandler: func() {
			once.Do(func() {
				close(enteredEnsure)
				<-releaseEnsure
			})
		},
	}

	errFirstCh := make(chan error, 1)
	errSecondCh := make(chan error, 1)

	go func() {
		_, err := m.AddEventListener("EVT", first)
		errFirstCh <- err
	}()

	select {
	case <-enteredEnsure:
	case <-time.After(time.Second):
		t.Fatal("first AddEventListener did not reach ensureEventHandler")
	}

	go func() {
		_, err := m.AddEventListener("EVT", second)
		errSecondCh <- err
	}()

	select {
	case <-errSecondCh:
		t.Fatal("second AddEventListener returned before first registration finished")
	case <-time.After(150 * time.Millisecond):
	}

	close(releaseEnsure)

	errFirst := <-errFirstCh
	errSecond := <-errSecondCh
	require.Error(t, errFirst)
	require.Error(t, errSecond)

	m.mu.Lock()
	defer m.mu.Unlock()
	require.Empty(t, m.listeners["EVT"])
	_, exists := m.handlers["EVT"]
	require.False(t, exists)
}

func TestUnregisterRollbackOnCancelFailure(t *testing.T) {
	t.Parallel()

	h := &gdsEventHandler{
		manager:   &Manager{},
		eventName: "EVT",
		state:     eventHandlerStateRegistered,
	}

	// Force CancelEvent failure (nil handle).
	err := h.unregisterOnConn(context.Background(), &fbconn.FbConn{})
	require.Error(t, err)

	h.mu.Lock()
	require.Equal(t, eventHandlerStateRegistered, h.state)
	h.mu.Unlock()

	// Best-effort local cancellation when connection is unavailable.
	err = h.unregisterOnConn(context.Background(), nil)
	require.NoError(t, err)

	h.mu.Lock()
	require.Equal(t, eventHandlerStateCancelled, h.state)
	h.mu.Unlock()
}

func TestWaitForEventIgnoresStaleAsyncErrorAfterSuccessfulAwait(t *testing.T) {
	t.Parallel()

	dummy := EventListenerFunc(func(DatabaseEvent) {})
	m := &Manager{
		connected:         true,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
		asyncErr:          errors.New("stale async error"),
		asyncErrSignal:    make(chan struct{}),
	}
	addRegisteredListenerForTest(m, "EVT", dummy)

	notifyDone := make(chan struct{})
	go func() {
		defer close(notifyDone)

		deadline := time.Now().Add(time.Second)
		for {
			m.mu.Lock()
			count := len(m.listeners["EVT"])
			m.mu.Unlock()
			if count == 2 {
				break
			}
			if time.Now().After(deadline) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		m.notify(DatabaseEvent{EventName: "EVT", EventCount: 7})
	}()

	count, err := m.WaitForEvent(context.Background(), "EVT", time.Second)
	require.NoError(t, err)
	require.Equal(t, 7, count)

	select {
	case <-notifyDone:
	case <-time.After(time.Second):
		t.Fatal("notify goroutine did not finish")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	require.Len(t, m.listeners["EVT"], 1)
}

func TestHandleConnLifecycleTerminationDisconnectsAndSignalsAsyncError(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		mode:              managerModeManaged,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
		asyncErrSignal:    make(chan struct{}),
		lifecycleWatchGen: 1,
	}

	m.handleConnLifecycleTermination(nil, 1)

	m.mu.Lock()
	require.False(t, m.connected)
	m.mu.Unlock()
	err := m.LastError()
	require.Error(t, err)
	require.ErrorContains(t, err, "underlying connection detached")
}

func TestHandleConnLifecycleTerminationIgnoresStaleGeneration(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected:         true,
		mode:              managerModeManaged,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
		asyncErrSignal:    make(chan struct{}),
		lifecycleWatchGen: 2,
	}

	m.handleConnLifecycleTermination(nil, 1)

	m.mu.Lock()
	require.True(t, m.connected)
	m.mu.Unlock()
	require.NoError(t, m.LastError())
}
