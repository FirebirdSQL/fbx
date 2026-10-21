package stmtcache

import (
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
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

func TestLRUCacheRemoveInvalidatedReleasesStatements(t *testing.T) {
	cache := NewLRUCache(1)
	st := newTestStatement("select 1")
	cache.Put(st)
	cache.Invalidate(st.Description.SQL)

	cache.RemoveInvalidated()

	for i, retained := range cache.invalidStmts[:cap(cache.invalidStmts)] {
		if retained != nil {
			t.Fatalf("invalidated statement slot %d retains %p", i, retained)
		}
	}
}

func TestLRUCacheRemoveChecksStatementIdentity(t *testing.T) {
	cache := NewLRUCache(1)
	cached := newTestStatement("select 1")
	other := newTestStatement(cached.Description.SQL)
	cache.Put(cached)

	cache.Remove(other)
	if got := cache.Get(cached.Description.SQL); got != cached {
		t.Fatalf("Get(%q) after removing other statement = %p, want %p", cached.Description.SQL, got, cached)
	}

	cache.Remove(cached)
	if got := cache.Get(cached.Description.SQL); got != nil {
		t.Fatalf("Get(%q) after removing cached statement = %p, want nil", cached.Description.SQL, got)
	}
	if invalidated := cache.GetInvalidated(); len(invalidated) != 0 {
		t.Fatalf("Remove added invalidated statements: %v", invalidated)
	}

	mutated := newTestStatement("select 2")
	cache.Put(mutated)
	originalSQL := mutated.Description.SQL
	mutated.Description.SQL = "select changed"
	cache.Remove(mutated)
	if got := cache.Get(originalSQL); got != nil {
		t.Fatalf("Get(%q) after removing mutated statement = %p, want nil", originalSQL, got)
	}
}

func TestLRUCacheRemoveInvalidatedChecksStatementIdentity(t *testing.T) {
	cache := NewLRUCache(1)
	cached := newTestStatement("select 1")
	other := newTestStatement(cached.Description.SQL)
	cache.Put(cached)
	cache.Invalidate(cached.Description.SQL)

	cache.Remove(other)
	if got := cache.GetInvalidatedBySQL(cached.Description.SQL); got != cached {
		t.Fatalf("GetInvalidatedBySQL(%q) after removing other statement = %p, want %p", cached.Description.SQL, got, cached)
	}

	originalSQL := cached.Description.SQL
	cached.Description.SQL = "select changed"
	cache.Remove(cached)
	if got := cache.GetInvalidatedBySQL(originalSQL); got != nil {
		t.Fatalf("GetInvalidatedBySQL(%q) after removing cached statement = %p, want nil", originalSQL, got)
	}
	if invalidated := cache.GetInvalidated(); len(invalidated) != 0 {
		t.Fatalf("invalidated after removing cached statement = %v, want empty", invalidated)
	}

	replacement := newTestStatement(originalSQL)
	cache.Put(replacement)
	if got := cache.Get(originalSQL); got != replacement {
		t.Fatalf("Get(%q) after removing invalidated statement = %p, want %p", originalSQL, got, replacement)
	}
}

func TestLRUCacheRemoveOneOfMultipleInvalidatedStatements(t *testing.T) {
	cache := NewLRUCache(3)
	first := newTestStatement("select 1")
	second := newTestStatement("select 2")
	third := newTestStatement("select 3")
	cache.Put(first)
	cache.Put(second)
	cache.Put(third)
	cache.InvalidateAll()

	cache.Remove(second)
	if got := cache.GetInvalidatedBySQL(second.Description.SQL); got != nil {
		t.Fatalf("GetInvalidatedBySQL(%q) after Remove = %p, want nil", second.Description.SQL, got)
	}
	invalidated := cache.GetInvalidated()
	if len(invalidated) != 2 || invalidated[0] == second || invalidated[1] == second {
		t.Fatalf("invalidated after removing middle statement = %v, want first and third", invalidated)
	}

	for _, st := range append([]*fbconn.Statement(nil), invalidated...) {
		cache.Remove(st)
	}
	if invalidated := cache.GetInvalidated(); len(invalidated) != 0 {
		t.Fatalf("invalidated after removing snapshot = %v, want empty", invalidated)
	}
	if got := cache.GetInvalidatedBySQL(first.Description.SQL); got != nil {
		t.Fatalf("GetInvalidatedBySQL(%q) = %p, want nil", first.Description.SQL, got)
	}
	if got := cache.GetInvalidatedBySQL(third.Description.SQL); got != nil {
		t.Fatalf("GetInvalidatedBySQL(%q) = %p, want nil", third.Description.SQL, got)
	}
}
