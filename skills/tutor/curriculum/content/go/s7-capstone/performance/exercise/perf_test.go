package capstone

import (
	"testing"
)

const (
	minPerfBytes    = 800
	minSectionBytes = 120
	minResultNums   = 2
)

// Criterion 3: PERF.md exists and every required section is actually written.
// The headings are matched loosely — your own wording survives — but an empty
// section, or one still holding template text, is not a write-up.
func TestPerfDocIsComplete(t *testing.T) {
	doc := perf(t)

	if len(doc.raw) < minPerfBytes {
		t.Errorf("%s is %d bytes, want at least %d — this is the document that "+
			"has to convince a reviewer who was not there",
			perfFileName, len(doc.raw), minPerfBytes)
	}

	for _, req := range requiredSections {
		s, ok := findSection(doc.sections, req.Match)
		if !ok {
			t.Errorf("%s has no `%s` section — it should hold %s",
				perfFileName, req.Heading, req.Wants)
			continue
		}
		if len(s.Body) < minSectionBytes {
			t.Errorf("%s:%d: `%s` is %d bytes, want at least %d — %s",
				perfFileName, s.Line, s.Heading, len(s.Body), minSectionBytes, req.Wants)
		}
		if m := placeholderRe.FindString(s.Body); m != "" {
			t.Errorf("%s:%d: `%s` still holds template text (%q) — replace it or delete it",
				perfFileName, s.Line, s.Heading, m)
		}
	}

	if len(doc.sections) == 0 {
		t.Fatalf("%s has no markdown headings at all — start from "+
			"exercise/PERF-template.md", perfFileName)
	}
}

// Criterion 4: the result is stated as before/after numbers with units.
// "Much faster" is not a result; it is an opinion about a result.
func TestPerfResultHasNumbers(t *testing.T) {
	s := requireSection(t, "result")

	nums := numberRe.FindAllString(s.Body, -1)
	if len(nums) < minResultNums {
		t.Errorf("%s:%d: `%s` contains %d number(s), want at least %d — "+
			"state the before value and the after value, not a verdict",
			perfFileName, s.Line, s.Heading, len(nums), minResultNums)
	}
	if !unitRe.MatchString(s.Body) {
		t.Errorf("%s:%d: `%s` has numbers with no units — ns/op, B/op, allocs/op, "+
			"ms, MB, req/s or %% all say something; a bare number says nothing",
			perfFileName, s.Line, s.Heading)
	}
}
