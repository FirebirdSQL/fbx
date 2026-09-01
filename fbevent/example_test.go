package fbevent_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbevent"
)

func postEvent(ctx context.Context, conn *fbx.Conn, eventName string) error {
	escaped := strings.ReplaceAll(eventName, "'", "''")
	_, err := conn.Exec(ctx, "execute block as begin post_event '"+escaped+"'; end")
	return err
}

func ExampleManager_WaitForEvent() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	manager, err := fbevent.Connect(ctx, dsn)
	if err != nil {
		return
	}
	defer func() { _ = manager.Close(context.Background()) }()

	notifier, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = notifier.Close(ctx) }()

	eventName := "FBX_EXAMPLE_WAIT"
	postErr := make(chan error, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		postErr <- postEvent(ctx, notifier, eventName)
	}()

	count, err := manager.WaitForEvent(ctx, eventName, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}
	if count == -1 {
		return
	}
	if err := <-postErr; err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s count delta: %d\n", eventName, count)

	// Output:
	// FBX_EXAMPLE_WAIT count delta: 1
}

func ExampleManager_Subscribe() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	manager, err := fbevent.Connect(ctx, dsn)
	if err != nil {
		return
	}
	defer func() { _ = manager.Close(context.Background()) }()

	notifier, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = notifier.Close(ctx) }()

	eventName := "FBX_EXAMPLE_SUBSCRIBE"
	sub, err := manager.Subscribe(eventName, 16)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = sub.Close() }()

	if err := postEvent(ctx, notifier, eventName); err != nil {
		fmt.Println(err)
		return
	}

	select {
	case event := <-sub.Events():
		fmt.Printf("%s count delta: %d\n", event.EventName, event.EventCount)
	case <-time.After(5 * time.Second):
		fmt.Println("timeout")
	}

	// Output:
	// FBX_EXAMPLE_SUBSCRIBE count delta: 1
}

func ExampleManager_WaitForNotification() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	manager, err := fbevent.Connect(ctx, dsn)
	if err != nil {
		return
	}
	defer func() { _ = manager.Close(context.Background()) }()

	notifier, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = notifier.Close(ctx) }()

	eventName := "FBX_EXAMPLE_NOTIFICATION"
	if err := manager.Listen(ctx, eventName); err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = manager.Unlisten(eventName) }()

	if err := postEvent(ctx, notifier, eventName); err != nil {
		fmt.Println(err)
		return
	}

	notification, err := manager.WaitForNotification(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s count delta: %d\n", notification.Channel, notification.Count)

	// Output:
	// FBX_EXAMPLE_NOTIFICATION count delta: 1
}

func ExampleManager_AddEventListener() {
	ctx := context.Background()
	dsn := "firebird://sysdba:masterkey@localhost:3050/employee"

	manager, err := fbevent.Connect(ctx, dsn)
	if err != nil {
		return
	}
	defer func() { _ = manager.Close(context.Background()) }()

	notifier, err := fbx.Connect(ctx, dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = notifier.Close(ctx) }()

	events := make(chan fbevent.DatabaseEvent, 1)
	listener := fbevent.EventListenerFunc(func(event fbevent.DatabaseEvent) {
		events <- event
	})

	eventName := "FBX_EXAMPLE_LISTENER"
	listenerCloser, err := manager.AddEventListener(eventName, listener)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = listenerCloser.Close() }()

	if err := postEvent(ctx, notifier, eventName); err != nil {
		fmt.Println(err)
		return
	}

	select {
	case event := <-events:
		fmt.Printf("%s count delta: %d\n", event.EventName, event.EventCount)
	case <-time.After(5 * time.Second):
		fmt.Println("timeout")
	}

	// Output:
	// FBX_EXAMPLE_LISTENER count delta: 1
}
