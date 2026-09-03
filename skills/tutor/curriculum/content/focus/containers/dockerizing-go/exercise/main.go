// Command timesvc is the small HTTP service you will containerize.
//
// It is given to you finished: this lesson grades your Dockerfile, not your
// Go. Server design proper comes later in the roadmap — read this as a
// worked example, change nothing, and spend your time on the image.
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	// Go does not embed the IANA timezone database unless you ask. Importing
	// it means time.LoadLocation keeps working on a base image that ships no
	// /usr/share/zoneinfo — which is every image in this lesson worth using.
	_ "time/tzdata"
)

const defaultPort = "8080"

type nowResponse struct {
	Service string `json:"service"`
	Zone    string `json:"zone"`
	Now     string `json:"now"`
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func now(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "UTC"
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		http.Error(w, "unknown time zone: "+zone, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nowResponse{
		Service: "timesvc",
		Zone:    zone,
		Now:     time.Now().In(loc).Format(time.RFC3339),
	})
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /now", now)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Info("timesvc listening", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
