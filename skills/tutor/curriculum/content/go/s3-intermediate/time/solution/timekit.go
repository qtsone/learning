// Package timekit is a small timestamp toolkit: parse operator-entered
// times in their local zone, render them canonically in UTC, and compare
// instants safely.
package timekit

import (
	"fmt"
	"time"
)

// ScheduleLayout is the input format operators type: minute precision,
// no zone — the zone travels separately (see ParseLocal).
const ScheduleLayout = "2006-01-02 15:04"

const utcLayout = "2006-01-02 15:04 MST"

// ParseLocal parses value against ScheduleLayout in the named IANA zone
// (e.g. "Europe/Bucharest") and returns the resulting instant.
// An unknown zone or a value that does not match the layout yields a
// non-nil error naming the offending input.
func ParseLocal(value, zone string) (time.Time, error) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load zone %q: %w", zone, err)
	}
	t, err := time.ParseInLocation(ScheduleLayout, value, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %q in %s: %w", value, zone, err)
	}
	return t, nil
}

// FormatUTC renders t as "2006-01-02 15:04 MST" after converting it to
// UTC, so the same instant always renders the same regardless of the
// zone it arrived in.
func FormatUTC(t time.Time) string {
	return t.UTC().Format(utcLayout)
}

// Halfway returns the instant exactly midway between start and end.
func Halfway(start, end time.Time) time.Time {
	return start.Add(end.Sub(start) / 2)
}

// SameInstant reports whether a and b name the same instant, even when
// their zones differ.
func SameInstant(a, b time.Time) bool {
	return a.Equal(b)
}

// InWindow reports whether t falls inside [start, end): inclusive of
// start, exclusive of end.
func InWindow(t, start, end time.Time) bool {
	return !t.Before(start) && t.Before(end)
}
