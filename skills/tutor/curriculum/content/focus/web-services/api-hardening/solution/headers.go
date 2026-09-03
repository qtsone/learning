package harden

import "net/http"

// SecurityHeaders sets the response headers that do real work for a JSON API.
//
// What each one buys:
//
//   - nosniff stops a browser from ignoring the declared Content-Type and
//     guessing. An endpoint that echoes attacker-controlled text and is
//     sniffed as HTML executes it;
//   - default-src 'none' means a response of this API may load nothing, so a
//     document rendered from it cannot fetch scripts. frame-ancestors 'none'
//     means nobody may frame it, which is what X-Frame-Options used to say;
//   - no-referrer keeps ids in your URLs out of the Referer header of
//     whatever page a user visits next;
//   - HSTS makes the browser refuse plaintext to this host for a year, which
//     is only meaningful — and only honoured — over TLS.
//
// It deliberately does NOT set X-XSS-Protection: the auditing tools that ask
// for it are out of date, no current browser implements it, and the filter it
// used to enable introduced vulnerabilities of its own.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		h.Set("Referrer-Policy", "no-referrer")
		// Only over TLS the server itself terminated. Behind a TLS-terminating
		// proxy r.TLS is nil even though the client's connection was HTTPS, so
		// a real deployment either sets the header at the proxy or trusts a
		// forwarded-proto header — and trusting that header is only safe when
		// the proxy is the sole way in and overwrites it.
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
