package fbio

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestReaderReadsSequentially(t *testing.T) {
	for _, tt := range []struct {
		name   string
		src    []byte
		read16 func(*Reader) uint16
		read32 func(*Reader) uint32
		read64 func(*Reader) uint64
	}{
		{
			name: "BE",
			src: []byte{
				0x01,
				0x01, 0x02,
				0x03, 0x04, 0x05, 0x06,
				0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
				0xde, 0xad,
			},
			read16: (*Reader).Uint16BE, read32: (*Reader).Uint32BE, read64: (*Reader).Uint64BE,
		},
		{
			name: "LE",
			src: []byte{
				0x01,
				0x02, 0x01,
				0x06, 0x05, 0x04, 0x03,
				0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08, 0x07,
				0xde, 0xad,
			},
			read16: (*Reader).Uint16LE, read32: (*Reader).Uint32LE, read64: (*Reader).Uint64LE,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(tt.src)
			if b := r.Byte(); b != 1 {
				t.Errorf("Byte() => %v, want 1", b)
			}
			if n := tt.read16(r); n != 0x0102 {
				t.Errorf("Uint16 => %#x, want 0x0102", n)
			}
			if n := tt.read32(r); n != 0x03040506 {
				t.Errorf("Uint32 => %#x, want 0x03040506", n)
			}
			if n := tt.read64(r); n != 0x0708090a0b0c0d0e {
				t.Errorf("Uint64 => %#x, want 0x0708090a0b0c0d0e", n)
			}
			if b := r.Bytes(2); !bytes.Equal(b, []byte{0xde, 0xad}) {
				t.Errorf("Bytes(2) => %x, want dead", b)
			}
			if err := r.Finish(); err != nil {
				t.Errorf("Finish() => %v, want nil", err)
			}
		})
	}
}

func TestReaderNegativeIntegers(t *testing.T) {
	for _, tt := range []struct {
		name   string
		src    []byte
		read16 func(*Reader) int16
		read32 func(*Reader) int32
		read64 func(*Reader) int64
	}{
		{
			name: "BE",
			src: []byte{
				0xff, 0xfe,
				0xff, 0xff, 0xff, 0xfd,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfc,
			},
			read16: (*Reader).Int16BE, read32: (*Reader).Int32BE, read64: (*Reader).Int64BE,
		},
		{
			name: "LE",
			src: []byte{
				0xfe, 0xff,
				0xfd, 0xff, 0xff, 0xff,
				0xfc, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
			read16: (*Reader).Int16LE, read32: (*Reader).Int32LE, read64: (*Reader).Int64LE,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(tt.src)
			if n := tt.read16(r); n != -2 {
				t.Errorf("Int16 => %v, want -2", n)
			}
			if n := tt.read32(r); n != -3 {
				t.Errorf("Int32 => %v, want -3", n)
			}
			if n := tt.read64(r); n != -4 {
				t.Errorf("Int64 => %v, want -4", n)
			}
			if err := r.Finish(); err != nil {
				t.Errorf("Finish() => %v, want nil", err)
			}
		})
	}
}

func TestReaderMixedByteOrders(t *testing.T) {
	r := NewReader([]byte{1, 2, 3, 4, 5, 6})
	if n := r.Uint16BE(); n != 0x0102 {
		t.Errorf("Uint16BE() => %#x, want 0x0102", n)
	}
	if n := r.Uint32LE(); n != 0x06050403 {
		t.Errorf("Uint32LE() => %#x, want 0x06050403", n)
	}
	if err := r.Finish(); err != nil {
		t.Errorf("Finish() => %v, want nil", err)
	}
}

func TestReaderInsufficientBytesIsSticky(t *testing.T) {
	for _, tt := range []struct {
		name string
		size int
		read func(*Reader) uint64
	}{
		{"Byte", 1, func(r *Reader) uint64 { return uint64(r.Byte()) }},
		{"Uint16BE", 2, func(r *Reader) uint64 { return uint64(r.Uint16BE()) }},
		{"Int16BE", 2, func(r *Reader) uint64 { return uint64(r.Int16BE()) }},
		{"Uint32BE", 4, func(r *Reader) uint64 { return uint64(r.Uint32BE()) }},
		{"Int32BE", 4, func(r *Reader) uint64 { return uint64(r.Int32BE()) }},
		{"Uint64BE", 8, (*Reader).Uint64BE},
		{"Int64BE", 8, func(r *Reader) uint64 { return uint64(r.Int64BE()) }},
		{"Uint16LE", 2, func(r *Reader) uint64 { return uint64(r.Uint16LE()) }},
		{"Int16LE", 2, func(r *Reader) uint64 { return uint64(r.Int16LE()) }},
		{"Uint32LE", 4, func(r *Reader) uint64 { return uint64(r.Uint32LE()) }},
		{"Int32LE", 4, func(r *Reader) uint64 { return uint64(r.Int32LE()) }},
		{"Uint64LE", 8, (*Reader).Uint64LE},
		{"Int64LE", 8, func(r *Reader) uint64 { return uint64(r.Int64LE()) }},
	} {
		for n := 0; n < tt.size; n++ {
			t.Run(fmt.Sprintf("%s/%d_bytes", tt.name, n), func(t *testing.T) {
				r := NewReader(bytes.Repeat([]byte{0xff}, n+1))
				r.Byte() // Start the failing read at a nonzero offset.
				if got := tt.read(r); got != 0 {
					t.Errorf("short read => %v, want 0", got)
				}
				firstErr := r.Err()
				if !errors.Is(firstErr, ErrInsufficientBytes) {
					t.Fatalf("Err() => %v, want ErrInsufficientBytes", firstErr)
				}
				if r.Remaining() != n {
					t.Fatalf("failed read advanced: %d bytes remain, want %d", r.Remaining(), n)
				}
				if got := r.Byte(); got != 0 {
					t.Errorf("Byte() after error => %v, want 0", got)
				}
				if got := r.Bytes(1); got != nil {
					t.Errorf("Bytes(1) after error => %v, want nil", got)
				}
				r.Bytes(-1) // A different error must not replace the first one.
				if r.Err() != firstErr || r.Finish() != firstErr || r.Remaining() != n {
					t.Error("reads after failure changed the error or position")
				}
			})
		}
	}
}

