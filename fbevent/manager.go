package fbevent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
)

type managerMode uint8

const (
	managerModeDefault managerMode = iota
	managerModeManaged
)

const (
	defaultManagerWaitTimeout        = time.Second
	managerOperationTimeout          = 5 * time.Second
	managerDispatcherShutdownTimeout = 5 * time.Second
	defaultMaxEventDeliveryQueue     = 4096
)

type listenerRegistration struct {
	handle   ListenerHandle
	listener EventListener
}

// Manager owns Firebird event registrations and delivers matching events to
// listeners.
//
// Managers should be constructed with [NewManager], [NewManagerConfig], or
// [CreateFor]. Call [Manager.Connect] before registering listeners, and call
// [Manager.Close] or [Manager.Disconnect] when event processing is no longer
// needed.
type Manager struct {
	mu sync.Mutex
	// listenerOpMu serializes listener add/remove operations that can cause
	// first-registration / last-unregistration network I/O. This avoids races
	// where a listener could be visible in memory without a successful server
	// registration for that event name.
	listenerOpMu sync.Mutex

	mode       managerMode
	connString string
	config     *fbx.ConnConfig
	conn       *fbx.Conn
	ownConn    bool

	connected bool

	waitTimeout time.Duration

	listeners         map[string][]listenerRegistration
	listenerEventName map[ListenerHandle]string
	nextListenerID    ListenerHandle
	handlers          map[string]*fbEventHandler
	notificationOpMu  sync.Mutex
	notifications     *notificationQueue
	notificationNames map[string]ListenerHandle

	eventQueue     chan DatabaseEvent
	delivery       *eventDeliveryWorker
	dispatcherStop chan struct{}
	dispatcherDone chan struct{}

	lifecycleWatchStop chan struct{}
	lifecycleWatchGen  uint64

	asyncErr error
	// asyncErrSignal is closed when the first asynchronous manager error is set.
	// Waiters can listen on this channel to avoid hanging indefinitely.
	asyncErrSignal chan struct{}

	// beforeEnsureEventHandler is a test hook used by internal tests to control
	// interleavings around first-listener registration.
	beforeEnsureEventHandler func()
}

// NewManager parses connString and returns a manager that owns its database
// connection.
//
// The connection is opened by [Manager.Connect], not by NewManager. Closing or
// disconnecting the manager also closes the owned connection.
func NewManager(connString string) (*Manager, error) {
	cfg, err := fbx.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	return NewManagerConfig(cfg), nil
}

// NewManagerConfig returns a manager that owns a connection created from cfg.
//
// The config is copied before it is stored, so later changes to cfg do not
// affect the manager. The connection is opened by [Manager.Connect], not by
// NewManagerConfig.
func NewManagerConfig(cfg *fbx.ConnConfig) *Manager {
	var cfgCopy *fbx.ConnConfig
	var connString string
	if cfg != nil {
		cfgCopy = cfg.Copy()
		connString = cfg.ConnString()
	}
	return &Manager{
		mode:              managerModeDefault,
		config:            cfgCopy,
		connString:        connString,
		ownConn:           true,
		waitTimeout:       defaultManagerWaitTimeout,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
}

// CreateFor returns a manager that uses an existing connection.
//
// The returned manager does not own conn. [Manager.Close] and
// [Manager.Disconnect] unregister event listeners but leave conn open.
func CreateFor(conn *fbx.Conn) *Manager {
	return &Manager{
		mode:              managerModeManaged,
		conn:              conn,
		ownConn:           false,
		waitTimeout:       defaultManagerWaitTimeout,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*fbEventHandler),
	}
}

