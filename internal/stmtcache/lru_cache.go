package stmtcache

import (
	"rdb.red-soft.ru/fbx/fbconn"
)

// lruNode is a typed doubly-linked list node with freelist support.
type lruNode struct {
	key  string
	st   *fbconn.Statement
	prev *lruNode
	next *lruNode
}

// LRUCache implements Cache with a Least Recently Used (LRU) cache.
type LRUCache struct {
	m    map[string]*lruNode
	head *lruNode

	tail     *lruNode
	len      int
	cap      int
	freelist *lruNode

	invalidStmts []*fbconn.Statement
	invalid      map[string]*fbconn.Statement
}

// NewLRUCache creates a new LRUCache. cap is the maximum size of the cache.
func NewLRUCache(cap int) *LRUCache {
	head := &lruNode{}
	tail := &lruNode{}
	head.next = tail
	tail.prev = head

	return &LRUCache{
		cap:     cap,
		m:       make(map[string]*lruNode, cap),
		head:    head,
		tail:    tail,
		invalid: make(map[string]*fbconn.Statement),
	}
}

// Get returns the statement for sql. It returns nil if the statement is not found.
func (c *LRUCache) Get(key string) *fbconn.Statement {
	node, ok := c.m[key]
	if !ok {
		return nil
	}
	c.moveToFront(node)
	return node.st
}

// Put stores st in the cache. Put panics if st.Description.SQL is empty.
// Put does nothing if the SQL is already cached or has been invalidated and
// RemoveInvalidated has not been called yet.
func (c *LRUCache) Put(st *fbconn.Statement) {
	if st.Description.SQL == "" {
		panic("cannot store statement description with empty SQL")
	}

	if _, present := c.m[st.Description.SQL]; present {
		return
	}

	// The statement may have been invalidated but not yet handled. Do not readd it to the cache.
	if _, invalidated := c.invalid[st.Description.SQL]; invalidated {
		return
	}

	if c.len == c.cap {
		c.invalidateOldest()
	}

	node := c.allocNode()
	node.key = st.Description.SQL
	node.st = st
	c.insertAfter(c.head, node)
	c.m[node.key] = node
	c.len++
}

// Remove removes st from the active cache or the invalidation queue.
// It does nothing if a different statement with the same SQL is stored.
func (c *LRUCache) Remove(st *fbconn.Statement) {
	key := st.Description.SQL
	node := c.m[key]
	if node == nil || node.st != st {
		node = nil
		for _, cachedNode := range c.m {
			if cachedNode.st == st {
				node = cachedNode
				break
			}
		}
	}

	if node != nil {
		delete(c.m, node.key)
		c.unlink(node)
		c.len--
		c.freeNode(node)
		return
	}

	invalidated := c.invalid[key]
	if invalidated != st {
		key = ""
		for invalidatedSQL, candidate := range c.invalid {
			if candidate == st {
				key = invalidatedSQL
				break
			}
		}
		if key == "" {
			return
		}
	}

	delete(c.invalid, key)
	for i, candidate := range c.invalidStmts {
		if candidate == st {
			copy(c.invalidStmts[i:], c.invalidStmts[i+1:])
			last := len(c.invalidStmts) - 1
			c.invalidStmts[last] = nil
			c.invalidStmts = c.invalidStmts[:last]
			return
		}
	}
}

// Invalidate invalidates the statement identified by sql.
// It does nothing if the statement is not found.
func (c *LRUCache) Invalidate(sql string) {
	node, ok := c.m[sql]
	if !ok {
		return
	}
	delete(c.m, sql)
	c.invalidStmts = append(c.invalidStmts, node.st)
	c.invalid[sql] = node.st
	c.unlink(node)
	c.len--
	c.freeNode(node)
}

// InvalidateAll invalidates all statements.
func (c *LRUCache) InvalidateAll() {
	for node := c.head.next; node != c.tail; {
		next := node.next
		c.invalidStmts = append(c.invalidStmts, node.st)
		c.invalid[node.key] = node.st
		c.freeNode(node)
		node = next
	}

	clear(c.m)
	c.head.next = c.tail
	c.tail.prev = c.head
	c.len = 0
}

// GetInvalidated returns all statements invalidated since the last call to RemoveInvalidated.
func (c *LRUCache) GetInvalidated() []*fbconn.Statement {
	return c.invalidStmts
}

// GetInvalidatedBySQL returns the invalidated statement for sql.
// It returns nil if the statement is not found.
func (c *LRUCache) GetInvalidatedBySQL(sql string) *fbconn.Statement {
	return c.invalid[sql]
}

// RemoveInvalidated removes all invalidated statements. No other calls to Cache must be made between a
// call to GetInvalidated and RemoveInvalidated or RemoveInvalidated may remove statements that were
// never seen by the call to GetInvalidated.
func (c *LRUCache) RemoveInvalidated() {
	clear(c.invalidStmts)
	c.invalidStmts = c.invalidStmts[:0]
	clear(c.invalid)
}

// Len returns the number of cached prepared statements.
func (c *LRUCache) Len() int {
	return c.len
}

// Cap returns the maximum number of cached prepared statements.
func (c *LRUCache) Cap() int {
	return c.cap
}

func (c *LRUCache) invalidateOldest() {
	node := c.tail.prev
	if node == c.head {
		return
	}
	c.invalidStmts = append(c.invalidStmts, node.st)
	c.invalid[node.key] = node.st
	delete(c.m, node.key)
	c.unlink(node)
	c.len--
	c.freeNode(node)
}

// List operations - sentinel nodes eliminate nil checks.

func (c *LRUCache) insertAfter(at, node *lruNode) {
	node.prev = at
	node.next = at.next
	at.next.prev = node
	at.next = node
}

func (c *LRUCache) unlink(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (c *LRUCache) moveToFront(node *lruNode) {
	if node.prev == c.head {
		return
	}
	c.unlink(node)
	c.insertAfter(c.head, node)
}

// Node pool operations - reuse evicted nodes to avoid allocations.

func (c *LRUCache) allocNode() *lruNode {
	if c.freelist != nil {
		node := c.freelist
		c.freelist = node.next
		node.next = nil
		node.prev = nil
		return node
	}
	return &lruNode{}
}

func (c *LRUCache) freeNode(node *lruNode) {
	node.key = ""
	node.st = nil
	node.prev = nil
	node.next = c.freelist
	c.freelist = node
}
