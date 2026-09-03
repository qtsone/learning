package logkit

import (
	"context"
	"errors"
	"maps"
	"testing"
)

// skipIfShort keeps the fast feedback loop fast: `go test -short ./...` runs
// only the unit tests, the full command runs everything.
func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs a real SQLite database on disk")
	}
}

// newTestStore returns a Store backed by a database file of its own, inside
// the directory TestMain created, and registers its teardown.
//
// TODO: implement it.
//   - Build a path under integrationDir that is unique per call, so two
//     stores in one test never share data.
//   - Open the store, failing the test with t.Fatalf if that errors.
//   - Register cleanup with t.Cleanup: close the handle, and report a close
//     error with t.Errorf rather than swallowing it.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	t.Fatalf("newTestStore is not implemented")
	return nil
}

func TestStoreRoundTrip(t *testing.T) {
	skipIfShort(t)
	ctx := context.Background()
	st := newTestStore(t)

	events := []Event{
		{"info", "api", "started"},
		{"error", "db", `pipes | and \ backslashes`},
		{"info", "worker", "line one\nline two"},
	}
	for _, ev := range events {
		if err := st.Insert(ctx, ev); err != nil {
			t.Fatalf("Insert(%#v) = %v", ev, err)
		}
	}

	got, err := st.All(ctx)
	if err != nil {
		t.Fatalf("All() = %v", err)
	}
	if len(got) != len(events) {
		t.Fatalf("All() returned %d events, want %d", len(got), len(events))
	}
	for i := range events {
		if got[i] != events[i] {
			t.Errorf("All()[%d] = %#v, want %#v", i, got[i], events[i])
		}
	}
}

func TestStoreCountsByLevel(t *testing.T) {
	skipIfShort(t)
	ctx := context.Background()
	st := newTestStore(t)

	for _, ev := range []Event{
		{"info", "api", "a"},
		{"info", "worker", "b"},
		{"error", "db", "c"},
	} {
		if err := st.Insert(ctx, ev); err != nil {
			t.Fatalf("Insert(%#v) = %v", ev, err)
		}
	}

	got, err := st.CountsByLevel(ctx)
	if err != nil {
		t.Fatalf("CountsByLevel() = %v", err)
	}
	want := map[string]int{"info": 2, "error": 1}
	if !maps.Equal(got, want) {
		t.Errorf("CountsByLevel() = %v, want %v", got, want)
	}
}

func TestStoreRejectsUnknownLevel(t *testing.T) {
	skipIfShort(t)
	ctx := context.Background()
	st := newTestStore(t)

	err := st.Insert(ctx, Event{"trace", "api", "nope"})
	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("Insert with level \"trace\" = %v, want an error matching ErrMalformed", err)
	}
	counts, err := st.CountsByLevel(ctx)
	if err != nil {
		t.Fatalf("CountsByLevel() = %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("rejected event was stored anyway: %v", counts)
	}
}

// Fixtures must not leak into each other: this is the test that catches a
// newTestStore which hands every caller the same database file.
func TestStoreFixturesAreIsolated(t *testing.T) {
	skipIfShort(t)
	ctx := context.Background()
	first, second := newTestStore(t), newTestStore(t)

	if err := first.Insert(ctx, Event{"info", "api", "only in the first store"}); err != nil {
		t.Fatalf("Insert() = %v", err)
	}
	counts, err := second.CountsByLevel(ctx)
	if err != nil {
		t.Fatalf("CountsByLevel() = %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("second store sees %v; each fixture needs its own database", counts)
	}
}
