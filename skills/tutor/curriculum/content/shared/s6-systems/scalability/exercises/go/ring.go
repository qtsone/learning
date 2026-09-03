package scalability

import (
	"crypto/sha256"
	"encoding/binary"
)

// Ring is a consistent-hash ring with virtual nodes. Each physical node is
// placed on the ring replicas times (its virtual nodes); a key belongs to
// the first virtual node clockwise from the key's hash.
type Ring struct {
	replicas int
	hashes   []uint64          // sorted virtual-node positions
	owner    map[uint64]string // virtual-node position -> physical node name
}

func NewRing(replicas int) *Ring {
	return &Ring{replicas: replicas, owner: make(map[uint64]string)}
}

// AddNode places name's virtual nodes on the ring.
func (r *Ring) AddNode(name string) {
	// TODO: hash replicas distinct labels derived from name (e.g. "name#0",
	// "name#1", …), record each position's owner, and keep hashes sorted.
}

// RemoveNode takes all of name's virtual nodes off the ring — this is what
// worker failure looks like to the ring.
func (r *Ring) RemoveNode(name string) {
	// TODO: drop every position owned by name from hashes and owner.
}

// Get returns the node that owns key; ok is false when the ring is empty.
func (r *Ring) Get(key string) (string, bool) {
	// TODO: binary-search for the first virtual node at or clockwise past
	// hashOf(key), wrapping past the top of the ring back to index 0.
	return "", false
}

// MovedFraction reports the fraction of keys whose owner differs between two
// assignment snapshots (key -> node). It is 0 for an empty before map.
func MovedFraction(before, after map[string]string) float64 {
	// TODO: count keys whose owner changed and divide by len(before).
	return 0
}

// hashOf is the ring's hash function: the first 8 bytes of SHA-256. It is
// deterministic across runs (stable distribution checks in the tests), and
// its avalanche behavior scatters similar inputs — with a weak hash such as
// FNV, "key-101" and "key-102" land next to each other on the ring and load
// clumps instead of spreading.
func hashOf(s string) uint64 {
	sum := sha256.Sum256([]byte(s))
	return binary.BigEndian.Uint64(sum[:8])
}
