package apiperf

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func page(s string) CachedPage { return CachedPage{Body: []byte(s), ETag: `"` + s + `"`} }

func TestCacheHitBeforeTTL(t *testing.T) {
	clk := newFakeClock()
	c := NewCache(4, 30*time.Second, clk)
	c.Set("a", page("alpha"))

	clk.Advance(29 * time.Second)
	got, ok := c.Get("a")
	if !ok {
		t.Fatal("entry expired 29s into a 30s TTL; it must still be a hit")
	}
	if string(got.Body) != "alpha" {
		t.Errorf("got body %q, want %q", got.Body, "alpha")
	}
}

func TestCacheExpiresAtExactlyTTL(t *testing.T) {
	clk := newFakeClock()
	c := NewCache(4, 30*time.Second, clk)
	c.Set("a", page("alpha"))

	clk.Advance(30 * time.Second)
	if _, ok := c.Get("a"); ok {
		t.Error("an entry stored at T is expired from T+ttl onwards, not after it")
	}
	if n := c.Len(); n != 0 {
		t.Errorf("Len() = %d after reading an expired entry, want 0 — an expired read must drop the entry, not just report a miss", n)
	}
}

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	clk := newFakeClock()
	c := NewCache(2, time.Minute, clk)
	c.Set("a", page("alpha"))
	c.Set("b", page("bravo"))

	if _, ok := c.Get("a"); !ok { // "a" is now the most recently used
		t.Fatal("a should still be cached")
	}
	c.Set("c", page("charlie"))

	if n := c.Len(); n != 2 {
		t.Errorf("Len() = %d, want 2 — the cache must never exceed its bound", n)
	}
	if _, ok := c.Get("b"); ok {
		t.Error("b was the least recently used entry and should have been evicted")
	}
	for _, key := range []string{"a", "c"} {
		if _, ok := c.Get(key); !ok {
			t.Errorf("%q should still be cached", key)
		}
	}
}

func TestCacheSetReplacesWithoutGrowing(t *testing.T) {
	clk := newFakeClock()
	c := NewCache(2, 30*time.Second, clk)
	c.Set("a", page("first"))
	clk.Advance(20 * time.Second)
	c.Set("a", page("second"))

	if n := c.Len(); n != 1 {
		t.Errorf("Len() = %d after re-Setting one key, want 1", n)
	}
	clk.Advance(20 * time.Second) // 40s after the first Set, 20s after the second
	got, ok := c.Get("a")
	if !ok {
		t.Fatal("re-Setting a key must refresh its expiry")
	}
	if string(got.Body) != "second" {
		t.Errorf("got body %q, want %q", got.Body, "second")
	}
}

func TestCacheStatsAndPurge(t *testing.T) {
	clk := newFakeClock()
	c := NewCache(4, time.Minute, clk)
	c.Set("a", page("alpha"))
	c.Get("a")
	c.Get("a")
	c.Get("missing")

	hits, misses := c.Stats()
	if hits != 2 || misses != 1 {
		t.Errorf("Stats() = (%d, %d), want (2, 1)", hits, misses)
	}

	c.Purge()
	if n := c.Len(); n != 0 {
		t.Errorf("Len() = %d after Purge, want 0", n)
	}
	if _, ok := c.Get("a"); ok {
		t.Error("Purge must drop every entry — this is what the write path relies on")
	}
	hits, misses = c.Stats()
	if hits != 2 || misses != 2 {
		t.Errorf("Stats() = (%d, %d) after Purge, want (2, 2) — counters measure the process, not the contents", hits, misses)
	}
}

// TestCacheIsSafeForConcurrentUse has no assertions beyond "does not race and
// stays bounded": it exists so `go test -race` sees the cache from many
// goroutines, which is how an http.Server will always use it.
func TestCacheIsSafeForConcurrentUse(t *testing.T) {
	clk := newFakeClock()
	c := NewCache(8, time.Minute, clk)

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range 50 {
				key := fmt.Sprintf("key-%d", (g+i)%16)
				c.Set(key, page(key))
				c.Get(key)
				c.Stats()
				c.Len()
			}
		}(g)
	}
	wg.Wait()

	if n := c.Len(); n > 8 {
		t.Errorf("Len() = %d, want at most 8", n)
	}
}
