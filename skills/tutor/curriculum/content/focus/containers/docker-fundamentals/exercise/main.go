// Command clockwork is a tiny HTTP service: it answers every request with the
// current UTC time and logs each request to standard output. It exists to give
// you something real to containerize; you are not asked to change it.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := os.Getenv("CLOCKWORK_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request path=%s remote=%s", r.URL.Path, r.RemoteAddr)
		fmt.Fprintf(w, "clockwork: %s\n", time.Now().UTC().Format(time.RFC3339))
	})

	log.Printf("clockwork listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
