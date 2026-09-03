// Package config loads the service configuration from the environment and
// validates it at startup. Every variable read here is documented in
// README.md; a knob nobody can find is a knob nobody can turn at 3am.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"
)

// Defaults, chosen so the service starts with no configuration at all.
const (
	defaultAddr            = ":8080"
	defaultShutdownTimeout = 15 * time.Second
)

// ErrInvalidValue wraps every startup failure, so a caller can tell a bad
// deployment manifest from a bug without reading the message text.
var ErrInvalidValue = errors.New("invalid configuration value")

// Config is the whole configuration surface of the service.
type Config struct {
	Addr            string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
}

// Load reads the environment. It returns an error rather than falling back to
// a default on bad input: a typo in a deployment manifest should stop the
// rollout, not surface as strange behaviour under load an hour later.
func Load() (Config, error) {
	cfg := Config{Addr: defaultAddr, ShutdownTimeout: defaultShutdownTimeout}

	// The listen address is validated here rather than left to
	// ListenAndServe: a typo should stop the rollout while the old version is
	// still serving, not surface once the new one is already taking traffic.
	if v := strings.TrimSpace(os.Getenv("TINYSVC_ADDR")); v != "" {
		if _, _, err := net.SplitHostPort(v); err != nil {
			return Config{}, fmt.Errorf("TINYSVC_ADDR=%q: %w: want host:port", v, ErrInvalidValue)
		}
		cfg.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("TINYSVC_LOG_LEVEL")); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			return Config{}, fmt.Errorf("TINYSVC_LOG_LEVEL=%q: %w: want debug, info, warn or error", v, ErrInvalidValue)
		}
	}
	if v := strings.TrimSpace(os.Getenv("TINYSVC_SHUTDOWN_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("TINYSVC_SHUTDOWN_TIMEOUT=%q: %w: want a duration like 15s", v, ErrInvalidValue)
		}
		if d <= 0 {
			return Config{}, fmt.Errorf("TINYSVC_SHUTDOWN_TIMEOUT=%q: %w: must be positive", v, ErrInvalidValue)
		}
		cfg.ShutdownTimeout = d
	}
	return cfg, nil
}
