package syncex

import (
	"maps"
	"sync"
	"sync/atomic"
)

// Counter counts named events and is safe for concurrent use.
// The zero value is not ready — call NewCounter.
type Counter struct {
	mu     sync.Mutex
	counts map[string]int
}

// NewCounter returns a Counter ready for concurrent use.
func NewCounter() *Counter {
	return &Counter{counts: make(map[string]int)}
}

// Inc adds one to the count for name.
func (c *Counter) Inc(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[name]++
}

// Value returns the current count for name (zero if never incremented).
func (c *Counter) Value(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[name]
}

// Snapshot returns a copy of all counts. The caller may keep or mutate the
// returned map freely without affecting the Counter.
func (c *Counter) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return maps.Clone(c.counts)
}

// Hits is a lock-free hit counter backed by sync/atomic.
// The zero value is ready to use.
type Hits struct {
	n atomic.Int64
}

// Inc records one hit.
func (h *Hits) Inc() {
	h.n.Add(1)
}

// Value reports the total hits recorded so far.
func (h *Hits) Value() int64 {
	return h.n.Load()
}
