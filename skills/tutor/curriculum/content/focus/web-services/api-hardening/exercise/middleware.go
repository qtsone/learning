package harden

import (
	"net"
	"net/http"
	"time"
)

// Middleware wraps a handler with behavior that runs around it. Same shape as
// the chains you built in S5.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so the FIRST one listed is outermost:
// Chain(h, A, B) serves requests as A(B(h)).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// Clock is the only way this package reads the current time. Production passes
// RealClock; tests pass a clock they move by hand, so a rate limiter's
// behavior is a function of its inputs instead of a function of how busy the
// machine was.
type Clock interface {
	Now() time.Time
}

// RealClock is the Clock backed by the operating system.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// RemoteIP returns the peer address without its port. It deliberately ignores
// X-Forwarded-For: that header is client-controlled, so trusting it turns
// per-client rate limiting into no rate limiting at all. Behind a proxy you
// control, parse it in a middleware that knows which hops are trusted.
func RemoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// timeoutBody is what a client gets when Timeout fires: the same envelope as
// every other error, so a client only has to know one shape.
const timeoutBody = `{"error":{"message":"request timeout"}}`

// Timeout bounds how long a handler may take, as in S5. TimeoutHandler runs
// the handler on its own goroutine with a deadline on the request context and
// buffers its writes; it answers the client, it does not kill the goroutine.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, timeoutBody)
	}
}
