package patterns

import "context"

// Group is a miniature errgroup: it runs functions in goroutines, remembers
// the first error, cancels its context on that first error, and waits for
// every function to finish. Create one with WithContext.
type Group struct {
	// TODO: you need a sync.WaitGroup to track the goroutines, a sync.Once
	// plus an error field to remember only the FIRST failure, and the
	// context.CancelFunc for the derived context.
}

// WithContext returns a Group and a context derived from ctx. The first
// error passed to a Go function cancels the derived context; Wait cancels
// it too, so no resources leak on the success path.
func WithContext(ctx context.Context) (*Group, context.Context) {
	// TODO: derive a cancelable context and hand its cancel to the Group.
	return &Group{}, ctx
}

// Go runs fn in a new goroutine tracked by the group.
//
// TODO: register with the WaitGroup before starting the goroutine. If fn
// returns a non-nil error, record it and cancel the group context — but
// only for the first error; later ones are dropped.
func (g *Group) Go(fn func() error) {
}

// Wait blocks until every function started with Go has returned, then
// returns the first error (or nil). It also cancels the group context, so
// callers on the success path don't leak it.
//
// TODO: wait, cancel, return the recorded error.
func (g *Group) Wait() error {
	return nil
}
