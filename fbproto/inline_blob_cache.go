package fbproto

import "sync"

type inlineBlobCache struct {
	mu      sync.Mutex
	maxSize int64
	size    int64
	byTx    map[int]map[int64][]byte
	pool    *inlineBlobPool
}

func newInlineBlobCache(maxSize int, pool *inlineBlobPool) *inlineBlobCache {
	cacheSize := int64(maxSize)
	if cacheSize <= 0 {
		cacheSize = -1
	}
	return &inlineBlobCache{
		maxSize: cacheSize,
		byTx:    make(map[int]map[int64][]byte, 4),
		pool:    pool,
	}
}

func (c *inlineBlobCache) add(txHandle int, blobId int64, data []byte) bool {
	if c == nil {
		panic("inlineBlobCache is nil")
	}
	if c.maxSize <= 0 {
		return false
	}
	dataLen := int64(len(data))
	if dataLen == 0 && c.maxSize <= 0 {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxSize <= 0 {
		return false
	}
	if c.size+dataLen > c.maxSize {
		return false
	}
	byBlob := c.byTx[txHandle]
	if byBlob == nil {
		byBlob = make(map[int64][]byte)
		c.byTx[txHandle] = byBlob
	}
	if _, exists := byBlob[blobId]; exists {
		return false
	}
	byBlob[blobId] = data
	c.size += dataLen
	return true
}

func (c *inlineBlobCache) getAndRemove(txHandle int, blobId int64) ([]byte, bool) {
	if c == nil {
		panic("inlineBlobCache is nil")
	}
	if c.maxSize <= 0 {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	byBlob := c.byTx[txHandle]
	if byBlob == nil {
		return nil, false
	}
	data, ok := byBlob[blobId]
	if !ok {
		return nil, false
	}
	delete(byBlob, blobId)
	c.size -= int64(len(data))
	if c.size < 0 {
		c.size = 0
	}
	if len(byBlob) == 0 {
		delete(c.byTx, txHandle)
	}
	return data, true
}

func (c *inlineBlobCache) clearTx(txHandle int) {
	if c == nil {
		panic("inlineBlobCache is nil")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	byBlob := c.byTx[txHandle]
	if byBlob == nil {
		return
	}
	var removed int64
	for _, data := range byBlob {
		if c.pool != nil {
			c.pool.Put(data)
		}
		removed += int64(len(data))
	}
	delete(c.byTx, txHandle)
	c.size -= removed
	if c.size < 0 {
		c.size = 0
	}
}

func (c *inlineBlobCache) clearAll() {
	if c == nil {
		panic("inlineBlobCache is nil")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pool != nil {
		for _, byBlob := range c.byTx {
			for _, data := range byBlob {
				c.pool.Put(data)
			}
		}
	}
	c.byTx = make(map[int]map[int64][]byte, 4)
	c.size = 0
}
