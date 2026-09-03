package main

import (
	"fmt"
	"io"
	"net/http"
)

// NewMux routes the service's endpoints using Go 1.22 method-and-wildcard
// patterns, e.g. "GET /greet/{name}".
func NewMux(version string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", homeHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.Handle("GET /version", versionHandler{version: version})
	mux.HandleFunc("GET /greet/{name}", greetHandler)
	return mux
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "greeter service")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "ok")
}

// versionHandler reports the build version. It is a struct — the version
// travels with the handler — so it implements http.Handler directly.
type versionHandler struct {
	version string
}

func (h versionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, h.version)
}

func greetHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.PathValue("name"))
}
