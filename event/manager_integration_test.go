package event_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/event"
	"rdb.red-soft.ru/fbx/fbconn"
)

var eventNameCounter atomic.Uint64

func TestManagerWaitForEventTimeout(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)

	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	eventName := nextEventName("TIMEOUT")
	count, err := manager.WaitForEvent(ctx, eventName, 250*time.Millisecond)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)
	require.Equal(t, -1, count)
}

func TestManagerWaitForEventTriggered(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)

	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	notifier, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, notifier.Close(context.Background()))
	}()

	eventA := nextEventName("A")
	triggerErrA := make(chan error, 1)
	go func() {
		time.Sleep(200 * time.Millisecond)
		triggerErrA <- postEvents(ctx, notifier, eventA, eventA)
	}()

	countA, err := manager.WaitForEvent(ctx, eventA, 5*time.Second)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)
	require.Equal(t, 2, countA)
	require.NoError(t, <-triggerErrA)

	eventB := nextEventName("B")
	triggerErrB := make(chan error, 1)
	go func() {
		time.Sleep(200 * time.Millisecond)
		triggerErrB <- postEvents(ctx, notifier, eventB)
	}()

	countB, err := manager.WaitForEvent(ctx, eventB, 5*time.Second)
	require.NoError(t, err)
	require.Equal(t, 1, countB)
	require.NoError(t, <-triggerErrB)
}

func TestManagerAddRemoveEventListener(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	notifier, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, notifier.Close(context.Background()))
	}()

	eventName := nextEventName("LISTENER")
	events := make(chan event.DatabaseEvent, 4)
	listener := event.EventListenerFunc(func(e event.DatabaseEvent) {
		events <- e
	})

	listenerHandle, err := manager.AddEventListener(eventName, listener)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)

	require.NoError(t, postEvents(ctx, notifier, eventName))
	got := waitEvent(t, events, 5*time.Second)
	require.Equal(t, eventName, got.EventName)
	require.Equal(t, 1, got.EventCount)

	require.NoError(t, manager.RemoveEventListener(listenerHandle))
	require.NoError(t, postEvents(ctx, notifier, eventName))

	select {
	case unexpected := <-events:
		t.Fatalf("unexpected event after listener removal: %+v", unexpected)
	case <-time.After(500 * time.Millisecond):
	}
}

func TestManagerMultipleListeners(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	notifier, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, notifier.Close(context.Background()))
	}()

	eventName := nextEventName("MULTI")
	listenerAEvents := make(chan event.DatabaseEvent, 2)
	listenerBEvents := make(chan event.DatabaseEvent, 2)

	listenerA := event.EventListenerFunc(func(e event.DatabaseEvent) {
		listenerAEvents <- e
	})
	listenerB := event.EventListenerFunc(func(e event.DatabaseEvent) {
		listenerBEvents <- e
	})

	_, err = manager.AddEventListener(eventName, listenerA)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)
	_, err = manager.AddEventListener(eventName, listenerB)
	require.NoError(t, err)

	require.NoError(t, postEvents(ctx, notifier, eventName))

	receivedA := waitEvent(t, listenerAEvents, 5*time.Second)
	receivedB := waitEvent(t, listenerBEvents, 5*time.Second)
	require.Equal(t, eventName, receivedA.EventName)
	require.Equal(t, 1, receivedA.EventCount)
	require.Equal(t, eventName, receivedB.EventName)
	require.Equal(t, 1, receivedB.EventCount)
}

func TestManagerCreateForDisconnectsWhenConnectionClosed(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)

	manager := event.CreateFor(conn)
	require.NoError(t, manager.Connect(ctx))
	require.True(t, manager.IsConnected())

	require.NoError(t, conn.Close(ctx))
	require.Eventually(t, func() bool { return !manager.IsConnected() }, 2*time.Second, 50*time.Millisecond)
	require.NoError(t, manager.Close(context.Background()))
}

