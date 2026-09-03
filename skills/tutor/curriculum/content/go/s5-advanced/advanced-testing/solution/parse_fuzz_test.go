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
		ev, err := ParseLine(line)
		if err != nil {
			return // rejecting input is always allowed; crashing is not
		}
		if _, ok := LevelRank(ev.Level); !ok {
			t.Fatalf("ParseLine(%q) accepted unknown level %q", line, ev.Level)
		}
		encoded := Format(ev)
		again, err := ParseLine(encoded)
		if err != nil {
			t.Fatalf("ParseLine(%q) succeeded but its Format output %q does not parse: %v", line, encoded, err)
		}
		if again != ev {
			t.Fatalf("round trip changed the event: %q -> %#v -> %q -> %#v", line, ev, encoded, again)
		}
	})
}
