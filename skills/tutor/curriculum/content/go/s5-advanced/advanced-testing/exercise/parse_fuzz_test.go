package logkit

import "testing"

// FuzzParseLine feeds arbitrary input to ParseLine.
//
// Plain `go test` runs only the seed corpus below plus the files committed in
// testdata/fuzz/FuzzParseLine — fast and deterministic, so it is safe in CI.
// You run the fuzzing engine yourself, by hand:
//
//	go test -fuzz=FuzzParseLine -fuzztime=30s .
//
// A failing input is written to testdata/fuzz/FuzzParseLine/ — commit it, and
// it becomes a permanent regression case in the seed corpus.
func FuzzParseLine(f *testing.F) {
	f.Add(`info|api|started`)
	f.Add(`debug||`)
	f.Add(`error|db|a\|b`)
	f.Add(`warn|c:\\tmp|x`)
	f.Add(`info|api|one\ntwo`)
	f.Add(`trace|api|unknown level`)
	f.Add(`info|api`)
	f.Add(``)

	f.Fuzz(func(t *testing.T, line string) {
		// TODO: assert the properties from LESSON.md.
		//
		// 1. ParseLine must not panic on any input (you get this for free —
		//    a panic fails the test).
		// 2. If ParseLine returns an error, there is nothing else to check.
		// 3. If it succeeds, the event's Level must be a known level, and
		//    re-encoding it with Format must parse back to the same Event.
		_ = line
	})
}
