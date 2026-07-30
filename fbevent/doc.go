/*
Package fbevent provides high-level Firebird and RedDatabase event listening.

The package wraps the low-level Firebird event API with a [Manager] that handles
event registration, dispatch, and listener cleanup. A manager can either open
and own its database connection or listen through an existing [fbx.Conn].

# Creating a Manager

Use [Connect], [ConnectWithOptions], or [ConnectConfig] when the event manager
should own its connection:

	manager, err := fbevent.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer manager.Close(context.Background())

Use [ConnectFor] to listen through an existing [fbx.Conn]. Closing a manager
created by [ConnectFor] does not close the underlying connection.

# Listening with Callbacks

[Manager.AddEventListener] registers an [EventListener] for an event name on a
connected manager. The returned [ListenerCloser] removes that registration.

	listenerCloser, err := manager.AddEventListener("orders_changed", fbevent.EventListenerFunc(func(event fbevent.DatabaseEvent) {
		fmt.Printf("%s: %d\n", event.EventName, event.EventCount)
	}))
	if err != nil {
		return err
	}
	defer listenerCloser.Close()

# Channel Subscriptions

[Manager.Subscribe] registers a listener and exposes matching events through a
channel. Closing the returned [EventSubscription] removes the listener and closes
the channel.

	subscription, err := manager.Subscribe("orders_changed", 16)
	if err != nil {
		return err
	}
	defer subscription.Close()

	select {
	case event := <-subscription.Events():
		fmt.Println(event.EventName, event.EventCount)
	case <-ctx.Done():
		return ctx.Err()
	}

# Waiting for a Single Event

[Manager.WaitForEvent] registers a temporary listener, waits for one event, and
then removes the listener. A zero timeout waits until ctx is canceled, an event
arrives, or an event manager error occurs. A positive timeout returns count -1
and a nil error when no event arrives before the timeout.

	count, err := manager.WaitForEvent(ctx, "orders_changed", 5*time.Second)
	if err != nil {
		return err
	}
	if count >= 0 {
		fmt.Println("event count:", count)
	}

# Notification Style

[Manager.Listen] and [Manager.WaitForNotification] provide a single notification
queue for code that prefers a Listen/WaitForNotification workflow.

	if err := manager.Listen(ctx, "orders_changed"); err != nil {
		return err
	}
	notification, err := manager.WaitForNotification(ctx)
	if err != nil {
		return err
	}
	fmt.Println(notification.Channel, notification.Count)

# Delivery and Errors

Events are delivered sequentially for each manager. Slow callbacks can delay
later callbacks and subscriptions registered on the same manager, so callbacks
should return quickly or hand work off to another goroutine.

If the underlying event channel fails, waiting operations return the asynchronous
error, and [Manager.LastError] stores the first such error. Call [Manager.Close]
or [Manager.Disconnect] when event processing is no longer needed.
*/
package fbevent
