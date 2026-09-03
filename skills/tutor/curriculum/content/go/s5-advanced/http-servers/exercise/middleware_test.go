package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChainOrder(t *testing.T) {
	var order []string
	tag := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+">")
				next.ServeHTTP(w, r)
				order = append(order, "<"+name)
			})
		}
	}
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), tag("A"), tag("B"))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	want := "A>,B>,handler,<B,<A"
	if got := strings.Join(order, ","); got != want {
		t.Errorf("execution order = %s, want %s (the first middleware listed must be outermost)", got, want)
	}
}

func TestRequestID(t *testing.T) {
	var seenByHandler string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenByHandler = RequestIDFrom(r.Context())
	}))

	t.Run("generates an id when the caller sends none", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		id := rec.Header().Get(requestIDHeader)
		if id == "" {
			t.Fatalf("response has no %s header", requestIDHeader)
		}
		if seenByHandler != id {
			t.Errorf("handler read request id %q from its context but the response carries %q — they must match", seenByHandler, id)
		}
	})

	t.Run("propagates the caller's id unchanged", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(requestIDHeader, "abc-123")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get(requestIDHeader); got != "abc-123" {
			t.Errorf("response %s = %q, want the caller's %q", requestIDHeader, got, "abc-123")
		}
		if seenByHandler != "abc-123" {
			t.Errorf("handler read request id %q from its context, want the caller's %q", seenByHandler, "abc-123")
		}
	})

	t.Run("distinct requests get distinct ids", func(t *testing.T) {
		rec1 := httptest.NewRecorder()
		h.ServeHTTP(rec1, httptest.NewRequest("GET", "/", nil))
		rec2 := httptest.NewRecorder()
		h.ServeHTTP(rec2, httptest.NewRequest("GET", "/", nil))
		a, b := rec1.Header().Get(requestIDHeader), rec2.Header().Get(requestIDHeader)
		if a == b {
			t.Errorf("two requests got the same id %q", a)
		}
	})

	t.Run("leaves the caller's request untouched", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
		if got := RequestIDFrom(req.Context()); got != "" {
			t.Errorf("the request handed to RequestID now carries id %q — give next a copy from r.WithContext instead of mutating the original", got)
		}
	})
}

// logLines runs h behind the given middleware chain and returns every JSON
// log record the logger emitted.
func logLines(t *testing.T, target string, h http.HandlerFunc, wrap func(*slog.Logger, http.Handler) http.Handler) []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	wrap(logger, h).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", target, nil))

	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log output is not JSON records: %q (%v)", buf.String(), err)
		}
		records = append(records, m)
	}
	return records
}

func TestLogging(t *testing.T) {
	only := func(t *testing.T, records []map[string]any) map[string]any {
		t.Helper()
		if len(records) != 1 {
			t.Fatalf("got %d log records, want exactly one line per request", len(records))
		}
		return records[0]
	}
	plain := func(logger *slog.Logger, h http.Handler) http.Handler { return Logging(logger)(h) }

	t.Run("logs method, path, and explicit status", func(t *testing.T) {
		m := only(t, logLines(t, "/tea", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}, plain))
		if m["method"] != "GET" {
			t.Errorf(`log attr "method" = %v, want "GET"`, m["method"])
		}
		if m["path"] != "/tea" {
			t.Errorf(`log attr "path" = %v, want "/tea"`, m["path"])
		}
		if m["status"] != float64(http.StatusTeapot) {
			t.Errorf(`log attr "status" = %v, want %d`, m["status"], http.StatusTeapot)
		}
	})

	t.Run("handler that never calls WriteHeader logs status 200", func(t *testing.T) {
		m := only(t, logLines(t, "/implicit", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "hi")
		}, plain))
		if m["status"] != float64(http.StatusOK) {
			t.Errorf(`log attr "status" = %v, want 200 (initialize statusWriter for the implicit-200 case)`, m["status"])
		}
	})

	t.Run("logs the request id when RequestID is outside it", func(t *testing.T) {
		var want string
		chained := func(logger *slog.Logger, h http.Handler) http.Handler {
			return Chain(h, RequestID, Logging(logger))
		}
		m := only(t, logLines(t, "/traced", func(w http.ResponseWriter, r *http.Request) {
			want = RequestIDFrom(r.Context())
		}, chained))
		if want == "" {
			t.Fatal("the handler saw no request id at all — fix RequestID first")
		}
		if m["request_id"] != want {
			t.Errorf(`log attr "request_id" = %v, want %q (read it from the request context)`, m["request_id"], want)
		}
	})
}

func TestRecover(t *testing.T) {
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("panic escaped the Recover middleware: %v", p)
			}
		}()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/panic", nil))
	}()
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status after panic = %d, want 500", rec.Code)
	}
}
