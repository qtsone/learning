package logkit

import (
	"context"
	"errors"
	"time"
)

// ErrPermanent marks a failure that retrying cannot fix. Wrap it (with %w) in
// errors that must stop the loop immediately.
var ErrPermanent = errors.New("permanent failure")

// Clock is the only way the retry loop is allowed to touch time. Production
// code passes RealClock; tests pass a fake that records durations instead of
// sleeping, which is what keeps them instant and deterministic.
type Clock interface {
	// Sleep blocks for d, or until ctx is done — in which case it returns
	// ctx.Err() and stops waiting.
	Sleep(ctx context.Context, d time.Duration) error
}

// RealClock is the Clock backed by the operating system.
type RealClock struct{}

// Sleep waits for d or for ctx, whichever comes first.
func (RealClock) Sleep(ctx context.Context, d time.Duration) error {
	// TODO: wait for d without ignoring ctx. time.Sleep cannot be canceled;
	// a timer plus select can.
	return nil
}

// Policy configures Retry. Base and Max must both be positive.
type Policy struct {
	MaxAttempts int           // total calls of fn, including the first
	Base        time.Duration // delay after the first failed attempt
	Max         time.Duration // upper bound on any single delay
}

// Backoff returns the delay to wait after the given 1-based attempt failed:
// Base for attempt 1, doubling each time, never exceeding Max — and never
// overflowing, however large attempt gets.
func (p Policy) Backoff(attempt int) time.Duration {
	// TODO: implement capped exponential backoff.
	return 0
}

// Retry calls fn until it returns nil, the attempt budget is spent, or ctx is
// done, sleeping p.Backoff(attempt) on clk between attempts. It never sleeps
// after the final attempt. An error wrapping ErrPermanent stops the loop at
// once and is returned unchanged. When ctx ends the loop, the returned error
// matches both the context error and the last attempt's error.
//
// A Policy with MaxAttempts below 1 is a caller bug: Retry reports it as an
// error and never calls fn.
func Retry(ctx context.Context, clk Clock, p Policy, fn func(context.Context) error) error {
	// TODO: implement the loop described above.
	return nil
}
