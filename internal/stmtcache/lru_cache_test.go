package stmtcache

import (
	"testing"

	"rdb.red-soft.ru/fbx/fbconn"
)

func newTestStatement(sql string) *fbconn.Statement {
	return &fbconn.Statement{
		Description: &fbconn.StatementDescription{SQL: sql},
	}
}

func TestLRUCacheEvictsOldestAndTracksInvalidated(t *testing.T) {
	cache := NewLRUCache(2)
	a := newTestStatement("select 1")
	b := newTestStatement("select 2")
	c := newTestStatement("select 3")

	cache.Put(a)
	cache.Put(b)
	if got := cache.Get(a.Description.SQL); got != a {
		t.Fatalf("Get(%q) = %p, want %p", a.Description.SQL, got, a)
	}

	cache.Put(c)
	if got := cache.Get(b.Description.SQL); got != nil {
		t.Fatalf("evicted statement still in cache: %p", got)
	}

	invalidated := cache.GetInvalidated()
	if len(invalidated) != 1 || invalidated[0] != b {
		t.Fatalf("invalidated = %v, want [%p]", invalidated, b)
	}

	cache.Put(b)
	if got := cache.Get(b.Description.SQL); got != nil {
		t.Fatalf("invalidated statement was re-added before RemoveInvalidated: %p", got)
	}

	cache.RemoveInvalidated()
	cache.Put(b)
	if got := cache.Get(b.Description.SQL); got != b {
		t.Fatalf("Get(%q) after RemoveInvalidated = %p, want %p", b.Description.SQL, got, b)
	}
}
