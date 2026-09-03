package scalability

import (
	"fmt"
	"testing"
)

func testKeys(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
	return keys
}

func assignments(t *testing.T, r *Ring, keys []string) map[string]string {
	t.Helper()
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		node, ok := r.Get(k)
		if !ok {
			t.Fatalf("Get(%q) reported ok=false on a non-empty ring", k)
		}
		out[k] = node
	}
	return out
}

func TestEmptyRing(t *testing.T) {
	r := NewRing(100)
	if node, ok := r.Get("anything"); ok {
		t.Fatalf("Get on an empty ring = %q, true; want ok=false", node)
	}
}

func TestSingleNodeOwnsEverything(t *testing.T) {
	r := NewRing(100)
	r.AddNode("only")
	for _, k := range testKeys(50) {
		node, ok := r.Get(k)
		if !ok || node != "only" {
			t.Fatalf("Get(%q) = %q, %v; want %q, true", k, node, ok, "only")
		}
	}
}

func TestGetIsDeterministic(t *testing.T) {
	r := NewRing(100)
	r.AddNode("n1")
	r.AddNode("n2")
	r.AddNode("n3")
	for _, k := range testKeys(200) {
		first, _ := r.Get(k)
		second, _ := r.Get(k)
		if first != second {
			t.Fatalf("Get(%q) returned %q then %q; ownership must be stable", k, first, second)
		}
	}
}

func TestVirtualNodesSpreadLoad(t *testing.T) {
	r := NewRing(200)
	nodes := []string{"n1", "n2", "n3", "n4"}
	for _, n := range nodes {
		r.AddNode(n)
	}
	keys := testKeys(8000)
	counts := map[string]int{}
	for _, node := range assignments(t, r, keys) {
		counts[node]++
	}
	mean := len(keys) / len(nodes)
	for _, n := range nodes {
		if counts[n] < mean*7/10 || counts[n] > mean*13/10 {
			t.Errorf("node %s owns %d of %d keys; want within 30%% of the mean (%d) — "+
				"200 virtual nodes should spread load", n, counts[n], len(keys), mean)
		}
	}
}

func TestFewVirtualNodesMeansHotSpots(t *testing.T) {
	keys := testKeys(8000)
	maxShare := func(replicas int) int {
		r := NewRing(replicas)
		for _, n := range []string{"n1", "n2", "n3", "n4"} {
			r.AddNode(n)
		}
		counts := map[string]int{}
		for _, node := range assignments(t, r, keys) {
			counts[node]++
		}
		max := 0
		for _, c := range counts {
			if c > max {
				max = c
			}
		}
		return max
	}
	if one, many := maxShare(1), maxShare(200); one <= many {
		t.Fatalf("hottest node owns %d keys with 1 virtual node but %d with 200 — "+
			"more virtual nodes should smooth the arcs, not roughen them", one, many)
	}
}

func TestAddNodeMovesOnlyToNewNode(t *testing.T) {
	r := NewRing(100)
	nodes := []string{"n1", "n2", "n3", "n4", "n5"}
	for _, n := range nodes {
		r.AddNode(n)
	}
	keys := testKeys(5000)
	before := assignments(t, r, keys)

	r.AddNode("n6")
	after := assignments(t, r, keys)

	for _, k := range keys {
		if before[k] != after[k] && after[k] != "n6" {
			t.Fatalf("key %q moved from %q to %q when n6 joined — "+
				"keys may only move onto the new node, never between old ones", k, before[k], after[k])
		}
	}

	frac := MovedFraction(before, after)
	if frac < 0.08 || frac > 0.30 {
		t.Errorf("MovedFraction after adding a 6th node = %.3f; want roughly 1/6 (accepted: 0.08-0.30)", frac)
	}

	// Contrast with mod-N sharding, where the same change reshuffles nearly
	// every key.
	modOwner := func(n int, key string) string {
		return nodes6()[hashOf(key)%uint64(n)]
	}
	modBefore, modAfter := map[string]string{}, map[string]string{}
	for _, k := range keys {
		modBefore[k] = modOwner(5, k)
		modAfter[k] = modOwner(6, k)
	}
	if modMoved := MovedFraction(modBefore, modAfter); frac >= modMoved/2 {
		t.Errorf("consistent hashing moved %.3f of keys, hash-mod-N moved %.3f — "+
			"the ring should move far fewer", frac, modMoved)
	}
}

func nodes6() []string {
	return []string{"n1", "n2", "n3", "n4", "n5", "n6"}
}

func TestRemoveNodeMovesOnlyItsKeys(t *testing.T) {
	r := NewRing(100)
	for _, n := range []string{"n1", "n2", "n3", "n4", "n5"} {
		r.AddNode(n)
	}
	keys := testKeys(5000)
	before := assignments(t, r, keys)

	r.RemoveNode("n3") // the worker died
	after := assignments(t, r, keys)

	deadShare := 0
	for _, k := range keys {
		switch {
		case before[k] == "n3":
			deadShare++
			if after[k] == "n3" {
				t.Fatalf("key %q still assigned to removed node n3", k)
			}
		case before[k] != after[k]:
			t.Fatalf("key %q moved from %q to %q although n3 died — "+
				"only the dead node's keys may move", k, before[k], after[k])
		}
	}

	want := float64(deadShare) / float64(len(keys))
	if got := MovedFraction(before, after); got != want {
		t.Errorf("MovedFraction after removing n3 = %.4f; want exactly n3's share %.4f", got, want)
	}
}

func TestAddThenRemoveRestoresAssignment(t *testing.T) {
	r := NewRing(100)
	for _, n := range []string{"n1", "n2", "n3"} {
		r.AddNode(n)
	}
	keys := testKeys(1000)
	before := assignments(t, r, keys)

	r.AddNode("tmp")
	r.RemoveNode("tmp")
	after := assignments(t, r, keys)

	for _, k := range keys {
		if before[k] != after[k] {
			t.Fatalf("key %q owned by %q before tmp joined but %q after it left — "+
				"add then remove must restore the original assignment", k, before[k], after[k])
		}
	}
}

func TestMovedFraction(t *testing.T) {
	cases := []struct {
		name          string
		before, after map[string]string
		want          float64
	}{
		{"empty", map[string]string{}, map[string]string{}, 0},
		{"none moved",
			map[string]string{"a": "n1", "b": "n2"},
			map[string]string{"a": "n1", "b": "n2"}, 0},
		{"one of three",
			map[string]string{"a": "n1", "b": "n1", "c": "n2"},
			map[string]string{"a": "n1", "b": "n2", "c": "n2"}, 1.0 / 3.0},
		{"all moved",
			map[string]string{"a": "n1", "b": "n1"},
			map[string]string{"a": "n2", "b": "n2"}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MovedFraction(c.before, c.after); got != c.want {
				t.Errorf("MovedFraction = %v; want %v", got, c.want)
			}
		})
	}
}
