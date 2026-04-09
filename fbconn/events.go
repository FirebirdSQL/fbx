package fbconn

import (
	"context"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
)

// CreateEventHandle creates an event handle for a Firebird event name.
func (fbConn *FbConn) CreateEventHandle(eventName string, handler EventHandler) (*EventHandle, error) {
	if fbConn == nil {
		return nil, errors.New("fbConn is nil")
	}
	return newEventHandle(eventName, handler, fbConn.Encoding)
}

// QueueEvent queues an event handle on the server.
func (fbConn *FbConn) QueueEvent(ctx context.Context, handle *EventHandle) error {
	if handle == nil {
		return errors.New("event handle is nil")
	}
	if eventErr := fbConn.takeEventError(); eventErr != nil {
		return eventErr
	}
	if _, err := fbConn.ensureEventChannel(ctx); err != nil {
		return err
	}
	if err := fbConn.lock(); err != nil {
		return err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	eventID := handle.assignNewLocalID()
	fbConn.registerEventHandle(eventID, handle)

	msg := &fbproto.OpQueEvents{
		DatabaseHandle: 0,
		EventItems:     handle.toEventBlock(),
		EventID:        eventID,
	}
	if err := fbConn.client.WriteMessage(msg); err != nil {
		fbConn.unregisterEventHandle(eventID)
		handle.clearLocalID()
		fbConn.asyncClose()
		return err
	}

	if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.unregisterEventHandle(eventID)
		handle.clearLocalID()
		fbConn.asyncClose()
		return err
	}

	response, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.unregisterEventHandle(eventID)
		handle.clearLocalID()
		fbConn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}

	switch m := response.(type) {
	case *fbproto.OpResponse:
		return nil
	case *fbproto.ErrorResponse:
		fbConn.unregisterEventHandle(eventID)
		handle.clearLocalID()
		return ErrorResponseToFbError(m)
	default:
		fbConn.unregisterEventHandle(eventID)
		handle.clearLocalID()
		return fmt.Errorf("unexpected response message type %T while queueing event", m)
	}
}

// CancelEvent cancels an event handle on the server.
func (fbConn *FbConn) CancelEvent(ctx context.Context, handle *EventHandle) error {
	if handle == nil {
		return errors.New("event handle is nil")
	}
	if eventErr := fbConn.takeEventError(); eventErr != nil {
		return eventErr
	}
	if err := fbConn.lock(); err != nil {
		return err
	}
	defer fbConn.unlock()

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	eventID := handle.EventID()
	if eventID == 0 {
		return nil
	}

	fbConn.eventMu.Lock()
	channel := fbConn.eventChannel
	fbConn.eventMu.Unlock()
	if channel == nil {
		fbConn.unregisterEventHandle(eventID)
		handle.clearLocalID()
		return errors.New("event channel is not connected")
	}

	msg := &fbproto.OpCancelEvents{
		DatabaseHandle: 0,
		EventID:        eventID,
	}
	if err := fbConn.client.WriteMessage(msg); err != nil {
		fbConn.asyncClose()
		return err
	}

	if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		fbConn.asyncClose()
		return err
	}

	response, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return normalizeTimeoutError(ctx, err)
	}

	switch m := response.(type) {
	case *fbproto.OpResponse:
		fbConn.unregisterEventHandle(eventID)
		handle.clearLocalID()
		return nil
	case *fbproto.ErrorResponse:
		return ErrorResponseToFbError(m)
	default:
		return fmt.Errorf("unexpected response message type %T while canceling event", m)
	}
}

// CountEvents calculates event count delta for an event handle.
func (fbConn *FbConn) CountEvents(handle *EventHandle) error {
	if handle == nil {
		return errors.New("event handle is nil")
	}
	handle.calculateCount()
	return nil
}
