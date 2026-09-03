package bigo

import (
	"fmt"
	"hash/fnv"
	"testing"
)

// The expected classes are stored as fingerprints, not plaintext, so this
// file can't hand you the answers — derive each class by counting
// iterations. A wrong entry fails with a hint, never with the answer.
func fingerprint(fn, class string) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s=%s", fn, class)
	return fmt.Sprintf("%016x", h.Sum64())
}

func TestComplexities(t *testing.T) {
	valid := map[string]bool{
		O1: true, OLogN: true, ON: true, ONLogN: true, ON2: true, O2N: true,
	}
	cases := []struct {
		fn   string
		want string // fingerprint of the correct entry
		hint string
	}{
		{"First", "1b34dfa398241085", "how much work for 3 elements? for 3 million?"},
		{"Sum", "01fd0c4dd78333b1", "one loop, one visit per element"},
		{"SumTwice", "230ecb263b02a0cd", "sequential passes ADD — what does 2n simplify to?"},
		{"HasPairSum", "181d2aad8eb2a59b", "for every element, look at every later element; count the pairs in the worst case"},
		{"Halving", "d9ec60824ac8be0f", "each iteration halves n — how many halvings from n to 1?"},
		{"HalvingPerItem", "37bd3270145fa809", "a halving loop runs once per element; nested loops MULTIPLY"},
		{"FirstTen", "b6eb4396047a08d9", "look hard at the inner loop's bound — does it scale with the input?"},
		{"Combos", "26788b28c3b2b1f3", "total doubles once per item, then the loop runs total times"},
	}
	if len(Complexities) != len(cases) {
		t.Fatalf("Complexities has %d entries, want %d — don't add or remove keys", len(Complexities), len(cases))
	}
	for _, c := range cases {
		t.Run(c.fn, func(t *testing.T) {
			got, ok := Complexities[c.fn]
			if !ok {
				t.Fatalf("Complexities has no entry for %q — keep the original keys", c.fn)
			}
			if got == "" {
				t.Fatalf("Complexities[%q] is empty — fill it with one of O1, OLogN, ON, ONLogN, ON2, O2N", c.fn)
			}
			if !valid[got] {
				t.Fatalf("Complexities[%q] = %q — use one of the six constants (O1, OLogN, ON, ONLogN, ON2, O2N), not a hand-written string", c.fn, got)
			}
			if fingerprint(c.fn, got) != c.want {
				t.Errorf("Complexities[%q] = %q is not the right class — hint: %s", c.fn, got, c.hint)
			}
		})
	}
}
