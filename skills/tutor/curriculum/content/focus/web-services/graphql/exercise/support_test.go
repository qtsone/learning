package gql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

// The queries the HTTP tests share. Each one fits the default limits below;
// the queries that do not are written out where they are used.
const (
	qPostsWithAuthors = `{ posts(first: 10) { title author { name } } }`
	qDeepTree         = `{ posts(first: 10) { title author { name } comments(first: 10) { body author { name } } } }`
	qAuthorPosts      = `{ posts(first: 10) { author { name posts(first: 10) { title } } } }`
)

func testLimits() Limits { return Limits{MaxDepth: 5, MaxComplexity: 1000} }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// startServer boots the endpoint over httptest — no sockets you own, no ports
// to pick, and it dies with the test.
func startServer(t *testing.T, store *Store, schema *Schema) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(NewHandler(Config{
		Schema: schema,
		Store:  store,
		Limits: testLimits(),
		Logger: discardLogger(),
	}))
	t.Cleanup(ts.Close)
	return ts
}

// batchedServer is the usual setup: fixture store, batched schema, default
// limits. It returns the store so a test can count what the query cost.
func batchedServer(t *testing.T) (*httptest.Server, *Store) {
	t.Helper()
	store := NewStore()
	return startServer(t, store, NewSchema(store)), store
}

func postQuery(t *testing.T, ts *httptest.Server, query string) (int, map[string]any) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return postRaw(t, ts, "application/json", body)
}

func postRaw(t *testing.T, ts *httptest.Server, contentType string, body []byte) (int, map[string]any) {
	t.Helper()
	resp, err := ts.Client().Post(ts.URL+"/graphql", contentType, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, out
}

// at walks a decoded response: at(t, body, "data", "posts", 4, "author").
func at(t *testing.T, v any, path ...any) any {
	t.Helper()
	for i, step := range path {
		switch s := step.(type) {
		case string:
			m, ok := v.(map[string]any)
			if !ok {
				t.Fatalf("%v: expected an object, got %T", path[:i+1], v)
			}
			v = m[s]
		case int:
			l, ok := v.([]any)
			if !ok {
				t.Fatalf("%v: expected a list, got %T", path[:i+1], v)
			}
			if s >= len(l) {
				t.Fatalf("%v: index %d out of range (len %d)", path[:i+1], s, len(l))
			}
			v = l[s]
		}
	}
	return v
}

type respError struct {
	Message string
	Path    string
}

// responseErrors flattens the errors array. Paths become "posts.4.author",
// which is far easier to assert on than []any{"posts", float64(4), "author"}.
func responseErrors(t *testing.T, body map[string]any) []respError {
	t.Helper()
	raw, ok := body["errors"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("errors is %T, want a list", raw)
	}
	out := make([]respError, 0, len(list))
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("error entry is %T, want an object", e)
		}
		msg, _ := m["message"].(string)
		var parts []string
		if p, ok := m["path"].([]any); ok {
			for _, step := range p {
				parts = append(parts, fmt.Sprintf("%v", step))
			}
		}
		out = append(out, respError{Message: msg, Path: strings.Join(parts, ".")})
	}
	return out
}

func mustParse(t *testing.T, s *Schema, query string) *Operation {
	t.Helper()
	op, err := Parse(query)
	if err != nil {
		t.Fatalf("parse %q: %v", query, err)
	}
	if err := Validate(s, op); err != nil {
		t.Fatalf("validate %q: %v", query, err)
	}
	return op
}

func requireStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}

func requireCallCount(t *testing.T, store *Store, method string, want int) {
	t.Helper()
	if got := store.CallCount(method); got != want {
		t.Errorf("%s called %d time(s), want %d", method, got, want)
	}
}
