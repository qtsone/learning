package apiperf

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

// TestFeedRoundTripsDoNotGrowWithThePage is the N+1 detector. A handler that
// asks for one author per article scales its database round trips with the
// page size; a batched one does not. The number is the assertion, not a
// stopwatch.
func TestFeedRoundTripsDoNotGrowWithThePage(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 5)
	seedArticles(t, st, clk, authors, 60, time.Second)

	for _, limit := range []int{5, 20, 50} {
		st.ResetQueries()
		p, err := svc.Feed(context.Background(), limit, "")
		if err != nil {
			t.Fatalf("Feed(limit=%d): %v", limit, err)
		}
		if len(p.Items) != limit {
			t.Fatalf("Feed(limit=%d) returned %d items, want %d", limit, len(p.Items), limit)
		}
		if n := st.Queries(); n != 2 {
			t.Errorf("Feed(limit=%d) made %d database round trips, want 2 (articles, then their authors in one batch)", limit, n)
		}
	}
}

func TestFeedOnAnEmptyFeed(t *testing.T) {
	svc, st, _ := newTestService(t)

	st.ResetQueries()
	p, err := svc.Feed(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if n := st.Queries(); n != 1 {
		t.Errorf("Feed on an empty feed made %d round trips, want 1 — no articles means no authors to batch", n)
	}
	if p.NextCursor != "" {
		t.Errorf("NextCursor = %q on an empty page, want \"\"", p.NextCursor)
	}
	body, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if want := `{"items":[],"next_cursor":""}`; string(body) != want {
		t.Errorf("empty page marshals to %s, want %s — Items must be an empty slice, not nil", body, want)
	}
}

func TestFeedJoinsAuthors(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 2)
	articles := seedArticles(t, st, clk, authors, 4, time.Second)

	p, err := svc.Feed(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(p.Items) != 4 {
		t.Fatalf("got %d items, want 4", len(p.Items))
	}
	byID := make(map[int64]Author, len(authors))
	for _, a := range authors {
		byID[a.ID] = a
	}
	for i, item := range p.Items {
		article := articles[3-i] // newest first
		want := byID[article.AuthorID]
		if item.Author != want {
			t.Errorf("item %d (article %d): author = %+v, want %+v", i, item.ID, item.Author, want)
		}
		if item.Title != article.Title {
			t.Errorf("item %d: title = %q, want %q", i, item.Title, article.Title)
		}
	}
}

func TestFeedPagesThroughEverythingExactlyOnce(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 3)
	articles := seedArticles(t, st, clk, authors, 13, time.Second)

	var seen []int64
	cursor := ""
	for pages := 0; ; pages++ {
		if pages > 10 {
			t.Fatal("still paging after 10 requests over 13 articles — NextCursor is not advancing")
		}
		p, err := svc.Feed(context.Background(), 5, cursor)
		if err != nil {
			t.Fatalf("Feed(cursor=%q): %v", cursor, err)
		}
		seen = append(seen, itemIDs(p.Items)...)
		if p.NextCursor == "" {
			if len(p.Items) != 3 {
				t.Errorf("last page held %d items, want 3", len(p.Items))
			}
			break
		}
		cursor = p.NextCursor
	}

	want := make([]int64, 0, len(articles))
	for i := len(articles) - 1; i >= 0; i-- {
		want = append(want, articles[i].ID)
	}
	if !reflect.DeepEqual(seen, want) {
		t.Errorf("paging saw %v, want %v (every article once, newest first)", seen, want)
	}
}

func TestFeedLastFullPageHasNoCursor(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 1)
	seedArticles(t, st, clk, authors, 4, time.Second)

	p, err := svc.Feed(context.Background(), 4, "")
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(p.Items) != 4 {
		t.Fatalf("got %d items, want 4", len(p.Items))
	}
	if p.NextCursor != "" {
		t.Error("a full page that happens to be the last one must not hand out a cursor; " +
			"ask the store for limit+1 rows so you know whether there is more")
	}
}

func TestFeedRejectsBadCursor(t *testing.T) {
	svc, _, _ := newTestService(t)
	if _, err := svc.Feed(context.Background(), 10, "definitely-not-a-cursor"); !errors.Is(err, ErrBadCursor) {
		t.Errorf("Feed with a garbage cursor returned %v, want an error matching ErrBadCursor", err)
	}
}

func TestFeedJSONServesHitsWithoutTouchingTheDatabase(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 2)
	seedArticles(t, st, clk, authors, 5, time.Second)

	first, err := svc.FeedJSON(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("FeedJSON: %v", err)
	}
	st.ResetQueries()
	second, err := svc.FeedJSON(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("FeedJSON: %v", err)
	}

	if n := st.Queries(); n != 0 {
		t.Errorf("a cache hit made %d database round trips, want 0", n)
	}
	if string(first.Body) != string(second.Body) || first.ETag != second.ETag {
		t.Error("a cache hit must return the same bytes and the same ETag as the miss that filled it")
	}
	if hits, _ := svc.Cache.Stats(); hits != 1 {
		t.Errorf("cache hits = %d, want 1", hits)
	}
}

func TestFeedJSONRefetchesAfterTTL(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 1)
	seedArticles(t, st, clk, authors, 3, time.Second)

	if _, err := svc.FeedJSON(context.Background(), 10, ""); err != nil {
		t.Fatalf("FeedJSON: %v", err)
	}
	clk.Advance(31 * time.Second)
	st.ResetQueries()
	if _, err := svc.FeedJSON(context.Background(), 10, ""); err != nil {
		t.Fatalf("FeedJSON: %v", err)
	}
	if n := st.Queries(); n == 0 {
		t.Error("the cached page was 31s old with a 30s TTL; it must be refetched")
	}
}

func TestFeedJSONKeysPagesSeparately(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 1)
	seedArticles(t, st, clk, authors, 10, time.Second)

	firstPage, err := svc.FeedJSON(context.Background(), 4, "")
	if err != nil {
		t.Fatalf("FeedJSON: %v", err)
	}
	var p Page
	if err := json.Unmarshal(firstPage.Body, &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	secondPage, err := svc.FeedJSON(context.Background(), 4, p.NextCursor)
	if err != nil {
		t.Fatalf("FeedJSON: %v", err)
	}
	if string(firstPage.Body) == string(secondPage.Body) {
		t.Error("two different cursors returned the same cached bytes — the cache key must include the cursor")
	}

	otherLimit, err := svc.FeedJSON(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("FeedJSON: %v", err)
	}
	if string(otherLimit.Body) == string(firstPage.Body) {
		t.Error("two different limits returned the same cached bytes — the cache key must include the limit")
	}
}
