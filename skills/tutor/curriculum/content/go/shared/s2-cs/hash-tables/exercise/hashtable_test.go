package hashtable

import (
	"fmt"
	"testing"
)

func TestFNV1a(t *testing.T) {
	// Published FNV-1a 64-bit vectors — cross-check with hash/fnv if curious.
	cases := []struct {
		in   string
		want uint64
	}{
		{"", 14695981039346656037},
		{"a", 12638187200555641996},
		{"b", 12638190499090526629},
		{"foobar", 9625390261332436968},
		{"hash table", 16255662506852573157},
		{"Go", 649144001569241323},
	}
	for _, c := range cases {
		if got := fnv1a(c.in); got != c.want {
			t.Errorf("fnv1a(%q) = %d, want %d — start from the offset basis, then XOR each byte in BEFORE multiplying by the prime", c.in, got, c.want)
		}
	}
}

func TestPutGet(t *testing.T) {
	h := NewHashTable()
	h.Put("alice", 30)
	h.Put("bob", 25)
	if got, ok := h.Get("alice"); !ok || got != 30 {
		t.Errorf(`Get("alice") = (%d, %v), want (30, true)`, got, ok)
	}
	if got, ok := h.Get("bob"); !ok || got != 25 {
		t.Errorf(`Get("bob") = (%d, %v), want (25, true)`, got, ok)
	}
	if h.Len() != 2 {
		t.Errorf("Len() = %d after 2 distinct Puts, want 2", h.Len())
	}
}

func TestZeroValueVsMissing(t *testing.T) {
	h := NewHashTable()
	h.Put("zero", 0)
	if got, ok := h.Get("zero"); !ok || got != 0 {
		t.Errorf(`Get("zero") = (%d, %v), want (0, true) — a stored zero is not a missing key`, got, ok)
	}
	if got, ok := h.Get("missing"); ok || got != 0 {
		t.Errorf(`Get("missing") = (%d, %v), want (0, false)`, got, ok)
	}
}

func TestPutReplacesExistingKey(t *testing.T) {
	h := NewHashTable()
	h.Put("counter", 1)
	h.Put("counter", 2)
	if got, ok := h.Get("counter"); !ok || got != 2 {
		t.Errorf(`Get("counter") = (%d, %v) after a second Put, want (2, true)`, got, ok)
	}
	if h.Len() != 1 {
		t.Errorf("Len() = %d after putting the same key twice, want 1 — Put must replace in place, not append a duplicate", h.Len())
	}
}

func TestDelete(t *testing.T) {
	h := NewHashTable()
	h.Put("alice", 1)
	h.Put("bob", 2)
	if !h.Delete("alice") {
		t.Fatalf(`Delete("alice") = false, want true for a present key`)
	}
	if _, ok := h.Get("alice"); ok {
		t.Errorf(`Get("alice") still succeeds after Delete("alice")`)
	}
	if h.Len() != 1 {
		t.Errorf("Len() = %d after deleting 1 of 2 keys, want 1", h.Len())
	}
	if h.Delete("alice") {
		t.Errorf(`Delete("alice") a second time = true, want false — the key is already gone`)
	}
	if got, ok := h.Get("bob"); !ok || got != 2 {
		t.Errorf(`Get("bob") = (%d, %v) after deleting a different key, want (2, true)`, got, ok)
	}
}

func TestDeleteFromMiddleOfChain(t *testing.T) {
	// "banana", "cherry", and "grape" all hash to the same bucket in an
	// 8-bucket table, so this forces a 3-entry chain.
	h := NewHashTable()
	h.Put("banana", 1)
	h.Put("cherry", 2)
	h.Put("grape", 3)
	if !h.Delete("cherry") {
		t.Fatalf(`Delete("cherry") = false, want true`)
	}
	if got, ok := h.Get("banana"); !ok || got != 1 {
		t.Errorf(`Get("banana") = (%d, %v) after deleting its chain neighbor, want (1, true) — unlinking must not drop the rest of the chain`, got, ok)
	}
	if got, ok := h.Get("grape"); !ok || got != 3 {
		t.Errorf(`Get("grape") = (%d, %v) after deleting its chain neighbor, want (3, true) — unlinking must not drop the rest of the chain`, got, ok)
	}
	if _, ok := h.Get("cherry"); ok {
		t.Errorf(`Get("cherry") still succeeds after Delete("cherry")`)
	}
	if h.Len() != 2 {
		t.Errorf("Len() = %d after deleting 1 of 3 keys, want 2", h.Len())
	}
}

func TestResizeKeepsEveryKey(t *testing.T) {
	h := NewHashTable()
	const n = 1000
	for i := 0; i < n; i++ {
		h.Put(fmt.Sprintf("key-%d", i), i)
	}
	if h.Len() != n {
		t.Fatalf("Len() = %d after %d distinct Puts, want %d", h.Len(), n, n)
	}
	if len(h.buckets) <= initialBuckets {
		t.Fatalf("still %d buckets after %d inserts — the table never resized", len(h.buckets), n)
	}
	if lf := float64(h.size) / float64(len(h.buckets)); lf > maxLoadFactor {
		t.Errorf("load factor = %.2f, want <= %v — Put must resize before the table gets this full", lf, maxLoadFactor)
	}
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%d", i)
		if got, ok := h.Get(key); !ok || got != i {
			t.Fatalf("Get(%q) = (%d, %v) after resizing, want (%d, true) — did rehashing re-insert every entry under its new bucket index?", key, got, ok, i)
		}
	}
}

func TestChainsStayShort(t *testing.T) {
	h := NewHashTable()
	for i := 0; i < 1000; i++ {
		h.Put(fmt.Sprintf("key-%d", i), i)
	}
	if got := maxChainLen(h); got > 12 {
		t.Errorf("longest chain has %d entries — with a uniform hash and resizing, chains stay a few entries long; long chains are exactly what turns O(1) average into O(n)", got)
	}
}

func maxChainLen(h *HashTable) int {
	longest := 0
	for _, e := range h.buckets {
		n := 0
		for ; e != nil; e = e.next {
			n++
		}
		if n > longest {
			longest = n
		}
	}
	return longest
}
