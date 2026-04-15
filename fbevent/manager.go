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
)

type listenerRegistration struct {
	handle   ListenerHandle
	listener EventListener
}

// Manager is a high-level Firebird event manager.
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
	handlers          map[string]*gdsEventHandler

	eventQueue     chan DatabaseEvent
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

// NewManager creates a manager for a connection string.
func NewManager(connString string) (*Manager, error) {
	cfg, err := fbx.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	return NewManagerConfig(cfg), nil
}

// NewManagerConfig creates a manager for a connection config.
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
		handlers:          make(map[string]*gdsEventHandler),
	}
}

// CreateFor creates a manager over an existing connection.
func CreateFor(conn *fbx.Conn) *Manager {
	return &Manager{
		mode:              managerModeManaged,
		conn:              conn,
		ownConn:           false,
		waitTimeout:       defaultManagerWaitTimeout,
		listeners:         make(map[string][]listenerRegistration),
		listenerEventName: make(map[ListenerHandle]string),
		handlers:          make(map[string]*gdsEventHandler),
	}
}

// Connect initializes manager event processing.
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
	m.dispatcherStop = make(chan struct{})
	m.dispatcherDone = make(chan struct{})
	m.lifecycleWatchStop = make(chan struct{})
	m.lifecycleWatchGen++
	stop := m.dispatcherStop
	done := m.dispatcherDone
	lifecycleStop := m.lifecycleWatchStop
	lifecycleGen := m.lifecycleWatchGen
	m.mu.Unlock()

	m.bindConnEventErrorHandler(conn)
	m.startConnLifecycleWatcher(conn, lifecycleStop, lifecycleGen)
	go m.dispatcherLoop(stop, done)
	return nil
}

// Disconnect stops manager event processing.
func (m *Manager) Disconnect(ctx context.Context) error {
	return m.disconnect(ctx, true, true, errors.New("event manager disconnected"))
}

// Close disconnects manager if connected.
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

	handlers := make([]*gdsEventHandler, 0, len(m.handlers))
	for _, handler := range m.handlers {
		handlers = append(handlers, handler)
	}
	m.handlers = make(map[string]*gdsEventHandler)
	m.listeners = make(map[string][]listenerRegistration)
	m.listenerEventName = make(map[ListenerHandle]string)
	m.nextListenerID = InvalidListenerHandle

	stop := m.dispatcherStop
	done := m.dispatcherDone
	m.dispatcherStop = nil
	m.dispatcherDone = nil
	m.eventQueue = nil
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

	m.unbindConnEventErrorHandler(conn)
	if lifecycleStop != nil {
		close(lifecycleStop)
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

// IsConnected reports manager connection state.
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

// LastError returns the first asynchronous manager error, if any.
func (m *Manager) LastError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.asyncErr
}

// GetWaitTimeout returns dispatcher wait timeout.
func (m *Manager) GetWaitTimeout() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.waitTimeout
}

// SetWaitTimeout sets dispatcher wait timeout.
func (m *Manager) SetWaitTimeout(timeout time.Duration) {
	if timeout <= 0 {
		timeout = time.Second
	}
	m.mu.Lock()
	m.waitTimeout = timeout
	m.mu.Unlock()
}

// AddEventListener adds listener for event name and returns registration handle.
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
	m.listeners[eventName] = append(currentListeners, listenerRegistration{
		handle:   handle,
		listener: listener,
	})
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

