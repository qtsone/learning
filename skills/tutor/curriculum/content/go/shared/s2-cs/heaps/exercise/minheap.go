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
	// TODO: the minimum of a min-heap lives at one known index.
	return 0, false
}

// Push adds x to the heap.
func (h *MinHeap) Push(x int) {
	// TODO: append x (keeps the shape complete), then restore the heap
	// property with siftUp.
}

// Pop removes and returns the smallest element.
// The second return is false when the heap is empty.
func (h *MinHeap) Pop() (int, bool) {
	// TODO: save the root, move the LAST element into its place, shrink
	// by one, then restore the heap property with siftDown.
	return 0, false
}

// siftUp moves the element at index i toward the root, swapping with its
// parent while the parent is larger.
func (h *MinHeap) siftUp(i int) {
	// TODO: the parent of i is (i-1)/2; stop at the root or when the
	// parent is no larger.
}

// siftDown moves the element at index i toward the leaves, swapping with
// its SMALLER child while either child is smaller.
func (h *MinHeap) siftDown(i int) {
	// TODO: children of i are 2i+1 and 2i+2 — mind indices past the end;
	// stop when no child is smaller.
}

// TopK returns the k largest values in nums, in descending order.
// k <= 0 yields an empty result; k >= len(nums) yields all values.
// Use a MinHeap holding at most k elements: O(n log k) time, O(k) space.
func TopK(nums []int, k int) []int {
	// TODO: push each value; whenever the heap grows past k, pop (evicting
	// the weakest survivor). Then drain the heap into the result
	// back-to-front so the largest value ends up first.
	return nil
}
