// Command app is the tiny HTTP service you wire up with Compose in this
// lesson. It is provided for you — the lesson is about compose.yaml, not about
// server design, which a later stage covers properly.
//
// It reads two environment variables:
//
//	PORT          the port to listen on (default 8080)
//	DATABASE_URL  postgres://user:password@host:port/dbname
//
// It never speaks SQL: /dbping opens a TCP connection to the host and port in
// DATABASE_URL and reports whether it answered. That is enough to prove two
// things you are about to learn — that a service name resolves to the right
// container, and that your startup ordering works.
package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbAddr, err := databaseAddr(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error("bad DATABASE_URL", "err", err)
		os.Exit(1)
	}
	addr := ":" + env("PORT", "8080")
	log.Info("starting", "listen", addr, "database", dbAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /dbping", func(w http.ResponseWriter, r *http.Request) {
		conn, err := net.DialTimeout("tcp", dbAddr, 2*time.Second)
		if err != nil {
			log.Error("database unreachable", "database", dbAddr, "err", err)
			http.Error(w, "database unreachable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer conn.Close()
		fmt.Fprintf(w, "reached %s\n", dbAddr)
	})

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// databaseAddr reduces a postgres:// URL to the host:port the app dials.
func databaseAddr(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("DATABASE_URL is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", raw, err)
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("no host in %q", raw)
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	return net.JoinHostPort(host, port), nil
}
