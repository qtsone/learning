package syncex

// Store is a read-mostly string key/value store safe for concurrent use.
// Reads vastly outnumber writes, so it uses a sync.RWMutex: any number of
// readers may hold the read lock together, while a writer waits for
// exclusive access.
type Store struct {
	// TODO: a sync.RWMutex and the map[string]string it guards.
}

// NewStore returns a Store ready for concurrent use.
func NewStore() *Store {
	// TODO: initialize the map.
	return &Store{}
}

// Get returns the value stored under key and whether the key was present.
// It takes the read lock, so concurrent Gets never block each other.
func (s *Store) Get(key string) (string, bool) {
	// TODO: RLock/RUnlock around the map read.
	return "", false
}

// Set stores value under key, replacing any previous value.
func (s *Store) Set(key, value string) {
	// TODO: Lock/Unlock — writers need the whole lock.
}

// Len reports how many keys the store currently holds.
func (s *Store) Len() int {
	// TODO: which of the two lock methods does a read-only operation take?
	return 0
}
