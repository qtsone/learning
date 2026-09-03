package apiperf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock is the only clock this suite uses. Nothing here sleeps and nothing
// asserts how long anything took: every time-dependent rule in the exercise —
// cache expiry, created_at ordering — is a comparison against Clock.Now(), so
// a test moves time instead of waiting for it.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// newTestStore opens a fresh SQLite database in the test's temp directory.
func newTestStore(t testing.TB) (*Store, *fakeClock) {
	t.Helper()
	clk := newFakeClock()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "apiperf.db"), 4)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewStore(db, clk), clk
}

// newTestService wires a service with an 8-entry cache and a 30s TTL.
func newTestService(t testing.TB) (*Service, *Store, *fakeClock) {
	t.Helper()
	st, clk := newTestStore(t)
	return NewService(st, 8, 30*time.Second, clk), st, clk
}

func seedAuthors(t testing.TB, st *Store, n int) []Author {
	t.Helper()
	out := make([]Author, 0, n)
	for i := 1; i <= n; i++ {
		a, err := st.CreateAuthor(context.Background(), fmt.Sprintf("author-%d", i))
		if err != nil {
			t.Fatalf("CreateAuthor: %v", err)
		}
		out = append(out, a)
	}
	return out
}

// seedArticles creates n articles round-robin across authors, advancing the
// clock by step between each so created_at is strictly increasing (step 0
// makes every article a tie, which the keyset tests use on purpose).
// The result is in creation order: oldest first, newest last.
func seedArticles(t testing.TB, st *Store, clk *fakeClock, authors []Author, n int, step time.Duration) []Article {
	t.Helper()
	out := make([]Article, 0, n)
	for i := 1; i <= n; i++ {
		author := authors[(i-1)%len(authors)]
		a, err := st.CreateArticle(context.Background(), author.ID,
			fmt.Sprintf("article-%d", i), strings.Repeat("body ", 20))
		if err != nil {
			t.Fatalf("CreateArticle: %v", err)
		}
		out = append(out, a)
		clk.Advance(step)
	}
	return out
}

func articleIDs(articles []Article) []int64 {
	out := make([]int64, 0, len(articles))
	for _, a := range articles {
		out = append(out, a.ID)
	}
	return out
}

func itemIDs(items []FeedItem) []int64 {
	out := make([]int64, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

// request drives the handler through httptest. No socket, no network, no
// server: the whole suite is in-process, which is what makes it fast enough to
// run on every save.
func request(t testing.TB, h http.Handler, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func decodePage(t testing.TB, rec *httptest.ResponseRecorder) Page {
	t.Helper()
	var p Page
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("response body is not a Page: %v\nbody: %s", err, rec.Body.String())
	}
	return p
}
