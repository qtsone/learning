package auth

import "time"

// Clock is the only way this package is allowed to read the current time.
// Production wiring passes RealClock; tests pass a fake they can advance, which
// is what makes session expiry assertions deterministic under `go test -race`.
type Clock interface {
	Now() time.Time
}

// RealClock is the Clock backed by the operating system.
type RealClock struct{}

// Now reports the current wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }
