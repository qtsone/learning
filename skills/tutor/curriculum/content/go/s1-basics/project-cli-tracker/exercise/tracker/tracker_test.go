package tracker

import (
	"errors"
	"testing"
)

func mustAdd(t *testing.T, tr *Tracker, title string) Task {
	t.Helper()
	task, err := tr.Add(title)
	if err != nil {
		t.Fatalf("Add(%q) returned error %v — this test needs a working Add first", title, err)
	}
	return task
}

func TestAddAssignsIncreasingIDs(t *testing.T) {
	tr := New()
	first := mustAdd(t, tr, "write tests")
	second := mustAdd(t, tr, "make them pass")
	if first.ID != 1 || second.ID != 2 {
		t.Errorf("IDs = %d, %d, want 1, 2 — ids start at 1 and increase by one", first.ID, second.ID)
	}
	if first.Title != "write tests" || first.Done {
		t.Errorf("first task = %+v, want Title %q and Done false", first, "write tests")
	}
}

func TestAddTitles(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"plain title", "buy milk", "buy milk", false},
		{"surrounding spaces are trimmed", "  stretch  ", "stretch", false},
		{"empty title is rejected", "", "", true},
		{"spaces-only title is rejected", "   ", "", true},
		{"the separator character is allowed", "read | write", "read | write", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			task, err := New().Add(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("Add(%q) = %+v, want an error — a task needs a title", c.in, task)
				}
				return
			}
			if err != nil {
				t.Fatalf("Add(%q) returned unexpected error: %v", c.in, err)
			}
			if task.Title != c.want {
				t.Errorf("Add(%q).Title = %q, want %q", c.in, task.Title, c.want)
			}
		})
	}
}

func TestListKeepsInsertionOrder(t *testing.T) {
	tr := New()
	titles := []string{"one", "two", "three"}
	for _, title := range titles {
		mustAdd(t, tr, title)
	}
	got := tr.List()
	if len(got) != len(titles) {
		t.Fatalf("List() has %d tasks, want %d", len(got), len(titles))
	}
	for i, task := range got {
		if task.Title != titles[i] {
			t.Errorf("List()[%d].Title = %q, want %q — keep insertion order", i, task.Title, titles[i])
		}
	}
}

func TestListReturnsACopy(t *testing.T) {
	tr := New()
	mustAdd(t, tr, "original")
	tr.List()[0].Title = "mutated"
	if got := tr.List()[0].Title; got != "original" {
		t.Errorf("List()[0].Title = %q after mutating an earlier List result, want %q — List must return a copy, not the tracker's own slice", got, "original")
	}
}

func TestCompleteMarksOnlyThatTask(t *testing.T) {
	tr := New()
	mustAdd(t, tr, "one")
	second := mustAdd(t, tr, "two")
	if err := tr.Complete(second.ID); err != nil {
		t.Fatalf("Complete(%d) returned error %v, want nil", second.ID, err)
	}
	got := tr.List()
	if got[0].Done {
		t.Errorf("task #%d is done, but only #%d was completed", got[0].ID, second.ID)
	}
	if !got[1].Done {
		t.Errorf("task #%d is still open after Complete(%d) — does Complete change the tracker's own task, or a copy?", second.ID, second.ID)
	}
}

func TestCompleteUnknownID(t *testing.T) {
	tr := New()
	mustAdd(t, tr, "only task")
	err := tr.Complete(42)
	if err == nil {
		t.Fatal("Complete(42) = nil, want an error for an id that does not exist")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Complete(42) error = %v, want errors.Is(err, ErrNotFound) to hold — return the sentinel, or wrap it with %%w", err)
	}
}

func TestCompleteTwiceIsNotAnError(t *testing.T) {
	tr := New()
	task := mustAdd(t, tr, "repeat")
	if err := tr.Complete(task.ID); err != nil {
		t.Fatalf("first Complete(%d): %v", task.ID, err)
	}
	if err := tr.Complete(task.ID); err != nil {
		t.Errorf("second Complete(%d) = %v, want nil — completing a done task is fine", task.ID, err)
	}
}

func TestSummary(t *testing.T) {
	cases := []struct {
		name     string
		titles   []string
		complete []int
		want     map[string]int
	}{
		{"empty tracker", nil, nil, map[string]int{"open": 0, "done": 0}},
		{"all open", []string{"a", "b"}, nil, map[string]int{"open": 2, "done": 0}},
		{"mixed", []string{"a", "b", "c"}, []int{2}, map[string]int{"open": 2, "done": 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr := New()
			for _, title := range c.titles {
				mustAdd(t, tr, title)
			}
			for _, id := range c.complete {
				if err := tr.Complete(id); err != nil {
					t.Fatalf("Complete(%d): %v", id, err)
				}
			}
			got := tr.Summary()
			if len(got) != len(c.want) || got["open"] != c.want["open"] || got["done"] != c.want["done"] {
				t.Errorf("Summary() = %v, want %v — both keys must always be present", got, c.want)
			}
		})
	}
}
