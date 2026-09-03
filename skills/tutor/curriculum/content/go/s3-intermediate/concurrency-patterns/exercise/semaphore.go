package patterns

// Semaphore is a counting semaphore: at most n holders at any moment.
type Semaphore struct {
	// TODO: a buffered channel of struct{} is the entire data structure —
	// its capacity is the limit, its length is the current holder count.
}

// NewSemaphore returns a semaphore admitting at most n concurrent holders.
// n is at least 1.
func NewSemaphore(n int) *Semaphore {
	// TODO: make the channel with capacity n.
	return &Semaphore{}
}

// Acquire takes a slot, blocking while all n are held.
//
// TODO: one channel operation.
func (s *Semaphore) Acquire() {
}

// Release frees a slot previously taken with Acquire or TryAcquire.
//
// TODO: the mirror-image channel operation.
func (s *Semaphore) Release() {
}

// TryAcquire takes a slot only if one is free right now; it reports whether
// it succeeded and never blocks.
//
// TODO: select with a default case.
func (s *Semaphore) TryAcquire() bool {
	return false
}
