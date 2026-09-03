package apiperf

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCursorRoundTrips(t *testing.T) {
	cases := []Cursor{
		{CreatedAt: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC), ID: 1},
		{CreatedAt: time.Date(2031, 7, 9, 4, 5, 6, 123456789, time.UTC), ID: 987654321},
	}
	for _, want := range cases {
		encoded := EncodeCursor(want)
		got, err := DecodeCursor(encoded)
		if err != nil {
			t.Fatalf("DecodeCursor(EncodeCursor(%v)) = error %v", want, err)
		}
		if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
			t.Errorf("round trip: got %v/%d, want %v/%d", got.CreatedAt, got.ID, want.CreatedAt, want.ID)
		}
	}
}

func TestCursorIsOpaqueAndURLSafe(t *testing.T) {
	encoded := EncodeCursor(Cursor{CreatedAt: time.Unix(0, 1700000000000000000).UTC(), ID: 42})
	if encoded == "" {
		t.Fatal("EncodeCursor returned an empty string")
	}
	if strings.ContainsAny(encoded, ":+/=") {
		t.Errorf("cursor %q must be URL-safe base64 without padding, and must not expose its raw shape", encoded)
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("cursor is not base64.RawURLEncoding: %v", err)
	}
	if want := "1700000000000000000:42"; string(raw) != want {
		t.Errorf("decoded payload = %q, want %q", raw, want)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	cases := map[string]string{
		"empty":           "",
		"not base64":      "not a cursor!!",
		"no separator":    base64.RawURLEncoding.EncodeToString([]byte("1700000000000000000")),
		"too many fields": base64.RawURLEncoding.EncodeToString([]byte("1:2:3")),
		"bad timestamp":   base64.RawURLEncoding.EncodeToString([]byte("nope:42")),
		"bad id":          base64.RawURLEncoding.EncodeToString([]byte("1700000000000000000:nope")),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCursor(in); !errors.Is(err, ErrBadCursor) {
				t.Errorf("DecodeCursor(%q) error = %v, want one that errors.Is ErrBadCursor", in, err)
			}
		})
	}
}

func TestListArticlesFirstPageIsNewestFirstAndOneQuery(t *testing.T) {
	st, clk := newTestStore(t)
	authors := seedAuthors(t, st, 2)
	articles := seedArticles(t, st, clk, authors, 5, time.Second)

	st.ResetQueries()
	got, err := st.ListArticles(context.Background(), 3, nil)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if n := st.Queries(); n != 1 {
		t.Errorf("ListArticles made %d round trips, want 1", n)
	}
	want := []int64{articles[4].ID, articles[3].ID, articles[2].ID}
	if !reflect.DeepEqual(articleIDs(got), want) {
		t.Errorf("first page = %v, want %v (newest first)", articleIDs(got), want)
	}
}

