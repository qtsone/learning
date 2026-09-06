package scalability

// Bounded is a FIFO queue that refuses new items once it holds capacity
// items — the backpressure primitive.
//
// The storage is the ring buffer S2's stacks-and-queues lesson pointed at:
// capacity slots allocated once, reused in a circle. The tempting one-liner
// there, `q.items = q.items[1:]`, is O(1) but leaves every taken item
// reachable through the backing array — in a Bounded[*Job] that pins whole
// jobs long after they were served, in the one type whose entire promise is
// that it never grows. Take zeroes the slot it vacates instead, so a taken
// item is garbage immediately.
type Bounded[T any] struct {
	items []T // capacity slots; head walks around them
	head  int // index of the oldest item
	count int
}

func NewBounded[T any](capacity int) *Bounded[T] {
	return &Bounded[T]{items: make([]T, capacity)}
}

// Offer stores v and reports true, or reports false when the queue is
// full — the caller sheds the load instead of queueing it.
func (q *Bounded[T]) Offer(v T) bool {
	if q.count == len(q.items) {
		return false
	}
	q.items[(q.head+q.count)%len(q.items)] = v
	q.count++
	return true
}

// Take removes and returns the oldest item; ok is false when empty.
func (q *Bounded[T]) Take() (T, bool) {
	var zero T
	if q.count == 0 {
		return zero, false
	}
	v := q.items[q.head]
	q.items[q.head] = zero // drop the queue's reference, not just its index
	q.head = (q.head + 1) % len(q.items)
	q.count--
	return v, true
}

func (q *Bounded[T]) Len() int {
	return q.count
}

// LoadResult describes one simulated overload run.
type LoadResult struct {
	Served   int // items a worker completed
	Shed     int // items rejected by a full queue
	MaxQueue int // largest queue length observed right after an arrival burst
	Backlog  int // items still queued when the run ends
}

// SimulateLoad drives a Bounded[int] of the given capacity for ticks rounds.
// Each tick, arrivalsPerTick items arrive and are Offered one at a time
// (every failed Offer counts as one shed item); then a worker Takes up to
// servicePerTick items. MaxQueue samples Len after each arrival burst,
// before the worker runs.
func SimulateLoad(capacity, arrivalsPerTick, servicePerTick, ticks int) LoadResult {
	q := NewBounded[int](capacity)
	var res LoadResult
	for tick := 0; tick < ticks; tick++ {
		for i := 0; i < arrivalsPerTick; i++ {
			if !q.Offer(tick) {
				res.Shed++
			}
		}
		if q.Len() > res.MaxQueue {
			res.MaxQueue = q.Len()
		}
		for i := 0; i < servicePerTick; i++ {
			if _, ok := q.Take(); !ok {
				break
			}
			res.Served++
		}
	}
	res.Backlog = q.Len()
	return res
}