func TestReaderBytes(t *testing.T) {
	src := []byte{1, 2}
	r := NewReader(src)
	b := r.Bytes(2)
	if !bytes.Equal(b, src) || r.Finish() != nil {
		t.Fatalf("Bytes(2) => %v, Finish() => %v", b, r.Finish())
	}
	b[0] = 3
	if src[0] != 3 {
		t.Error("Bytes must return a slice of the source, not a copy")
	}
	if b := r.Bytes(0); len(b) != 0 || r.Err() != nil {
		t.Errorf("Bytes(0) at end => %v, err %v", b, r.Err())
	}

	for _, src := range [][]byte{nil, {1, 2}} {
		r := NewReader(src)
		if b := r.Bytes(len(src) + 1); b != nil || !errors.Is(r.Err(), ErrInsufficientBytes) {
			t.Errorf("short Bytes read => %v, err %v", b, r.Err())
		}
		if r.Remaining() != len(src) {
			t.Error("short Bytes read advanced the position")
		}
	}
}

func TestReaderBytesNegativeCount(t *testing.T) {
	r := NewReader([]byte{0x01})
	if b := r.Bytes(-1); b != nil {
		t.Errorf("Bytes(-1) => %v, want nil", b)
	}
	if r.Err() == nil || r.Remaining() != 1 {
		t.Errorf("Bytes(-1): Err() => %v, Remaining() => %d", r.Err(), r.Remaining())
	}
}

func TestReaderFinishTrailingBytes(t *testing.T) {
	r := NewReader([]byte{0x01, 0x02})
	r.Byte()
	if err := r.Finish(); err == nil {
		t.Error("Finish() with unread bytes => nil, want error")
	}
}

func TestUintExact(t *testing.T) {
	for _, tt := range []struct {
		name string
		size int
		read func([]byte) (uint64, error)
		want uint64
	}{
		{"Uint16BEExact", 2, func(b []byte) (uint64, error) { n, err := Uint16BEExact(b); return uint64(n), err }, 0x0102},
		{"Uint32BEExact", 4, func(b []byte) (uint64, error) { n, err := Uint32BEExact(b); return uint64(n), err }, 0x01020304},
		{"Uint64BEExact", 8, Uint64BEExact, 0x0102030405060708},
		{"Uint16LEExact", 2, func(b []byte) (uint64, error) { n, err := Uint16LEExact(b); return uint64(n), err }, 0x0201},
		{"Uint32LEExact", 4, func(b []byte) (uint64, error) { n, err := Uint32LEExact(b); return uint64(n), err }, 0x04030201},
		{"Uint64LEExact", 8, Uint64LEExact, 0x0807060504030201},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := []byte{1, 2, 3, 4, 5, 6, 7, 8}
			if got, err := tt.read(src[:tt.size]); got != tt.want || err != nil {
				t.Errorf("read => %#x, %v, want %#x, nil", got, err, tt.want)
			}
			// A single value must occupy the entire source, so reject both
			// truncated input and trailing bytes.
			for n := 0; n <= tt.size+1; n++ {
				if n == tt.size {
					continue
				}
				got, err := tt.read(bytes.Repeat([]byte{0xff}, n))
				if got != 0 || !errors.Is(err, ErrInvalidLength) {
					t.Errorf("read(%d bytes) => %v, %v, want 0, ErrInvalidLength", n, got, err)
				}
			}
		})
	}
}

func TestReadUint16BE(t *testing.T) {
	if got := ReadUint16BE([]byte{0, 1}); got != 1 {
		t.Errorf("ReadUint16BE([0,1]) => %d, want 1", got)
	}
}

func TestReadInt32BE(t *testing.T) {
	if got := ReadInt32BE([]byte{0, 0, 0, 1}); got != 1 {
		t.Errorf("ReadInt32BE([0,0,0,1]) => %d, want 1", got)
	}
}

func TestReadInt64BE(t *testing.T) {
	if got := ReadInt64BE([]byte{0, 0, 0, 0, 0, 0, 0, 1}); got != 1 {
		t.Errorf("ReadInt64BE([0,0,0,0,0,0,0,1]) => %d, want 1", got)
	}
}

func TestReadInt16LE(t *testing.T) {
	if got := ReadInt16LE([]byte{1, 0}); got != 1 {
		t.Errorf("ReadInt16LE([1,0]) => %d, want 1", got)
	}
}

func TestReadInt32LE(t *testing.T) {
	if got := ReadInt32LE([]byte{1, 0, 0, 0}); got != 1 {
		t.Errorf("ReadInt32LE([1,0,0,0]) => %d, want 1", got)
	}
}

func TestReadInt64LE(t *testing.T) {
	if got := ReadInt64LE([]byte{1, 0, 0, 0, 0, 0, 0, 0}); got != 1 {
		t.Errorf("ReadInt64LE([1,0,0,0,0,0,0,0]) => %d, want 1", got)
	}
}
