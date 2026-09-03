package main

import "net/http"

// NewMux routes the service's endpoints using Go 1.22 method-and-wildcard
// patterns, e.g. "GET /greet/{name}".
func NewMux(version string) *http.ServeMux {
	mux := http.NewServeMux()
	// TODO: register the four routes from the acceptance criteria:
	//   GET /{$}          -> homeHandler   (only "/" itself, not every path)
	//   GET /health       -> healthHandler
	//   GET /version      -> versionHandler{version: version}
	//   GET /greet/{name} -> greetHandler
	return mux
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: respond 200 with the body "greeter service".
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: respond 200 with the body "ok".
}

// versionHandler reports the build version. It is a struct — the version
// travels with the handler — so it implements http.Handler directly.
type versionHandler struct {
	version string
}

// TODO: implement ServeHTTP on versionHandler: respond 200 with h.version.

func greetHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: respond 200 with "Hello, <name>!" where <name> is the {name}
	// path segment. r.PathValue is how you read a wildcard.
}
