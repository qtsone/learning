package main

import "testing"

// snapshot returns an independent copy of s, so tests can check that a
// function did not modify its input.
func snapshot(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// withSpareCap builds a slice whose backing array has room to grow, so an
// implementation that writes into the input's backing array gets caught by
// the "input unchanged" checks instead of passing by luck.
func withSpareCap(vals ...int) []int {
	s := make([]int, 0, len(vals)+4)
	return append(s, vals...)
}

func TestClone(t *testing.T) {
	orig := []int{1, 2, 3}
	got := Clone(orig)
	if !equal(got, orig) {
		t.Fatalf("Clone(%v) = %v, want %v", orig, got, orig)
	}
	got[0] = 99
	if orig[0] != 1 {
		t.Errorf("mutating the clone changed the original to %v — the two slices share a backing array", orig)
	}
}

func TestCloneEmpty(t *testing.T) {
	if got := Clone([]int{}); len(got) != 0 {
		t.Errorf("Clone of an empty slice has len %d, want 0", len(got))
	}
}

func TestInsert(t *testing.T) {
	cases := []struct {
		name string
		s    []int
		i, v int
		want []int
	}{
		{"front", []int{1, 2, 3}, 0, 9, []int{9, 1, 2, 3}},
		{"middle", []int{1, 2, 3}, 1, 9, []int{1, 9, 2, 3}},
		{"end appends", []int{1, 2, 3}, 3, 9, []int{1, 2, 3, 9}},
		{"empty", []int{}, 0, 9, []int{9}},
		{"spare capacity", withSpareCap(4, 5, 6), 1, 9, []int{4, 9, 5, 6}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := snapshot(c.s)
			got := Insert(c.s, c.i, c.v)
			if !equal(got, c.want) {
				t.Fatalf("Insert(%v, %d, %d) = %v, want %v", before, c.i, c.v, got, c.want)
			}
			if !equal(c.s, before) {
				t.Fatalf("Insert modified its input: %v became %v", before, c.s)
			}
			got[0] = -1
			if !equal(c.s, before) {
				t.Errorf("Insert's result shares a backing array with the input: mutating it changed %v to %v", before, c.s)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	cases := []struct {
		name string
		s    []int
		i    int
		want []int
	}{
		{"front", []int{1, 2, 3}, 0, []int{2, 3}},
		{"middle", []int{1, 2, 3}, 1, []int{1, 3}},
		{"last", []int{1, 2, 3}, 2, []int{1, 2}},
		{"single element", []int{7}, 0, []int{}},
		{"spare capacity", withSpareCap(4, 5, 6), 1, []int{4, 6}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := snapshot(c.s)
			got := Remove(c.s, c.i)
			if !equal(got, c.want) {
				t.Fatalf("Remove(%v, %d) = %v, want %v", before, c.i, got, c.want)
			}
			if !equal(c.s, before) {
				t.Fatalf("Remove modified its input: %v became %v", before, c.s)
			}
			if len(got) > 0 {
				got[0] = -1
				if !equal(c.s, before) {
					t.Errorf("Remove's result shares a backing array with the input: mutating it changed %v to %v", before, c.s)
				}
			}
		})
	}
}

func TestKeepAbove(t *testing.T) {
	cases := []struct {
		name  string
		s     []int
		limit int
		want  []int
	}{
		{"some kept", []int{72, 90, 45, 88}, 80, []int{90, 88}},
		{"boundary is excluded", []int{80, 81}, 80, []int{81}},
		{"all kept", []int{5, 6, 7}, 0, []int{5, 6, 7}},
		{"none kept", []int{1, 2, 3}, 10, []int{}},
		{"empty input", []int{}, 10, []int{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := snapshot(c.s)
			got := KeepAbove(c.s, c.limit)
			if !equal(got, c.want) {
				t.Fatalf("KeepAbove(%v, %d) = %v, want %v", before, c.limit, got, c.want)
			}
			if !equal(c.s, before) {
				t.Fatalf("KeepAbove modified its input: %v became %v", before, c.s)
			}
			if len(got) > 0 {
				got[0] = -1
				if !equal(c.s, before) {
					t.Errorf("KeepAbove's result shares a backing array with the input: mutating it changed %v to %v", before, c.s)
				}
			}
		})
	}
}
