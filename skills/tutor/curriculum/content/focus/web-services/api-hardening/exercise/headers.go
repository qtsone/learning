package harden

import "net/http"

// SecurityHeaders sets the response headers that do real work for a JSON API:
//
//	X-Content-Type-Options: nosniff
//	Content-Security-Policy: default-src 'none'; frame-ancestors 'none'; base-uri 'none'
//	Referrer-Policy: no-referrer
//
// and, only when the request arrived over TLS,
//
//	Strict-Transport-Security: max-age=31536000; includeSubDomains
//
// It must NOT set X-XSS-Protection: the auditing tools that ask for it are
// out of date, no current browser implements it, and the filter it used to
// enable introduced vulnerabilities of its own.
func SecurityHeaders(next http.Handler) http.Handler {
	// TODO: set the headers above and call next. Check r.TLS for the HSTS
	// condition, and be ready to explain in review why that condition exists.
	return next
}
