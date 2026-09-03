package main

import (
	"errors"
	"strings"
	"testing"

	"tutor.local/project-cli-tracker/tracker"
)

func seeded(t *testing.T, titles ...string) *tracker.Tracker {
	t.Helper()
	tr := tracker.New()
	for _, title := range titles {
		if _, err := tr.Add(title); err != nil {
			t.Fatalf("seeding: Add(%q) = %v — make the tracker package pass its own tests first", title, err)
		}
	}
	return tr
}

func TestDispatchAdd(t *testing.T) {
	got, err := dispatch(seeded(t), []string{"add", "buy", "milk"})
	if err != nil {
		t.Fatalf("dispatch(add buy milk) = %v, want nil", err)
	}
	if want := "added #1: buy milk"; got != want {
		t.Errorf("dispatch(add buy milk) = %q, want %q — join the words after 'add' with spaces", got, want)
	}
}

func TestDispatchAddWithoutTitle(t *testing.T) {
	if _, err := dispatch(seeded(t), []string{"add"}); err == nil {
		t.Error("dispatch(add) with no title = nil error, want an error")
	}
}

func TestDispatchList(t *testing.T) {
	t.Run("empty tracker", func(t *testing.T) {
		got, err := dispatch(seeded(t), []string{"list"})
		if err != nil {
			t.Fatalf("dispatch(list) = %v, want nil", err)
		}
		if want := "no tasks yet"; got != want {
			t.Errorf("dispatch(list) on an empty tracker = %q, want %q", got, want)
		}
	})
	t.Run("open and done tasks", func(t *testing.T) {
		tr := seeded(t, "buy milk", "stretch")
		if err := tr.Complete(2); err != nil {
			t.Fatalf("Complete(2): %v", err)
		}
		got, err := dispatch(tr, []string{"list"})
		if err != nil {
			t.Fatalf("dispatch(list) = %v, want nil", err)
		}
		want := strings.Join([]string{"[ ] #1 buy milk", "[x] #2 stretch"}, "\n")
		if got != want {
			t.Errorf("dispatch(list) =\n%s\nwant:\n%s", got, want)
		}
	})
}

func TestDispatchComplete(t *testing.T) {
	tr := seeded(t, "one", "two")
	got, err := dispatch(tr, []string{"complete", "2"})
	if err != nil {
		t.Fatalf("dispatch(complete 2) = %v, want nil", err)
	}
	if want := "completed #2"; got != want {
		t.Errorf("dispatch(complete 2) = %q, want %q", got, want)
	}
	if !tr.List()[1].Done {
		t.Error("task #2 is still open after dispatch(complete 2)")
	}
}

func TestDispatchCompleteUnknownID(t *testing.T) {
	_, err := dispatch(seeded(t, "only"), []string{"complete", "9"})
	if !errors.Is(err, tracker.ErrNotFound) {
		t.Errorf("dispatch(complete 9) error = %v, want errors.Is(err, tracker.ErrNotFound) to hold — pass the tracker's error through (wrapping is fine)", err)
	}
}

func TestDispatchSummary(t *testing.T) {
	tr := seeded(t, "a", "b", "c")
	if err := tr.Complete(3); err != nil {
		t.Fatalf("Complete(3): %v", err)
	}
	got, err := dispatch(tr, []string{"summary"})
	if err != nil {
		t.Fatalf("dispatch(summary) = %v, want nil", err)
	}
	if want := "2 open, 1 done"; got != want {
		t.Errorf("dispatch(summary) = %q, want %q", got, want)
	}
}

func TestDispatchBadInput(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no arguments", nil},
		{"unknown subcommand", []string{"frobnicate"}},
		{"complete without an id", []string{"complete"}},
		{"complete with a non-number", []string{"complete", "two"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := dispatch(seeded(t, "task"), c.args); err == nil {
				t.Errorf("dispatch(%v) = nil error, want an error telling the user what went wrong", c.args)
			}
		})
	}
}
