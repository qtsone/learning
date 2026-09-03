package logkit

import (
	"fmt"
	"slices"
	"strings"
)

// Report renders the level summary for events. skipped is the number of input
// lines that could not be parsed. The exact layout is specified in LESSON.md:
//
//	level  count  sources
//	info       2  api, worker
//	error      1  db
//
//	total: 3 events, 1 skipped
//
// Rows appear in severity order, levels with no events are omitted, and each
// row's sources are deduplicated and sorted.
func Report(events []Event, skipped int) string {
	counts := make(map[string]int, len(Levels))
	sources := make(map[string]map[string]bool, len(Levels))
	for _, ev := range events {
		counts[ev.Level]++
		if sources[ev.Level] == nil {
			sources[ev.Level] = make(map[string]bool)
		}
		sources[ev.Level][ev.Source] = true
	}

	var b strings.Builder
	b.WriteString("level  count  sources\n")
	for _, level := range Levels {
		n := counts[level]
		if n == 0 {
			continue
		}
		names := make([]string, 0, len(sources[level]))
		for name := range sources[level] {
			names = append(names, name)
		}
		slices.Sort(names)
		fmt.Fprintf(&b, "%-5s  %5d  %s\n", level, n, strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "\ntotal: %d events, %d skipped\n", len(events), skipped)
	return b.String()
}
