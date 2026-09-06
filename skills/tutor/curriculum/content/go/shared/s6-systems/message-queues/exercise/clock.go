package queue

import "time"

// Clock supplies the current time. Production code uses SystemClock; the
// tests inject a fake so redelivery is driven by the test, never by sleeping.
type Clock interface {
	Now() time.Time
}

// SystemClock is the real wall clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
