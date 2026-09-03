package main

// Clone returns a new slice with the same elements as s.
// The result shares no backing array with s.
func Clone(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

// Insert returns a new slice with v inserted at index i (0 <= i <= len(s)).
// Inserting at len(s) appends to the end. s itself is never modified.
func Insert(s []int, i, v int) []int {
	out := make([]int, 0, len(s)+1)
	out = append(out, s[:i]...)
	out = append(out, v)
	return append(out, s[i:]...)
}

// Remove returns a new slice with the element at index i removed
// (0 <= i < len(s)). s itself is never modified.
func Remove(s []int, i int) []int {
	out := make([]int, 0, len(s)-1)
	out = append(out, s[:i]...)
	return append(out, s[i+1:]...)
}

// KeepAbove returns the elements of s strictly greater than limit,
// in their original order. s itself is never modified.
func KeepAbove(s []int, limit int) []int {
	var out []int
	for _, v := range s {
		if v > limit {
			out = append(out, v)
		}
	}
	return out
}
