// Package report aggregates HTTP access-log entries into a per-user
// traffic report. It works — and it is slow. Your job is not to guess
// why: benchmark it, profile it, let the profile point at the guilty
// lines, then fix them without changing behavior.
package report

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Entry is one parsed access-log line.
type Entry struct {
	User   string // as logged — casing varies ("Alice", "ALICE", "alice")
	Path   string
	Status int
	Bytes  int
}

// UserStat is the aggregate for one user. User holds the lowercase form.
type UserStat struct {
	User     string
	Requests int
	Bytes    int
}

// TopUsers aggregates entries per user (case-insensitively — "Alice" and
// "ALICE" are the same user, reported in lowercase), sorts by request
// count descending (ties broken by user ascending), and returns at most
// n stats.
//
// TODO: this version passes every correctness test and fails the
// allocation probe. Profile it (see LESSON.md), find out where the CPU
// and the allocations go, and fix it. Behavior must not change.
func TopUsers(entries []Entry, n int) []UserStat {
	var stats []UserStat
	for _, e := range entries {
		found := false
		for i := range stats {
			if stats[i].User == strings.ToLower(e.User) {
				stats[i].Requests++
				stats[i].Bytes += e.Bytes
				found = true
				break
			}
		}
		if !found {
			stats = append(stats, UserStat{User: strings.ToLower(e.User), Requests: 1, Bytes: e.Bytes})
		}
	}
	sortStats(stats)
	if len(stats) > n {
		stats = stats[:n]
	}
	return stats
}

// Render formats stats as one line per user:
//
//	alice requests=42 bytes=90210\n
//
// TODO: this version is also correct and also fails its allocation
// probe. The heap profile will tell you why. Fix it; the output must
// stay byte-for-byte identical.
func Render(stats []UserStat) string {
	out := ""
	for _, s := range stats {
		out += fmt.Sprintf("%s requests=%d bytes=%d\n", s.User, s.Requests, s.Bytes)
	}
	return out
}

func sortStats(stats []UserStat) {
	slices.SortFunc(stats, func(a, b UserStat) int {
		if a.Requests != b.Requests {
			return cmp.Compare(b.Requests, a.Requests)
		}
		return cmp.Compare(a.User, b.User)
	})
}
