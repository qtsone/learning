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
	nd := &node{val: v}
	if q.tail == nil {
		q.head = nd
	} else {
		q.tail.next = nd
	}
	q.tail = nd
	q.n++
}

// Dequeue removes and returns the front element.
// The second return value is false if the queue is empty.
func (q *Queue) Dequeue() (int, bool) {
	if q.head == nil {
		return 0, false
	}
	v := q.head.val
	q.head = q.head.next
	if q.head == nil {
		q.tail = nil
	}
	q.n--
	return v, true
}

// Peek returns the front element without removing it.
// The second return value is false if the queue is empty.
func (q *Queue) Peek() (int, bool) {
	if q.head == nil {
		return 0, false
	}
	return q.head.val, true
}

// Len reports how many elements are in the queue.
func (q *Queue) Len() int {
	return q.n
}
