package apiperf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListArticlesAnswersWithETagAndCacheControl(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 2)
	seedArticles(t, st, clk, authors, 5, time.Second)
	h := svc.Routes()

	rec := request(t, h, http.MethodGet, "/articles", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	etag := rec.Header().Get("ETag")
	if !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) || len(etag) != 66 {
		t.Errorf("ETag = %q, want a quoted sha256 hex digest", etag)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=30" {
		t.Errorf("Cache-Control = %q, want %q", cc, "public, max-age=30")
	}
	if got := len(decodePage(t, rec).Items); got != 5 {
		t.Errorf("got %d items, want 5", got)
	}
}

func TestConditionalGetReturns304(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 1)
	seedArticles(t, st, clk, authors, 3, time.Second)
	h := svc.Routes()

	first := request(t, h, http.MethodGet, "/articles", "", nil)
	etag := first.Header().Get("ETag")

	for _, header := range []string{etag, `W/` + etag, "*", `"other", ` + etag} {
		rec := request(t, h, http.MethodGet, "/articles", "", map[string]string{"If-None-Match": header})
		if rec.Code != http.StatusNotModified {
			t.Errorf("If-None-Match: %s → status %d, want 304", header, rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("If-None-Match: %s → %d bytes of body, want none", header, rec.Body.Len())
		}
		if got := rec.Header().Get("ETag"); got != etag {
			t.Errorf("If-None-Match: %s → ETag %q on the 304, want %q", header, got, etag)
		}
		if got := rec.Header().Get("Cache-Control"); got != "public, max-age=30" {
			t.Errorf("If-None-Match: %s → Cache-Control %q on the 304, want it set", header, got)
		}
	}
}

func TestStaleETagStillGetsTheBody(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 1)
	seedArticles(t, st, clk, authors, 2, time.Second)
	h := svc.Routes()

	rec := request(t, h, http.MethodGet, "/articles", "", map[string]string{"If-None-Match": `"stale"`})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(decodePage(t, rec).Items) != 2 {
		t.Error("a non-matching If-None-Match must be served the full body")
	}
}

func TestRepeatedRequestIsServedFromTheCache(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 2)
	seedArticles(t, st, clk, authors, 6, time.Second)
	h := svc.Routes()

	request(t, h, http.MethodGet, "/articles?limit=3", "", nil)
	st.ResetQueries()
	rec := request(t, h, http.MethodGet, "/articles?limit=3", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if n := st.Queries(); n != 0 {
		t.Errorf("the second identical request made %d database round trips, want 0", n)
	}
}

func TestCreateArticleInvalidatesTheCache(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 1)
	seedArticles(t, st, clk, authors, 3, time.Second)
	h := svc.Routes()

	before := request(t, h, http.MethodGet, "/articles", "", nil)
	etag := before.Header().Get("ETag")

	body := fmt.Sprintf(`{"author_id":%d,"title":"brand new","body":"hello"}`, authors[0].ID)
	created := request(t, h, http.MethodPost, "/articles", body, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("POST /articles status = %d, want 201 (body: %s)", created.Code, created.Body.String())
	}
	var article Article
	if err := json.Unmarshal(created.Body.Bytes(), &article); err != nil {
		t.Fatalf("POST response is not an Article: %v", err)
	}
	if article.ID == 0 || article.Title != "brand new" {
		t.Errorf("POST returned %+v, want the created article", article)
	}

	after := request(t, h, http.MethodGet, "/articles", "", map[string]string{"If-None-Match": etag})
	if after.Code != http.StatusOK {
		t.Fatalf("status = %d after a write, want 200 — a client holding the old ETag must not get a 304", after.Code)
	}
	items := decodePage(t, after).Items
	if len(items) != 4 {
		t.Fatalf("feed after the write held %d items, want 4", len(items))
	}
	if items[0].Title != "brand new" {
		t.Errorf("newest item is %q, want the article just created", items[0].Title)
	}
	if after.Header().Get("ETag") == etag {
		t.Error("the ETag must change when the body does")
	}
}

func TestCreateArticleRejectsBadInput(t *testing.T) {
	svc, st, _ := newTestService(t)
	authors := seedAuthors(t, st, 1)
	h := svc.Routes()

	cases := []struct {
		name string
		body string
		want int
	}{
		{"blank title", fmt.Sprintf(`{"author_id":%d,"title":"  ","body":"x"}`, authors[0].ID), http.StatusBadRequest},
		{"unknown author", `{"author_id":4242,"title":"t","body":"x"}`, http.StatusBadRequest},
		{"unknown field", fmt.Sprintf(`{"author_id":%d,"title":"t","body":"x","oops":1}`, authors[0].ID), http.StatusBadRequest},
		{"malformed", `{`, http.StatusBadRequest},
		{"oversized", `{"title":"` + strings.Repeat("a", 70<<10) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := request(t, h, http.MethodPost, "/articles", tc.body, nil)
			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestBadCursorIsBadRequest(t *testing.T) {
	svc, _, _ := newTestService(t)
	rec := request(t, svc.Routes(), http.MethodGet, "/articles?cursor=nonsense", "", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 — an unreadable cursor is the caller's mistake, not a 500", rec.Code)
	}
}

func TestLimitParameter(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 2)
	seedArticles(t, st, clk, authors, MaxLimit+10, time.Second)
	h := svc.Routes()

	for _, bad := range []string{"abc", "0", "-3"} {
		rec := request(t, h, http.MethodGet, "/articles?limit="+bad, "", nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("?limit=%s → status %d, want 400", bad, rec.Code)
		}
	}

	rec := request(t, h, http.MethodGet, "/articles?limit=100000", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("?limit=100000 → status %d, want 200", rec.Code)
	}
	if got := len(decodePage(t, rec).Items); got != MaxLimit {
		t.Errorf("?limit=100000 returned %d items, want it clamped to %d", got, MaxLimit)
	}

	rec = request(t, h, http.MethodGet, "/articles", "", nil)
	if got := len(decodePage(t, rec).Items); got != DefaultLimit {
		t.Errorf("no ?limit returned %d items, want the default %d", got, DefaultLimit)
	}
}

func TestCursorParameterPagesTheEndpoint(t *testing.T) {
	svc, st, clk := newTestService(t)
	authors := seedAuthors(t, st, 2)
	seedArticles(t, st, clk, authors, 7, time.Second)
	h := svc.Routes()

	first := decodePage(t, request(t, h, http.MethodGet, "/articles?limit=4", "", nil))
	if first.NextCursor == "" {
		t.Fatal("page 1 of 7 articles at limit 4 must carry a cursor")
	}
	second := decodePage(t, request(t, h, http.MethodGet, "/articles?limit=4&cursor="+first.NextCursor, "", nil))

	if len(second.Items) != 3 || second.NextCursor != "" {
		t.Fatalf("page 2 = %d items, cursor %q; want 3 items and no cursor", len(second.Items), second.NextCursor)
	}
	seen := map[int64]bool{}
	for _, id := range append(itemIDs(first.Items), itemIDs(second.Items)...) {
		if seen[id] {
			t.Errorf("article %d appeared on both pages", id)
		}
		seen[id] = true
	}
	if len(seen) != 7 {
		t.Errorf("the two pages covered %d articles, want 7", len(seen))
	}
}
