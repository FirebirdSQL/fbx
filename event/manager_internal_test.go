package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
)

type managerInternalTestListener struct {
	id int
}

func (l *managerInternalTestListener) EventOccurred(DatabaseEvent) {}

func TestListenersEqual(t *testing.T) {
	t.Parallel()

	fnA := EventListenerFunc(func(DatabaseEvent) {})
	fnB := EventListenerFunc(func(DatabaseEvent) {})
	require.True(t, listenersEqual(fnA, fnA))
	require.False(t, listenersEqual(fnA, fnB))

	l1 := &managerInternalTestListener{id: 1}
	l2 := l1
	l3 := &managerInternalTestListener{id: 2}
	require.True(t, listenersEqual(l1, l2))
	require.False(t, listenersEqual(l1, l3))

	require.False(t, listenersEqual(l1, fnA))
	require.True(t, listenersEqual(nil, nil))
	require.False(t, listenersEqual(nil, l1))
}

func TestListenersEqualDifferentClosuresFromSameLiteral(t *testing.T) {
	t.Parallel()

	functions := make([]EventListenerFunc, 0, 2)
	for _, id := range []int{1, 2} {
		listenerID := id
		functions = append(functions, EventListenerFunc(func(DatabaseEvent) {
			_ = listenerID
		}))
	}

	require.False(t, listenersEqual(functions[0], functions[1]))
	require.True(t, listenersEqual(functions[0], functions[0]))
}

func TestIsConnectedManagedDisconnectedDoesNotBlock(t *testing.T) {
	t.Parallel()

	stop := make(chan struct{})
	done := make(chan struct{})

	m := &Manager{
		mode:           managerModeManaged,
		connected:      true,
		listeners:      make(map[string][]EventListener),
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
		listeners:      make(map[string][]EventListener),
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

func TestAddEventListenerSkipsDuplicate(t *testing.T) {
	t.Parallel()

	listener := EventListenerFunc(func(DatabaseEvent) {})
	m := &Manager{
		connected: true,
		listeners: map[string][]EventListener{
			"EVT": {listener},
		},
		handlers: make(map[string]*gdsEventHandler),
	}

	err := m.AddEventListener("EVT", listener)
	require.NoError(t, err)
	require.Len(t, m.listeners["EVT"], 1)
}

func TestRemoveEventListenerRemovesAllDuplicates(t *testing.T) {
	t.Parallel()

	target := EventListenerFunc(func(DatabaseEvent) {})
	other := EventListenerFunc(func(DatabaseEvent) {})
	m := &Manager{
		connected: true,
		listeners: map[string][]EventListener{
			"EVT": {target, other, target},
		},
		handlers: make(map[string]*gdsEventHandler),
	}

	err := m.RemoveEventListener("EVT", target)
	require.NoError(t, err)
	require.Len(t, m.listeners["EVT"], 1)
	require.True(t, listenersEqual(m.listeners["EVT"][0], other))
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
		connected: true,
		listeners: map[string][]EventListener{
			"EVT": {first},
		},
		handlers: make(map[string]*gdsEventHandler),
	}

	require.NoError(t, m.AddEventListener("EVT", second))
	require.Len(t, m.listeners["EVT"], 2)

	require.NoError(t, m.RemoveEventListener("EVT", first))
	require.Len(t, m.listeners["EVT"], 1)
	require.True(t, listenersEqual(m.listeners["EVT"][0], second))
}

func TestWaitForEventReturnsAsyncErrorWithoutTimeout(t *testing.T) {
	t.Parallel()

	m := &Manager{
		connected: true,
		listeners: map[string][]EventListener{
			"EVT": {
				EventListenerFunc(func(DatabaseEvent) {}),
			},
		},
		handlers:       make(map[string]*gdsEventHandler),
		asyncErrSignal: make(chan struct{}),
	}

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

func TestAddEventListenerSerializesConcurrentFirstRegistration(t *testing.T) {
	t.Parallel()

	first := EventListenerFunc(func(DatabaseEvent) {})
	second := EventListenerFunc(func(DatabaseEvent) {})

	enteredEnsure := make(chan struct{})
	releaseEnsure := make(chan struct{})
	var once sync.Once

	m := &Manager{
		connected: true,
		listeners: make(map[string][]EventListener),
		handlers:  make(map[string]*gdsEventHandler),
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
		errFirstCh <- m.AddEventListener("EVT", first)
	}()

	select {
	case <-enteredEnsure:
	case <-time.After(time.Second):
		t.Fatal("first AddEventListener did not reach ensureEventHandler")
	}

	go func() {
		errSecondCh <- m.AddEventListener("EVT", second)
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
		connected: true,
		listeners: map[string][]EventListener{
			"EVT": {dummy},
		},
		handlers:       make(map[string]*gdsEventHandler),
		asyncErr:       errors.New("stale async error"),
		asyncErrSignal: make(chan struct{}),
	}

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
