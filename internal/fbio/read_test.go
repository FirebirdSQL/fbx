package fbio

import "testing"

func TestReadUint16BE(t *testing.T) {
	got := ReadUint16BE([]byte{0, 1})
	if got != 1 {
		t.Errorf("ReadUint16BE([0,1]) => %d, want 1", got)
	}
}

func TestReadInt32BE(t *testing.T) {
	got := ReadInt32BE([]byte{0, 0, 0, 1})
	if got != 1 {
		t.Errorf("ReadInt32BE([0,0,0,1]) => %d, want 1", got)
	}
}

func TestReadInt64BE(t *testing.T) {
	got := ReadInt64BE([]byte{0, 0, 0, 0, 0, 0, 0, 1})
	if got != 1 {
		t.Errorf("ReadInt64BE([0,0,0,0,0,0,0,1]) => %d, want 1", got)
	}
}

func TestReadInt16LE(t *testing.T) {
	got := ReadInt16LE([]byte{1, 0})
	if got != 1 {
		t.Errorf("ReadInt16LE([1,0]) => %d, want 1", got)
	}
}

func TestReadInt32LE(t *testing.T) {
	got := ReadInt32LE([]byte{1, 0, 0, 0})
	if got != 1 {
		t.Errorf("ReadInt32LE([1,0,0,0]) => %d, want 1", got)
	}
}

func TestReadInt64LE(t *testing.T) {
	got := ReadInt64LE([]byte{1, 0, 0, 0, 0, 0, 0, 0})
	if got != 1 {
		t.Errorf("ReadInt64LE([1,0,0,0,0,0,0,0]) => %d, want 1", got)
	}
}
