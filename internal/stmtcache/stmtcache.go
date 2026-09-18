// Package stmtcache provides a cache for prepared statements.
package stmtcache

import (
	"crypto/sha256"
	"encoding/hex"

	"rdb.red-soft.ru/fbx/fbconn"
)

// StatementName returns a statement name that will be stable for sql across multiple connections and program
// executions.
func StatementName(sql string) string {
	digest := sha256.Sum256([]byte(sql))
	return "stmtcache_" + hex.EncodeToString(digest[0:24])
}

// Cache stores prepared statements by SQL text.
type Cache interface {
	// Get returns the statement for sql. It returns nil if the statement is not found.
	Get(sql string) *fbconn.Statement

	// Put stores st in the cache. Put panics if st.Description.SQL is empty.
	// Put does nothing if st.Description.SQL already exists in the cache.
	Put(st *fbconn.Statement)

	// Remove removes st from the active cache or the invalidation queue.
	// It does nothing if a different statement with the same SQL is stored.
	Remove(st *fbconn.Statement)

	// Invalidate invalidates the statement identified by sql. It does nothing if the statement is not found.
	Invalidate(sql string)

	// InvalidateAll invalidates all statements.
	InvalidateAll()

	// GetInvalidated returns all statements invalidated since the last call to RemoveInvalidated.
	GetInvalidated() []*fbconn.Statement

	// GetInvalidatedBySQL returns the invalidated statement for sql.
	// It returns nil if the statement is not found.
	GetInvalidatedBySQL(sql string) *fbconn.Statement

	// RemoveInvalidated removes all invalidated statements. No other calls to Cache must be made between a
	// call to GetInvalidated and RemoveInvalidated or RemoveInvalidated may remove statements that were
	// never seen by the call to GetInvalidated.
	RemoveInvalidated()

	// Len returns the number of cached prepared statements.
	Len() int

	// Cap returns the maximum number of cached prepared statements.
	Cap() int
}
