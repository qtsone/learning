package apiperf

import (
	"container/list"
	"sync"
	"time"
)

// CachedPage is one rendered feed page: the exact bytes to write and the ETag
// that identifies them. They are stored together because they must never
// disagree.
type CachedPage struct {
	Body []byte
	ETag string
}

// cacheEntry is what the list holds. It keeps its own key so eviction from the
// back of the list can delete the map entry too.
type cacheEntry struct {
	key       string
	value     CachedPage
	expiresAt time.Time
}

// Cache is a bounded, TTL'd, in-process cache keyed by string.
//
// The TTL bounds staleness; the entry count bounds memory. Both are needed:
// a cache keyed by client-controlled input and bounded only by time is a way
// for a stranger to fill your heap one distinct key at a time.
type Cache struct {
	mu      sync.Mutex
	max     int
	ttl     time.Duration
	clock   Clock
	entries map[string]*list.Element
	order   *list.List // front = most recently used
	hits    int64
	misses  int64
}

// NewCache returns a cache holding at most max entries, each valid for ttl,
// reading time through clock.
func NewCache(max int, ttl time.Duration, clock Clock) *Cache {
	if max < 1 {
		max = 1
	}
	return &Cache{
		max:     max,
		ttl:     ttl,
		clock:   clock,
		entries: make(map[string]*list.Element, max),
		order:   list.New(),
	}
}

// Get returns the entry for key if it is present and unexpired.
func (c *Cache) Get(key string) (CachedPage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.entries[key]
	if !ok {
		c.misses++
		return CachedPage{}, false
	}
	entry := el.Value.(*cacheEntry)
	if !c.clock.Now().Before(entry.expiresAt) {
		// Dropping it here, rather than waiting for eviction pressure, is what
		// keeps a cold key from holding its bytes alive indefinitely.
		c.remove(el)
		c.misses++
		return CachedPage{}, false
	}
	c.order.MoveToFront(el)
	c.hits++
	return entry.value, true
}

// Set stores value under key as the most recently used entry, evicting the
// least recently used one when that puts the cache over its bound.
func (c *Cache) Set(key string, value CachedPage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := c.clock.Now().Add(c.ttl)
	if el, ok := c.entries[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.value, entry.expiresAt = value, expiresAt
		c.order.MoveToFront(el)
		return
	}
	c.entries[key] = c.order.PushFront(&cacheEntry{key: key, value: value, expiresAt: expiresAt})
	for c.order.Len() > c.max {
		c.remove(c.order.Back())
	}
}

// Purge drops every entry. Counters survive: hit rate is a property of the
// process, not of the current contents.
func (c *Cache) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*list.Element, c.max)
	c.order.Init()
}

// Len reports how many entries are held, expired ones included: it measures
// memory, not usefulness.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// Stats reports hits and misses since the process started.
func (c *Cache) Stats() (hits, misses int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses
}

// remove unlinks an element from both the list and the map. Callers hold c.mu.
func (c *Cache) remove(el *list.Element) {
	c.order.Remove(el)
	delete(c.entries, el.Value.(*cacheEntry).key)
}
