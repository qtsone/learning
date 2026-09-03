// Package report aggregates HTTP access-log entries into a per-user
// traffic report.
package report

import (
	"cmp"
	"slices"
	"strconv"
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
// The CPU profile blamed the starter's inner scan: strings.ToLower ran
// once per *comparison* (O(entries x users) calls, each allocating). A
// map index makes each entry one lookup, and lowering happens once per
// entry.
func TopUsers(entries []Entry, n int) []UserStat {
	index := make(map[string]int)
	var stats []UserStat
	for _, e := range entries {
		user := strings.ToLower(e.User)
		i, ok := index[user]
		if !ok {
			i = len(stats)
			stats = append(stats, UserStat{User: user})
			index[user] = i
		}
		stats[i].Requests++
		stats[i].Bytes += e.Bytes
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
// The heap profile blamed += (a fresh, longer string per line: O(n)
// allocations, O(n²) bytes copied) and fmt.Sprintf (boxing every
// argument). One Builder plus append-style formatting into a reused
// buffer renders everything with a handful of allocations.
func Render(stats []UserStat) string {
	var b strings.Builder
	b.Grow(len(stats) * 48)
	line := make([]byte, 0, 64)
	for _, s := range stats {
		line = line[:0]
		line = append(line, s.User...)
		line = append(line, " requests="...)
		line = strconv.AppendInt(line, int64(s.Requests), 10)
		line = append(line, " bytes="...)
		line = strconv.AppendInt(line, int64(s.Bytes), 10)
		line = append(line, '\n')
		b.Write(line)
	}
	return b.String()
}

func sortStats(stats []UserStat) {
	slices.SortFunc(stats, func(a, b UserStat) int {
		if a.Requests != b.Requests {
			return cmp.Compare(b.Requests, a.Requests)
		}
		return cmp.Compare(a.User, b.User)
	})
}
