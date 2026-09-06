// Package hashtable is a separate-chaining hash table built from scratch —
// the built-in map must not appear anywhere in this file.
package hashtable

// FNV-1a 64-bit parameters (see LESSON.md and the canonical FNV reference).
const (
	offset64 uint64 = 14695981039346656037
	prime64  uint64 = 1099511628211
)

const (
	initialBuckets = 8
	maxLoadFactor  = 0.75
)

// fnv1a returns the 64-bit FNV-1a hash of key: start from offset64, then for
// each byte XOR it into the hash and multiply by prime64.
func fnv1a(key string) uint64 {
	// TODO: implement the FNV-1a recipe from LESSON.md.
	return 0
}

// entry is one key/value pair in a bucket's chain.
// The tests inspect these fields — do not rename or remove them.
type entry struct {
	key   string
	value int
	next  *entry
}

// HashTable maps string keys to int values using separate chaining.
// The tests inspect buckets and size — do not rename or remove them.
type HashTable struct {
	buckets []*entry
	size    int
}

// NewHashTable returns an empty table with initialBuckets buckets.
func NewHashTable() *HashTable {
	return &HashTable{buckets: make([]*entry, initialBuckets)}
}

// Len returns the number of stored keys.
func (h *HashTable) Len() int { return h.size }

// bucketIndex returns the bucket key currently belongs to.
func (h *HashTable) bucketIndex(key string) int {
	return int(fnv1a(key) % uint64(len(h.buckets)))
}

// Put stores value under key, replacing the old value if key already exists.
// When storing a NEW key would push the load factor past maxLoadFactor, it
// first doubles the bucket count and rehashes every entry.
func (h *HashTable) Put(key string, value int) {
	// TODO: walk the key's chain; if the key exists, update it in place.
	// TODO: otherwise link in a new entry and grow size — resizing first
	// if (size+1)/buckets would exceed maxLoadFactor.
}

// Get returns the value stored under key and whether the key exists.
func (h *HashTable) Get(key string) (int, bool) {
	// TODO: hash to a bucket, walk its chain comparing keys.
	return 0, false
}

// Delete removes key and reports whether it was present.
func (h *HashTable) Delete(key string) bool {
	// TODO: unlink the entry from its chain (careful with the middle of a
	// chain — its neighbors must survive) and decrement size.
	return false
}
