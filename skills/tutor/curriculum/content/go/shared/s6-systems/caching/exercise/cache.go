// Package cache implements a concurrency-safe in-memory cache with TTL
// expiry, LRU eviction, and singleflight loading.
package cache

import "time"

// Cache holds at most capacity entries. Every entry expires ttl after the
// Set that last wrote it. When a new key is inserted into a full cache, the
// least-recently-used entry is evicted; Get hits and Set both count as use.
// Time is read through the injected now func so tests control the clock.
type Cache[K comparable, V any] struct {
	// TODO: you need at least — a mutex, capacity, ttl, the now func,
	// a map for O(1) lookup, a recency ordering (container/list is in
	// the standard library, or roll your own), and per-key in-flight
	// load tracking for GetOrLoad.
}

// New returns an empty cache. A capacity below 1 is clamped to 1 — an
// unbounded cache is a memory leak, so the type refuses to build one. A nil
// now defaults to time.Now.
func New[K comparable, V any](capacity int, ttl time.Duration, now func() time.Time) *Cache[K, V] {
	if now == nil {
		now = time.Now
	}
	// TODO: initialize your fields.
	return &Cache[K, V]{}
}

// Get returns the live value for key. It reports false when the key is
// absent or expired — an entry is expired once its age reaches ttl (a Get
// at exactly ttl after the write misses). A hit marks the entry most
// recently used.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	// TODO: look up, expire if due, refresh recency.
	var zero V
	return zero, false
}

// Set stores value under key, resetting the entry's TTL and marking it most
// recently used. Inserting a new key into a full cache evicts the
// least-recently-used entry first.
func (c *Cache[K, V]) Set(key K, value V) {
	// TODO
}

// Delete removes key if present. Deleting an absent key is a no-op.
func (c *Cache[K, V]) Delete(key K) {
	// TODO
}

// GetOrLoad returns the cached value for key, or calls loader to produce
// it. Concurrent calls for the same missing key share one loader call and
// all receive its result. A successful load is cached; an error is
// returned to every sharing caller and nothing is cached.
func (c *Cache[K, V]) GetOrLoad(key K, loader func() (V, error)) (V, error) {
	// TODO: check the cache; join an in-flight load for this key if one
	// exists; otherwise become the loader. Never hold your mutex while
	// the loader runs — other keys must stay readable meanwhile.
	var zero V
	return zero, nil
}
