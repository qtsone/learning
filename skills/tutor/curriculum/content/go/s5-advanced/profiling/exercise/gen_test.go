package report

import (
	"fmt"
	"math/rand/v2"
)

// genEntries builds a deterministic workload: n entries spread over 100
// distinct users whose logged names are mixed-case (so case-insensitive
// matching has real work to do). Same seed, same data, every run —
// benchmarks and allocation probes stay comparable.
func genEntries(n int) []Entry {
	users := make([]string, 100)
	for i := range users {
		users[i] = fmt.Sprintf("User%03d", i)
	}
	r := rand.New(rand.NewPCG(1, 2))
	entries := make([]Entry, n)
	for i := range entries {
		entries[i] = Entry{
			User:   users[r.IntN(len(users))],
			Path:   fmt.Sprintf("/items/%d", r.IntN(500)),
			Status: 200,
			Bytes:  r.IntN(2000),
		}
	}
	return entries
}

func genStats(n int) []UserStat {
	r := rand.New(rand.NewPCG(3, 4))
	stats := make([]UserStat, n)
	for i := range stats {
		stats[i] = UserStat{
			User:     fmt.Sprintf("user%05d", i),
			Requests: 1 + r.IntN(9000),
			Bytes:    r.IntN(5_000_000),
		}
	}
	return stats
}
