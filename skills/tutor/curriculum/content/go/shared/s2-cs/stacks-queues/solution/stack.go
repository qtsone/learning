// Package sq implements a stack and a queue from scratch.
package sq

// Stack is a last-in, first-out collection of runes backed by a slice.
// The zero value is an empty stack, ready to use.
type Stack struct {
	items []rune
}

// Push places r on top of the stack.
func (s *Stack) Push(r rune) {
	s.items = append(s.items, r)
}

// Pop removes and returns the top element.
// The second return value is false if the stack is empty.
func (s *Stack) Pop() (rune, bool) {
	if len(s.items) == 0 {
		return 0, false
	}
	r := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return r, true
}

// Peek returns the top element without removing it.
// The second return value is false if the stack is empty.
func (s *Stack) Peek() (rune, bool) {
	if len(s.items) == 0 {
		return 0, false
	}
	return s.items[len(s.items)-1], true
}

// Len reports how many elements are on the stack.
func (s *Stack) Len() int {
	return len(s.items)
}
