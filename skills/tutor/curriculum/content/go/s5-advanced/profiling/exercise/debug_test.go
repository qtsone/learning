package report

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The CPU endpoint (/debug/pprof/profile) is deliberately not requested
// here: it blocks for its ?seconds= duration. Mount it anyway — the
// index test proves the subtree wiring, and cmdline proves you
// registered the non-index handlers.
func TestDebugMuxServesPprof(t *testing.T) {
	mux := NewDebugMux()
	for _, path := range []string{
		"/debug/pprof/",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine",
		"/debug/pprof/cmdline",
	} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d — is the pprof handler mounted on this mux?",
					path, rec.Code, http.StatusOK)
			}
			if rec.Body.Len() == 0 {
				t.Errorf("GET %s returned an empty body", path)
			}
		})
	}
}

func TestDebugMuxIndexListsProfiles(t *testing.T) {
	mux := NewDebugMux()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	body := rec.Body.String()
	for _, name := range []string{"heap", "goroutine", "block", "mutex"} {
		if !strings.Contains(body, name) {
			t.Errorf("index page does not mention the %q profile — expected the net/http/pprof index", name)
		}
	}
}
