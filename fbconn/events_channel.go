package fbconn

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/vax"
)

type eventChannel struct {
	conn      net.Conn
	closeOnce sync.Once
}

func newEventChannel(conn net.Conn) *eventChannel {
	return &eventChannel{conn: conn}
}

func (ch *eventChannel) close() error {
	if ch == nil || ch.conn == nil {
		return nil
	}
	var closeErr error
	ch.closeOnce.Do(func() {
		closeErr = ch.conn.Close()
	})
	return closeErr
}

func (fbConn *FbConn) ensureEventChannel(ctx context.Context) (*eventChannel, error) {
	fbConn.eventMu.Lock()
	channel := fbConn.eventChannel
	fbConn.eventMu.Unlock()
	if channel != nil {
		return channel, nil
	}

	if err := fbConn.client.WriteMessage(&fbproto.OpConnectRequest{}); err != nil {
		return nil, err
	}
	if err := fbConn.flushWithPotentialWriteReadDeadlock(); err != nil {
		return nil, err
	}

	msg, err := fbConn.receiveMessage()
	if err != nil {
		return nil, normalizeTimeoutError(ctx, err)
	}

	var auxPort int
	switch m := msg.(type) {
	case *fbproto.OpResponse:
		auxPort, err = parseAuxEventPort(m.Data)
		if err != nil {
			return nil, err
		}
	case *fbproto.ErrorResponse:
		return nil, ErrorResponseToFbError(m)
	default:
		return nil, fmt.Errorf("unexpected response message type %T while requesting event channel", m)
	}

	host := fbConn.eventDialHost()
	address := net.JoinHostPort(host, strconv.Itoa(auxPort))
	auxConn, err := fbConn.config.DialFunc(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	channel = newEventChannel(auxConn)
	fbConn.eventMu.Lock()
	fbConn.eventChannel = channel
	fbConn.eventErr = nil
	if fbConn.eventHandles == nil {
		fbConn.eventHandles = make(map[int]*EventHandle)
	}
	fbConn.eventMu.Unlock()

	go fbConn.readEventChannelLoop(channel)
	return channel, nil
}

func (fbConn *FbConn) eventDialHost() string {
	host := fbConn.eventHost
	if host == "" {
		host = fbConn.config.Host
	}
	if host == "" || isAbsolutePath(host) {
		return "localhost"
	}
	return host
}

func parseAuxEventPort(data []byte) (int, error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("invalid aux connect response data length: %d", len(data))
	}
	port := int(binary.BigEndian.Uint16(data[2:4]))
	if port <= 0 {
		return 0, fmt.Errorf("invalid aux event port: %d", port)
	}
	return port, nil
}

func (fbConn *FbConn) readEventChannelLoop(channel *eventChannel) {
	defer fbConn.releaseEventChannel(channel)

	xdr := fbproto.NewXDRReader(channel.conn)
	var eventBufferScratch []byte
	for {
		opcode, err := xdr.ReadInt32()
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				fbConn.setEventError(fmt.Errorf("event channel read failed: %w", err))
				if closeErr := channel.close(); closeErr != nil {
					fbConn.setEventError(fmt.Errorf("event channel close after read failure: %w", closeErr))
				}
			}
			return
		}

		switch opcode {
		case fbproto.Op_dummy:
			continue
		case fbproto.Op_disconnect, fbproto.Op_exit:
			if closeErr := channel.close(); closeErr != nil {
				fbConn.setEventError(fmt.Errorf("event channel close on server disconnect failed: %w", closeErr))
			}
			return
		case fbproto.Op_event:
			eventID, eventCount, readErr := readSingleEventFromXDR(xdr, &eventBufferScratch)
			if readErr != nil {
				fbConn.setEventError(fmt.Errorf("event message parse failed: %w", readErr))
				if closeErr := channel.close(); closeErr != nil {
					fbConn.setEventError(fmt.Errorf("event channel close after parse failure: %w", closeErr))
				}
				return
			}
			fbConn.dispatchEvent(eventID, eventCount)
		default:
			fbConn.setEventError(fmt.Errorf("unexpected event opcode %d", opcode))
			if closeErr := channel.close(); closeErr != nil {
				fbConn.setEventError(fmt.Errorf("event channel close after unexpected opcode failed: %w", closeErr))
			}
			return
		}
	}
}

func readSingleEventFromXDR(xdr *fbproto.XDRReader, scratch *[]byte) (eventID int, eventCount int, err error) {
	// p_event_database (ignored)
	if _, err = xdr.ReadInt32(); err != nil {
		return 0, 0, err
	}

	buffer, err := xdr.ReadBufferInto(scratch)
	if err != nil {
		return 0, 0, err
	}

	if len(buffer) > 4 {
		if buffer[0] != fbproto.EPB_version1 {
			return 0, 0, fmt.Errorf("invalid event buffer version: %d", buffer[0])
		}
		eventCount = vax.Integer(buffer, len(buffer)-4, 4)
	}

	// p_event_ast + p_event_arg (ignored)
	if _, err = xdr.ReadInt64(); err != nil {
		return 0, 0, err
	}

	eventID, err = xdr.ReadInt32()
	if err != nil {
		return 0, 0, err
	}

	return eventID, eventCount, nil
}

func (fbConn *FbConn) dispatchEvent(eventID int, eventCount int) {
	fbConn.eventMu.Lock()
	handle := fbConn.eventHandles[eventID]
	delete(fbConn.eventHandles, eventID)
	fbConn.eventMu.Unlock()

	if handle == nil {
		return
	}
	handle.setInternalCount(eventCount)
	handle.notify()
}

func (fbConn *FbConn) registerEventHandle(eventID int, handle *EventHandle) {
	fbConn.eventMu.Lock()
	if fbConn.eventHandles == nil {
		fbConn.eventHandles = make(map[int]*EventHandle)
	}
	for id, h := range fbConn.eventHandles {
		if h == handle {
			delete(fbConn.eventHandles, id)
		}
	}
	fbConn.eventHandles[eventID] = handle
	fbConn.eventMu.Unlock()
}

func (fbConn *FbConn) unregisterEventHandle(eventID int) {
	fbConn.eventMu.Lock()
	delete(fbConn.eventHandles, eventID)
	fbConn.eventMu.Unlock()
}

func (fbConn *FbConn) releaseEventChannel(channel *eventChannel) {
	fbConn.eventMu.Lock()
	if fbConn.eventChannel == channel {
		fbConn.eventChannel = nil
		fbConn.eventHandles = nil
	}
	fbConn.eventMu.Unlock()
}

func (fbConn *FbConn) closeEventChannel() {
	fbConn.eventMu.Lock()
	channel := fbConn.eventChannel
	fbConn.eventChannel = nil
	fbConn.eventHandles = nil
	fbConn.eventMu.Unlock()
	if closeErr := channel.close(); closeErr != nil {
		fbConn.setEventError(fmt.Errorf("event channel close failed: %w", closeErr))
	}
}

func (fbConn *FbConn) setEventError(err error) {
	if err == nil {
		return
	}
	fbConn.eventMu.Lock()
	fbConn.eventErr = errors.Join(fbConn.eventErr, err)
	fbConn.eventMu.Unlock()
}

func (fbConn *FbConn) takeEventError() error {
	fbConn.eventMu.Lock()
	err := fbConn.eventErr
	fbConn.eventErr = nil
	fbConn.eventMu.Unlock()
	return err
}
