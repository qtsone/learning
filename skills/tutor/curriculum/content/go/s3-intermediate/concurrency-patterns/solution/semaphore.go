package patterns

// Semaphore is a counting semaphore: at most n holders at any moment. The
// buffered channel is the entire data structure — its capacity is the
// limit, its length is the current holder count.
type Semaphore struct {
	slots chan struct{}
}

// NewSemaphore returns a semaphore admitting at most n concurrent holders.
// n is at least 1.
func NewSemaphore(n int) *Semaphore {
	return &Semaphore{slots: make(chan struct{}, n)}
}

// Acquire takes a slot, blocking while all n are held.
func (s *Semaphore) Acquire() {
	s.slots <- struct{}{}
}

// Release frees a slot previously taken with Acquire or TryAcquire.
func (s *Semaphore) Release() {
	<-s.slots
}

// TryAcquire takes a slot only if one is free right now; it reports whether
// it succeeded and never blocks.
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.slots <- struct{}{}:
		return true
	default:
		return false
	}
}
