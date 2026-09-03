package report

import (
	"net/http"
)

// NewDebugMux returns the mux a production service would serve on its
// private diagnostics port: the full set of net/http/pprof endpoints
// under /debug/pprof/.
//
// TODO: mount the pprof handlers (index, cmdline, profile, symbol,
// trace) on this mux with Go 1.22 method+path patterns. Remember: the
// index handler owns the whole /debug/pprof/ subtree — it also serves
// the named runtime profiles (heap, goroutine, allocs, block, mutex).
func NewDebugMux() *http.ServeMux {
	mux := http.NewServeMux()
	return mux
}
