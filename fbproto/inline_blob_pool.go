package fbproto

import (
	"fmt"
	"sync"
)

type inlineBlobPool struct {
	pool    sync.Pool
	maxSize int
}

// newInlineBlobPool creates a pool with a size cap (in bytes).
func newInlineBlobPool(maxSize int) *inlineBlobPool {
	if maxSize < 0 {
		panic(fmt.Sprintf("inlineBlobPool: invalid maxSize %d", maxSize))
	}
	if maxSize == 0 {
		return nil
	}
	return &inlineBlobPool{maxSize: maxSize}
}

// Get returns a buffer with at least the requested size.
func (p *inlineBlobPool) Get(size int) []byte {
	if p == nil {
		panic("inlineBlobPool is nil")
	}
	if size <= 0 {
		panic(fmt.Sprintf("inlineBlobPool.Get: invalid size %d", size))
	}
	if v := p.pool.Get(); v != nil {
		buf := *v.(*[]byte)
		if cap(buf) >= size {
			return buf[:size]
		}
	}
	return make([]byte, size)
}

// Put returns the buffer to the pool if it fits the configured max size.
func (p *inlineBlobPool) Put(buf []byte) {
	if p == nil {
		panic("inlineBlobPool is nil")
	}
	if buf == nil {
		return
	}
	if p.maxSize > 0 && cap(buf) > p.maxSize {
		return
	}
	buf = buf[:0]
	p.pool.Put(&buf)
}
