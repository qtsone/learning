package scalability

// Bounded is a FIFO queue that refuses new items once it holds capacity
// items — the backpressure primitive.
type Bounded[T any] struct {
	items    []T
	capacity int
}

func NewBounded[T any](capacity int) *Bounded[T] {
	return &Bounded[T]{capacity: capacity}
}

// Offer appends v and reports true, or reports false when the queue is
// full — the caller sheds the load instead of queueing it.
func (q *Bounded[T]) Offer(v T) bool {
	// TODO: reject when Len() == capacity; otherwise append.
	return false
}

// Take removes and returns the oldest item; ok is false when empty.
func (q *Bounded[T]) Take() (T, bool) {
	// TODO: remove and return the oldest item — without leaving it reachable
	// afterwards (S2 named `q.items = q.items[1:]` as the trap).
	var zero T
	return zero, false
}

func (q *Bounded[T]) Len() int {
	// TODO
	return 0
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
	// TODO: run the loop above and tally the LoadResult.
	return LoadResult{}
}
