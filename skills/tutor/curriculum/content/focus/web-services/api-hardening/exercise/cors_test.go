package harden

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testPolicy() CORSPolicy {
	return CORSPolicy{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Retry-After", "ETag"},
		AllowCredentials: true,
		MaxAge:           10 * time.Minute,
	}
}

func mustCORS(t *testing.T, p CORSPolicy) Middleware {
	t.Helper()
	mw, err := NewCORS(p)
	if err != nil {
		t.Fatalf("NewCORS(%+v) = %v, want a middleware", p, err)
	}
	return mw
}

// The bug that hands every website on the internet a logged-in session.
func TestNewCORSRejectsWildcardWithCredentials(t *testing.T) {
	_, err := NewCORS(CORSPolicy{AllowedOrigins: []string{"*"}, AllowCredentials: true})
	if !errors.Is(err, ErrCredentialedWildcard) {
		t.Fatalf("NewCORS = %v, want ErrCredentialedWildcard: refuse the combination at startup", err)
	}
	if _, err := NewCORS(CORSPolicy{AllowedOrigins: []string{"*"}}); err != nil {
		t.Errorf("a wildcard without credentials is legitimate for a public read-only API, got %v", err)
	}
}

func TestCORSAllowsAConfiguredOrigin(t *testing.T) {
	h := mustCORS(t, testPolicy())(okHandler(http.StatusOK, "data"))

	r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	r.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the origin echoed back exactly", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
	// A page reads only the seven safelisted response headers unless the
	// response names more, so an unexposed Retry-After might as well be absent.
	if got := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(got, "Retry-After") {
		t.Errorf("Access-Control-Expose-Headers = %q, want it to include Retry-After", got)
	}
	if rec.Body.String() != "data" {
		t.Errorf("body = %q, want the handler's response", rec.Body.String())
	}
}

// Without Vary: Origin a shared cache can hand one origin's headers to
// another, turning a correct policy into an incorrect one at the cache.
func TestCORSAlwaysVariesOnOrigin(t *testing.T) {
	h := mustCORS(t, testPolicy())(okHandler(http.StatusOK, "data"))

	for _, origin := range []string{"", "https://app.example.com", "https://evil.example"} {
		r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)

		if !strings.Contains(rec.Header().Get("Vary"), "Origin") {
			t.Errorf("Origin %q: Vary = %q, want it to include Origin", origin, rec.Header().Get("Vary"))
		}
	}
}

// CORS is a browser rule, not access control: an unlisted origin still reaches
// the handler, it just cannot read the answer. Authentication is what stops
// the request.
func TestCORSWithholdsHeadersFromAnUnlistedOrigin(t *testing.T) {
	served := false
	h := mustCORS(t, testPolicy())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	}))

	r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	r.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent for an unlisted origin", got)
	}
	if !served {
		t.Error("the handler did not run: CORS withholds headers, it does not block requests")
	}
}

func TestCORSWithoutAnOriginHeaderIsUntouched(t *testing.T) {
	h := mustCORS(t, testPolicy())(okHandler(http.StatusOK, "data"))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent: this is not a cross-origin request", got)
	}
	if rec.Code != http.StatusOK || rec.Body.String() != "data" {
		t.Errorf("got %d %q, want 200 %q", rec.Code, rec.Body.String(), "data")
	}
}

func preflight(t *testing.T, h http.Handler, origin, method string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodOptions, "/tasks", nil)
	r.Header.Set("Origin", origin)
	r.Header.Set("Access-Control-Request-Method", method)
	r.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestCORSAnswersAnAllowedPreflight(t *testing.T) {
	h := mustCORS(t, testPolicy())(unreachable(t))

	rec := preflight(t, h, "https://app.example.com", http.MethodPost)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: a preflight asks permission, it does not do work", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Errorf("Access-Control-Allow-Methods = %q, want it to include POST", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
		t.Errorf("Access-Control-Allow-Headers = %q, want it to include Content-Type", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Errorf("Access-Control-Max-Age = %q, want %q (whole seconds)", got, "600")
	}
}

func TestCORSRefusesAPreflightFromAnUnlistedOrigin(t *testing.T) {
	h := mustCORS(t, testPolicy())(unreachable(t))

	rec := preflight(t, h, "https://evil.example", http.MethodPost)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("Access-Control-Allow-Methods = %q, want it absent", got)
	}
}

func TestCORSRefusesAPreflightForAnUnlistedMethod(t *testing.T) {
	h := mustCORS(t, testPolicy())(unreachable(t))

	rec := preflight(t, h, "https://app.example.com", http.MethodDelete)

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("Access-Control-Allow-Methods = %q, want it absent: DELETE is not in the policy", got)
	}
}

func TestCORSWildcardWithoutCredentials(t *testing.T) {
	h := mustCORS(t, CORSPolicy{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet},
	})(okHandler(http.StatusOK, "public"))

	r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	r.Header.Set("Origin", "https://anywhere.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want it absent", got)
	}
}
