package apiperf

import "time"

// CachedPage is one rendered feed page: the exact bytes to write and the ETag
// that identifies them.
//
// Caching the *rendered* response rather than the Page struct is deliberate.
// A hit then costs a map lookup and a write; caching the struct would still
// pay for json.Marshal and the hash on every request, which on a hot endpoint
// is most of what is left after the database is gone.
type CachedPage struct {
	Body []byte
	ETag string
}

// Cache is a bounded, TTL'd, in-process cache keyed by string.
//
// Two limits, and both matter. The TTL bounds *staleness* — how wrong an
// answer may be. The entry count bounds *memory* — without it, a cache keyed
// by anything a client controls (here: limit and cursor) is a way for a
// stranger to fill your heap, one distinct key at a time.
//
// It is safe for concurrent use: one http.Server serves every request in its
// own goroutine, so an unsynchronised map here is a data race the moment two
// clients arrive together.
type Cache struct {
	// TODO: your fields. You need a map for lookup, something that remembers
	// use order for eviction (container/list is the usual answer), a mutex,
	// the two limits, the clock, and the hit/miss counters.
}

// NewCache returns a cache holding at most max entries, each valid for ttl,
// reading time through clock. A max below 1 is treated as 1.
func NewCache(max int, ttl time.Duration, clock Clock) *Cache {
	// TODO
	return &Cache{}
}

// Get returns the entry for key if it is present and not expired, and reports
// whether it was a hit.
//
// An entry stored at T is valid while now is before T+ttl, and expired from
// T+ttl onwards. An expired entry is a miss *and* is removed — a cache that
// only drops entries when it runs out of room keeps dead bytes alive for as
// long as it has space.
//
// A hit also marks the entry as most recently used.
func (c *Cache) Get(key string) (CachedPage, bool) {
	// TODO
	return CachedPage{}, false
}

// Set stores value under key, most recently used, expiring ttl from now.
// Storing a key that is already present replaces it and refreshes its expiry
// without growing the cache. Once the cache is over its limit, the least
// recently used entry is evicted.
func (c *Cache) Set(key string, value CachedPage) {
	// TODO
}

// Purge drops every entry. This is invalidation: the write path calls it so a
// client cannot be served a page that predates its own POST. Counters survive
// a purge — hit rate is a property of the process, not of the current
// contents.
func (c *Cache) Purge() {
	// TODO
}

// Len reports how many entries are held, expired ones included: it measures
// memory, not usefulness.
func (c *Cache) Len() int {
	// TODO
	return 0
}

// Stats reports hits and misses since the process started.
//
// A cache with no hit-rate instrument is a guess with a mutex. Two numbers
// tell you whether the TTL is worth its staleness, and a hit rate that falls
// off a cliff after a deploy is usually a key that grew a new dimension.
func (c *Cache) Stats() (hits, misses int64) {
	// TODO
	return 0, 0
}
