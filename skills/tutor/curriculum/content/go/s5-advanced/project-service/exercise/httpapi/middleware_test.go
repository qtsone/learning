package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

// logRecords parses the JSON lines a handler logged.
func logRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

func okHandler(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		io.WriteString(w, body)
	})
}

func TestChainAppliesTheFirstMiddlewareOutermost(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-in")
				next.ServeHTTP(w, r)
				order = append(order, name+"-out")
			})
		}
	}
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), mark("A"), mark("B"))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := "A-in B-in handler B-out A-out"
	if got := strings.Join(order, " "); got != want {
		t.Errorf("order = %q, want %q", got, want)
	}
}

func TestAuthAcceptsAConfiguredTokenAndNamesTheClient(t *testing.T) {
	h := Auth(map[string]string{"tests": testToken})(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, ClientFrom(r.Context()))
		}))

	for _, header := range []string{"Bearer " + testToken, "bearer " + testToken} {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req.Header.Set("Authorization", header)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status for %q = %d, want 200 (the scheme is case-insensitive)", header, rec.Code)
		}
		if rec.Body.String() != "tests" {
			t.Errorf("handler saw client %q, want %q: put the authenticated name on the context",
				rec.Body.String(), "tests")
		}
	}
}

func TestAuthRejectsEverythingElse(t *testing.T) {
	h := Auth(map[string]string{"tests": testToken})(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t.Error("the handler ran for a request that failed authentication")
		}))

	for _, header := range []string{"", "Bearer", "Bearer wrong", "Basic " + testToken, testToken} {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status for Authorization %q = %d, want 401", header, rec.Code)
		}
	}
}

// An empty token map means "nobody is configured", which must fail closed.
func TestAuthWithNoTokensRejectsEveryone(t *testing.T) {
	h := Auth(nil)(okHandler(http.StatusOK, "reached"))

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401: an unconfigured service authenticates nobody", rec.Code)
	}
}

func TestRecoverTurnsPanicsInto500(t *testing.T) {
	logger, logs := newLogger()
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("handler exploded")
	}), RequestID, Recover(logger))

	rec := httptest.NewRecorder()
	// If the panic escapes, this call panics and the test fails loudly.
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":{"message":"internal error"}}` {
		t.Errorf("body = %s, want the standard error envelope %s", got, `{"error":{"message":"internal error"}}`)
	}
	records := logRecords(t, logs)
	if len(records) != 1 {
		t.Fatalf("logged %d lines, want 1", len(records))
	}
	if lvl, _ := records[0]["level"].(string); lvl != "ERROR" {
		t.Errorf("level = %q, want ERROR: a panic is not routine", lvl)
	}
	if id, _ := records[0]["request_id"].(string); id == "" {
		t.Error("the panic log line has no request_id; RequestID must sit outside Recover")
	}
	if !strings.Contains(logs.String(), "handler exploded") {
		t.Errorf("the panic value is missing from the log: %s", logs.String())
	}
}

func TestAccessLogWritesOneStructuredLine(t *testing.T) {
	logger, logs := newLogger()
	h := Chain(okHandler(http.StatusCreated, "made"), RequestID, AccessLog(logger))

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set(requestIDHeader, "req-42")
	req.Header.Set("Authorization", "Bearer "+testToken)
	h.ServeHTTP(httptest.NewRecorder(), req)

	records := logRecords(t, logs)
	if len(records) != 1 {
		t.Fatalf("logged %d lines, want exactly 1 per request", len(records))
	}
	rec := records[0]
	for attr, want := range map[string]any{
		"method":     "POST",
		"path":       "/tasks",
		"status":     float64(http.StatusCreated), // JSON numbers decode as float64
		"request_id": "req-42",
	} {
		if got := rec[attr]; got != want {
			t.Errorf("attribute %q = %v, want %v", attr, got, want)
		}
	}
	if _, ok := rec["duration_ms"]; !ok {
		t.Error("no duration_ms attribute: an access log without latency cannot answer the D in RED")
	}
	if strings.Contains(logs.String(), testToken) {
		t.Error("the access log contains the caller's token; credentials never go in logs")
	}
}

// A handler that writes a body without calling WriteHeader produced a 200,
// and the log has to say so.
func TestAccessLogReportsTheImplicit200(t *testing.T) {
	logger, logs := newLogger()
	h := AccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "no explicit status")
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	records := logRecords(t, logs)
	if len(records) != 1 {
		t.Fatalf("logged %d lines, want 1", len(records))
	}
	if got := records[0]["status"]; got != float64(http.StatusOK) {
		t.Errorf("status = %v, want 200", got)
	}
}

func TestTimeoutAnswers503(t *testing.T) {
	// 50ms is not a performance assertion: the handler waits for the
	// timeout to fire, however long the race detector makes that take.
	h := Timeout(50 * time.Millisecond)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done() // the timeout cancels the request context
		}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != timeoutBody {
		t.Errorf("body = %s, want %s", got, timeoutBody)
	}
}

func TestTimeoutLetsFastHandlersThrough(t *testing.T) {
	h := Timeout(10 * time.Second)(okHandler(http.StatusOK, "fast"))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "fast" {
		t.Errorf("got %d %q, want 200 %q", rec.Code, rec.Body.String(), "fast")
	}
}

// The documented chain order has to hold together: a panic deep inside must
// still produce a 500 response AND an access-log line that records it.
func TestProductionChainLogsPanicsAsFailedRequests(t *testing.T) {
	logger, logs := newLogger()
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}), RequestID, AccessLog(logger), Recover(logger), Timeout(5*time.Second))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var access map[string]any
	for _, r := range logRecords(t, logs) {
		if _, ok := r["status"]; ok {
			access = r
		}
	}
	if access == nil {
		t.Fatalf("no access-log line for a panicking request: %s", logs.String())
	}
	if got := access["status"]; got != float64(http.StatusInternalServerError) {
		t.Errorf("access log status = %v, want 500: AccessLog must sit outside Recover", got)
	}
}
