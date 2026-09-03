package apiperf

import (
	"context"
	"testing"
)

func TestAuthorsByIDsIsOneRoundTrip(t *testing.T) {
	st, _ := newTestStore(t)
	authors := seedAuthors(t, st, 3)
	ids := []int64{authors[0].ID, authors[1].ID, authors[2].ID}

	st.ResetQueries()
	got, err := st.AuthorsByIDs(context.Background(), ids)
	if err != nil {
		t.Fatalf("AuthorsByIDs: %v", err)
	}

	if n := st.Queries(); n != 1 {
		t.Errorf("AuthorsByIDs(%d ids) made %d database round trips, want 1 — one statement, not a loop", len(ids), n)
	}
	if len(got) != 3 {
		t.Fatalf("got %d authors, want 3", len(got))
	}
	for _, want := range authors {
		if got[want.ID].Name != want.Name {
			t.Errorf("author %d: got name %q, want %q", want.ID, got[want.ID].Name, want.Name)
		}
	}
}

func TestAuthorsByIDsDedupesAndSkipsMissing(t *testing.T) {
	st, _ := newTestStore(t)
	authors := seedAuthors(t, st, 2)

	st.ResetQueries()
	ids := []int64{authors[0].ID, authors[0].ID, authors[1].ID, 9999}
	got, err := st.AuthorsByIDs(context.Background(), ids)
	if err != nil {
		t.Fatalf("AuthorsByIDs: %v", err)
	}

	if n := st.Queries(); n != 1 {
		t.Errorf("made %d round trips, want 1", n)
	}
	if len(got) != 2 {
		t.Fatalf("got %d authors, want 2 (duplicates collapse, the missing id is simply absent)", len(got))
	}
	if _, ok := got[9999]; ok {
		t.Errorf("id 9999 has no row, so it must be absent from the map, not present with a zero Author")
	}
}

func TestAuthorsByIDsWithNoIDsMakesNoQuery(t *testing.T) {
	st, _ := newTestStore(t)
	seedAuthors(t, st, 1)

	for _, ids := range [][]int64{nil, {}} {
		st.ResetQueries()
		got, err := st.AuthorsByIDs(context.Background(), ids)
		if err != nil {
			t.Fatalf("AuthorsByIDs(%v): %v", ids, err)
		}
		if n := st.Queries(); n != 0 {
			t.Errorf("AuthorsByIDs(%v) made %d round trips, want 0 — an empty batch is not a query", ids, n)
		}
		if got == nil {
			t.Errorf("AuthorsByIDs(%v) returned a nil map; return an empty one so callers can index it", ids)
		}
		if len(got) != 0 {
			t.Errorf("AuthorsByIDs(%v) returned %d authors, want 0", ids, len(got))
		}
	}
}
