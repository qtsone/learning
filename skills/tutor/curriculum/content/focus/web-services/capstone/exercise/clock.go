package board

import "time"

// Clock is the only way this service is allowed to read time. Production wiring
// passes RealClock; the tests pass a clock they advance by hand and a ticker
// they fire by hand, which is what makes session expiry, lease expiry, rate
// limiting and heartbeats deterministic under `go test -race` with no sleeps.
//
// NewTicker returns the channel ticks arrive on and a function that releases the
// ticker; a real *time.Ticker leaks its runtime timer until it is stopped.
type Clock interface {
	Now() time.Time
	NewTicker(d time.Duration) (<-chan time.Time, func())
}

// RealClock is the Clock backed by the operating system.
type RealClock struct{}

// Now reports the current wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }

// NewTicker starts a real ticker firing every d.
func (RealClock) NewTicker(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}

// unixNano encodes a time for storage. The zero time stores as 0, so "no lease"
// and "no deadline" are a plain zero in the database.
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
