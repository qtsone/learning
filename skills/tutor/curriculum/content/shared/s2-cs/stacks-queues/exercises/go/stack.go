// Package sq implements a stack and a queue from scratch.
package sq

// Stack is a last-in, first-out collection of runes backed by a slice.
// The zero value is an empty stack, ready to use.
type Stack struct {
	items []rune
}

// Push places r on top of the stack.
func (s *Stack) Push(r rune) {
	// TODO: append r to s.items.
}

// Pop removes and returns the top element.
// The second return value is false if the stack is empty.
func (s *Stack) Pop() (rune, bool) {
	// TODO: guard the empty case, then return the last element of s.items
	// and shrink the slice by one.
	return 0, false
}

// Peek returns the top element without removing it.
// The second return value is false if the stack is empty.
func (s *Stack) Peek() (rune, bool) {
	// TODO: like Pop, but leave the stack unchanged.
	return 0, false
}

// Len reports how many elements are on the stack.
func (s *Stack) Len() int {
	// TODO
	return 0
}
