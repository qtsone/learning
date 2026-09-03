// Package timekit is a small timestamp toolkit: parse operator-entered
// times in their local zone, render them canonically in UTC, and compare
// instants safely.
package timekit

import "time"

// ScheduleLayout is the input format operators type: minute precision,
// no zone — the zone travels separately (see ParseLocal).
const ScheduleLayout = "2006-01-02 15:04"

// ParseLocal parses value against ScheduleLayout in the named IANA zone
// (e.g. "Europe/Bucharest") and returns the resulting instant.
// An unknown zone or a value that does not match the layout yields a
// non-nil error naming the offending input.
func ParseLocal(value, zone string) (time.Time, error) {
	// TODO: load the zone, then parse value in it. Two steps, two
	// failure paths — wrap each error with context (%w).
	return time.Time{}, nil
}

// FormatUTC renders t as "2006-01-02 15:04 MST" after converting it to
// UTC, so the same instant always renders the same regardless of the
// zone it arrived in.
func FormatUTC(t time.Time) string {
	// TODO: convert to UTC, then format.
	return ""
}

// Halfway returns the instant exactly midway between start and end.
func Halfway(start, end time.Time) time.Time {
	// TODO: pure Time/Duration arithmetic — no Unix-seconds juggling.
	return time.Time{}
}

// SameInstant reports whether a and b name the same instant, even when
// their zones differ.
func SameInstant(a, b time.Time) bool {
	// TODO: compare instants, not representations.
	return false
}

// InWindow reports whether t falls inside [start, end): inclusive of
// start, exclusive of end.
func InWindow(t, start, end time.Time) bool {
	// TODO: compare instants, not representations.
	return false
}