func TestManagerLifecycleDefaultConnectDisconnectReconnect(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)

	require.NoError(t, manager.Connect(ctx))
	require.True(t, manager.IsConnected())

	err = manager.Connect(ctx)
	require.ErrorContains(t, err, "connect called while already connected")

	require.NoError(t, manager.Disconnect(ctx))
	require.False(t, manager.IsConnected())

	err = manager.Disconnect(ctx)
	require.ErrorContains(t, err, "disconnect called while not connected")

	require.NoError(t, manager.Close(ctx))

	require.NoError(t, manager.Connect(ctx))
	require.True(t, manager.IsConnected())
	require.NoError(t, manager.Close(ctx))
}

func TestManagerCreateForCloseKeepsUnderlyingConnectionAlive(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close(context.Background()))
	}()

	manager := event.CreateFor(conn)
	require.NoError(t, manager.Connect(ctx))
	require.True(t, manager.IsConnected())

	require.NoError(t, manager.Close(ctx))
	require.False(t, manager.IsConnected())
	require.False(t, conn.IsClosed())
	require.NoError(t, conn.Ping(ctx))
}

func TestManagerWaitForEventContextDeadlineExceeded(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	waitCtx, waitCancel := context.WithTimeout(ctx, 350*time.Millisecond)
	defer waitCancel()

	count, err := manager.WaitForEvent(waitCtx, nextEventName("CTX_TIMEOUT"), 0)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.Equal(t, -1, count)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestManagerDuplicateSameListenerDeliveredPerRegistration(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	notifier, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, notifier.Close(context.Background()))
	}()

	eventName := nextEventName("DEDUP")
	var delivered atomic.Int64
	listener := event.EventListenerFunc(func(e event.DatabaseEvent) {
		delivered.Add(int64(normalizeIntegrationEventCount(e.EventCount)))
	})
	asyncErrs := make(chan error, 1)

	_, err = manager.AddEventListener(eventName, listener)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)
	_, err = manager.AddEventListener(eventName, listener)
	require.NoError(t, err)

	primeTimeout := 8 * time.Second
	require.NoError(t, primeEventCounterIntegration(ctx, notifier, eventName, &delivered, primeTimeout, asyncErrs, manager))
	base := delivered.Load()

	const posted = int64(5)
	const registrationCount = int64(2)
	require.NoError(t, postEventCountIntegration(ctx, notifier, eventName, int(posted)))
	require.NoError(t, waitForCounterAtLeastIntegration(ctx, &delivered, base+posted*registrationCount, 10*time.Second, asyncErrs, manager, "duplicate registration"))

	select {
	case <-time.After(300 * time.Millisecond):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	require.Equal(t, posted*registrationCount, delivered.Load()-base)
}

