package realtime

import "time"

// Clock is the only way this package is allowed to read time. Production wiring
// passes RealClock; tests pass a fake whose ticker fires when the test says so,
// which is what makes heartbeat assertions deterministic under `go test -race`.
//
// NewTicker returns the channel ticks arrive on and a function that releases
// the ticker. A real *time.Ticker leaks its runtime timer until it is stopped,
// so every caller defers the stop function.
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
