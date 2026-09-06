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
	h := offset64
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime64
	}
	return h
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
	i := h.bucketIndex(key)
	for e := h.buckets[i]; e != nil; e = e.next {
		if e.key == key {
			e.value = value
			return
		}
	}
	if float64(h.size+1)/float64(len(h.buckets)) > maxLoadFactor {
		h.resize()
		i = h.bucketIndex(key)
	}
	h.buckets[i] = &entry{key: key, value: value, next: h.buckets[i]}
	h.size++
}

// resize doubles the bucket count and rehashes every entry: bucket indexes
// are hash mod bucket-count, so all of them change with the count.
func (h *HashTable) resize() {
	old := h.buckets
	h.buckets = make([]*entry, 2*len(old))
	for _, e := range old {
		for e != nil {
			next := e.next
			i := h.bucketIndex(e.key)
			e.next = h.buckets[i]
			h.buckets[i] = e
			e = next
		}
	}
}

// Get returns the value stored under key and whether the key exists.
func (h *HashTable) Get(key string) (int, bool) {
	for e := h.buckets[h.bucketIndex(key)]; e != nil; e = e.next {
		if e.key == key {
			return e.value, true
		}
	}
	return 0, false
}

// Delete removes key and reports whether it was present.
func (h *HashTable) Delete(key string) bool {
	i := h.bucketIndex(key)
	var prev *entry
	for e := h.buckets[i]; e != nil; e = e.next {
		if e.key == key {
			if prev == nil {
				h.buckets[i] = e.next
			} else {
				prev.next = e.next
			}
			h.size--
			return true
		}
		prev = e
	}
	return false
}
