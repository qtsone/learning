package jobs

import "time"

// Clock is the only way this package is allowed to read the current time.
// Production wiring passes RealClock; tests pass a clock they can advance by
// hand, which is what makes lease expiry, delayed jobs and backoff assertions
// deterministic under `go test -race` with no sleeps anywhere.
type Clock interface {
	Now() time.Time
}

// RealClock is the Clock backed by the operating system.
type RealClock struct{}

// Now reports the current wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }

// unixNano encodes a time for storage. The zero time stores as 0 so that
// "no lease" and "no deadline" are a plain zero in the database.
func unixNano(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// fromUnixNano decodes a stored time, mapping 0 back to the zero time.
func fromUnixNano(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n).UTC()
}