// RemoveEventListener removes listener by registration handle.
func (m *Manager) RemoveEventListener(handle ListenerHandle) error {
	if handle == InvalidListenerHandle {
		return errors.New("listener handle must not be invalid")
	}
	m.listenerOpMu.Lock()
	defer m.listenerOpMu.Unlock()

	var handler *gdsEventHandler
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
	removed := false
	for _, existing := range list {
		if existing.handle == handle {
			removed = true
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
		return nil
	}

	delete(m.listeners, eventName)
	handler = m.handlers[eventName]
	delete(m.handlers, eventName)
	m.mu.Unlock()

	if handler != nil {
		return handler.unregister(context.Background())
	}
	return nil
}

// WaitForEvent waits for one event occurrence for eventName.
func (m *Manager) WaitForEvent(ctx context.Context, eventName string, timeout time.Duration) (int, error) {
	listener := newOneTimeEventListener()
	handle, err := m.AddEventListener(eventName, listener)
	if err != nil {
		return -1, err
	}

	count, awaitErr := listener.awaitWithAsyncError(ctx, timeout, m.asyncErrorSignal(), m.LastError)
	removeErr := m.RemoveEventListener(handle)

	if awaitErr != nil {
		if removeErr != nil {
			return -1, errors.Join(awaitErr, removeErr)
		}
		return -1, awaitErr
	}
	if removeErr != nil {
		return -1, removeErr
	}
	// Keep wait semantics local to this wait operation: once this listener got
	// its event (or timeout), we don't fail the call because of unrelated
	// manager-wide async errors that may have happened earlier.
	return count, nil
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

	handler := &gdsEventHandler{
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
	defer m.mu.Unlock()

	eventName, exists := m.listenerEventName[handle]
	if !exists {
		return
	}

	list := m.listeners[eventName]
	for i, existing := range list {
		if existing.handle == handle {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}
	delete(m.listenerEventName, handle)
	if len(list) == 0 {
		delete(m.listeners, eventName)
		return
	}
	m.listeners[eventName] = list
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

func (m *Manager) dispatcherLoop(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		timeout := m.GetWaitTimeout()
		timer := time.NewTimer(timeout)
		select {
		case <-stop:
			timer.Stop()
			return
		case event, ok := <-m.eventQueue:
			timer.Stop()
			if !ok {
				return
			}
			m.notify(event)
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

func (m *Manager) currentConn() *fbconn.FbConn {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected || m.conn == nil || m.conn.IsClosed() {
		return nil
	}
	return m.conn.FbConn()
}

func (m *Manager) bindConnEventErrorHandler(conn *fbx.Conn) {
	if conn == nil || conn.IsClosed() {
		return
	}
	conn.FbConn().SetEventErrorHandler(m.onConnEventError)
}

func (m *Manager) unbindConnEventErrorHandler(conn *fbx.Conn) {
	if conn == nil || conn.IsClosed() {
		return
	}
	conn.FbConn().SetEventErrorHandler(nil)
}

func (m *Manager) onConnEventError(err error) {
	if err == nil {
		return
	}
	m.setAsyncError(fmt.Errorf("event channel failed: %w", err))
}

func (m *Manager) setAsyncError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	m.setAsyncErrorLocked(err)
	m.mu.Unlock()
}

func (m *Manager) asyncErrorSignal() <-chan struct{} {
	m.mu.Lock()
	ch := m.asyncErrSignal
	m.mu.Unlock()
	return ch
}

func (m *Manager) startConnLifecycleWatcher(conn *fbx.Conn, stop <-chan struct{}, generation uint64) {
	if conn == nil {
		return
	}
	cleanupDone := conn.FbConn().CleanupDone()
	go func() {
		select {
		case <-stop:
			return
		case <-cleanupDone:
		}

		// Explicit manager disconnect closes stop channel first; do not treat
		// that path as an asynchronous lifecycle failure.
		select {
		case <-stop:
			return
		default:
		}

		m.handleConnLifecycleTermination(conn, generation)
	}()
}

func (m *Manager) handleConnLifecycleTermination(conn *fbx.Conn, generation uint64) {
	m.mu.Lock()
	if generation != m.lifecycleWatchGen || !m.connected || m.conn != conn {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	lifecycleErr := errors.New("underlying connection detached")
	if err := m.disconnect(context.Background(), false, false, nil); err != nil {
		lifecycleErr = errors.Join(lifecycleErr, fmt.Errorf("manager disconnect failed after detaching: %w", err))
	}
	m.setAsyncError(lifecycleErr)
}

func (m *Manager) setAsyncErrorLocked(err error) {
	if err == nil || m.asyncErr != nil {
		return
	}
	m.asyncErr = err
	closeSignalIfOpen(m.asyncErrSignal)
}

func closeSignalIfOpen(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case <-ch:
		return
	default:
		close(ch)
	}
}

func (m *Manager) operationContext(parent context.Context) (context.Context, context.CancelFunc) {
	return contextWithBoundedTimeout(parent, managerOperationTimeout)
}

func (m *Manager) dispatcherStopContext(parent context.Context) (context.Context, context.CancelFunc) {
	return contextWithBoundedTimeout(parent, managerDispatcherShutdownTimeout)
}

func contextWithBoundedTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

type gdsEventHandler struct {
	manager   *Manager
	eventName string
	handle    *fbconn.EventHandle

	// mu protects state transitions and initialized flag.
	mu sync.Mutex
	// opMu serializes register/re-register/unregister network operations so state
	// transitions cannot overlap around server calls.
	opMu sync.Mutex

	initialized bool
	state       eventHandlerState
}

type eventHandlerState uint8

const (
	// eventHandlerStateCreated means handler exists locally but is not registered
	// on the server yet.
	eventHandlerStateCreated eventHandlerState = iota
	// eventHandlerStateRegistering means a queue/register call is in progress.
	eventHandlerStateRegistering
	// eventHandlerStateRegistered means server-side registration is active.
	eventHandlerStateRegistered
	// eventHandlerStateCancelling means cancel call is in progress.
	eventHandlerStateCancelling
	// eventHandlerStateCancelled means handler is permanently cancelled.
	eventHandlerStateCancelled
)

func (h *gdsEventHandler) register(ctx context.Context) error {
	h.opMu.Lock()
	defer h.opMu.Unlock()

	h.mu.Lock()
	switch h.state {
	case eventHandlerStateCancelled, eventHandlerStateCancelling:
		h.mu.Unlock()
		return errors.New("trying to register cancelled event handler")
	default:
	}
	prevState := h.state
	h.state = eventHandlerStateRegistering
	handle := h.handle
	h.mu.Unlock()

	conn := h.manager.currentConn()
	if conn == nil {
		h.mu.Lock()
		if h.state == eventHandlerStateRegistering {
			h.state = prevState
		}
		h.mu.Unlock()
		return errors.New("manager disconnected")
	}
	opCtx, cancel := h.manager.operationContext(ctx)
	defer cancel()
	err := conn.QueueEvent(opCtx, handle)

	h.mu.Lock()
	// register can fail; on failure we rollback to previous state instead of
	// pretending cancellation happened.
	if h.state == eventHandlerStateRegistering {
		if err == nil {
			h.state = eventHandlerStateRegistered
		} else {
			h.state = prevState
		}
	}
	h.mu.Unlock()
	return err
}

func (h *gdsEventHandler) unregister(ctx context.Context) error {
	return h.unregisterOnConn(ctx, h.manager.currentConn())
}

func (h *gdsEventHandler) unregisterOnConn(ctx context.Context, conn *fbconn.FbConn) error {
	h.opMu.Lock()
	defer h.opMu.Unlock()

	h.mu.Lock()
	switch h.state {
	case eventHandlerStateCancelled, eventHandlerStateCancelling:
		h.mu.Unlock()
		return nil
	default:
	}
	prevState := h.state
	// Transition to cancelling before network I/O; if cancel fails we roll back
	// to previous state (typically registered), so failed cancel does not leave
	// handler in a permanently cancelled local state.
	h.state = eventHandlerStateCancelling
	handle := h.handle
	h.mu.Unlock()

	if conn == nil {
		// Connection already unavailable: treat as best-effort local cancellation.
		h.mu.Lock()
		if h.state == eventHandlerStateCancelling {
			h.state = eventHandlerStateCancelled
		}
		h.mu.Unlock()
		return nil
	}
	opCtx, cancel := h.manager.operationContext(ctx)
	defer cancel()
	err := conn.CancelEvent(opCtx, handle)

	h.mu.Lock()
	if h.state == eventHandlerStateCancelling {
		if err == nil {
			h.state = eventHandlerStateCancelled
		} else {
			h.state = prevState
		}
	}
	h.mu.Unlock()
	return err
}

func (h *gdsEventHandler) EventOccurred(handle *fbconn.EventHandle) {
	if h.isStopping() {
		return
	}

	conn := h.manager.currentConn()
	if conn == nil {
		return
	}

	if err := conn.CountEvents(handle); err != nil {
		h.manager.setAsyncError(fmt.Errorf("count events failed for %q: %w", h.eventName, err))
		return
	}

	h.mu.Lock()
	if h.state == eventHandlerStateCancelling || h.state == eventHandlerStateCancelled {
		h.mu.Unlock()
		return
	}
	emitEvent := h.initialized
	h.initialized = true
	h.mu.Unlock()

	if h.isStopping() {
		return
	}
	if err := h.reRegister(handle); err != nil {
		h.manager.setAsyncError(fmt.Errorf("re-register failed for %q: %w", h.eventName, err))
	}
	if !emitEvent || h.isStopping() {
		return
	}
	// Keep server-side subscription active before potentially blocking on
	// manager event queue backpressure.
	h.manager.enqueueEvent(DatabaseEvent{
		EventName:  handle.EventName(),
		EventCount: handle.EventCount(),
	})
}

func (h *gdsEventHandler) reRegister(handle *fbconn.EventHandle) error {
	const maxAttempts = 3
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		h.opMu.Lock()

		h.mu.Lock()
		if h.state == eventHandlerStateCancelling || h.state == eventHandlerStateCancelled {
			h.mu.Unlock()
			h.opMu.Unlock()
			return nil
		}
		prevState := h.state
		h.state = eventHandlerStateRegistering
		h.mu.Unlock()

		conn := h.manager.currentConn()
		if conn == nil {
			h.mu.Lock()
			if h.state == eventHandlerStateRegistering {
				h.state = prevState
			}
			h.mu.Unlock()
			h.opMu.Unlock()
			return errors.New("manager disconnected")
		}

		opCtx, cancel := h.manager.operationContext(context.Background())
		err := conn.QueueEvent(opCtx, handle)
		cancel()

		h.mu.Lock()
		if h.state == eventHandlerStateRegistering {
			if err == nil {
				h.state = eventHandlerStateRegistered
			} else {
				h.state = prevState
			}
		}
		h.mu.Unlock()

		if err != nil {
			lastErr = err
			h.opMu.Unlock()
		} else {
			h.opMu.Unlock()
			return nil
		}

		time.Sleep(time.Duration(25*(attempt+1)) * time.Millisecond)
	}

	if lastErr == nil {
		return errors.New("re-register failed for unknown reason")
	}
	return lastErr
}

func (h *gdsEventHandler) isStopping() bool {
	h.mu.Lock()
	state := h.state
	h.mu.Unlock()
	return state == eventHandlerStateCancelling || state == eventHandlerStateCancelled
}
