// Package heaps implements a binary min-heap from scratch and a small
// priority queue on top of Go's container/heap.
package heaps

// MinHeap is a binary min-heap of ints stored compactly in a slice:
// the children of index i sit at 2i+1 and 2i+2, its parent at (i-1)/2.
// The zero value is an empty, ready-to-use heap.
type MinHeap struct {
	data []int
}

// Len returns the number of elements currently in the heap.
func (h *MinHeap) Len() int { return len(h.data) }

// Peek returns the smallest element without removing it.
// The second return is false when the heap is empty.
func (h *MinHeap) Peek() (int, bool) {
	if len(h.data) == 0 {
		return 0, false
	}
	return h.data[0], true
}

// Push adds x to the heap.
func (h *MinHeap) Push(x int) {
	h.data = append(h.data, x)
	h.siftUp(len(h.data) - 1)
}

// Pop removes and returns the smallest element.
// The second return is false when the heap is empty.
func (h *MinHeap) Pop() (int, bool) {
	if len(h.data) == 0 {
		return 0, false
	}
	root := h.data[0]
	last := len(h.data) - 1
	h.data[0] = h.data[last]
	h.data = h.data[:last]
	h.siftDown(0)
	return root, true
}

// siftUp moves the element at index i toward the root, swapping with its
// parent while the parent is larger.
func (h *MinHeap) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h.data[parent] <= h.data[i] {
			return
		}
		h.data[i], h.data[parent] = h.data[parent], h.data[i]
		i = parent
	}
}

// siftDown moves the element at index i toward the leaves, swapping with
// its SMALLER child while either child is smaller.
func (h *MinHeap) siftDown(i int) {
	for {
		smallest := i
		if l := 2*i + 1; l < len(h.data) && h.data[l] < h.data[smallest] {
			smallest = l
		}
		if r := 2*i + 2; r < len(h.data) && h.data[r] < h.data[smallest] {
			smallest = r
		}
		if smallest == i {
			return
		}
		h.data[i], h.data[smallest] = h.data[smallest], h.data[i]
		i = smallest
	}
}

// TopK returns the k largest values in nums, in descending order.
// k <= 0 yields an empty result; k >= len(nums) yields all values.
// Use a MinHeap holding at most k elements: O(n log k) time, O(k) space.
func TopK(nums []int, k int) []int {
	if k <= 0 {
		return nil
	}
	var h MinHeap
	for _, x := range nums {
		h.Push(x)
		if h.Len() > k {
			h.Pop()
		}
	}
	out := make([]int, h.Len())
	for i := len(out) - 1; i >= 0; i-- {
		out[i], _ = h.Pop()
	}
	return out
}
