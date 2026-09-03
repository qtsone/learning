package harden

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersSetsWhatMattersForJSON(t *testing.T) {
	h := SecurityHeaders(okHandler(http.StatusOK, `{"ok":true}`))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q: without it a browser may sniff JSON as HTML", got, "nosniff")
	}
	csp := rec.Header().Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("Content-Security-Policy = %q, want it to contain %q", csp, want)
		}
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want %q", got, "no-referrer")
	}
	if rec.Body.String() != `{"ok":true}` {
		t.Errorf("body = %q, want the handler's response", rec.Body.String())
	}
}

// HSTS is a promise about TLS. A browser ignores it on a plaintext response,
// so sending it there is noise that hides the real question: is this
// connection encrypted?
func TestSecurityHeadersSendsHSTSOnlyOverTLS(t *testing.T) {
	h := SecurityHeaders(okHandler(http.StatusOK, "ok"))

	plain := httptest.NewRecorder()
	h.ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/tasks", nil))
	if got := plain.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS on a plaintext request = %q, want it absent", got)
	}

	r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	r.TLS = &tls.ConnectionState{}
	secure := httptest.NewRecorder()
	h.ServeHTTP(secure, r)
	hsts := secure.Header().Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "max-age=") {
		t.Errorf("HSTS over TLS = %q, want a max-age directive", hsts)
	}
}

// Cargo cult is a cost: every header you send is bytes on every response and a
// claim you have to defend in the next audit.
func TestSecurityHeadersOmitsTheDeadOnes(t *testing.T) {
	h := SecurityHeaders(okHandler(http.StatusOK, "ok"))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if got := rec.Header().Get("X-XSS-Protection"); got != "" {
		t.Errorf("X-XSS-Protection = %q, want it absent: no current browser implements it and the filter caused bugs of its own", got)
	}
}
