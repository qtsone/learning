package syncex

// Counter counts named events and is safe for concurrent use.
// The zero value is not ready — call NewCounter.
type Counter struct {
	// TODO: a sync.Mutex and the map[string]int it guards.
	// Convention: declare the mutex directly above the fields it protects.
}

// NewCounter returns a Counter ready for concurrent use.
func NewCounter() *Counter {
	// TODO: initialize the map.
	return &Counter{}
}

// Inc adds one to the count for name.
func (c *Counter) Inc(name string) {
	// TODO: lock, increment, unlock (defer). Why must the receiver be a
	// pointer? go vet knows.
}

// Value returns the current count for name (zero if never incremented).
func (c *Counter) Value(name string) int {
	// TODO: reads need the lock too — run the tests with -race to see why.
	return 0
}

// Snapshot returns a copy of all counts. The caller may keep or mutate the
// returned map freely without affecting the Counter.
func (c *Counter) Snapshot() map[string]int {
	// TODO: copy the map while holding the lock, then return the copy.
	return nil
}

// Hits is a lock-free hit counter backed by sync/atomic.
// The zero value is ready to use.
type Hits struct {
	// TODO: a single atomic.Int64 — no mutex.
}

// Inc records one hit.
func (h *Hits) Inc() {
	// TODO: one atomic call.
}

// Value reports the total hits recorded so far.
func (h *Hits) Value() int64 {
	// TODO: one atomic call.
	return 0
}
