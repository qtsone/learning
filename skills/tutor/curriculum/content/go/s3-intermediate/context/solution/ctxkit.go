// Package ctxkit is the context-handling core of a small job-processing
// client: await a result until told to stop, run a cancelable worker,
// classify why work stopped, and carry a request ID the idiomatic way.
package ctxkit

import (
	"context"
	"errors"
)

// Await returns the first value to arrive on ch. If ctx is done before a
// value arrives, it returns the zero value and ctx.Err().
func Await(ctx context.Context, ch <-chan string) (string, error) {
	select {
	case v := <-ch:
		return v, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Square reads jobs and sends j*j on results, in order. When jobs is
// closed it returns nil (natural end). When ctx is done it returns
// ctx.Err() promptly — both while waiting for a job and while blocked
// sending a result.
func Square(ctx context.Context, jobs <-chan int, results chan<- int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case j, ok := <-jobs:
			if !ok {
				return nil
			}
			select {
			case results <- j * j:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Retryable reports whether err is worth retrying: true when the work
// ran out of time (transient), false when it was deliberately canceled,
// is nil, or is any other error. err may arrive wrapped.
func Retryable(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// requestIDKey is unexported so no other package can construct it —
// collisions with foreign context values are structurally impossible.
type requestIDKey struct{}

// WithRequestID returns a child of ctx carrying id, retrievable only via
// RequestID — a key from another package must not collide with it.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the request ID carried by ctx and whether one is set.
func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}
