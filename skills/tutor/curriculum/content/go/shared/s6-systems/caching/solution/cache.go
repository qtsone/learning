// Package cache implements a concurrency-safe in-memory cache with TTL
// expiry, LRU eviction, and singleflight loading.
package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// flight is one in-progress load. The owner writes val/err, then closes
// done; waiters read them only after <-done, so the close is the
// happens-before edge and no lock is needed on the fields.
type flight[V any] struct {
	done chan struct{}
	val  V
	err  error
}

// Cache holds at most capacity entries. Every entry expires ttl after the
// Set that last wrote it. When a new key is inserted into a full cache, the
// least-recently-used entry is evicted; Get hits and Set both count as use.
// Time is read through the injected now func so tests control the clock.
type Cache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	now      func() time.Time
	items    map[K]*list.Element
	order    *list.List // front = most recently used
	flights  map[K]*flight[V]
}

// New returns an empty cache. A capacity below 1 is clamped to 1 — an
// unbounded cache is a memory leak, so the type refuses to build one. A nil
// now defaults to time.Now.
func New[K comparable, V any](capacity int, ttl time.Duration, now func() time.Time) *Cache[K, V] {
	if now == nil {
		now = time.Now
	}
	if capacity < 1 {
		capacity = 1
	}
	return &Cache[K, V]{
		capacity: capacity,
		ttl:      ttl,
		now:      now,
		items:    make(map[K]*list.Element),
		order:    list.New(),
		flights:  make(map[K]*flight[V]),
	}
}

// Get returns the live value for key. It reports false when the key is
// absent or expired — an entry is expired once its age reaches ttl (a Get
// at exactly ttl after the write misses). A hit marks the entry most
// recently used.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.get(key)
}

// Set stores value under key, resetting the entry's TTL and marking it most
// recently used. Inserting a new key into a full cache evicts the
// least-recently-used entry first.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.set(key, value)
}

// Delete removes key if present. Deleting an absent key is a no-op.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.remove(el)
	}
}

// GetOrLoad returns the cached value for key, or calls loader to produce
// it. Concurrent calls for the same missing key share one loader call and
// all receive its result. A successful load is cached; an error is
// returned to every sharing caller and nothing is cached.
func (c *Cache[K, V]) GetOrLoad(key K, loader func() (V, error)) (V, error) {
	c.mu.Lock()
	if v, ok := c.get(key); ok {
		c.mu.Unlock()
		return v, nil
	}
	if f, ok := c.flights[key]; ok {
		c.mu.Unlock()
		<-f.done
		return f.val, f.err
	}
	f := &flight[V]{done: make(chan struct{})}
	c.flights[key] = f
	c.mu.Unlock()

	// The load runs outside the lock: other keys stay readable, and a slow
	// origin never blocks the whole cache.
	f.val, f.err = loader()

	c.mu.Lock()
	if f.err == nil {
		c.set(key, f.val)
	}
	delete(c.flights, key)
	c.mu.Unlock()
	close(f.done)
	return f.val, f.err
}

// get, set, and remove require c.mu to be held.

func (c *Cache[K, V]) get(key K) (V, bool) {
	var zero V
	el, ok := c.items[key]
	if !ok {
		return zero, false
	}
	en := el.Value.(*entry[K, V])
	if !c.now().Before(en.expiresAt) {
		c.remove(el)
		return zero, false
	}
	c.order.MoveToFront(el)
	return en.value, true
}

func (c *Cache[K, V]) set(key K, value V) {
	expiresAt := c.now().Add(c.ttl)
	if el, ok := c.items[key]; ok {
		en := el.Value.(*entry[K, V])
		en.value, en.expiresAt = value, expiresAt
		c.order.MoveToFront(el)
		return
	}
	if len(c.items) >= c.capacity {
		if oldest := c.order.Back(); oldest != nil {
			c.remove(oldest)
		}
	}
	c.items[key] = c.order.PushFront(&entry[K, V]{key: key, value: value, expiresAt: expiresAt})
}

func (c *Cache[K, V]) remove(el *list.Element) {
	c.order.Remove(el)
	delete(c.items, el.Value.(*entry[K, V]).key)
}
