// Package list implements a singly linked list of ints from scratch.
package list

// node is one link in the chain: a value plus a reference to the next node.
// The last node's next is nil.
type node struct {
	value int
	next  *node
}

// List is a singly linked list. head and tail let both ends be reached in
// O(1); length makes Len O(1) instead of an O(n) walk.
type List struct {
	head   *node
	tail   *node
	length int
}

// New returns an empty list, ready to use.
func New() *List {
	return &List{}
}

// Len reports how many values the list holds. Must be O(1).
func (l *List) Len() int {
	// TODO: return the tracked length (keep it accurate in every method
	// that adds or removes a node).
	return -1
}

// Prepend inserts v at the front of the list in O(1).
func (l *List) Prepend(v int) {
	// TODO: make a new node pointing at the current head and make it the
	// new head. Careful: if the list was empty, this node is also the tail.
}

// Append inserts v at the back of the list. Must be O(1) — use the tail
// pointer, do not walk the list.
func (l *List) Append(v int) {
	// TODO: link a new node after the current tail and make it the new
	// tail. Careful: if the list was empty, this node is also the head.
}

// Contains reports whether v is anywhere in the list.
func (l *List) Contains(v int) bool {
	// TODO: walk from head, following next until nil.
	return false
}

// Delete removes the first node holding v and reports whether anything was
// removed. It must handle deleting the head, the tail, and the only element.
func (l *List) Delete(v int) bool {
	// TODO: walk with two references (previous node and current node) so
	// you can rewire around the match. Update head, tail, and length as
	// needed — deleting the tail must move the tail pointer back.
	return false
}

// Values returns the list's values front-to-back as a slice. An empty list
// yields an empty slice.
func (l *List) Values() []int {
	// TODO: walk the list and collect the values in order.
	return nil
}
