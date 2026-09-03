package note_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"tutor.local/capstone-reference/internal/note"
)

func TestNewValidates(t *testing.T) {
	when := time.Date(2026, 3, 1, 12, 0, 0, 0, time.FixedZone("CET", 3600))
	cases := []struct {
		name    string
		title   string
		wantErr error
	}{
		{"plain", "buy milk", nil},
		{"trimmed", "  buy milk  ", nil},
		{"empty", "", note.ErrTitleEmpty},
		{"blank", "   ", note.ErrTitleEmpty},
		{"too long", strings.Repeat("x", note.MaxTitleRunes+1), note.ErrTitleTooLong},
		{"at the limit", strings.Repeat("x", note.MaxTitleRunes), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := note.New("n1", c.title, nil, when)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("New(%q) error = %v, want %v", c.title, err, c.wantErr)
			}
			if c.wantErr != nil {
				return
			}
			if want := strings.TrimSpace(c.title); got.Title != want {
				t.Errorf("Title = %q, want %q", got.Title, want)
			}
			if got.Created.Location() != time.UTC {
				t.Errorf("Created = %v, want a UTC time", got.Created)
			}
			if got.Tags == nil {
				t.Error("Tags = nil, want an empty slice")
			}
		})
	}
}

func TestNormaliseTags(t *testing.T) {
	got := note.NormaliseTags([]string{"Work", " home ", "work", "", "Home", "urgent"})
	want := []string{"home", "urgent", "work"}
	if len(got) != len(want) {
		t.Fatalf("NormaliseTags() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormaliseTags() = %v, want %v", got, want)
		}
	}
}

func TestHasTag(t *testing.T) {
	n, err := note.New("n1", "ship it", []string{"Work"}, time.Now())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if !n.HasTag("WORK") {
		t.Error("HasTag(\"WORK\") = false, want true (comparison is normalised)")
	}
	if n.HasTag("home") {
		t.Error("HasTag(\"home\") = true, want false")
	}
}

func TestString(t *testing.T) {
	when := time.Now()
	tagged, err := note.New("n1", "ship it", []string{"work"}, when)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got, want := tagged.String(), "n1  ship it  [work]"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	bare, err := note.New("n2", "think", nil, when)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got, want := bare.String(), "n2  think"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
