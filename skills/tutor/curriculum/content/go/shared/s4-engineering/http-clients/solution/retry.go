package client

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
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
	d := p.BaseDelay
	for i := 0; i < attempt; i++ {
		d *= 2
		// d <= 0 means the doubling overflowed; the cap is the answer.
		if d >= p.MaxDelay || d <= 0 {
			return p.MaxDelay
		}
	}
	if d > p.MaxDelay {
		return p.MaxDelay
	}
	return d
}

// Jitter returns a uniformly random duration in [0, d] (full jitter),
// so simultaneous failures do not become simultaneous retries.
func Jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return rand.N(d + 1)
}

// ShouldRetry reports whether the failure is worth another attempt.
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	// The caller gave up — retrying past their deadline helps nobody.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
	}
	// Anything else is transport-level (refused, reset, timeout): the
	// request may never have arrived, so trying again can succeed.
	return true
}

// GetJSONRetry is GetJSON with up to p.MaxAttempts attempts, waiting
// Jitter(Backoff(...)) between attempts via c.sleep.
func (c *Client) GetJSONRetry(ctx context.Context, path string, v any, p RetryPolicy) error {
	var lastErr error
	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if attempt > 0 {
			c.sleep(Jitter(Backoff(p, attempt-1)))
		}
		lastErr = c.GetJSON(ctx, path, v)
		if lastErr == nil {
			return nil
		}
		if !ShouldRetry(lastErr) {
			return lastErr
		}
	}
	return lastErr
}
