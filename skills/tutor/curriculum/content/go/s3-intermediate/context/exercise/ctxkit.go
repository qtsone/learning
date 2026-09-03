// Package ctxkit is the context-handling core of a small job-processing
// client: await a result until told to stop, run a cancelable worker,
// classify why work stopped, and carry a request ID the idiomatic way.
package ctxkit

import (
	"context"
)

// Await returns the first value to arrive on ch. If ctx is done before a
// value arrives, it returns the zero value and ctx.Err().
func Await(ctx context.Context, ch <-chan string) (string, error) {
	// TODO: race the two channels against each other with select.
	return "", nil
}

// Square reads jobs and sends j*j on results, in order. When jobs is
// closed it returns nil (natural end). When ctx is done it returns
// ctx.Err() promptly — both while waiting for a job and while blocked
// sending a result.
func Square(ctx context.Context, jobs <-chan int, results chan<- int) error {
	// TODO: the worker-loop shape from LESSON.md — cancellation must
	// cover every blocking operation, the send included.
	return nil
}

// Retryable reports whether err is worth retrying: true when the work
// ran out of time (transient), false when it was deliberately canceled,
// is nil, or is any other error. err may arrive wrapped.
func Retryable(err error) bool {
	// TODO: compare against the two context sentinel errors — through
	// wrapping, so == won't do.
	return false
}

// WithRequestID returns a child of ctx carrying id, retrievable only via
// RequestID — a key from another package must not collide with it.
func WithRequestID(ctx context.Context, id string) context.Context {
	// TODO: store id under a key no other package can construct.
	return ctx
}

// RequestID returns the request ID carried by ctx and whether one is set.
func RequestID(ctx context.Context) (string, bool) {
	// TODO: retrieve and type-assert with the , ok form — a missing
	// value must not panic.
	return "", false
}