// Connect starts event processing for the manager.
//
// For a manager created with [NewManager] or [NewManagerConfig], Connect opens
// the database connection. For a manager created with [CreateFor], Connect uses
// the existing connection and returns an error if it is nil or closed. Connect
// returns an error when called on an already connected manager.
func (m *Manager) Connect(ctx context.Context) error {
	m.mu.Lock()
	if m.connected {
		m.mu.Unlock()
		return errors.New("connect called while already connected")
	}
	mode := m.mode
	cfg := m.config
	connString := m.connString
	conn := m.conn
	m.mu.Unlock()

	switch mode {
	case managerModeDefault:
		var err error
		if cfg != nil {
			conn, err = fbx.ConnectConfig(ctx, cfg.Copy())
		} else {
			conn, err = fbx.Connect(ctx, connString)
		}
		if err != nil {
			return err
		}
	case managerModeManaged:
		if conn == nil || conn.IsClosed() {
			return errors.New("not connected to server")
		}
	default:
		return errors.New("unknown manager mode")
	}

	m.mu.Lock()
	m.conn = conn
	m.connected = true
	m.asyncErr = nil
	m.asyncErrSignal = make(chan struct{})
	m.eventQueue = make(chan DatabaseEvent, 256)
	m.delivery = newEventDeliveryWorker(m)
	m.dispatcherStop = make(chan struct{})
	m.dispatcherDone = make(chan struct{})
	m.lifecycleWatchStop = make(chan struct{})
	m.lifecycleWatchGen++
	queue := m.eventQueue
	delivery := m.delivery
	stop := m.dispatcherStop
	done := m.dispatcherDone
	lifecycleStop := m.lifecycleWatchStop
	lifecycleGen := m.lifecycleWatchGen
	m.mu.Unlock()

	m.bindConnEventErrorHandler(conn)
	m.startConnLifecycleWatcher(conn, lifecycleStop, lifecycleGen)
	go delivery.run()
	go m.dispatcherLoop(stop, done, queue, delivery)
	return nil
}

// Disconnect stops event processing and unregisters all listeners.
//
// For a manager that owns its connection, Disconnect also closes that
// connection. For a manager created with [CreateFor], the underlying connection
// is left open. Disconnect returns an error when called on an already
// disconnected manager.
func (m *Manager) Disconnect(ctx context.Context) error {
	return m.disconnect(ctx, true, true, errors.New("event manager disconnected"))
}

// Close stops event processing if the manager is connected.
//
// Close has the same cleanup behavior as [Manager.Disconnect], but it is safe to
// call when the manager is already disconnected.
func (m *Manager) Close(ctx context.Context) error {
	return m.disconnect(ctx, false, true, errors.New("event manager disconnected"))
}

func (m *Manager) disconnect(ctx context.Context, strict bool, waitDispatcher bool, disconnectErr error) error {
	m.mu.Lock()
	if !m.connected {
		m.mu.Unlock()
		if strict {
			return errors.New("disconnect called while not connected")
		}
		return nil
	}

	handlers := make([]*fbEventHandler, 0, len(m.handlers))
	for _, handler := range m.handlers {
		handlers = append(handlers, handler)
	}
	var allRegs []listenerRegistration
	for _, regs := range m.listeners {
		allRegs = append(allRegs, regs...)
	}
	m.handlers = make(map[string]*fbEventHandler)
	m.listeners = make(map[string][]listenerRegistration)
	m.listenerEventName = make(map[ListenerHandle]string)
	m.notificationNames = make(map[string]ListenerHandle)
	if m.notifications != nil {
		m.notifications.close()
		m.notifications = nil
	}
	m.nextListenerID = InvalidListenerHandle

	stop := m.dispatcherStop
	done := m.dispatcherDone
	delivery := m.delivery
	m.dispatcherStop = nil
	m.dispatcherDone = nil
	m.eventQueue = nil
	m.delivery = nil
	lifecycleStop := m.lifecycleWatchStop
	m.lifecycleWatchStop = nil
	if disconnectErr != nil {
		m.setAsyncErrorLocked(disconnectErr)
	}

	conn := m.conn
	ownConn := m.ownConn
	var lowConn *fbconn.FbConn
	if conn != nil && !conn.IsClosed() {
		lowConn = conn.FbConn()
	}

	m.connected = false
	if ownConn {
		m.conn = nil
	}
	m.mu.Unlock()

	for _, reg := range allRegs {
		closeListenerIfNeeded(reg.listener)
	}
	m.unbindConnEventErrorHandler(conn)
	if lifecycleStop != nil {
		close(lifecycleStop)
	}
	if delivery != nil {
		close(delivery.stop)
	}
	if stop != nil {
		close(stop)
	}
	var allErr error
	if waitDispatcher && done != nil {
		waitCtx, waitCancel := m.dispatcherStopContext(ctx)
		select {
		case <-done:
		case <-waitCtx.Done():
			allErr = errors.Join(allErr, fmt.Errorf("event dispatcher stop failed: %w", waitCtx.Err()))
		}
		waitCancel()
	}

	for _, handler := range handlers {
		unregisterCtx, unregisterCancel := m.operationContext(ctx)
		err := handler.unregisterOnConn(unregisterCtx, lowConn)
		unregisterCancel()
		if err != nil {
			allErr = errors.Join(allErr, err)
		}
	}

	if ownConn && conn != nil {
		closeCtx, closeCancel := m.operationContext(ctx)
		err := conn.Close(closeCtx)
		closeCancel()
		if err != nil {
			allErr = errors.Join(allErr, err)
		}
	}

	return allErr
}

