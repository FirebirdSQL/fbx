package fbio

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ErrInsufficientBytes is wrapped by all Reader errors caused by a read past
// the end of the source.
var ErrInsufficientBytes = errors.New("insufficient bytes")

// Reader reads binary data with bounds checks. It is
// designed so that decoders of untrusted input cannot forget a length check:
// every read validates against the remaining bytes, and the first failure
// sticks. After a failure all subsequent reads return zero values, so a
// decoder can read an entire structure without intermediate error checks and
// inspect Err or Finish once at the end.
//
// Reads never panic. A decoder that branches on a value it just read (e.g. an
// element count used to size an allocation) should check Err before acting on
// the value.
type Reader struct {
	s   []byte
	rp  int
	err error
}

// NewReader reads from s without copying it.
func NewReader(s []byte) *Reader {
	return &Reader{s: s}
}

func (r *Reader) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

// need reports whether n more bytes are available, recording an error if not.
//
// Error formatting is kept in failShort to keep the successful read path small.
func (r *Reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if len(r.s)-r.rp < n {
		r.failShort(n)
		return false
	}
	return true
}

func (r *Reader) failShort(n int) {
	r.fail(fmt.Errorf("%w: %d needed at offset %d, %d remain", ErrInsufficientBytes, n, r.rp, len(r.s)-r.rp))
}

// Err returns the first error encountered, if any.
func (r *Reader) Err() error {
	return r.err
}

// Remaining returns the number of unread bytes.
func (r *Reader) Remaining() int {
	return len(r.s) - r.rp
}

// Byte reads the next byte.
func (r *Reader) Byte() byte {
	if !r.need(1) {
		return 0
	}
	b := r.s[r.rp]
	r.rp += 1
	return b
}

// Uint16BE reads the next two bytes as a big-endian uint16.
func (r *Reader) Uint16BE() uint16 {
	if !r.need(2) {
		return 0
	}
	n := binary.BigEndian.Uint16(r.s[r.rp:])
	r.rp += 2
	return n
}

// Int16BE reads the next two bytes as a big-endian int16.
func (r *Reader) Int16BE() int16 {
	return int16(r.Uint16BE())
}

// Uint32BE reads the next four bytes as a big-endian uint32.
func (r *Reader) Uint32BE() uint32 {
	if !r.need(4) {
		return 0
	}
	n := binary.BigEndian.Uint32(r.s[r.rp:])
	r.rp += 4
	return n
}

// Int32BE reads the next four bytes as a big-endian int32.
func (r *Reader) Int32BE() int32 {
	return int32(r.Uint32BE())
}

// Uint64BE reads the next eight bytes as a big-endian uint64.
func (r *Reader) Uint64BE() uint64 {
	if !r.need(8) {
		return 0
	}
	n := binary.BigEndian.Uint64(r.s[r.rp:])
	r.rp += 8
	return n
}

// Int64BE reads the next eight bytes as a big-endian int64.
func (r *Reader) Int64BE() int64 {
	return int64(r.Uint64BE())
}

// Uint16LE reads the next two bytes as a little-endian uint16.
func (r *Reader) Uint16LE() uint16 {
	if !r.need(2) {
		return 0
	}
	n := binary.LittleEndian.Uint16(r.s[r.rp:])
	r.rp += 2
	return n
}

// Int16LE reads the next two bytes as a little-endian int16.
func (r *Reader) Int16LE() int16 {
	return int16(r.Uint16LE())
}

// Uint32LE reads the next four bytes as a little-endian uint32.
func (r *Reader) Uint32LE() uint32 {
	if !r.need(4) {
		return 0
	}
	n := binary.LittleEndian.Uint32(r.s[r.rp:])
	r.rp += 4
	return n
}

// Int32LE reads the next four bytes as a little-endian int32.
func (r *Reader) Int32LE() int32 {
	return int32(r.Uint32LE())
}

// Uint64LE reads the next eight bytes as a little-endian uint64.
func (r *Reader) Uint64LE() uint64 {
	if !r.need(8) {
		return 0
	}
	n := binary.LittleEndian.Uint64(r.s[r.rp:])
	r.rp += 8
	return n
}

// Int64LE reads the next eight bytes as a little-endian int64.
func (r *Reader) Int64LE() int64 {
	return int64(r.Uint64LE())
}

// Bytes reads the next n bytes. The returned slice aliases the source; it is
// not a copy.
func (r *Reader) Bytes(n int) []byte {
	if n < 0 {
		r.fail(fmt.Errorf("invalid byte count %d at offset %d", n, r.rp))
		return nil
	}
	if !r.need(n) {
		return nil
	}
	b := r.s[r.rp : r.rp+n]
	r.rp += n
	return b
}

// Finish returns the first error encountered, or an error if unread bytes
// remain. Decoders that must consume the entire source should end with Finish.
func (r *Reader) Finish() error {
	if r.err != nil {
		return r.err
	}
	if r.rp != len(r.s) {
		return r.errTrailing()
	}
	return nil
}

func (r *Reader) errTrailing() error {
	return fmt.Errorf("%d unexpected trailing bytes at offset %d", len(r.s)-r.rp, r.rp)
}

// ErrInvalidLength is wrapped by the errors returned from the exact-length
// read functions below.
var ErrInvalidLength = errors.New("invalid length")

func errLength(want, got int) error {
	return fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidLength, want, got)
}

// The Uint*Exact functions read a single fixed-size value occupying the entire
// source. They check its exact length without maintaining a read position.

// Uint16BEExact returns the big-endian uint16 in src, which must be exactly 2 bytes.
func Uint16BEExact(src []byte) (uint16, error) {
	if len(src) != 2 {
		return 0, errLength(2, len(src))
	}
	return binary.BigEndian.Uint16(src), nil
}

// Uint32BEExact returns the big-endian uint32 in src, which must be exactly 4 bytes.
func Uint32BEExact(src []byte) (uint32, error) {
	if len(src) != 4 {
		return 0, errLength(4, len(src))
	}
	return binary.BigEndian.Uint32(src), nil
}

// Uint64BEExact returns the big-endian uint64 in src, which must be exactly 8 bytes.
func Uint64BEExact(src []byte) (uint64, error) {
	if len(src) != 8 {
		return 0, errLength(8, len(src))
	}
	return binary.BigEndian.Uint64(src), nil
}

// Uint16LEExact returns the little-endian uint16 in src, which must be exactly 2 bytes.
func Uint16LEExact(src []byte) (uint16, error) {
	if len(src) != 2 {
		return 0, errLength(2, len(src))
	}
	return binary.LittleEndian.Uint16(src), nil
}

// Uint32LEExact returns the little-endian uint32 in src, which must be exactly 4 bytes.
func Uint32LEExact(src []byte) (uint32, error) {
	if len(src) != 4 {
		return 0, errLength(4, len(src))
	}
	return binary.LittleEndian.Uint32(src), nil
}

// Uint64LEExact returns the little-endian uint64 in src, which must be exactly 8 bytes.
func Uint64LEExact(src []byte) (uint64, error) {
	if len(src) != 8 {
		return 0, errLength(8, len(src))
	}
	return binary.LittleEndian.Uint64(src), nil
}

// The Read* helpers read the leading bytes without checking length.
// Callers must validate the input length before calling them.

func ReadUint16BE(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func ReadInt32BE(b []byte) int32 {
	return int32(binary.BigEndian.Uint32(b))
}

func ReadInt64BE(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func ReadInt16LE(b []byte) int16 {
	return int16(binary.LittleEndian.Uint16(b))
}

func ReadInt32LE(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b))
}

func ReadInt64LE(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}
