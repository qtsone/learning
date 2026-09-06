package heaps

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// checkHeapArray looks straight at the backing slice and verifies the
// array-embedded tree: the parent of index i lives at (i-1)/2, and a
// min-heap demands parent <= child at every index.
func checkHeapArray(t *testing.T, data []int) {
	t.Helper()
	for i := 1; i < len(data); i++ {
		parent := (i - 1) / 2
		if data[parent] > data[i] {
			t.Fatalf("heap property broken in the array: data[%d]=%d (parent) > data[%d]=%d (child)",
				parent, data[parent], i, data[i])
		}
	}
}

func mustPop(t *testing.T, h *MinHeap, want int) {
	t.Helper()
	got, ok := h.Pop()
	if !ok {
		t.Fatalf("Pop() reported empty, want %d", want)
	}
	if got != want {
		t.Fatalf("Pop() = %d, want %d (the current minimum)", got, want)
	}
}

func TestPushPopDrainsAscending(t *testing.T) {
	pushes := []int{42, 7, 93, 15, 7, 3, 27, 81, 50, 12, 99, 34, 3, 61, 76}
	var h MinHeap
	for _, x := range pushes {
		h.Push(x)
	}
	if h.Len() != len(pushes) {
		t.Fatalf("Len() = %d after %d pushes, want %d", h.Len(), len(pushes), len(pushes))
	}
	want := slices.Clone(pushes)
	slices.Sort(want)
	var got []int
	for h.Len() > 0 {
		x, ok := h.Pop()
		if !ok {
			t.Fatalf("Pop() reported empty after %d pops, want %d elements in total", len(got), len(pushes))
		}
		got = append(got, x)
	}
	if !slices.Equal(got, want) {
		t.Errorf("popping everything = %v, want ascending %v (each Pop returns the minimum, duplicates included)", got, want)
	}
}

func TestPeekReturnsMinWithoutRemoving(t *testing.T) {
	var h MinHeap
	if _, ok := h.Peek(); ok {
		t.Error("Peek() on an empty heap reports ok=true, want false")
	}
	h.Push(9)
	h.Push(2)
	h.Push(5)
	if got, ok := h.Peek(); !ok || got != 2 {
		t.Errorf("Peek() = %d, %v, want 2, true (the minimum)", got, ok)
	}
	if h.Len() != 3 {
		t.Errorf("Len() = %d after Peek, want 3 (Peek must not remove)", h.Len())
	}
}

func TestPopOnEmpty(t *testing.T) {
	var h MinHeap
	if _, ok := h.Pop(); ok {
		t.Error("Pop() on an empty heap reports ok=true, want false")
	}
	h.Push(1)
	mustPop(t, &h, 1)
	if _, ok := h.Pop(); ok {
		t.Error("Pop() on a drained heap reports ok=true, want false")
	}
}

func TestInterleavedPushPop(t *testing.T) {
	var h MinHeap
	h.Push(5)
	h.Push(1)
	h.Push(8)
	mustPop(t, &h, 1)
	h.Push(2)
	mustPop(t, &h, 2)
	mustPop(t, &h, 5)
	h.Push(3)
	mustPop(t, &h, 3)
	mustPop(t, &h, 8)
	if h.Len() != 0 {
		t.Errorf("Len() = %d after popping everything, want 0", h.Len())
	}
}

func TestArrayLayoutInvariant(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	var h MinHeap
	for i := 0; i < 200; i++ {
		h.Push(r.IntN(1000))
		checkHeapArray(t, h.data)
	}
	for i := 0; i < 100; i++ {
		if _, ok := h.Pop(); !ok {
			t.Fatalf("Pop() reported empty after %d pops, want 200 elements present", i)
		}
		checkHeapArray(t, h.data)
	}
	if h.Len() != 100 {
		t.Fatalf("Len() = %d after 200 pushes and 100 pops, want 100", h.Len())
	}
}

func TestTopK(t *testing.T) {
	cases := []struct {
		name string
		nums []int
		k    int
		want []int
	}{
		{"basic", []int{5, 1, 9, 3, 7}, 3, []int{9, 7, 5}},
		{"k equals len", []int{2, 8, 4}, 3, []int{8, 4, 2}},
		{"k larger than len", []int{2, 8, 4}, 10, []int{8, 4, 2}},
		{"k zero", []int{1, 2, 3}, 0, nil},
		{"k negative", []int{1, 2, 3}, -1, nil},
		{"duplicates", []int{5, 5, 1, 5, 2}, 3, []int{5, 5, 5}},
		{"empty input", nil, 3, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := TopK(c.nums, c.k)
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("TopK(%v, %d) = %v, want %v (the k largest, descending)", c.nums, c.k, got, c.want)
			}
		})
	}
}

// 100k values, k = 5. A heap bounded at k elements does n·log k work and
// holds 5 ints; sorting a copy would do n·log n and hold all 100k.
func TestTopKLargeStream(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	nums := make([]int, 100_000)
	for i := range nums {
		nums[i] = r.IntN(1_000_000)
	}
	want := slices.Clone(nums)
	slices.Sort(want)
	slices.Reverse(want)
	want = want[:5]
	if got := TopK(nums, 5); !slices.Equal(got, want) {
		t.Errorf("TopK over 100k values = %v, want %v", got, want)
	}
}
