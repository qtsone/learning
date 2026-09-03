package store_test

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"tutor.local/capstone-reference/internal/note"
	"tutor.local/capstone-reference/internal/store"
)

func mustNote(t *testing.T, id, title string, tags ...string) note.Note {
	t.Helper()
	n, err := note.New(id, title, tags, time.Now())
	if err != nil {
		t.Fatalf("note.New(%q) error = %v", id, err)
	}
	return n
}

func TestAddAndGet(t *testing.T) {
	s := store.NewMemory()
	if err := s.Add(mustNote(t, "n1", "first", "work")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	got, err := s.Get("n1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != "first" {
		t.Errorf("Get().Title = %q, want %q", got.Title, "first")
	}
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1", s.Len())
	}
}

func TestAddRejects(t *testing.T) {
	s := store.NewMemory()
	if err := s.Add(mustNote(t, "n1", "first")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := s.Add(mustNote(t, "n1", "again")); !errors.Is(err, store.ErrDuplicate) {
		t.Errorf("Add(duplicate) error = %v, want ErrDuplicate", err)
	}
	if err := s.Add(note.Note{Title: "no id"}); err == nil {
		t.Error("Add(no id) error = nil, want an error")
	}
}

func TestGetMissing(t *testing.T) {
	s := store.NewMemory()
	if _, err := s.Get("nope"); !errors.Is(err, note.ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want note.ErrNotFound", err)
	}
}

func TestListFiltersAndKeepsOrder(t *testing.T) {
	s := store.NewMemory()
	for _, n := range []note.Note{
		mustNote(t, "n1", "first", "work"),
		mustNote(t, "n2", "second", "home"),
		mustNote(t, "n3", "third", "work", "urgent"),
	} {
		if err := s.Add(n); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}
	all := s.List("")
	if len(all) != 3 || all[0].ID != "n1" || all[2].ID != "n3" {
		t.Fatalf("List(\"\") = %v, want n1,n2,n3 in order", all)
	}
	work := s.List("work")
	if len(work) != 2 || work[0].ID != "n1" || work[1].ID != "n3" {
		t.Errorf("List(\"work\") = %v, want n1,n3", work)
	}
	if got := s.List("missing"); len(got) != 0 {
		t.Errorf("List(\"missing\") = %v, want empty", got)
	}
}

// TestListByTagMatchesLinearScan is the correctness proof for the tag index:
// it cross-checks the fast path against the implementation the index replaced,
// over generated input, including queries that need normalising and one tag
// nobody used. If the two ever disagree, the optimization changed behaviour —
// which is the only interesting way this change can be wrong. See PERF.md.
func TestListByTagMatchesLinearScan(t *testing.T) {
	tags := []string{"work", "home", "urgent", "later", "reading"}
	rng := rand.New(rand.NewSource(7))
	s := store.NewMemory()
	for i := 0; i < 500; i++ {
		var chosen []string
		for _, tag := range tags {
			if rng.Intn(3) == 0 {
				chosen = append(chosen, tag)
			}
		}
		n := mustNote(t, fmt.Sprintf("n%03d", i), fmt.Sprintf("note %d", i), chosen...)
		if err := s.Add(n); err != nil {
			t.Fatalf("Add(%s) error = %v", n.ID, err)
		}
	}

	all := s.List("")
	queries := append(append([]string{}, tags...), "WORK", "  Urgent  ", "absent")
	for _, query := range queries {
		want := linearScan(all, query)
		got := s.List(query)
		if ids(got) != ids(want) {
			t.Errorf("List(%q) = %s, linear scan = %s", query, ids(got), ids(want))
		}
	}
}

// linearScan is the pre-index implementation of List, kept here as the oracle
// the indexed one has to agree with.
func linearScan(all []note.Note, tag string) []note.Note {
	out := make([]note.Note, 0, len(all))
	for _, n := range all {
		if n.HasTag(tag) {
			out = append(out, n)
		}
	}
	return out
}

func ids(notes []note.Note) string {
	out := make([]string, 0, len(notes))
	for _, n := range notes {
		out = append(out, n.ID)
	}
	return fmt.Sprint(out)
}

// The store is shared by concurrent callers, so the race detector gets a
// chance to prove the locking is right.
func TestConcurrentUse(t *testing.T) {
	s := store.NewMemory()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("n%02d", i)
			if err := s.Add(mustNoteConcurrent(id)); err != nil {
				t.Errorf("Add(%s) error = %v", id, err)
			}
			s.List("work")
			s.Len()
		}(i)
	}
	wg.Wait()
	if s.Len() != 16 {
		t.Errorf("Len() = %d, want 16", s.Len())
	}
}

func mustNoteConcurrent(id string) note.Note {
	n, err := note.New(id, "concurrent "+id, []string{"work"}, time.Now())
	if err != nil {
		panic("test setup is wrong: " + err.Error())
	}
	return n
}
