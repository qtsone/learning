package sq

type node struct {
	val  int
	next *node
}

// Queue is a first-in, first-out collection of ints backed by linked nodes.
// The zero value is an empty queue, ready to use.
type Queue struct {
	head *node
	tail *node
	n    int
}

// Enqueue adds v at the back of the queue.
func (q *Queue) Enqueue(v int) {
	// TODO: create a node and attach it after tail (it becomes both head
	// and tail when the queue is empty).
}

// Dequeue removes and returns the front element.
// The second return value is false if the queue is empty.
func (q *Queue) Dequeue() (int, bool) {
	// TODO: guard the empty case, advance head, and remember to reset tail
	// when you remove the last element.
	return 0, false
}

// Peek returns the front element without removing it.
// The second return value is false if the queue is empty.
func (q *Queue) Peek() (int, bool) {
	// TODO
	return 0, false
}

// Len reports how many elements are in the queue.
func (q *Queue) Len() int {
	// TODO
	return 0
}
