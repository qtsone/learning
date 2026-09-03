package syncex

import "sync"

// Store is a read-mostly string key/value store safe for concurrent use.
// Reads vastly outnumber writes, so it uses a sync.RWMutex: any number of
// readers may hold the read lock together, while a writer waits for
// exclusive access.
type Store struct {
	mu   sync.RWMutex
	vals map[string]string
}

// NewStore returns a Store ready for concurrent use.
func NewStore() *Store {
	return &Store{vals: make(map[string]string)}
}

// Get returns the value stored under key and whether the key was present.
// It takes the read lock, so concurrent Gets never block each other.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.vals[key]
	return v, ok
}

// Set stores value under key, replacing any previous value.
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vals[key] = value
}

// Len reports how many keys the store currently holds.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.vals)
}
