package note_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"tutor.local/capstone-reference/internal/note"
)

var parsedAt = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

func TestParseLine(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		wantTitle string
		wantTags  []string
		wantErr   error
	}{
		{"two fields", "n1|ship it", "ship it", nil, nil},
		{"three fields", "n1|ship it|Work, home", "ship it", []string{"home", "work"}, nil},
		{"empty tag field", "n1|ship it|", "ship it", nil, nil},
		{"padding is trimmed", "  n1 | ship it |  work ", "ship it", []string{"work"}, nil},
		{"one field", "n1", "", nil, note.ErrFieldCount},
		{"four fields", "n1|a|b|c", "", nil, note.ErrFieldCount},
		{"empty line", "", "", nil, note.ErrFieldCount},
		{"missing id", "|ship it|", "", nil, note.ErrInvalidID},
		{"id with a slash", "../../etc/passwd|ship it|", "", nil, note.ErrInvalidID},
		{"id too long", strings.Repeat("a", 65) + "|ship it", "", nil, note.ErrInvalidID},
		{"empty title", "n1| |work", "", nil, note.ErrTitleEmpty},
		{"title too long", "n1|" + strings.Repeat("x", note.MaxTitleRunes+1), "", nil, note.ErrTitleTooLong},
		{"line too long", "n1|" + strings.Repeat("x", note.MaxLineBytes), "", nil, note.ErrLineTooLong},
		{"invalid utf-8", "n1|ship \xff it", "", nil, note.ErrNotUTF8},
		{"nul byte", "n1|ship\x00it", "", nil, note.ErrControlChar},
		{"escape sequence", "n1|\x1b[31mred", "", nil, note.ErrControlChar},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := note.ParseLine(c.line, parsedAt)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("ParseLine(%q) error = %v, want %v", c.line, err, c.wantErr)
			}
			if c.wantErr != nil {
				return
			}
			if got.Title != c.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, c.wantTitle)
			}
			if len(got.Tags) != len(c.wantTags) {
				t.Fatalf("Tags = %v, want %v", got.Tags, c.wantTags)
			}
			for i := range c.wantTags {
				if got.Tags[i] != c.wantTags[i] {
					t.Fatalf("Tags = %v, want %v", got.Tags, c.wantTags)
				}
			}
		})
	}
}

// TestParseLineRejectsHostileInput is the regression test for the pass's one
// real finding: ids reached a file path unchecked, so "../.." escaped the
// store directory. It stays even though the store is in memory today.
func TestParseLineRejectsHostileInput(t *testing.T) {
	hostile := []string{
		"../../etc/passwd|owned",
		"..|owned",
		"n1/../../n2|owned",
		"n1\\..\\..\\n2|owned",
		"n1%2F..%2Fn2|owned",
		"n 1|owned",
		"n1;rm -rf /|owned",
		strings.Repeat("n", 65) + "|owned",
	}
	for _, line := range hostile {
		if _, err := note.ParseLine(line, parsedAt); !errors.Is(err, note.ErrInvalidID) {
			t.Errorf("ParseLine(%q) error = %v, want ErrInvalidID", line, err)
		}
	}
	if _, err := note.ParseLine(strings.Repeat("x", note.MaxLineBytes+1), parsedAt); err == nil {
		t.Error("ParseLine(oversized line) error = nil, want an error")
	}
}

func TestLineRoundTrips(t *testing.T) {
	n, err := note.ParseLine("n1|ship it|Work, home, work", parsedAt)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	if got, want := n.Line(), "n1|ship it|home,work"; got != want {
		t.Fatalf("Line() = %q, want %q", got, want)
	}
	again, err := note.ParseLine(n.Line(), parsedAt)
	if err != nil {
		t.Fatalf("re-parsing %q failed: %v", n.Line(), err)
	}
	if !again.Equal(n) {
		t.Errorf("round trip changed the note: %+v vs %+v", again, n)
	}
}

// FuzzParseLine states the properties that must hold for every input, not for
// the inputs we thought of. The seeds below plus testdata/fuzz/FuzzParseLine
// run on every plain `go test`; run it with -fuzz to hunt for more.
func FuzzParseLine(f *testing.F) {
	for _, seed := range []string{
		"n1|ship it",
		"n1|ship it|work,home",
		"n1|ship it|",
		"|ship it|",
		"n1|a|b|c",
		"../../etc/passwd|owned|",
		"n1|" + strings.Repeat("x", 200),
		"n1|ship\x00it",
		"n1|ship \xff it",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		n, err := note.ParseLine(line, parsedAt)
		if err != nil {
			return
		}

		if !note.ValidID(n.ID) {
			t.Fatalf("accepted id %q, which ValidID rejects", n.ID)
		}
		if n.Title == "" {
			t.Fatalf("accepted an empty title from %q", line)
		}
		if runes := len([]rune(n.Title)); runes > note.MaxTitleRunes {
			t.Fatalf("accepted a %d-rune title, limit is %d", runes, note.MaxTitleRunes)
		}
		for i, tag := range n.Tags {
			if tag == "" {
				t.Fatalf("accepted an empty tag from %q", line)
			}
			if tag != strings.ToLower(tag) {
				t.Fatalf("tag %q is not normalised", tag)
			}
			if i > 0 && n.Tags[i-1] >= tag {
				t.Fatalf("tags are not sorted and unique: %v", n.Tags)
			}
		}

		rendered := n.Line()
		if len(rendered) > note.MaxLineBytes {
			return // normalisation can grow a line; that is not a round-trip claim
		}
		again, err := note.ParseLine(rendered, parsedAt)
		if err != nil {
			t.Fatalf("ParseLine(%q) accepted, but re-parsing %q failed: %v", line, rendered, err)
		}
		if !again.Equal(n) {
			t.Fatalf("round trip changed the note:\n got %+v\nwant %+v", again, n)
		}
	})
}
