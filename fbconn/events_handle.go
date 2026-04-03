package fbconn

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"rdb.red-soft.ru/fbx/fbproto"
)

// EventHandler receives callbacks for queued Firebird events.
type EventHandler interface {
	EventOccurred(*EventHandle)
}

// EventHandle tracks event state for one Firebird event name.
type EventHandle struct {
	eventName      string
	eventNameBytes []byte
	handler        EventHandler

	mu                    sync.Mutex
	localID               int
	internalCount         int
	previousInternalCount int
	eventCount            int
}

var eventLocalIDCounter atomic.Int32

func newEventHandle(eventName string, handler EventHandler, encoding *fbproto.Encoding) (*EventHandle, error) {
	if eventName == "" {
		return nil, errors.New("eventName must not be empty")
	}
	if handler == nil {
		return nil, errors.New("handler must not be nil")
	}
	if encoding == nil {
		return nil, errors.New("encoding must not be nil")
	}

	encodedName, err := encoding.Encoder.String(eventName)
	if err != nil {
		return nil, err
	}
	nameBytes := []byte(encodedName)
	if len(nameBytes) > 256 {
		return nil, fmt.Errorf("event name as bytes too long: %d (max 255)", len(nameBytes))
	}

	return &EventHandle{
		eventName:      eventName,
		eventNameBytes: nameBytes,
		handler:        handler,
	}, nil
}

// EventName returns event name.
func (h *EventHandle) EventName() string {
	if h == nil {
		return ""
	}
	return h.eventName
}

// EventCount returns event count delta for the latest occurrence.
func (h *EventHandle) EventCount() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.eventCount
}

// EventID returns current local event id.
func (h *EventHandle) EventID() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.localID
}

func (h *EventHandle) assignNewLocalID() int {
	newID := int(eventLocalIDCounter.Add(1))
	h.mu.Lock()
	h.localID = newID
	h.mu.Unlock()
	return newID
}

func (h *EventHandle) clearLocalID() {
	h.mu.Lock()
	h.localID = 0
	h.mu.Unlock()
}

func (h *EventHandle) setInternalCount(internalCount int) {
	h.mu.Lock()
	h.internalCount = internalCount
	h.mu.Unlock()
}

func (h *EventHandle) calculateCount() {
	h.mu.Lock()
	h.eventCount = h.internalCount - h.previousInternalCount
	h.previousInternalCount = h.internalCount
	h.mu.Unlock()
}

func (h *EventHandle) notify() {
	if h == nil || h.handler == nil {
		return
	}
	h.handler.EventOccurred(h)
}

func (h *EventHandle) toEventBlock() []byte {
	h.mu.Lock()
	internalCount := h.internalCount
	nameBytes := h.eventNameBytes
	h.mu.Unlock()

	buf := make([]byte, 2+len(nameBytes)+4)
	buf[0] = fbproto.EPB_version1
	buf[1] = byte(len(nameBytes))
	copy(buf[2:], nameBytes)
	putVaxInt32(buf, 2+len(nameBytes), internalCount)
	return buf
}

func putVaxInt32(dst []byte, off, v int) {
	dst[off] = byte(v)
	dst[off+1] = byte(v >> 8)
	dst[off+2] = byte(v >> 16)
	dst[off+3] = byte(v >> 24)
}
