package apiperf

import "time"

// Clock is the only way this package is allowed to read the current time.
// Production wiring passes RealClock; tests pass a clock they advance by hand,
// which is what makes cache expiry assertable under `go test -race` with no
// sleeps anywhere. You met the pattern in S5 and used it again for the rate
// limiter in the hardening lesson.
type Clock interface {
	Now() time.Time
}

// RealClock is the Clock backed by the operating system.
type RealClock struct{}

// Now reports the current wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }

// unixNano encodes a time for storage. Times go into SQLite as integers rather
// than as datetime strings so that ordering, the keyset comparison and the
// index all agree on one representation.
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
