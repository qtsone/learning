package client

import (
	"context"
	"errors"
	"time"
)

// RetryPolicy bounds how often and how patiently a request is retried.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// Backoff returns the wait ceiling after the given 0-indexed failed
// attempt: BaseDelay doubled per attempt, capped at MaxDelay.
func Backoff(p RetryPolicy, attempt int) time.Duration {
	// TODO: implement exponential backoff with a cap.
	return 0
}

// Jitter returns a uniformly random duration in [0, d].
func Jitter(d time.Duration) time.Duration {
	// TODO: implement full jitter (math/rand/v2 is in the standard library).
	return 0
}

// ShouldRetry reports whether the failure is worth another attempt.
func ShouldRetry(err error) bool {
	// TODO: retry transport errors and 429/5xx APIErrors; never retry
	// other 4xx, context cancellation/expiry, or nil.
	return false
}

// GetJSONRetry is GetJSON with up to p.MaxAttempts attempts, waiting
// Jitter(Backoff(...)) between attempts via c.sleep.
func (c *Client) GetJSONRetry(ctx context.Context, path string, v any, p RetryPolicy) error {
	// TODO: loop attempts; stop early on success, a non-retryable error,
	// or a cancelled context (make no request once ctx is done). Do not
	// wait after the final attempt.
	return errors.New("TODO: implement GetJSONRetry")
}