func TestListArticlesAfterCursorIsStrictlyAfter(t *testing.T) {
	st, clk := newTestStore(t)
	authors := seedAuthors(t, st, 1)
	articles := seedArticles(t, st, clk, authors, 5, time.Second)

	last := articles[2] // the third-oldest; the page before it is {5,4,3}
	got, err := st.ListArticles(context.Background(), 10, &Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	want := []int64{articles[1].ID, articles[0].ID}
	if !reflect.DeepEqual(articleIDs(got), want) {
		t.Errorf("page after cursor = %v, want %v — the cursor row itself must not repeat", articleIDs(got), want)
	}
}

func TestListArticlesBreaksTiesOnID(t *testing.T) {
	st, clk := newTestStore(t)
	authors := seedAuthors(t, st, 1)
	// step 0: every article shares one created_at, so only the id tiebreaker
	// can produce a stable order.
	articles := seedArticles(t, st, clk, authors, 5, 0)

	var seen []int64
	var cursor *Cursor
	for range 3 {
		page, err := st.ListArticles(context.Background(), 2, cursor)
		if err != nil {
			t.Fatalf("ListArticles: %v", err)
		}
		if len(page) == 0 {
			break
		}
		seen = append(seen, articleIDs(page)...)
		last := page[len(page)-1]
		cursor = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}

	want := []int64{articles[4].ID, articles[3].ID, articles[2].ID, articles[1].ID, articles[0].ID}
	if !reflect.DeepEqual(seen, want) {
		t.Errorf("paging through tied timestamps saw %v, want %v — compare (created_at, id), not created_at alone", seen, want)
	}
}

// TestListArticlesSeeksInsteadOfScanning asks SQLite's planner what it will do
// with the cursor query. This is the one place the suite checks *how fast*
// something is, and it does it by reading a plan rather than a clock: a plan is
// deterministic, a duration on a shared CI machine is not.
func TestListArticlesSeeksInsteadOfScanning(t *testing.T) {
	st, clk := newTestStore(t)
	authors := seedAuthors(t, st, 1)
	articles := seedArticles(t, st, clk, authors, 20, time.Second)

	last := articles[10]
	if _, err := st.ListArticles(context.Background(), 5, &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}); err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	plan, err := st.ExplainLast(context.Background())
	if err != nil {
		t.Fatalf("ExplainLast: %v", err)
	}

	if !strings.Contains(plan, "SEARCH") || !strings.Contains(plan, "articles_feed") {
		t.Errorf("the cursor query plans as:\n\t%s\nwant a SEARCH into articles_feed.\n"+
			"A SCAN means the planner walked rows and threw them away — which is what OFFSET does, "+
			"so a keyset query that scans has bought you correctness and no speed. "+
			"Compare the row-value form `(created_at, id) < (?, ?)` with the OR form.", plan)
	}
}

func TestKeysetPagesAreStableUnderInserts(t *testing.T) {
	st, clk := newTestStore(t)
	authors := seedAuthors(t, st, 1)
	original := seedArticles(t, st, clk, authors, 6, time.Second)

	first, err := st.ListArticles(context.Background(), 3, nil)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if len(first) != 3 {
		t.Fatalf("page 1 held %d articles, want 3", len(first))
	}
	last := first[len(first)-1]

	// Three newer articles arrive between the two page requests, which is what
	// happens on any live feed.
	seedArticles(t, st, clk, authors, 3, time.Second)

	second, err := st.ListArticles(context.Background(), 3, &Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}

	wantFirst := []int64{original[5].ID, original[4].ID, original[3].ID}
	wantSecond := []int64{original[2].ID, original[1].ID, original[0].ID}
	if !reflect.DeepEqual(articleIDs(first), wantFirst) {
		t.Errorf("page 1 = %v, want %v", articleIDs(first), wantFirst)
	}
	if !reflect.DeepEqual(articleIDs(second), wantSecond) {
		t.Errorf("page 2 = %v, want %v — a cursor is anchored to a row, so inserts at the head cannot shift it",
			articleIDs(second), wantSecond)
	}
}

// TestOffsetPagesRepeatRowsUnderInserts documents the bug the keyset version
// does not have. It asserts the *broken* behaviour on purpose: OFFSET counts
// rows, so rows inserted at the head push the window back over page 1.
func TestOffsetPagesRepeatRowsUnderInserts(t *testing.T) {
	st, clk := newTestStore(t)
	authors := seedAuthors(t, st, 1)
	seedArticles(t, st, clk, authors, 6, time.Second)

	first, err := st.ListArticlesOffset(context.Background(), 3, 0)
	if err != nil {
		t.Fatalf("ListArticlesOffset: %v", err)
	}
	seedArticles(t, st, clk, authors, 3, time.Second)
	second, err := st.ListArticlesOffset(context.Background(), 3, 3)
	if err != nil {
		t.Fatalf("ListArticlesOffset: %v", err)
	}

	if !reflect.DeepEqual(articleIDs(first), articleIDs(second)) {
		t.Errorf("expected OFFSET page 2 (%v) to repeat page 1 (%v) after three inserts; "+
			"if this test fails the demonstration is wrong, not your code",
			articleIDs(second), articleIDs(first))
	}
}
