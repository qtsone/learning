package memlab

import (
	"sync"
	"sync/atomic"
)

// Meter counts events and remembers the most recent one. The two fields form
// a single invariant, so one mutex guards them both: separate atomics would
// keep each field individually intact while letting a Snapshot pair a new
// count with an old event.
type Meter struct {
	mu   sync.Mutex
	hits int
	last string
}

// NewMeter returns a Meter ready for use.
func NewMeter() *Meter { return &Meter{} }

// Record notes that event just happened.
func (m *Meter) Record(event string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits++
	m.last = event
}

// Snapshot returns the hit count and the most recent event as one
// consistent pair.
func (m *Meter) Snapshot() (hits int, last string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits, m.last
}

// Config is an immutable configuration snapshot. To change configuration,
// build a new Config and publish it — never mutate a published one.
type Config struct {
	Version  int
	Endpoint string
}

// Store publishes the current *Config to any goroutine that asks. The state
// is a single word (one pointer) and Configs are immutable once published,
// so atomic.Pointer is exactly enough: Store/Load create the happens-before
// edge that makes the pointed-to Config's fields visible to readers.
type Store struct {
	cfg atomic.Pointer[Config]
}

// NewStore returns a Store publishing c.
func NewStore(c *Config) *Store {
	s := &Store{}
	s.cfg.Store(c)
	return s
}

// Update publishes c as the current configuration.
func (s *Store) Update(c *Config) { s.cfg.Store(c) }

// Current returns the most recently published configuration.
func (s *Store) Current() *Config { return s.cfg.Load() }
