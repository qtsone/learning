package harden

import (
	"net/http"
	"time"
)

// HandlerTimeout is the per-request budget the Timeout middleware enforces. It
// must stay below the server's WriteTimeout, or the connection dies before the
// 503 that explains why can reach the client.
const HandlerTimeout = 10 * time.Second

// NewServer returns an http.Server with every timeout set.
//
// The four that matter, and the attack each one ends:
//
//   - ReadHeaderTimeout — a client that opens a connection and dribbles
//     header bytes forever, holding a goroutine per connection;
//   - ReadTimeout — a client that sends headers promptly and then the body one
//     byte per second;
//   - WriteTimeout — a client that stops reading, so writes block on a full
//     send buffer;
//   - IdleTimeout — keep-alive connections parked open and never reused.
//
// A zero value means "no limit", which is why the defaults are not defaults at
// all: they are an unbounded server waiting for its first bad afternoon.
func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// Larger than HandlerTimeout on purpose: TimeoutHandler needs a live
		// connection to deliver its 503.
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		// The same as the net/http default, written down so it is a decision
		// somebody made rather than one nobody noticed.
		MaxHeaderBytes: 1 << 20,
	}
}

// Options configures the edge stack.
type Options struct {
	// Limiter rate limits per key. Required.
	Limiter *Limiter
	// Key identifies the client. Nil means RemoteIP.
	Key func(*http.Request) string
	// CORS is the middleware NewCORS returned. Nil means no CORS handling.
	CORS Middleware
	// Timeout is the per-request budget. Zero means HandlerTimeout.
	Timeout time.Duration
}

// Harden wraps h with the edge stack, outermost first:
//
//	SecurityHeaders -> CORS -> RateLimit -> Timeout -> h
//
// The order is an argument, not a preference:
//
//   - SecurityHeaders is outermost so even a 429 or a 503 carries them;
//   - CORS sits outside RateLimit so a rejected cross-origin request still
//     carries Access-Control-Allow-Origin — otherwise the browser hides the
//     429 from the page and the frontend developer sees only "network error";
//   - RateLimit sits outside everything expensive, so a client over budget
//     costs a map lookup rather than a body read, a decode and a query.
func Harden(h http.Handler, opts Options) http.Handler {
	key := opts.Key
	if key == nil {
		key = RemoteIP
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = HandlerTimeout
	}

	mws := []Middleware{SecurityHeaders}
	if opts.CORS != nil {
		mws = append(mws, opts.CORS)
	}
	if opts.Limiter != nil {
		mws = append(mws, RateLimit(opts.Limiter, key))
	}
	mws = append(mws, Timeout(timeout))

	return Chain(h, mws...)
}
