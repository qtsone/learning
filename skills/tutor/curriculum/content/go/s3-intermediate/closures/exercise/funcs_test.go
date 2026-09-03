package closures

import (
	"slices"
	"testing"
)

func TestMakeAdders(t *testing.T) {
	offsets := []int{0, 10, 100}
	adders := MakeAdders(offsets)
	if len(adders) != len(offsets) {
		t.Fatalf("MakeAdders(%v) returned %d functions, want %d", offsets, len(adders), len(offsets))
	}
	cases := []struct {
		i, in, want int
	}{
		{0, 5, 5},
		{1, 5, 15},
		{2, 5, 105},
		{1, -3, 7},
	}
	for _, c := range cases {
		if got := adders[c.i](c.in); got != c.want {
			t.Errorf("adders[%d](%d) = %d, want %d (each closure must capture its own offset)", c.i, c.in, got, c.want)
		}
	}
}

func TestMakeAddersEmpty(t *testing.T) {
	if got := MakeAdders(nil); len(got) != 0 {
		t.Errorf("MakeAdders(nil) returned %d functions, want 0", len(got))
	}
}

func TestFilterInts(t *testing.T) {
	in := []int{1, 2, 3, 4, 5, 6}
	got := Filter(in, func(n int) bool { return n%2 == 0 })
	want := []int{2, 4, 6}
	if !slices.Equal(got, want) {
		t.Errorf("Filter(%v, even) = %v, want %v", in, got, want)
	}
}

func TestFilterStrings(t *testing.T) {
	in := []string{"go", "gopher", "if", "closure"}
	got := Filter(in, func(s string) bool { return len(s) > 2 })
	want := []string{"gopher", "closure"}
	if !slices.Equal(got, want) {
		t.Errorf("Filter(%v, len>2) = %v, want %v (order must be preserved)", in, got, want)
	}
}

func TestFilterKeepNone(t *testing.T) {
	got := Filter([]int{1, 3, 5}, func(n int) bool { return n%2 == 0 })
	if len(got) != 0 {
		t.Errorf("Filter with nothing to keep returned %v, want an empty result", got)
	}
}
