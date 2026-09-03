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
	// TODO: return a server carrying addr and h with all four timeouts set,
	// ReadHeaderTimeout no larger than ReadTimeout, and WriteTimeout larger
	// than HandlerTimeout. Set MaxHeaderBytes too, and know its default.
	return &http.Server{Addr: addr, Handler: h}
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
	// TODO: build the chain above with Chain, applying the documented
	// defaults for Key and Timeout and skipping a nil CORS.
	return h
}
