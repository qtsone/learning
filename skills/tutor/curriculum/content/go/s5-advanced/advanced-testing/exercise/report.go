package logkit

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
	// TODO: render the table described above.
	return ""
}
