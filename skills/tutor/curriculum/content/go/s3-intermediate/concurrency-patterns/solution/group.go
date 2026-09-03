package patterns

import (
	"context"
	"sync"
)

// Group is a miniature errgroup: it runs functions in goroutines, remembers
// the first error, cancels its context on that first error, and waits for
// every function to finish. Create one with WithContext.
type Group struct {
	wg     sync.WaitGroup
	cancel context.CancelFunc
	once   sync.Once
	err    error
}

// WithContext returns a Group and a context derived from ctx. The first
// error passed to a Go function cancels the derived context; Wait cancels
// it too, so no resources leak on the success path.
func WithContext(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{cancel: cancel}, ctx
}

// Go runs fn in a new goroutine tracked by the group. The first non-nil
// error is recorded and cancels the group context; later errors are dropped.
func (g *Group) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			g.once.Do(func() {
				g.err = err
				g.cancel()
			})
		}
	}()
}

// Wait blocks until every function started with Go has returned, then
// returns the first error (or nil). It also cancels the group context, so
// callers on the success path don't leak it.
func (g *Group) Wait() error {
	g.wg.Wait()
	g.cancel()
	// Safe to read without the Once: every write to g.err happened before
	// a wg.Done that wg.Wait observed.
	return g.err
}