// IsConnected reports whether the manager is currently connected.
//
// For a manager created with [CreateFor], IsConnected also observes whether the
// underlying connection has already been closed and updates the manager state if
// needed.
func (m *Manager) IsConnected() bool {
	m.mu.Lock()
	connected := m.connected
	mode := m.mode
	conn := m.conn
	m.mu.Unlock()

	if connected && mode == managerModeManaged && (conn == nil || conn.IsClosed()) {
		if err := m.disconnect(context.Background(), false, false, nil); err != nil {
			m.setAsyncError(fmt.Errorf("managed manager disconnect failed: %w", err))
		}
		return false
	}
	return connected
}

// LastError returns the first manager error recorded since the last successful
// [Manager.Connect].
//
// Recorded errors can come from the underlying event channel, event
// re-registration, notification queue overflow, delivery queue overflow, or
// manager shutdown that wakes waiting operations.
func (m *Manager) LastError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.asyncErr
}

// GetWaitTimeout returns the dispatcher polling interval.
func (m *Manager) GetWaitTimeout() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.waitTimeout
}

// SetWaitTimeout sets the dispatcher polling interval.
//
// If timeout is less than or equal to zero, SetWaitTimeout uses a one-second
// interval.
func (m *Manager) SetWaitTimeout(timeout time.Duration) {
	if timeout <= 0 {
		timeout = time.Second
	}
	m.mu.Lock()
	m.waitTimeout = timeout
	m.mu.Unlock()
}

// AddEventListener registers listener for eventName and returns its handle.
//
// The manager must be connected, eventName must be non-empty, and listener must
// be non-nil. Each successful call creates a separate registration, even when
// the same listener value is registered more than once. Pass the returned handle
// to [Manager.RemoveEventListener] to remove that registration.
func (m *Manager) AddEventListener(eventName string, listener EventListener) (ListenerHandle, error) {
	if eventName == "" {
		return InvalidListenerHandle, errors.New("eventName must not be empty")
	}
	if listener == nil {
		return InvalidListenerHandle, errors.New("listener must not be nil")
	}
	m.listenerOpMu.Lock()
	defer m.listenerOpMu.Unlock()

	m.mu.Lock()
	if !m.connected {
		m.mu.Unlock()
		return InvalidListenerHandle, errors.New("can't add event listeners to disconnected manager")
	}
	if m.listeners == nil {
		m.listeners = make(map[string][]listenerRegistration)
	}
	if m.listenerEventName == nil {
		m.listenerEventName = make(map[ListenerHandle]string)
	}
	m.nextListenerID++
	if m.nextListenerID == InvalidListenerHandle {
		m.nextListenerID++
	}
	handle := m.nextListenerID

	currentListeners := m.listeners[eventName]
	hasListeners := len(currentListeners) > 0
	reg := listenerRegistration{
		handle:   handle,
		listener: listener,
	}
	m.listeners[eventName] = append(currentListeners, reg)
	m.listenerEventName[handle] = eventName
	m.mu.Unlock()

	if hasListeners {
		return handle, nil
	}

	m.mu.Lock()
	beforeEnsureHook := m.beforeEnsureEventHandler
	m.mu.Unlock()
	if beforeEnsureHook != nil {
		beforeEnsureHook()
	}

	if err := m.ensureEventHandler(eventName); err != nil {
		m.removeListenerSilently(handle)
		return InvalidListenerHandle, err
	}
	return handle, nil
}

