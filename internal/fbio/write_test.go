package fbio

import (
	"reflect"
	"testing"
)

func TestAppendUint16BENilBuf(t *testing.T) {
	buf := AppendUint16BE(nil, 1)
	if !reflect.DeepEqual(buf, []byte{0, 1}) {
		t.Errorf("AppendUint16BE(nil, 1) => %v, want %v", buf, []byte{0, 1})
	}
}

func TestAppendUint16BEEmptyBuf(t *testing.T) {
	buf := []byte{}
	buf = AppendUint16BE(buf, 1)
	if !reflect.DeepEqual(buf, []byte{0, 1}) {
		t.Errorf("AppendUint16BE(nil, 1) => %v, want %v", buf, []byte{0, 1})
	}
}

func TestAppendUint16BEBufWithCapacityDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 4)
	AppendUint16BE(buf, 1)
	buf = buf[0:2]
	if !reflect.DeepEqual(buf, []byte{0, 1}) {
		t.Errorf("AppendUint16BE(nil, 1) => %v, want %v", buf, []byte{0, 1})
	}
}

func TestAppendUint32BENilBuf(t *testing.T) {
	buf := AppendUint32BE(nil, 1)
	if !reflect.DeepEqual(buf, []byte{0, 0, 0, 1}) {
		t.Errorf("AppendUint32BE(nil, 1) => %v, want %v", buf, []byte{0, 0, 0, 1})
	}
}

func TestAppendUint32BEEmptyBuf(t *testing.T) {
	buf := []byte{}
	buf = AppendUint32BE(buf, 1)
	if !reflect.DeepEqual(buf, []byte{0, 0, 0, 1}) {
		t.Errorf("AppendUint32BE(nil, 1) => %v, want %v", buf, []byte{0, 0, 0, 1})
	}
}

func TestAppendUint32BEBufWithCapacityDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 4)
	AppendUint32BE(buf, 1)
	buf = buf[0:4]
	if !reflect.DeepEqual(buf, []byte{0, 0, 0, 1}) {
		t.Errorf("AppendUint32BE(nil, 1) => %v, want %v", buf, []byte{0, 0, 0, 1})
	}
}

func TestAppendUint64BENilBuf(t *testing.T) {
	buf := AppendUint64BE(nil, 1)
	if !reflect.DeepEqual(buf, []byte{0, 0, 0, 0, 0, 0, 0, 1}) {
		t.Errorf("AppendUint64BE(nil, 1) => %v, want %v", buf, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	}
}

func TestAppendUint64BEEmptyBuf(t *testing.T) {
	buf := []byte{}
	buf = AppendUint64BE(buf, 1)
	if !reflect.DeepEqual(buf, []byte{0, 0, 0, 0, 0, 0, 0, 1}) {
		t.Errorf("AppendUint64BE(nil, 1) => %v, want %v", buf, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	}
}

func TestAppendUint64BEBufWithCapacityDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 8)
	AppendUint64BE(buf, 1)
	buf = buf[0:8]
	if !reflect.DeepEqual(buf, []byte{0, 0, 0, 0, 0, 0, 0, 1}) {
		t.Errorf("AppendUint64BE(nil, 1) => %v, want %v", buf, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	}
}

func TestAppendInt16LENilBuf(t *testing.T) {
	buf := AppendInt16LE(nil, 1)
	if !reflect.DeepEqual(buf, []byte{1, 0}) {
		t.Errorf("AppendInt16LE(nil, 1) => %v, want %v", buf, []byte{1, 0})
	}
}

func TestAppendInt16LEEmptyBuf(t *testing.T) {
	buf := []byte{}
	buf = AppendInt16LE(buf, 1)
	if !reflect.DeepEqual(buf, []byte{1, 0}) {
		t.Errorf("AppendInt16LE(nil, 1) => %v, want %v", buf, []byte{1, 0})
	}
}

func TestAppendInt16LEBufWithCapacityDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 2)
	AppendInt16LE(buf, 1)
	buf = buf[0:2]
	if !reflect.DeepEqual(buf, []byte{1, 0}) {
		t.Errorf("AppendInt16LE(nil, 1) => %v, want %v", buf, []byte{1, 0})
	}
}

func TestAppendInt32LENilBuf(t *testing.T) {
	buf := AppendInt32LE(nil, 1)
	if !reflect.DeepEqual(buf, []byte{1, 0, 0, 0}) {
		t.Errorf("AppendInt32LE(nil, 1) => %v, want %v", buf, []byte{1, 0, 0, 0})
	}
}

func TestAppendInt32LEEmptyBuf(t *testing.T) {
	buf := []byte{}
	buf = AppendInt32LE(buf, 1)
	if !reflect.DeepEqual(buf, []byte{1, 0, 0, 0}) {
		t.Errorf("AppendInt32LE(nil, 1) => %v, want %v", buf, []byte{1, 0, 0, 0})
	}
}

func TestAppendInt32LEBufWithCapacityDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 4)
	AppendInt32LE(buf, 1)
	buf = buf[0:4]
	if !reflect.DeepEqual(buf, []byte{1, 0, 0, 0}) {
		t.Errorf("AppendInt32LE(nil, 1) => %v, want %v", buf, []byte{1, 0, 0, 0})
	}
}

func TestAppendInt64LENilBuf(t *testing.T) {
	buf := AppendInt64LE(nil, 1)
	if !reflect.DeepEqual(buf, []byte{1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("AppendInt64LE(nil, 1) => %v, want %v", buf, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	}
}

func TestAppendInt64LEEmptyBuf(t *testing.T) {
	buf := []byte{}
	buf = AppendInt64LE(buf, 1)
	if !reflect.DeepEqual(buf, []byte{1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("AppendInt64LE(nil, 1) => %v, want %v", buf, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	}
}

func TestAppendInt64LEBufWithCapacityDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 8)
	AppendInt64LE(buf, 1)
	buf = buf[0:8]
	if !reflect.DeepEqual(buf, []byte{1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("AppendInt64LE(nil, 1) => %v, want %v", buf, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	}
}
