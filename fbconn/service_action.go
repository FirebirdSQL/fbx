package fbconn

import (
	"context"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
)

var errServiceActionActive = errors.New("a service action is already active")

func (c *ServiceConn) startServiceAction(ctx context.Context, actionSPB []byte) error {
	if c == nil || c.conn == nil {
		return errors.New("service connection is nil")
	}
	if c.actionActive {
		return errServiceActionActive
	}

	_, err := c.requestService(ctx, &fbproto.OpServiceStart{
		ServiceHandle: c.conn.attachmentHandle,
		ActionSPB:     actionSPB,
	})
	if err != nil {
		return fmt.Errorf("start service action: %w", err)
	}
	c.actionActive = true
	return nil
}

func (c *ServiceConn) queryService(
	ctx context.Context,
	sendItems []byte,
	receiveItems []byte,
	bufferLength int,
) ([]byte, error) {
	if c == nil || c.conn == nil {
		return nil, errors.New("service connection is nil")
	}
	if !c.actionActive {
		return nil, errors.New("no service action is active")
	}

	response, err := c.requestService(ctx, &fbproto.OpServiceInfo{
		ServiceHandle: c.conn.attachmentHandle,
		SendItems:     sendItems,
		ReceiveItems:  receiveItems,
		BufferLength:  bufferLength,
	})
	if err != nil {
		return nil, fmt.Errorf("query service action: %w", err)
	}
	return append([]byte(nil), response.Data...), nil
}

func (c *ServiceConn) requestService(ctx context.Context, request fbproto.ClientMessage) (*fbproto.OpResponse, error) {
	fbConn := c.conn
	if err := fbConn.lock(); err != nil {
		return nil, err
	}
	defer fbConn.unlock()

	if err := fbConn.beforeRequest(); err != nil {
		return nil, err
	}

	if ctx != context.Background() {
		select {
		case <-ctx.Done():
			return nil, newContextAlreadyDoneError(ctx)
		default:
		}
		fbConn.contextWatcher.Watch(ctx)
		defer fbConn.contextWatcher.Unwatch()
	}

	if err := fbConn.withWriteLockAndFlush(func() error {
		return fbConn.client.WriteMessage(request)
	}); err != nil {
		fbConn.asyncClose()
		return nil, normalizeTimeoutError(ctx, err)
	}

	message, err := fbConn.receiveMessage()
	if err != nil {
		fbConn.asyncClose()
		return nil, normalizeTimeoutError(ctx, err)
	}

	switch message := message.(type) {
	case *fbproto.OpResponse:
		return message, nil
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(message)
	default:
		fbConn.asyncClose()
		return nil, fmt.Errorf("received unexpected service response: %v", message)
	}
}
