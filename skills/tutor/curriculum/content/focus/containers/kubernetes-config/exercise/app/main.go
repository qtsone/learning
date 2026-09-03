// timesvc is the service you have been containerizing all pack, with the
// endpoints a scheduler needs. It is provided complete: this lesson grades your
// manifests, not your Go. Read it to learn the contract your probes and your
// configuration have to match.
//
//	GET /healthz  liveness  — process-local. 200 as long as this process can
//	                          still serve. It touches nothing outside the
//	                          process: no database, no upstream, no disk.
//	GET /readyz   readiness — 503 while this pod must not receive traffic:
//	                          during warm-up and when a dependency it needs is
//	                          unusable.
//	GET /now      the work  — requires the X-API-Key header to match API_KEY.
//
// Configuration comes from the environment (PORT, LOG_LEVEL, NOW_CACHE_TTL,
// API_KEY, CONFIG_DIR) and from a file, $CONFIG_DIR/features.yaml, which is
// re-read on every request so that editing it does not require a restart.
package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
)

type config struct {
	addr     string
	level    slog.Level
	cacheTTL time.Duration
	apiKey   string
	confDir  string
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func loadConfig() config {
	c := config{
		addr:     ":" + env("PORT", "8080"),
		level:    slog.LevelInfo,
		cacheTTL: 30 * time.Second,
		apiKey:   os.Getenv("API_KEY"),
		confDir:  env("CONFIG_DIR", "/etc/timesvc/config"),
	}
	if err := c.level.UnmarshalText([]byte(env("LOG_LEVEL", "info"))); err != nil {
		c.level = slog.LevelInfo
	}
	if d, err := time.ParseDuration(env("NOW_CACHE_TTL", "30s")); err == nil {
		c.cacheTTL = d
	}
	return c
}

type server struct {
	cfg   config
	log   *slog.Logger
	ready atomic.Bool
}

func (s *server) healthz(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "ok\n")
}

func (s *server) readyz(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		http.Error(w, "warming up", http.StatusServiceUnavailable)
		return
	}
	// A real service also checks the things it cannot serve without here —
	// a database ping with a short timeout, a cache connection. Never put
	// those checks behind /healthz.
	io.WriteString(w, "ready\n")
}

func (s *server) now(w http.ResponseWriter, r *http.Request) {
	if s.cfg.apiKey == "" || r.Header.Get("X-API-Key") != s.cfg.apiKey {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	features, err := os.ReadFile(filepath.Join(s.cfg.confDir, "features.yaml"))
	if err != nil {
		s.log.Warn("features file unreadable, using defaults", "err", err)
	}
	s.log.Info("now", "zone", r.URL.Query().Get("zone"), "features", len(features))
	// NOW_CACHE_TTL is a real knob: it is the freshness this endpoint promises
	// its callers, so a ConfigMap change actually changes what clients see.
	w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(int(s.cfg.cacheTTL.Seconds())))
	io.WriteString(w, time.Now().UTC().Format(time.RFC3339)+"\n")
}

func main() {
	cfg := loadConfig()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.level}))
	s := &server{cfg: cfg, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.HandleFunc("GET /now", s.now)

	// Warm-up: the pod is alive immediately but must not be sent traffic until
	// this finishes. That gap is exactly what readiness exists to express.
	go func() {
		time.Sleep(2 * time.Second)
		s.ready.Store(true)
		log.Info("ready", "addr", cfg.addr, "cache_ttl", cfg.cacheTTL.String())
	}()

	log.Info("listening", "addr", cfg.addr, "config_dir", cfg.confDir)
	// Reacting to SIGTERM is the next lesson's subject; this one stops here.
	if err := http.ListenAndServe(cfg.addr, mux); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
