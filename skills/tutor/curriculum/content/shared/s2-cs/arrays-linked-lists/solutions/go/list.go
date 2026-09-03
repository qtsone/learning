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

// Len reports how many values the list holds in O(1).
func (l *List) Len() int {
	return l.length
}

// Prepend inserts v at the front of the list in O(1).
func (l *List) Prepend(v int) {
	n := &node{value: v, next: l.head}
	l.head = n
	if l.tail == nil {
		l.tail = n
	}
	l.length++
}

// Append inserts v at the back of the list in O(1) via the tail pointer.
func (l *List) Append(v int) {
	n := &node{value: v}
	if l.tail == nil {
		l.head = n
	} else {
		l.tail.next = n
	}
	l.tail = n
	l.length++
}

// Contains reports whether v is anywhere in the list.
func (l *List) Contains(v int) bool {
	for cur := l.head; cur != nil; cur = cur.next {
		if cur.value == v {
			return true
		}
	}
	return false
}

// Delete removes the first node holding v and reports whether anything was
// removed.
func (l *List) Delete(v int) bool {
	var prev *node
	for cur := l.head; cur != nil; prev, cur = cur, cur.next {
		if cur.value != v {
			continue
		}
		if prev == nil {
			l.head = cur.next
		} else {
			prev.next = cur.next
		}
		if cur == l.tail {
			l.tail = prev
		}
		l.length--
		return true
	}
	return false
}

// Values returns the list's values front-to-back as a slice.
func (l *List) Values() []int {
	out := make([]int, 0, l.length)
	for cur := l.head; cur != nil; cur = cur.next {
		out = append(out, cur.value)
	}
	return out
}