// RemoveEventListener removes the listener registration identified by handle.
//
// Removing an unknown handle is a no-op. Removing [InvalidListenerHandle]
// returns an error. When the last listener for an event name is removed, the
// manager unregisters that event from the server.
func (m *Manager) RemoveEventListener(handle ListenerHandle) error {
	if handle == InvalidListenerHandle {
		return errors.New("listener handle must not be invalid")
	}
	m.listenerOpMu.Lock()
	defer m.listenerOpMu.Unlock()

	var handler *fbEventHandler
	m.mu.Lock()
	eventName, exists := m.listenerEventName[handle]
	if !exists {
		m.mu.Unlock()
		return nil
	}

	list := m.listeners[eventName]
	if len(list) == 0 {
		delete(m.listenerEventName, handle)
		m.mu.Unlock()
		return nil
	}

	updated := make([]listenerRegistration, 0, len(list))
	var removedReg listenerRegistration
	removed := false
	for _, existing := range list {
		if existing.handle == handle {
			removed = true
			removedReg = existing
			continue
		}
		updated = append(updated, existing)
	}
	if !removed {
		delete(m.listenerEventName, handle)
		m.mu.Unlock()
		return nil
	}
	delete(m.listenerEventName, handle)

	if len(updated) > 0 {
		m.listeners[eventName] = updated
		m.mu.Unlock()
		closeListenerIfNeeded(removedReg.listener)
		return nil
	}

	delete(m.listeners, eventName)
	handler = m.handlers[eventName]
	delete(m.handlers, eventName)
	m.mu.Unlock()

	closeListenerIfNeeded(removedReg.listener)
	if handler != nil {
		return handler.unregister(context.Background())
	}
	return nil
}

func (m *Manager) ensureEventHandler(eventName string) error {
	m.mu.Lock()
	if !m.connected {
		m.mu.Unlock()
		return errors.New("manager disconnected")
	}
	if _, ok := m.handlers[eventName]; ok {
		m.mu.Unlock()
		return nil
	}
	conn := m.conn
	m.mu.Unlock()

	if conn == nil || conn.IsClosed() {
		return errors.New("not connected to server")
	}

	handler := &fbEventHandler{
		manager:   m,
		eventName: eventName,
	}
	handle, err := conn.FbConn().CreateEventHandle(eventName, handler)
	if err != nil {
		return err
	}
	handler.handle = handle
	if err := handler.register(context.Background()); err != nil {
		return err
	}

	m.mu.Lock()
	if !m.connected {
		m.mu.Unlock()
		unregErr := handler.unregister(context.Background())
		if unregErr != nil {
			return errors.Join(errors.New("manager disconnected"), unregErr)
		}
		return errors.New("manager disconnected")
	}
	listeners := m.listeners[eventName]
	if len(listeners) == 0 {
		m.mu.Unlock()
		if err := handler.unregister(context.Background()); err != nil {
			return err
		}
		return nil
	}
	if _, exists := m.handlers[eventName]; exists {
		m.mu.Unlock()
		if err := handler.unregister(context.Background()); err != nil {
			return err
		}
		return nil
	}
	m.handlers[eventName] = handler
	m.mu.Unlock()

	return nil
}

func (m *Manager) removeListenerSilently(handle ListenerHandle) {
	m.mu.Lock()

	eventName, exists := m.listenerEventName[handle]
	if !exists {
		m.mu.Unlock()
		return
	}

	var removedListener EventListener
	list := m.listeners[eventName]
	for i, existing := range list {
		if existing.handle == handle {
			removedListener = existing.listener
			list = append(list[:i], list[i+1:]...)
			break
		}
	}
	delete(m.listenerEventName, handle)
	if len(list) == 0 {
		delete(m.listeners, eventName)
	} else {
		m.listeners[eventName] = list
	}
	m.mu.Unlock()

	closeListenerIfNeeded(removedListener)
}

func (m *Manager) enqueueEvent(event DatabaseEvent) {
	m.mu.Lock()
	queue := m.eventQueue
	stop := m.dispatcherStop
	connected := m.connected
	m.mu.Unlock()
	if !connected || queue == nil {
		return
	}
	select {
	case queue <- event:
	case <-stop:
	}
}

func (m *Manager) dispatcherLoop(stop <-chan struct{}, done chan<- struct{}, queue <-chan DatabaseEvent, delivery *eventDeliveryWorker) {
	defer close(done)
	for {
		timeout := m.GetWaitTimeout()
		timer := time.NewTimer(timeout)
		select {
		case <-stop:
			timer.Stop()
			return
		case event, ok := <-queue:
			timer.Stop()
			if !ok {
				return
			}
			if delivery != nil {
				delivery.send(event)
			}
		case <-timer.C:
			if !m.IsConnected() {
				return
			}
		}
	}
}

func (m *Manager) notify(event DatabaseEvent) {
	m.mu.Lock()
	registrations := append([]listenerRegistration(nil), m.listeners[event.EventName]...)
	m.mu.Unlock()
	for _, registration := range registrations {
		registration.listener.EventOccurred(event)
	}
}