func TestManagerDifferentClosureListenersSelectiveRemove(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	notifier, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, notifier.Close(context.Background()))
	}()

	eventName := nextEventName("CLOSURE")
	var deliveredA atomic.Int64
	var deliveredB atomic.Int64
	asyncErrs := make(chan error, 1)

	makeListener := func(counter *atomic.Int64) event.EventListenerFunc {
		return event.EventListenerFunc(func(e event.DatabaseEvent) {
			counter.Add(int64(normalizeIntegrationEventCount(e.EventCount)))
		})
	}
	listenerA := makeListener(&deliveredA)
	listenerB := makeListener(&deliveredB)

	listenerAHandle, err := manager.AddEventListener(eventName, listenerA)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)
	_, err = manager.AddEventListener(eventName, listenerB)
	require.NoError(t, err)

	primeTimeout := 8 * time.Second
	require.NoError(t, primeEventCounterIntegration(ctx, notifier, eventName, &deliveredA, primeTimeout, asyncErrs, manager))
	require.NoError(t, primeEventCounterIntegration(ctx, notifier, eventName, &deliveredB, primeTimeout, asyncErrs, manager))

	baseA := deliveredA.Load()
	baseB := deliveredB.Load()

	const phase1Posted = int64(3)
	require.NoError(t, postEventCountIntegration(ctx, notifier, eventName, int(phase1Posted)))
	require.NoError(t, waitForCounterAtLeastIntegration(ctx, &deliveredA, baseA+phase1Posted, 10*time.Second, asyncErrs, manager, "closure phase1 A"))
	require.NoError(t, waitForCounterAtLeastIntegration(ctx, &deliveredB, baseB+phase1Posted, 10*time.Second, asyncErrs, manager, "closure phase1 B"))

	select {
	case <-time.After(300 * time.Millisecond):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	require.Equal(t, phase1Posted, deliveredA.Load()-baseA)
	require.Equal(t, phase1Posted, deliveredB.Load()-baseB)

	require.NoError(t, manager.RemoveEventListener(listenerAHandle))

	const phase2Posted = int64(3)
	require.NoError(t, postEventCountIntegration(ctx, notifier, eventName, int(phase2Posted)))
	require.NoError(t, waitForCounterAtLeastIntegration(ctx, &deliveredB, baseB+phase1Posted+phase2Posted, 10*time.Second, asyncErrs, manager, "closure phase2 B"))

	select {
	case <-time.After(300 * time.Millisecond):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	require.Equal(t, phase1Posted, deliveredA.Load()-baseA)
	require.Equal(t, phase1Posted+phase2Posted, deliveredB.Load()-baseB)
}

func TestManagerWorkflowScenarioFromCmdEvents(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	producer, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, producer.Close(context.Background()))
	}()

	billingPoster, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, billingPoster.Close(context.Background()))
	}()

	shippingPoster, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, shippingPoster.Close(context.Background()))
	}()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	prefix := nextEventName("REAL_WORKFLOW")
	orderCreated := prefix + "_ORDER_CREATED"
	paymentConfirmed := prefix + "_PAYMENT_CONFIRMED"
	orderShipped := prefix + "_ORDER_SHIPPED"

	var createdCount atomic.Int64
	var paidCount atomic.Int64
	var shippedCount atomic.Int64
	var auditCount atomic.Int64

	asyncErrs := make(chan error, 1)
	reportAsyncErr := func(err error) {
		if err == nil {
			return
		}
		select {
		case asyncErrs <- err:
		default:
		}
	}

	createdListener := event.EventListenerFunc(func(dbEvent event.DatabaseEvent) {
		eventDelta := normalizeIntegrationEventCount(dbEvent.EventCount)
		createdCount.Add(int64(eventDelta))

		postCtx, postCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer postCancel()
		reportAsyncErr(postEventCountIntegration(postCtx, billingPoster, paymentConfirmed, eventDelta))
	})

	paymentListener := event.EventListenerFunc(func(dbEvent event.DatabaseEvent) {
		eventDelta := normalizeIntegrationEventCount(dbEvent.EventCount)
		paidCount.Add(int64(eventDelta))

		postCtx, postCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer postCancel()
		reportAsyncErr(postEventCountIntegration(postCtx, shippingPoster, orderShipped, eventDelta))
	})

	auditListener := event.EventListenerFunc(func(dbEvent event.DatabaseEvent) {
		eventDelta := normalizeIntegrationEventCount(dbEvent.EventCount)
		auditCount.Add(int64(eventDelta))
	})

	shippedListener := event.EventListenerFunc(func(dbEvent event.DatabaseEvent) {
		eventDelta := normalizeIntegrationEventCount(dbEvent.EventCount)
		shippedCount.Add(int64(eventDelta))
	})

	_, err = manager.AddEventListener(orderCreated, createdListener)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)

	_, err = manager.AddEventListener(paymentConfirmed, paymentListener)
	require.NoError(t, err)
	auditHandle, err := manager.AddEventListener(paymentConfirmed, auditListener)
	require.NoError(t, err)
	_, err = manager.AddEventListener(orderShipped, shippedListener)
	require.NoError(t, err)

	primeTimeout := 8 * time.Second
	require.NoError(t, primeEventCounterIntegration(ctx, producer, orderCreated, &createdCount, primeTimeout, asyncErrs, manager))
	require.NoError(t, primeEventCounterIntegration(ctx, producer, paymentConfirmed, &paidCount, primeTimeout, asyncErrs, manager))
	require.NoError(t, primeEventCounterIntegration(ctx, producer, orderShipped, &shippedCount, primeTimeout, asyncErrs, manager))

	baseCreated := createdCount.Load()
	basePaid := paidCount.Load()
	baseShipped := shippedCount.Load()
	baseAudit := auditCount.Load()

	const ordersPhase1 = 4
	const ordersPhase2 = 2
	require.NoError(t, emitOrdersIntegration(ctx, producer, orderCreated, ordersPhase1, 2, 100*time.Millisecond))

	phase1Target := baseShipped + ordersPhase1
	require.NoError(t, waitForCounterAtLeastIntegration(ctx, &shippedCount, phase1Target, 12*time.Second, asyncErrs, manager, "phase1 shipped"))

	auditBeforeRemove := auditCount.Load()
	require.NoError(t, manager.RemoveEventListener(auditHandle))

	require.NoError(t, emitOrdersIntegration(ctx, producer, orderCreated, ordersPhase2, 2, 100*time.Millisecond))

	phase2Target := phase1Target + ordersPhase2
	require.NoError(t, waitForCounterAtLeastIntegration(ctx, &shippedCount, phase2Target, 12*time.Second, asyncErrs, manager, "phase2 shipped"))

	select {
	case <-time.After(250 * time.Millisecond):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	if err := readAsyncErrIntegration(asyncErrs); err != nil {
		t.Fatalf("async listener error: %v", err)
	}
	if err := manager.LastError(); err != nil {
		t.Fatalf("manager async error: %v", err)
	}

	totalExpected := int64(ordersPhase1 + ordersPhase2)
	totalCreated := createdCount.Load() - baseCreated
	totalPaid := paidCount.Load() - basePaid
	totalShipped := shippedCount.Load() - baseShipped
	totalAudit := auditCount.Load() - baseAudit

	require.GreaterOrEqual(t, totalCreated, totalExpected)
	require.GreaterOrEqual(t, totalPaid, totalExpected)
	require.GreaterOrEqual(t, totalShipped, totalExpected)
	require.Equal(t, auditBeforeRemove, auditCount.Load())
	require.GreaterOrEqual(t, totalAudit, int64(0))
}

