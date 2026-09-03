package report

import (
	"net/http"
	"net/http/pprof"
)

// NewDebugMux returns the mux a production service would serve on its
// private diagnostics port: the full set of net/http/pprof endpoints
// under /debug/pprof/.
//
// Index owns the subtree: any /debug/pprof/<name> it receives is looked
// up among the runtime profiles (heap, goroutine, allocs, block, mutex,
// threadcreate). The other four are separate handlers, so the exact
// patterns below win over the subtree pattern under the 1.22 mux rules.
func NewDebugMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	return mux
}
