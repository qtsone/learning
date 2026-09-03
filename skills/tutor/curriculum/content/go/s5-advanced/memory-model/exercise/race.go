package memlab

// Meter counts events and remembers the most recent one. The two fields form
// a single invariant: a Snapshot must describe one moment in time.
//
// TODO: Meter is not safe for concurrent use — Record and Snapshot touch
// both fields with no happens-before edge. Make it safe while keeping hits
// and last consistent with each other (criteria 1-2 in LESSON.md).
type Meter struct {
	hits int
	last string
}

// NewMeter returns a Meter ready for use.
func NewMeter() *Meter { return &Meter{} }

// Record notes that event just happened.
func (m *Meter) Record(event string) {
	m.hits++
	m.last = event
}

// Snapshot returns the hit count and the most recent event as one
// consistent pair.
func (m *Meter) Snapshot() (hits int, last string) {
	return m.hits, m.last
}

// Config is an immutable configuration snapshot. To change configuration,
// build a new Config and publish it — never mutate a published one.
type Config struct {
	Version  int
	Endpoint string
}

// Store publishes the current *Config to any goroutine that asks.
//
// TODO: this publication is a data race — Update writes s.cfg while Current
// reads it, with no happens-before edge between them. "Pointer assignment
// is atomic" is not a memory-model guarantee. Fix it (criterion 3).
type Store struct {
	cfg *Config
}

// NewStore returns a Store publishing c.
func NewStore(c *Config) *Store { return &Store{cfg: c} }

// Update publishes c as the current configuration.
func (s *Store) Update(c *Config) { s.cfg = c }

// Current returns the most recently published configuration.
func (s *Store) Current() *Config { return s.cfg }