func TestManagerWaitForEventOneShotFromCmdEvents(t *testing.T) {
	connString := requireIntegrationConnString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	manager, err := event.NewManager(connString)
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	defer func() {
		require.NoError(t, manager.Close(context.Background()))
	}()

	poster, err := fbx.Connect(ctx, connString)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, poster.Close(context.Background()))
	}()

	eventName := nextEventName("WAIT_ONE_SHOT")
	postDone := make(chan error, 1)

	go func() {
		time.Sleep(200 * time.Millisecond)
		if err := postEventCountIntegration(ctx, poster, eventName, 1); err != nil {
			postDone <- err
			return
		}
		time.Sleep(200 * time.Millisecond)
		postDone <- postEventCountIntegration(ctx, poster, eventName, 1)
	}()

	count, err := manager.WaitForEvent(ctx, eventName, 8*time.Second)
	if skipIfAuxEventPortUnavailable(t, err) {
		return
	}
	require.NoError(t, err)
	require.NoError(t, <-postDone)
	require.Greater(t, count, 0)
}

func requireIntegrationConnString(t testing.TB) string {
	t.Helper()
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "FBX_TEST_DATABASE")
	}
	return connString
}

func nextEventName(prefix string) string {
	id := eventNameCounter.Add(1)
	return fmt.Sprintf("FBX_EVT_%s_%d", prefix, id)
}

func postEvents(ctx context.Context, conn *fbx.Conn, eventNames ...string) error {
	var sb strings.Builder
	sb.WriteString("execute block as begin ")
	for _, eventName := range eventNames {
		escaped := strings.ReplaceAll(eventName, "'", "''")
		sb.WriteString("post_event '")
		sb.WriteString(escaped)
		sb.WriteString("'; ")
	}
	sb.WriteString("end")

	_, err := conn.Exec(ctx, sb.String())
	return err
}

