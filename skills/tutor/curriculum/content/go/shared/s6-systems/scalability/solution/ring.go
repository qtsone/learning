package scalability

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
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
	for i := 0; i < r.replicas; i++ {
		h := hashOf(fmt.Sprintf("%s#%d", name, i))
		r.owner[h] = name
		r.hashes = append(r.hashes, h)
	}
	slices.Sort(r.hashes)
}

// RemoveNode takes all of name's virtual nodes off the ring — this is what
// worker failure looks like to the ring.
func (r *Ring) RemoveNode(name string) {
	r.hashes = slices.DeleteFunc(r.hashes, func(h uint64) bool {
		return r.owner[h] == name
	})
	for h, n := range r.owner {
		if n == name {
			delete(r.owner, h)
		}
	}
}

// Get returns the node that owns key; ok is false when the ring is empty.
func (r *Ring) Get(key string) (string, bool) {
	if len(r.hashes) == 0 {
		return "", false
	}
	i, _ := slices.BinarySearch(r.hashes, hashOf(key))
	if i == len(r.hashes) {
		i = 0 // wrap past the top of the ring
	}
	return r.owner[r.hashes[i]], true
}

// MovedFraction reports the fraction of keys whose owner differs between two
// assignment snapshots (key -> node). It is 0 for an empty before map.
func MovedFraction(before, after map[string]string) float64 {
	if len(before) == 0 {
		return 0
	}
	moved := 0
	for k, owner := range before {
		if after[k] != owner {
			moved++
		}
	}
	return float64(moved) / float64(len(before))
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
