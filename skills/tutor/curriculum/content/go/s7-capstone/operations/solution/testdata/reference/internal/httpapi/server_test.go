package httpapi_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tutor.local/capstone-reference/internal/httpapi"
)

func newServer() *httpapi.Server {
	return httpapi.New(slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealthzIsAlwaysUpWhileTheProcessAnswers(t *testing.T) {
	s := newServer()
	if got := get(t, s.Routes(), "/healthz").Code; got != http.StatusOK {
		t.Errorf("GET /healthz = %d, want %d", got, http.StatusOK)
	}
}

func TestReadyzFollowsTheReadinessFlag(t *testing.T) {
	s := newServer()
	h := s.Routes()

	if got := get(t, h, "/readyz").Code; got != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz before Ready() = %d, want %d", got, http.StatusServiceUnavailable)
	}
	s.Ready()
	if got := get(t, h, "/readyz").Code; got != http.StatusOK {
		t.Errorf("GET /readyz after Ready() = %d, want %d", got, http.StatusOK)
	}
	s.NotReady()
	if got := get(t, h, "/readyz").Code; got != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz after NotReady() = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestWorkRejectsOversizedIDs(t *testing.T) {
	h := newServer().Routes()
	if got := get(t, h, "/work/"+strings.Repeat("x", 65)).Code; got != http.StatusBadRequest {
		t.Errorf("GET /work/<65 chars> = %d, want %d", got, http.StatusBadRequest)
	}
	if got := get(t, h, "/work/42").Code; got != http.StatusOK {
		t.Errorf("GET /work/42 = %d, want %d", got, http.StatusOK)
	}
}

func TestMetricsAreExposed(t *testing.T) {
	rec := get(t, newServer().Routes(), "/metrics")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "http_requests_total") {
		t.Errorf("GET /metrics body does not publish http_requests_total:\n%s", rec.Body.String())
	}
}