func waitEvent(t testing.TB, events <-chan event.DatabaseEvent, timeout time.Duration) event.DatabaseEvent {
	t.Helper()
	select {
	case received := <-events:
		return received
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for event (%s)", timeout)
		return event.DatabaseEvent{}
	}
}

func normalizeIntegrationEventCount(eventCount int) int {
	if eventCount <= 0 {
		return 1
	}
	return eventCount
}

func emitOrdersIntegration(
	ctx context.Context,
	conn *fbx.Conn,
	eventName string,
	total int64,
	batchSize int64,
	delay time.Duration,
) error {
	remaining := total
	for remaining > 0 {
		n := batchSize
		if remaining < n {
			n = remaining
		}

		if err := postEventCountIntegration(ctx, conn, eventName, int(n)); err != nil {
			return err
		}
		remaining -= n

		if remaining > 0 && delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func postEventCountIntegration(ctx context.Context, conn *fbx.Conn, eventName string, count int) error {
	if count <= 0 {
		return nil
	}

	escaped := strings.ReplaceAll(eventName, "'", "''")
	const maxPerStatement = 64
	remaining := count

	for remaining > 0 {
		n := maxPerStatement
		if remaining < n {
			n = remaining
		}

		var sb strings.Builder
		sb.WriteString("execute block as begin ")
		for i := 0; i < n; i++ {
			sb.WriteString("post_event '")
			sb.WriteString(escaped)
			sb.WriteString("'; ")
		}
		sb.WriteString("end")

		if _, err := conn.Exec(ctx, sb.String()); err != nil {
			return err
		}
		remaining -= n
	}

	return nil
}

func readAsyncErrIntegration(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	default:
		return nil
	}
}

func primeEventCounterIntegration(
	ctx context.Context,
	conn *fbx.Conn,
	eventName string,
	counter *atomic.Int64,
	timeout time.Duration,
	asyncErrs <-chan error,
	manager *event.Manager,
) error {
	start := counter.Load()
	deadline := time.Now().Add(timeout)

	for counter.Load() <= start {
		if err := readAsyncErrIntegration(asyncErrs); err != nil {
			return err
		}
		if err := manager.LastError(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("failed to prime event counter for %q (stayed at %d)", eventName, start)
		}

		if err := postEventCountIntegration(ctx, conn, eventName, 1); err != nil {
			return err
		}

		waitUntil := time.Now().Add(350 * time.Millisecond)
		for counter.Load() <= start && time.Now().Before(waitUntil) {
			if err := readAsyncErrIntegration(asyncErrs); err != nil {
				return err
			}
			if err := manager.LastError(); err != nil {
				return err
			}

			select {
			case <-time.After(25 * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func waitForCounterAtLeastIntegration(
	ctx context.Context,
	counter *atomic.Int64,
	target int64,
	timeout time.Duration,
	asyncErrs <-chan error,
	manager *event.Manager,
	label string,
) error {
	if counter.Load() >= target {
		return nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := readAsyncErrIntegration(asyncErrs); err != nil {
			return err
		}
		if err := manager.LastError(); err != nil {
			return fmt.Errorf("manager async error while waiting for %s: %w", label, err)
		}
		if counter.Load() >= target {
			return nil
		}

		select {
		case <-ticker.C:
		case <-timer.C:
			return fmt.Errorf("wait timeout for %s (current=%d target=%d)", label, counter.Load(), target)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func skipIfAuxEventPortUnavailable(t testing.TB, err error) bool {
	t.Helper()
	if err == nil {
		return false
	}

	var fbErr *fbconn.FbError
	if !errors.As(err, &fbErr) {
		return false
	}

	for _, gdsCode := range fbErr.GdsCodes {
		switch gdsCode {
		case 335544724, 335544725, 335544975:
			t.Skipf("skipping due to unavailable auxiliary event port (gds code %d): %v", gdsCode, err)
			return true
		}
	}
	return false
}
