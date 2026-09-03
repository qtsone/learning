package main

// Clone returns a new slice with the same elements as s.
// The result shares no backing array with s.
func Clone(s []int) []int {
	// TODO: allocate a slice of the right length and copy the elements over.
	return nil
}

// Insert returns a new slice with v inserted at index i (0 <= i <= len(s)).
// Inserting at len(s) appends to the end. s itself is never modified.
func Insert(s []int, i, v int) []int {
	// TODO: build a fresh slice out of s[:i], v, and s[i:].
	return nil
}

// Remove returns a new slice with the element at index i removed
// (0 <= i < len(s)). s itself is never modified.
func Remove(s []int, i int) []int {
	// TODO: build a fresh slice out of s[:i] and s[i+1:].
	return nil
}

// KeepAbove returns the elements of s strictly greater than limit,
// in their original order. s itself is never modified.
func KeepAbove(s []int, limit int) []int {
	// TODO: loop over s with range and append the elements you keep.
	return nil
}
